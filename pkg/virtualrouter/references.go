package virtualrouter

import appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"

// extractVirtualNodeReferences extracts all virtualNodeReferences for this virtualRouter
func extractVirtualNodeReferences(vr *appmesh.VirtualRouter) []appmesh.VirtualNodeReference {
	var vnRefs []appmesh.VirtualNodeReference
	for _, route := range vr.Spec.Routes {
		if route.GRPCRoute != nil {
			for _, target := range route.GRPCRoute.Action.WeightedTargets {
				vnRefs = append(vnRefs, target.VirtualNodeRef)
			}
		}
		if route.HTTPRoute != nil {
			for _, target := range route.HTTPRoute.Action.WeightedTargets {
				vnRefs = append(vnRefs, target.VirtualNodeRef)
			}
		}
		if route.HTTP2Route != nil {
			for _, target := range route.HTTP2Route.Action.WeightedTargets {
				vnRefs = append(vnRefs, target.VirtualNodeRef)
			}
		}
		if route.TCPRoute != nil {
			for _, target := range route.TCPRoute.Action.WeightedTargets {
				vnRefs = append(vnRefs, target.VirtualNodeRef)
			}
		}
	}
	return vnRefs
}
