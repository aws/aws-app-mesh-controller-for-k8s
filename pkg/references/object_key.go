package references

import (
	"fmt"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// ObjectKeyForVirtualGatewayReference returns the key of referenced VirtualGateway CR.
func ObjectKeyForVirtualGatewayReference(obj metav1.Object, vgRef appmesh.VirtualGatewayReference) types.NamespacedName {
	namespace := obj.GetNamespace()
	if vgRef.Namespace != nil && len(*vgRef.Namespace) != 0 {
		namespace = *vgRef.Namespace
	}
	return types.NamespacedName{Namespace: namespace, Name: vgRef.Name}
}

// ObjectKeyForVirtualNodeReference returns the key of referenced VirtualNode CR.
func ObjectKeyForVirtualNodeReference(obj metav1.Object, vnRef appmesh.VirtualNodeReference) types.NamespacedName {
	namespace := obj.GetNamespace()
	if vnRef.Namespace != nil && len(*vnRef.Namespace) != 0 {
		namespace = *vnRef.Namespace
	}
	return types.NamespacedName{Namespace: namespace, Name: vnRef.Name}
}

// ObjectKeyForVirtualServiceReference returns the key of referenced VirtualService CR.
func ObjectKeyForVirtualServiceReference(obj metav1.Object, vsRef appmesh.VirtualServiceReference) types.NamespacedName {
	namespace := obj.GetNamespace()
	if vsRef.Namespace != nil && len(*vsRef.Namespace) != 0 {
		namespace = *vsRef.Namespace
	}
	return types.NamespacedName{Namespace: namespace, Name: vsRef.Name}
}

// ObjectKeyForVirtualRouterReference returns the key of referenced VirtualRouter CR.
func ObjectKeyForVirtualRouterReference(obj metav1.Object, vrRef appmesh.VirtualRouterReference) types.NamespacedName {
	namespace := obj.GetNamespace()
	if vrRef.Namespace != nil && len(*vrRef.Namespace) != 0 {
		namespace = *vrRef.Namespace
	}
	return types.NamespacedName{Namespace: namespace, Name: vrRef.Name}
}

// ObjectKeyForBackendGroupReference returns the key of referenced BackendGroup CR.
func ObjectKeyForBackendGroupReference(obj metav1.Object, bgRef appmesh.BackendGroupReference) types.NamespacedName {
	namespace := obj.GetNamespace()
	if bgRef.Namespace != nil && len(*bgRef.Namespace) != 0 {
		namespace = *bgRef.Namespace
	}
	return types.NamespacedName{Namespace: namespace, Name: bgRef.Name}
}

// KeyForVirtualServiceOfaVirtualNode returns the key of referenced VirtualService CR.
func KeyForVirtualServiceOfaVirtualNode(vsRef appmesh.VirtualServiceReference) (*types.NamespacedName, error) {
	if vsRef.Namespace != nil && len(*vsRef.Namespace) != 0 {
		namespace := *vsRef.Namespace
		return &types.NamespacedName{Namespace: namespace, Name: vsRef.Name}, nil
	} else {
		return nil, fmt.Errorf("namespace: %v, name: %v", vsRef.Namespace, vsRef.Name)
	}
}
