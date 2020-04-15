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

const apiPathMutateAppMeshVirtualRouter = "/mutate-appmesh-k8s-aws-v1beta2-virtualrouter"

// NewVirtualRouterMutator returns a mutator for VirtualRouter.
func NewVirtualRouterMutator(meshMembershipDesignator mesh.MembershipDesignator) *virtualRouterMutator {
	return &virtualRouterMutator{
		meshMembershipDesignator: meshMembershipDesignator,
	}
}

var _ webhook.Mutator = &virtualRouterMutator{}

type virtualRouterMutator struct {
	meshMembershipDesignator mesh.MembershipDesignator
}

func (m *virtualRouterMutator) Prototype(req admission.Request) (runtime.Object, error) {
	return &appmesh.VirtualRouter{}, nil
}

func (m *virtualRouterMutator) MutateCreate(ctx context.Context, obj runtime.Object) (runtime.Object, error) {
	vr := obj.(*appmesh.VirtualRouter)
	if err := m.designateMeshMembership(ctx, vr); err != nil {
		return nil, err
	}
	if err := m.defaultingAWSName(vr); err != nil {
		return nil, err
	}

	return vr, nil
}

func (m *virtualRouterMutator) MutateUpdate(ctx context.Context, obj runtime.Object, oldObj runtime.Object) (runtime.Object, error) {
	return obj, nil
}

func (m *virtualRouterMutator) defaultingAWSName(vr *appmesh.VirtualRouter) error {
	if vr.Spec.AWSName == nil || len(*vr.Spec.AWSName) == 0 {
		awsName := fmt.Sprintf("%s_%s", vr.Name, vr.Namespace)
		vr.Spec.AWSName = &awsName
	}
	return nil
}

func (m *virtualRouterMutator) designateMeshMembership(ctx context.Context, vr *appmesh.VirtualRouter) error {
	if vr.Spec.MeshRef != nil {
		return errors.Errorf("%s create may not specify read-only field: %s", "VirtualRouter", "spec.meshRef")
	}
	mesh, err := m.meshMembershipDesignator.Designate(ctx, vr)
	if err != nil {
		return err
	}
	vr.Spec.MeshRef = &appmesh.MeshReference{
		Name: mesh.Name,
		UID:  mesh.UID,
	}
	return nil
}

// +kubebuilder:webhook:path=/mutate-appmesh-k8s-aws-v1beta2-virtualrouter,mutating=true,failurePolicy=fail,groups=appmesh.k8s.aws,resources=virtualrouters,verbs=create;update,versions=v1beta2,name=mvirtualrouter.appmesh.k8s.aws

func (m *virtualRouterMutator) SetupWithManager(mgr ctrl.Manager) {
	mgr.GetWebhookServer().Register(apiPathMutateAppMeshVirtualRouter, webhook.MutatingWebhookForMutator(m))
}
