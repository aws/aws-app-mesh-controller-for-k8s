package appmesh

import (
	"context"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/webhook"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const apiPathMutateAppMeshMesh = "/mutate-appmesh-k8s-aws-v1beta2-mesh"

// NewMeshMutator returns a mutator for Mesh.
func NewMeshMutator() *meshMutator {
	return &meshMutator{}
}

var _ webhook.Mutator = &meshMutator{}

type meshMutator struct {
}

func (m *meshMutator) Prototype(req admission.Request) (runtime.Object, error) {
	return &appmesh.Mesh{}, nil
}

func (m *meshMutator) MutateCreate(ctx context.Context, obj runtime.Object) (runtime.Object, error) {
	mesh := obj.(*appmesh.Mesh)
	if err := m.defaultingAWSName(mesh); err != nil {
		return nil, err
	}
	return mesh, nil
}

func (m *meshMutator) MutateUpdate(ctx context.Context, obj runtime.Object, oldObj runtime.Object) (runtime.Object, error) {
	return obj, nil
}

func (m *meshMutator) defaultingAWSName(mesh *appmesh.Mesh) error {
	if mesh.Spec.AWSName == nil || len(*mesh.Spec.AWSName) == 0 {
		awsName := mesh.Name
		mesh.Spec.AWSName = &awsName
	}
	return nil
}

// +kubebuilder:webhook:path=/mutate-appmesh-k8s-aws-v1beta2-mesh,mutating=true,failurePolicy=fail,groups=appmesh.k8s.aws,resources=meshes,verbs=create;update,versions=v1beta2,name=mmesh.appmesh.k8s.aws

func (m *meshMutator) SetupWithManager(mgr ctrl.Manager) {
	mgr.GetWebhookServer().Register(apiPathMutateAppMeshMesh, webhook.MutatingWebhookForMutator(m))
}
