package virtualservice

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/references"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

const (
	ReferenceKindVirtualNode   = "VirtualNode"
	ReferenceKindVirtualRouter = "VirtualRouter"
)

func VirtualNodeReferenceIndexFunc(obj runtime.Object) []types.NamespacedName {
	vs := obj.(*appmesh.VirtualService)
	if vs.Spec.Provider == nil || vs.Spec.Provider.VirtualNode == nil {
		return nil
	}
	vnRef := vs.Spec.Provider.VirtualNode.VirtualNodeRef
	vnKey := references.ObjectKeyForVirtualNodeReference(vs, vnRef)
	return []types.NamespacedName{vnKey}
}

func VirtualRouterReferenceIndexFunc(obj runtime.Object) []types.NamespacedName {
	vs := obj.(*appmesh.VirtualService)
	if vs.Spec.Provider == nil || vs.Spec.Provider.VirtualRouter == nil {
		return nil
	}
	vrRef := vs.Spec.Provider.VirtualRouter.VirtualRouterRef
	vrKey := references.ObjectKeyForVirtualRouterReference(vs, vrRef)
	return []types.NamespacedName{vrKey}
}
