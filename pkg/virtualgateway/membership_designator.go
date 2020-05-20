package virtualgateway

import (
	"context"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/k8s"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

// MembershipDesignator designates VirtualGateway membership for pods and namespaced AppMesh GatewayRoute CRs.
type MembershipDesignator interface {
	// DesignateForPod will choose a VirtualGateway for given pod.
	DesignateForPod(ctx context.Context, pod *corev1.Pod) (*appmesh.VirtualGateway, error)
	// DesignateForGatewayRoute will choose a VirtualGateway for given namespaced GatewayRoute CR.
	DesignateForGatewayRoute(ctx context.Context, obj *appmesh.GatewayRoute) (*appmesh.VirtualGateway, error)
}

// NewMembershipDesignator creates new MembershipDesignator.
func NewMembershipDesignator(k8sClient client.Client) MembershipDesignator {
	return &membershipDesignator{k8sClient: k8sClient}
}

var _ MembershipDesignator = &membershipDesignator{}

// virtualGatewaySelectorDesignator designates VirtualGateway membership based on selectors on VirtualGateway.
type membershipDesignator struct {
	k8sClient client.Client
}

//// +kubebuilder:rbac:groups=appmesh.k8s.aws,resources=virtualgateways,verbs=get;list;watch

func (d *membershipDesignator) DesignateForPod(ctx context.Context, pod *corev1.Pod) (*appmesh.VirtualGateway, error) {
	vgList := appmesh.VirtualGatewayList{}
	if err := d.k8sClient.List(ctx, &vgList); err != nil {
		return nil, errors.Wrap(err, "failed to list VirtualGateways in cluster")
	}

	var vgCandidates []*appmesh.VirtualGateway
	for _, vgObj := range vgList.Items {
		selector, err := metav1.LabelSelectorAsSelector(vgObj.Spec.PodSelector)
		if err != nil {
			return nil, err
		}
		if selector.Matches(labels.Set(pod.Labels)) {
			vgCandidates = append(vgCandidates, vgObj.DeepCopy())
		}
	}
	if len(vgCandidates) == 0 {
		return nil, nil
	}
	if len(vgCandidates) > 1 {
		var vgCandidatesNames []string
		for _, vg := range vgCandidates {
			vgCandidatesNames = append(vgCandidatesNames, k8s.NamespacedName(vg).String())
		}
		return nil, errors.Errorf("found multiple matching VirtualGateways for pod %s: %s",
			k8s.NamespacedName(pod).String(), strings.Join(vgCandidatesNames, ","))
	}
	return vgCandidates[0], nil
}

// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch

func (d *membershipDesignator) DesignateForGatewayRoute(ctx context.Context, obj *appmesh.GatewayRoute) (*appmesh.VirtualGateway, error) {
	objNS := corev1.Namespace{}
	if err := d.k8sClient.Get(ctx, types.NamespacedName{Name: obj.GetNamespace()}, &objNS); err != nil {
		return nil, errors.Wrapf(err, "failed to get namespace: %s", obj.GetNamespace())
	}
	vgList := appmesh.VirtualGatewayList{}
	if err := d.k8sClient.List(ctx, &vgList); err != nil {
		return nil, errors.Wrap(err, "failed to list virtualGateways in cluster")
	}

	var vgCandidates []*appmesh.VirtualGateway
	for _, vgObj := range vgList.Items {
		selector, err := metav1.LabelSelectorAsSelector(vgObj.Spec.NamespaceSelector)
		if err != nil {
			return nil, err
		}
		if selector.Matches(labels.Set(objNS.Labels)) {
			vgCandidates = append(vgCandidates, vgObj.DeepCopy())
		}
	}
	if len(vgCandidates) == 0 {
		return nil, errors.Errorf("failed to find matching virtualGateway for namespace: %s, expecting 1 but found %d",
			obj.GetNamespace(), 0)
	}
	if len(vgCandidates) > 1 {
		var vgCandidatesNames []string
		for _, vg := range vgCandidates {
			vgCandidatesNames = append(vgCandidatesNames, vg.Name)
		}
		return nil, errors.Errorf("found multiple matching virtualGateways for namespace: %s, expecting 1 but found %d: %s",
			obj.GetNamespace(), len(vgCandidates), strings.Join(vgCandidatesNames, ","))
	}
	return vgCandidates[0], nil
}
