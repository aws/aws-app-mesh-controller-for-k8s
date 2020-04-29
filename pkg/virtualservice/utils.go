package virtualservice

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	corev1 "k8s.io/api/core/v1"
)

// IsVirtualServiceActive checks whether given virtualService is active.
// virtualService is active when its VirtualServiceActive condition equals true.
func IsVirtualServiceActive(vs *appmesh.VirtualService) bool {
	for _, condition := range vs.Status.Conditions {
		if condition.Type == appmesh.VirtualServiceActive {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}
