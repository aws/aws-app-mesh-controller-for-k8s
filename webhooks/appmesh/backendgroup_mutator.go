package appmesh

import (
	"context"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/mesh"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/webhook"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const apiPathMutateAppMeshBackendGroup = "/mutate-appmesh-k8s-aws-v1beta2-backendgroup"

// NewBackendGroupMutator returns a mutator for BackendGroup.
func NewBackendGroupMutator(meshMembershipDesignator mesh.MembershipDesignator) *backendGroupMutator {
	return &backendGroupMutator{
		meshMembershipDesignator: meshMembershipDesignator,
	}
}

var _ webhook.Mutator = &backendGroupMutator{}

type backendGroupMutator struct {
	meshMembershipDesignator mesh.MembershipDesignator
}

func (m *backendGroupMutator) Prototype(req admission.Request) (runtime.Object, error) {
	return &appmesh.BackendGroup{}, nil
}

func (m *backendGroupMutator) MutateCreate(ctx context.Context, obj runtime.Object) (runtime.Object, error) {
	bg := obj.(*appmesh.BackendGroup)
	if err := m.designateMeshMembership(ctx, bg); err != nil {
		return nil, err
	}

	return bg, nil
}

func (m *backendGroupMutator) MutateUpdate(ctx context.Context, obj runtime.Object, oldObj runtime.Object) (runtime.Object, error) {
	return obj, nil
}

func (m *backendGroupMutator) designateMeshMembership(ctx context.Context, bg *appmesh.BackendGroup) error {
	if bg.Spec.MeshRef != nil {
		return errors.Errorf("%s create may not specify read-only field: %s", "BackendGroup", "spec.meshRef")
	}
	mesh, err := m.meshMembershipDesignator.Designate(ctx, bg)
	if err != nil {
		return err
	}
	bg.Spec.MeshRef = &appmesh.MeshReference{
		Name: mesh.Name,
		UID:  mesh.UID,
	}
	return nil
}

// +kubebuilder:webhook:path=/mutate-appmesh-k8s-aws-v1beta2-backendgroup,mutating=true,failurePolicy=fail,groups=appmesh.k8s.aws,resources=backendgroups,verbs=create;update,versions=v1beta2,name=mbackendgroup.appmesh.k8s.aws,sideEffects=None,webhookVersions=v1beta1

func (m *backendGroupMutator) SetupWithManager(mgr ctrl.Manager) {
	mgr.GetWebhookServer().Register(apiPathMutateAppMeshBackendGroup, webhook.MutatingWebhookForMutator(m))
}
