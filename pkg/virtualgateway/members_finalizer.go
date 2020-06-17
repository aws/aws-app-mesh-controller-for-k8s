// +build preview

package virtualgateway

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
	Finalize(ctx context.Context, vg *appmesh.VirtualGateway) error
}

//// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch;delete

func NewPendingMembersFinalizer(k8sClient client.Client, eventRecorder record.EventRecorder, log logr.Logger) MembersFinalizer {
	return &pendingMembersFinalizer{
		k8sClient:        k8sClient,
		eventRecorder:    eventRecorder,
		log:              log,
		evaluateInterval: pendingMembersFinalizerEvaluateInterval,
	}
}

// pendingMembersFinalizer is a MembersFinalizer that will pend virtualGateway deletion until all virtualGateway members are deleted.
type pendingMembersFinalizer struct {
	k8sClient     client.Client
	eventRecorder record.EventRecorder
	log           logr.Logger

	evaluateInterval time.Duration
}

func (m *pendingMembersFinalizer) Finalize(ctx context.Context, vg *appmesh.VirtualGateway) error {
	grMembers, err := m.findGatewayRouteMembers(ctx, vg)
	if err != nil {
		return err
	}
	if len(grMembers) == 0 {
		return nil
	}

	message := m.buildPendingMembersEventMessage(ctx, grMembers)
	m.eventRecorder.Eventf(vg, corev1.EventTypeWarning, "PendingMembersDeletion", message)
	return runtime.NewRequeueAfterError(errors.New("pending members deletion"), m.evaluateInterval)
}

// findGatewayRouteMembers find the GatewayRoute members for this virtualGateway.
func (m *pendingMembersFinalizer) findGatewayRouteMembers(ctx context.Context, vg *appmesh.VirtualGateway) ([]*appmesh.GatewayRoute, error) {
	grList := &appmesh.GatewayRouteList{}
	if err := m.k8sClient.List(ctx, grList); err != nil {
		return nil, err
	}
	members := make([]*appmesh.GatewayRoute, 0, len(grList.Items))
	for i := range grList.Items {
		gr := &grList.Items[i]
		if gr.Spec.VirtualGatewayRef == nil || !IsVirtualGatewayReferenced(vg, *gr.Spec.VirtualGatewayRef) {
			continue
		}
		members = append(members, gr)
	}
	return members, nil
}

func (m *pendingMembersFinalizer) buildPendingMembersEventMessage(ctx context.Context, grMembers []*appmesh.GatewayRoute) string {
	var messagePerObjectTypes []string
	if len(grMembers) != 0 {
		message := fmt.Sprintf("gatewayRoute: %v", len(grMembers))
		messagePerObjectTypes = append(messagePerObjectTypes, message)
	}

	return "objects belonging to this virtualGateway exist, please delete them to proceed. " + strings.Join(messagePerObjectTypes, ", ")
}
