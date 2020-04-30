package virtualservice

import (
	"context"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ReferenceResolver resolves references to virtualService CR.
type ReferenceResolver interface {
	// Resolve returns a virtualService CR based on vsRef
	Resolve(ctx context.Context, obj metav1.Object, vsRef appmesh.VirtualServiceReference) (*appmesh.VirtualService, error)
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

func (r *defaultReferenceResolver) Resolve(ctx context.Context, obj metav1.Object, vsRef appmesh.VirtualServiceReference) (*appmesh.VirtualService, error) {
	namespace := obj.GetNamespace()
	if vsRef.Namespace != nil && len(*vsRef.Namespace) != 0 {
		namespace = *vsRef.Namespace
	}
	vsKey := types.NamespacedName{Namespace: namespace, Name: vsRef.Name}
	vs := &appmesh.VirtualService{}
	if err := r.k8sClient.Get(ctx, vsKey, vs); err != nil {
		return nil, errors.Wrapf(err, "unable to fetch virtualService: %v", vsKey)
	}
	return vs, nil
}
