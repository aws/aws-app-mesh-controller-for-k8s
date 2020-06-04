package mesh

import (
	"context"
	"fmt"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/runtime"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"time"
)

const (
	pendingMembersFinalizerEvaluateInterval = 60 * time.Second
)

type MembersFinalizer interface {
	Finalize(ctx context.Context, ms *appmesh.Mesh) error
}

// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch;delete

func NewPendingMembersFinalizer(k8sClient client.Client, eventRecorder record.EventRecorder, log logr.Logger) MembersFinalizer {
	return &pendingMembersFinalizer{
		k8sClient:        k8sClient,
		eventRecorder:    eventRecorder,
		log:              log,
		evaluateInterval: pendingMembersFinalizerEvaluateInterval,
	}
}

// pendingMembersFinalizer is a MembersFinalizer that will pend mesh deletion until all mesh members are deleted.
type pendingMembersFinalizer struct {
	k8sClient     client.Client
	eventRecorder record.EventRecorder
	log           logr.Logger

	evaluateInterval time.Duration
}

func (m *pendingMembersFinalizer) Finalize(ctx context.Context, ms *appmesh.Mesh) error {
	vsMembers, err := m.findVirtualServiceMembers(ctx, ms)
	if err != nil {
		return err
	}
	vrMembers, err := m.findVirtualRouterMembers(ctx, ms)
	if err != nil {
		return err
	}
	vnMembers, err := m.findVirtualNodeMembers(ctx, ms)
	if err != nil {
		return err
	}
	vgMembers, err := m.findVirtualGatewayMembers(ctx, ms)
	if err != nil {
		return err
	}
	if len(vsMembers) == 0 && len(vrMembers) == 0 && len(vnMembers) == 0 && len(vgMembers) == 0 {
		return nil
	}

	message := m.buildPendingMembersEventMessage(ctx, vsMembers, vrMembers, vnMembers, vgMembers)
	m.eventRecorder.Eventf(ms, corev1.EventTypeWarning, "PendingMembersDeletion", message)
	return runtime.NewRequeueAfterError(errors.New("pending members deletion"), m.evaluateInterval)
}

// findVirtualServiceMembers find the VirtualService members for this mesh.
func (m *pendingMembersFinalizer) findVirtualServiceMembers(ctx context.Context, ms *appmesh.Mesh) ([]*appmesh.VirtualService, error) {
	vsList := &appmesh.VirtualServiceList{}
	if err := m.k8sClient.List(ctx, vsList); err != nil {
		return nil, err
	}
	members := make([]*appmesh.VirtualService, 0, len(vsList.Items))
	for i := range vsList.Items {
		vs := &vsList.Items[i]
		if vs.Spec.MeshRef == nil || !IsMeshReferenced(ms, *vs.Spec.MeshRef) {
			continue
		}
		members = append(members, vs)
	}
	return members, nil
}

// findVirtualRouterMembers find the VirtualRouter members for this mesh.
func (m *pendingMembersFinalizer) findVirtualRouterMembers(ctx context.Context, ms *appmesh.Mesh) ([]*appmesh.VirtualRouter, error) {
	vrList := &appmesh.VirtualRouterList{}
	if err := m.k8sClient.List(ctx, vrList); err != nil {
		return nil, err
	}
	members := make([]*appmesh.VirtualRouter, 0, len(vrList.Items))
	for i := range vrList.Items {
		vr := &vrList.Items[i]
		if vr.Spec.MeshRef == nil || !IsMeshReferenced(ms, *vr.Spec.MeshRef) {
			continue
		}
		members = append(members, vr)
	}
	return members, nil
}

// findVirtualNodeMembers find the VirtualNode members for this mesh.
func (m *pendingMembersFinalizer) findVirtualNodeMembers(ctx context.Context, ms *appmesh.Mesh) ([]*appmesh.VirtualNode, error) {
	vnList := &appmesh.VirtualNodeList{}
	if err := m.k8sClient.List(ctx, vnList); err != nil {
		return nil, err
	}
	members := make([]*appmesh.VirtualNode, 0, len(vnList.Items))
	for i := range vnList.Items {
		vn := &vnList.Items[i]
		if vn.Spec.MeshRef == nil || !IsMeshReferenced(ms, *vn.Spec.MeshRef) {
			continue
		}
		members = append(members, vn)
	}
	return members, nil
}

// findVirtualGatewayMembers find the VirtualGateway members for this mesh.
func (m *pendingMembersFinalizer) findVirtualGatewayMembers(ctx context.Context, ms *appmesh.Mesh) ([]*appmesh.VirtualGateway, error) {
	vgList := &appmesh.VirtualGatewayList{}
	if err := m.k8sClient.List(ctx, vgList); err != nil {
		return nil, err
	}
	members := make([]*appmesh.VirtualGateway, 0, len(vgList.Items))
	for i := range vgList.Items {
		vg := &vgList.Items[i]
		if vg.Spec.MeshRef == nil || !IsMeshReferenced(ms, *vg.Spec.MeshRef) {
			continue
		}
		members = append(members, vg)
	}
	return members, nil
}

func (m *pendingMembersFinalizer) buildPendingMembersEventMessage(ctx context.Context,
	vsMembers []*appmesh.VirtualService, vrMembers []*appmesh.VirtualRouter,
	vnMembers []*appmesh.VirtualNode, vgMembers []*appmesh.VirtualGateway) string {
	var messagePerObjectTypes []string
	if len(vsMembers) != 0 {
		message := fmt.Sprintf("virtualService: %v", len(vsMembers))
		messagePerObjectTypes = append(messagePerObjectTypes, message)
	}
	if len(vrMembers) != 0 {
		message := fmt.Sprintf("virtualRouter: %v", len(vrMembers))
		messagePerObjectTypes = append(messagePerObjectTypes, message)
	}
	if len(vnMembers) != 0 {
		message := fmt.Sprintf("virtualNode: %v", len(vnMembers))
		messagePerObjectTypes = append(messagePerObjectTypes, message)
	}
	if len(vgMembers) != 0 {
		message := fmt.Sprintf("virtualGateway: %v", len(vgMembers))
		messagePerObjectTypes = append(messagePerObjectTypes, message)
	}

	return "objects belong to this mesh exists, please delete them to proceed. " + strings.Join(messagePerObjectTypes, ", ")
}
