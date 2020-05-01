package virtualnode

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	corev1 "k8s.io/api/core/v1"
)

// IsVirtualNodeActive checks whether given virtualNode is active.
// virtualNode is active when its VirtualNodeActive condition equals true.
func IsVirtualNodeActive(vn *appmesh.VirtualNode) bool {
	for _, condition := range vn.Status.Conditions {
		if condition.Type == appmesh.VirtualNodeActive {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}
