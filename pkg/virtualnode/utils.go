package virtualnode

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	corev1 "k8s.io/api/core/v1"
)

<<<<<<< HEAD
// IsVirtualNodective tests whether given mesh is active.
// mesh is active when its MeshActive condition equals true.
=======
// IsVirtualNodeActive checks whether given virtualNode is active.
// virtualNode is active when its VirtualNodeActive condition equals true.
>>>>>>> 53da0cbbe9d678c402fd834fba93d747884081a0
func IsVirtualNodeActive(vn *appmesh.VirtualNode) bool {
	for _, condition := range vn.Status.Conditions {
		if condition.Type == appmesh.VirtualNodeActive {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}
<<<<<<< HEAD

// IsVirtualNodeReferenced tests whether given mesh is referenced by meshReference
func podBelongsToVirtualNode(vn *appmesh.VirtualNode, pod *corev1.Pod) bool {
	return vn.Name == reference.Name
}
=======
>>>>>>> 53da0cbbe9d678c402fd834fba93d747884081a0
