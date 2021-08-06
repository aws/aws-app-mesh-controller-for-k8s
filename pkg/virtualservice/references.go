package virtualservice

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/references"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ReferenceKindVirtualNode   = "VirtualNode"
	ReferenceKindVirtualRouter = "VirtualRouter"
)

func ExtractVirtualNodeReferences(vs *appmesh.VirtualService) []appmesh.VirtualNodeReference {
	if vs.Spec.Provider == nil || vs.Spec.Provider.VirtualNode == nil || vs.Spec.Provider.VirtualNode.VirtualNodeRef == nil {
		return nil
	}
	return []appmesh.VirtualNodeReference{*vs.Spec.Provider.VirtualNode.VirtualNodeRef}
}

func ExtractVirtualRouterReferences(vs *appmesh.VirtualService) []appmesh.VirtualRouterReference {
	if vs.Spec.Provider == nil || vs.Spec.Provider.VirtualRouter == nil || vs.Spec.Provider.VirtualRouter.VirtualRouterRef == nil {
		return nil
	}
	return []appmesh.VirtualRouterReference{*vs.Spec.Provider.VirtualRouter.VirtualRouterRef}
}

func VirtualNodeReferenceIndexFunc(obj client.Object) []types.NamespacedName {
	vs := obj.(*appmesh.VirtualService)
	vnRefs := ExtractVirtualNodeReferences(vs)

	var vnKeys []types.NamespacedName
	for _, vnRef := range vnRefs {
		vnKey := references.ObjectKeyForVirtualNodeReference(vs, vnRef)
		vnKeys = append(vnKeys, vnKey)
	}
	return vnKeys
}

func VirtualRouterReferenceIndexFunc(obj client.Object) []types.NamespacedName {
	vs := obj.(*appmesh.VirtualService)
	vrRefs := ExtractVirtualRouterReferences(vs)

	var vrKeys []types.NamespacedName
	for _, vrRef := range vrRefs {
		vrKey := references.ObjectKeyForVirtualRouterReference(vs, vrRef)
		vrKeys = append(vrKeys, vrKey)
	}
	return vrKeys
}
