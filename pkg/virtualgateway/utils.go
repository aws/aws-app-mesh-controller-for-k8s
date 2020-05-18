package virtualgateway

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	corev1 "k8s.io/api/core/v1"
)

// IsVirtualGatewayActive tests whether given virtualGateway is active.
// virtualGateway is active when its VirtualGatewayActive condition equals true.
func IsVirtualGatewayActive(vg *appmesh.VirtualGateway) bool {
	for _, condition := range vg.Status.Conditions {
		if condition.Type == appmesh.VirtualGatewayActive {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

// IsVirtualGatewayReferenced tests whether given virtualGateway is referenced by virtualGatewayReference
func IsVirtualGatewayReferenced(vg *appmesh.VirtualGateway, reference appmesh.VirtualGatewayReference) bool {
	return vg.Name == reference.Name && vg.Namespace == *reference.Namespace && vg.UID == reference.UID
}
