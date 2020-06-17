// +build preview

package gatewayroute

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	corev1 "k8s.io/api/core/v1"
)

// IsGatewayRouteActive tests whether given gatewayRoute is active.
// gatewayRoute is active when its GatewayRouteActive condition equals true.
func IsGatewayRouteActive(gr *appmesh.GatewayRoute) bool {
	for _, condition := range gr.Status.Conditions {
		if condition.Type == appmesh.GatewayRouteActive {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}
