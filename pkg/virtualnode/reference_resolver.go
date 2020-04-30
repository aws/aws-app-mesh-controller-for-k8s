package virtualnode

import (
	"context"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ReferenceResolver resolves references to virtualNode CR.
type ReferenceResolver interface {
	// Resolve returns a virtualNode CR based on vnRef
	Resolve(ctx context.Context, obj metav1.Object, vnRef appmesh.VirtualNodeReference) (*appmesh.VirtualNode, error)
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

func (r *defaultReferenceResolver) Resolve(ctx context.Context, obj metav1.Object, vnRef appmesh.VirtualNodeReference) (*appmesh.VirtualNode, error) {
	namespace := obj.GetNamespace()
	if vnRef.Namespace != nil && len(*vnRef.Namespace) != 0 {
		namespace = *vnRef.Namespace
	}

	vnKey := types.NamespacedName{Namespace: namespace, Name: vnRef.Name}
	vn := &appmesh.VirtualNode{}
	if err := r.k8sClient.Get(ctx, vnKey, vn); err != nil {
		return nil, errors.Wrapf(err, "unable to fetch virtualNode: %v", vnKey)
	}
	return vn, nil
}
