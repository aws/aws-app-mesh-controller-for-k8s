package gatewayroute

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/references"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

const (
	ReferenceKindVirtualService = "VirtualService"
)

// extractVirtualServiceReferences extracts all virtualServiceReferences for this gatewayRoute
func extractVirtualServiceReferences(gr *appmesh.GatewayRoute) []appmesh.VirtualServiceReference {
	var vsRefs []appmesh.VirtualServiceReference

	if gr.Spec.GRPCRoute != nil {
		vsRefs = append(vsRefs, gr.Spec.GRPCRoute.Action.Target.VirtualService.VirtualServiceRef)
	}
	if gr.Spec.HTTPRoute != nil {
		vsRefs = append(vsRefs, gr.Spec.HTTPRoute.Action.Target.VirtualService.VirtualServiceRef)
	}
	if gr.Spec.HTTP2Route != nil {
		vsRefs = append(vsRefs, gr.Spec.HTTP2Route.Action.Target.VirtualService.VirtualServiceRef)
	}

	return vsRefs
}

func VirtualServiceReferenceIndexFunc(obj runtime.Object) []types.NamespacedName {
	gr := obj.(*appmesh.GatewayRoute)
	vsRefs := extractVirtualServiceReferences(gr)
	var vsKeys []types.NamespacedName
	for _, vsRef := range vsRefs {
		vsKeys = append(vsKeys, references.ObjectKeyForVirtualServiceReference(gr, vsRef))
	}
	return vsKeys
}
