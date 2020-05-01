package virtualrouter

import (
	"context"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ReferenceResolver resolves references to virtualRouter CR.
type ReferenceResolver interface {
	// Resolve returns a virtualRouter CR based on vrRef
	Resolve(ctx context.Context, obj metav1.Object, vrRef appmesh.VirtualRouterReference) (*appmesh.VirtualRouter, error)
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

func (r *defaultReferenceResolver) Resolve(ctx context.Context, obj metav1.Object, vrRef appmesh.VirtualRouterReference) (*appmesh.VirtualRouter, error) {
	namespace := obj.GetNamespace()
	if vrRef.Namespace != nil && len(*vrRef.Namespace) != 0 {
		namespace = *vrRef.Namespace
	}

	vrKey := types.NamespacedName{Namespace: namespace, Name: vrRef.Name}
	vr := &appmesh.VirtualRouter{}
	if err := r.k8sClient.Get(ctx, vrKey, vr); err != nil {
		return nil, errors.Wrapf(err, "unable to fetch virtualRouter: %v", vrKey)
	}
	return vr, nil
}
