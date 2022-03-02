package virtualrouter

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/references"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ReferenceKindVirtualNode = "VirtualNode"
)

// ExtractVirtualNodeReferences extracts all virtualNodeReferences for this virtualRouter
func ExtractVirtualNodeReferences(vr *appmesh.VirtualRouter) []appmesh.VirtualNodeReference {
	var vnRefs []appmesh.VirtualNodeReference
	for _, route := range vr.Spec.Routes {
		if route.GRPCRoute != nil {
			for _, target := range route.GRPCRoute.Action.WeightedTargets {
				if target.VirtualNodeRef != nil {
					vnRefs = append(vnRefs, *target.VirtualNodeRef)
				}
			}
		}
		if route.HTTPRoute != nil {
			for _, target := range route.HTTPRoute.Action.WeightedTargets {
				if target.VirtualNodeRef != nil {
					vnRefs = append(vnRefs, *target.VirtualNodeRef)
				}
			}
		}
		if route.HTTP2Route != nil {
			for _, target := range route.HTTP2Route.Action.WeightedTargets {
				if target.VirtualNodeRef != nil {
					vnRefs = append(vnRefs, *target.VirtualNodeRef)
				}
			}
		}
		if route.TCPRoute != nil {
			for _, target := range route.TCPRoute.Action.WeightedTargets {
				if target.VirtualNodeRef != nil {
					vnRefs = append(vnRefs, *target.VirtualNodeRef)
				}
			}
		}
	}
	return vnRefs
}

func VirtualNodeReferenceIndexFunc(obj client.Object) []types.NamespacedName {
	vr := obj.(*appmesh.VirtualRouter)
	vnRefs := ExtractVirtualNodeReferences(vr)
	var vnKeys []types.NamespacedName
	for _, vnRef := range vnRefs {
		vnKeys = append(vnKeys, references.ObjectKeyForVirtualNodeReference(vr, vnRef))
	}
	return vnKeys
}
