package virtualnode

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	corev1 "k8s.io/api/core/v1"
)

// IsVirtualNodective tests whether given mesh is active.
// mesh is active when its MeshActive condition equals true.
func IsVirtualNodeActive(vn *appmesh.VirtualNode) bool {
	for _, condition := range vn.Status.Conditions {
		if condition.Type == appmesh.VirtualNodeActive {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

// IsVirtualNodeReferenced tests whether given mesh is referenced by meshReference
func podBelongsToVirtualNode(vn *appmesh.VirtualNode, pod *corev1.Pod) bool {
	return vn.Name == reference.Name
}
