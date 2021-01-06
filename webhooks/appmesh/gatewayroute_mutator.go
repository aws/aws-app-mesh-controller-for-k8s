package appmesh

import (
	"context"
	"fmt"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/mesh"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/virtualgateway"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/webhook"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const apiPathMutateAppMeshGatewayRoute = "/mutate-appmesh-k8s-aws-v1beta2-gatewayroute"

// NewGatewayRouteMutator returns a mutator for GatewayRoute.
func NewGatewayRouteMutator(meshMembershipDesignator mesh.MembershipDesignator, virtualGatewayMembershipDesignator virtualgateway.MembershipDesignator) *gatewayRouteMutator {
	return &gatewayRouteMutator{
		meshMembershipDesignator:           meshMembershipDesignator,
		virtualGatewayMembershipDesignator: virtualGatewayMembershipDesignator,
	}
}

var _ webhook.Mutator = &gatewayRouteMutator{}

type gatewayRouteMutator struct {
	meshMembershipDesignator           mesh.MembershipDesignator
	virtualGatewayMembershipDesignator virtualgateway.MembershipDesignator
}

func (m *gatewayRouteMutator) Prototype(req admission.Request) (runtime.Object, error) {
	return &appmesh.GatewayRoute{}, nil
}

func (m *gatewayRouteMutator) MutateCreate(ctx context.Context, obj runtime.Object) (runtime.Object, error) {
	gr := obj.(*appmesh.GatewayRoute)
	ms, err := m.designateMeshMembership(ctx, gr)
	if err != nil {
		return nil, err
	}
	vg, err := m.designateVirtualGatewayMembership(ctx, gr)
	if err != nil {
		return nil, err
	}

	if vg.Spec.MeshRef == nil || !mesh.IsMeshReferenced(ms, *vg.Spec.MeshRef) {
		return nil, errors.Errorf("virtualGateway referenced does not belong to the mesh referenced in GatewayRoute Create")
	}
	if err := m.defaultingAWSName(gr); err != nil {
		return nil, err
	}

	return gr, nil
}

func (m *gatewayRouteMutator) MutateUpdate(ctx context.Context, obj runtime.Object, oldObj runtime.Object) (runtime.Object, error) {
	return obj, nil
}

func (m *gatewayRouteMutator) defaultingAWSName(gr *appmesh.GatewayRoute) error {
	if gr.Spec.AWSName == nil || len(*gr.Spec.AWSName) == 0 {
		awsName := fmt.Sprintf("%s_%s", gr.Name, gr.Namespace)
		gr.Spec.AWSName = &awsName
	}
	return nil
}

func (m *gatewayRouteMutator) designateMeshMembership(ctx context.Context, gr *appmesh.GatewayRoute) (*appmesh.Mesh, error) {
	if gr.Spec.MeshRef != nil {
		return nil, errors.Errorf("%s create may not specify read-only field: %s", "GatewayRoute", "spec.meshRef")
	}
	mesh, err := m.meshMembershipDesignator.Designate(ctx, gr)
	if err != nil {
		return nil, err
	}
	gr.Spec.MeshRef = &appmesh.MeshReference{
		Name: mesh.Name,
		UID:  mesh.UID,
	}
	return mesh, nil
}

func (m *gatewayRouteMutator) designateVirtualGatewayMembership(ctx context.Context, gr *appmesh.GatewayRoute) (*appmesh.VirtualGateway, error) {
	if gr.Spec.VirtualGatewayRef != nil {
		return nil, errors.Errorf("%s create may not specify read-only field: %s", "GatewayRoute", "spec.virtualGatewayRef")
	}
	virtualGateway, err := m.virtualGatewayMembershipDesignator.DesignateForGatewayRoute(ctx, gr)
	if err != nil {
		return nil, err
	}
	gr.Spec.VirtualGatewayRef = &appmesh.VirtualGatewayReference{
		Namespace: &virtualGateway.Namespace,
		Name:      virtualGateway.Name,
		UID:       virtualGateway.UID,
	}
	return virtualGateway, nil
}

// +kubebuilder:webhook:path=/mutate-appmesh-k8s-aws-v1beta2-gatewayroute,mutating=true,failurePolicy=fail,groups=appmesh.k8s.aws,resources=gatewayroutes,verbs=create;update,versions=v1beta2,name=mgatewayroute.appmesh.k8s.aws,sideEffects=None,webhookVersions=v1beta1

func (m *gatewayRouteMutator) SetupWithManager(mgr ctrl.Manager) {
	mgr.GetWebhookServer().Register(apiPathMutateAppMeshGatewayRoute, webhook.MutatingWebhookForMutator(m))
}
