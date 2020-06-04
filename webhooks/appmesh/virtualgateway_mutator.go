package appmesh

import (
	"context"
	"fmt"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/mesh"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/webhook"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const apiPathMutateAppMeshVirtualGateway = "/mutate-appmesh-k8s-aws-v1beta2-virtualgateway"

// NewVirtualGatewayMutator returns a mutator for VirtualGateway.
func NewVirtualGatewayMutator(meshMembershipDesignator mesh.MembershipDesignator) *virtualGatewayMutator {
	return &virtualGatewayMutator{
		meshMembershipDesignator: meshMembershipDesignator,
	}
}

var _ webhook.Mutator = &virtualGatewayMutator{}

type virtualGatewayMutator struct {
	meshMembershipDesignator mesh.MembershipDesignator
}

func (m *virtualGatewayMutator) Prototype(req admission.Request) (runtime.Object, error) {
	return &appmesh.VirtualGateway{}, nil
}

func (m *virtualGatewayMutator) MutateCreate(ctx context.Context, obj runtime.Object) (runtime.Object, error) {
	vg := obj.(*appmesh.VirtualGateway)
	if err := m.designateMeshMembership(ctx, vg); err != nil {
		return nil, err
	}
	if err := m.defaultingAWSName(vg); err != nil {
		return nil, err
	}

	return vg, nil
}

func (m *virtualGatewayMutator) MutateUpdate(ctx context.Context, obj runtime.Object, oldObj runtime.Object) (runtime.Object, error) {
	return obj, nil
}

func (m *virtualGatewayMutator) defaultingAWSName(vg *appmesh.VirtualGateway) error {
	if vg.Spec.AWSName == nil || len(*vg.Spec.AWSName) == 0 {
		awsName := fmt.Sprintf("%s_%s", vg.Name, vg.Namespace)
		vg.Spec.AWSName = &awsName
	}
	return nil
}

func (m *virtualGatewayMutator) designateMeshMembership(ctx context.Context, vg *appmesh.VirtualGateway) error {
	if vg.Spec.MeshRef != nil {
		return errors.Errorf("%s create may not specify read-only field: %s", "VirtualGateway", "spec.meshRef")
	}
	mesh, err := m.meshMembershipDesignator.Designate(ctx, vg)
	if err != nil {
		return err
	}
	vg.Spec.MeshRef = &appmesh.MeshReference{
		Name: mesh.Name,
		UID:  mesh.UID,
	}
	return nil
}

// +kubebuilder:webhook:path=/mutate-appmesh-k8s-aws-v1beta2-virtualgateway,mutating=true,failurePolicy=fail,groups=appmesh.k8s.aws,resources=virtualgateways,verbs=create;update,versions=v1beta2,name=mvirtualgateway.appmesh.k8s.aws

func (m *virtualGatewayMutator) SetupWithManager(mgr ctrl.Manager) {
	mgr.GetWebhookServer().Register(apiPathMutateAppMeshVirtualGateway, webhook.MutatingWebhookForMutator(m))
}
