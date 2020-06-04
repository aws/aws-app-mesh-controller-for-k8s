package references

import (
	"context"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Resolver interface {
	// ResolveMeshReference returns a mesh CR based on ref
	ResolveMeshReference(ctx context.Context, ref appmesh.MeshReference) (*appmesh.Mesh, error)
	// ResolveVirtualGatewayReference returns a virtualGateway CR based on obj and ref
	ResolveVirtualGatewayReference(ctx context.Context, obj metav1.Object, ref appmesh.VirtualGatewayReference) (*appmesh.VirtualGateway, error)
	// ResolveVirtualNodeReference returns a virtualNode CR based on obj and ref
	ResolveVirtualNodeReference(ctx context.Context, obj metav1.Object, ref appmesh.VirtualNodeReference) (*appmesh.VirtualNode, error)
	// ResolveVirtualServiceReference returns a virtualService CR based obj and ref
	ResolveVirtualServiceReference(ctx context.Context, obj metav1.Object, ref appmesh.VirtualServiceReference) (*appmesh.VirtualService, error)
	// ResolveVirtualRouterReference returns a virtualRouter CR based obj and ref
	ResolveVirtualRouterReference(ctx context.Context, obj metav1.Object, ref appmesh.VirtualRouterReference) (*appmesh.VirtualRouter, error)
}

// NewDefaultResolver constructs new defaultResolver
func NewDefaultResolver(k8sClient client.Client, log logr.Logger) Resolver {
	return &defaultResolver{
		k8sClient: k8sClient,
		log:       log,
	}
}

// defaultResolver implements Resolver
type defaultResolver struct {
	k8sClient client.Client
	log       logr.Logger
}

func (r *defaultResolver) ResolveMeshReference(ctx context.Context, ref appmesh.MeshReference) (*appmesh.Mesh, error) {
	mesh := &appmesh.Mesh{}
	if err := r.k8sClient.Get(ctx, types.NamespacedName{Name: ref.Name}, mesh); err != nil {
		return nil, errors.Wrapf(err, "unable to fetch mesh: %s", ref.Name)
	}

	if mesh.UID != ref.UID {
		r.log.Error(nil, "mesh UID mismatch",
			"mesh", ref.Name,
			"expected UID", ref.UID,
			"actual UID", mesh.UID,
		)
		return nil, errors.Errorf("mesh UID mismatch: %s", ref.Name)
	}
	return mesh, nil
}

func (r *defaultResolver) ResolveVirtualGatewayReference(ctx context.Context, obj metav1.Object, ref appmesh.VirtualGatewayReference) (*appmesh.VirtualGateway, error) {
	vgKey := ObjectKeyForVirtualGatewayReference(obj, ref)
	vg := &appmesh.VirtualGateway{}
	if err := r.k8sClient.Get(ctx, vgKey, vg); err != nil {
		return nil, errors.Wrapf(err, "unable to fetch virtualGateway: %v", vgKey)
	}

	if vg.UID != ref.UID {
		r.log.Error(nil, "virtualGateway UID mismatch",
			"virtualGateway", ref.Name,
			"expected UID", ref.UID,
			"actual UID", vg.UID,
		)
		return nil, errors.Errorf("virtualGateway UID mismatch: %s", ref.Name)
	}
	return vg, nil
}

func (r *defaultResolver) ResolveVirtualNodeReference(ctx context.Context, obj metav1.Object, ref appmesh.VirtualNodeReference) (*appmesh.VirtualNode, error) {
	vnKey := ObjectKeyForVirtualNodeReference(obj, ref)
	vn := &appmesh.VirtualNode{}
	if err := r.k8sClient.Get(ctx, vnKey, vn); err != nil {
		return nil, errors.Wrapf(err, "unable to fetch virtualNode: %v", vnKey)
	}
	return vn, nil
}

func (r *defaultResolver) ResolveVirtualServiceReference(ctx context.Context, obj metav1.Object, ref appmesh.VirtualServiceReference) (*appmesh.VirtualService, error) {
	vsKey := ObjectKeyForVirtualServiceReference(obj, ref)
	vs := &appmesh.VirtualService{}
	if err := r.k8sClient.Get(ctx, vsKey, vs); err != nil {
		return nil, errors.Wrapf(err, "unable to fetch virtualService: %v", vsKey)
	}
	return vs, nil
}

func (r *defaultResolver) ResolveVirtualRouterReference(ctx context.Context, obj metav1.Object, ref appmesh.VirtualRouterReference) (*appmesh.VirtualRouter, error) {
	vrKey := ObjectKeyForVirtualRouterReference(obj, ref)
	vr := &appmesh.VirtualRouter{}
	if err := r.k8sClient.Get(ctx, vrKey, vr); err != nil {
		return nil, errors.Wrapf(err, "unable to fetch virtualRouter: %v", vrKey)
	}
	return vr, nil
}
