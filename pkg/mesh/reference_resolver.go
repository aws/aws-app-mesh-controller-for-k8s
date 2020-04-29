package mesh

import (
	"context"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ReferenceResolver resolves references to mesh CR.
type ReferenceResolver interface {
	// Resolve returns a mesh CR based on meshRef
	Resolve(ctx context.Context, meshRef appmesh.MeshReference) (*appmesh.Mesh, error)
}

func NewDefaultReferenceResolver(k8sClient client.Client, log logr.Logger) ReferenceResolver {
	return &defaultReferenceResolver{
		k8sClient: k8sClient,
		log:       log,
	}
}

var _ ReferenceResolver = &defaultReferenceResolver{}

// defaultReferenceResolver implements ReferenceResolver
type defaultReferenceResolver struct {
	k8sClient client.Client
	log       logr.Logger
}

// Resolve returns a mesh CR based on meshRef
func (r *defaultReferenceResolver) Resolve(ctx context.Context, meshRef appmesh.MeshReference) (*appmesh.Mesh, error) {
	mesh := &appmesh.Mesh{}
	if err := r.k8sClient.Get(ctx, types.NamespacedName{Name: meshRef.Name}, mesh); err != nil {
		return nil, errors.Wrapf(err, "unable to fetch mesh: %s", meshRef.Name)
	}

	if mesh.UID != meshRef.UID {
		r.log.Error(nil, "mesh UID mismatch", "expected UID", meshRef.Name, "actual UID", mesh.UID)
		return nil, errors.Errorf("mesh UID mismatch: %s", meshRef.Name)
	}
	return mesh, nil
}
