package virtualrouter

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	corev1 "k8s.io/api/core/v1"
)

// IsVirtualRouterActive checks whether given virtualRouter is active.
// virtualRouter is active when its VirtualRouterActive condition equals true.
func IsVirtualRouterActive(vr *appmesh.VirtualRouter) bool {
	for _, condition := range vr.Status.Conditions {
		if condition.Type == appmesh.VirtualRouterActive {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}
