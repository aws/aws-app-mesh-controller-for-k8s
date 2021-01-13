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

const apiPathMutateAppMeshVirtualService = "/mutate-appmesh-k8s-aws-v1beta2-virtualservice"

// NewVirtualServiceMutator returns a mutator for VirtualService.
func NewVirtualServiceMutator(meshMembershipDesignator mesh.MembershipDesignator) *virtualServiceMutator {
	return &virtualServiceMutator{
		meshMembershipDesignator: meshMembershipDesignator,
	}
}

var _ webhook.Mutator = &virtualServiceMutator{}

type virtualServiceMutator struct {
	meshMembershipDesignator mesh.MembershipDesignator
}

func (m *virtualServiceMutator) Prototype(req admission.Request) (runtime.Object, error) {
	return &appmesh.VirtualService{}, nil
}

func (m *virtualServiceMutator) MutateCreate(ctx context.Context, obj runtime.Object) (runtime.Object, error) {
	vs := obj.(*appmesh.VirtualService)
	if err := m.designateMeshMembership(ctx, vs); err != nil {
		return nil, err
	}
	if err := m.defaultingAWSName(vs); err != nil {
		return nil, err
	}

	return vs, nil
}

func (m *virtualServiceMutator) MutateUpdate(ctx context.Context, obj runtime.Object, oldObj runtime.Object) (runtime.Object, error) {
	return obj, nil
}

func (m *virtualServiceMutator) defaultingAWSName(vs *appmesh.VirtualService) error {
	if vs.Spec.AWSName == nil || len(*vs.Spec.AWSName) == 0 {
		awsName := fmt.Sprintf("%s.%s", vs.Name, vs.Namespace)
		vs.Spec.AWSName = &awsName
	}
	return nil
}

func (m *virtualServiceMutator) designateMeshMembership(ctx context.Context, vs *appmesh.VirtualService) error {
	if vs.Spec.MeshRef != nil {
		return errors.Errorf("%s create may not specify read-only field: %s", "VirtualService", "spec.meshRef")
	}
	mesh, err := m.meshMembershipDesignator.Designate(ctx, vs)
	if err != nil {
		return err
	}
	vs.Spec.MeshRef = &appmesh.MeshReference{
		Name: mesh.Name,
		UID:  mesh.UID,
	}
	return nil
}

// +kubebuilder:webhook:path=/mutate-appmesh-k8s-aws-v1beta2-virtualservice,mutating=true,failurePolicy=fail,groups=appmesh.k8s.aws,resources=virtualservices,verbs=create;update,versions=v1beta2,name=mvirtualservice.appmesh.k8s.aws,sideEffects=None,webhookVersions=v1beta1

func (m *virtualServiceMutator) SetupWithManager(mgr ctrl.Manager) {
	mgr.GetWebhookServer().Register(apiPathMutateAppMeshVirtualService, webhook.MutatingWebhookForMutator(m))
}
