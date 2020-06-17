// +build preview

package gatewayroute

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-sdk-go/aws"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// getCondition will get pointer to gatewayRoute's existing condition.
func getCondition(gr *appmesh.GatewayRoute, conditionType appmesh.GatewayRouteConditionType) *appmesh.GatewayRouteCondition {
	for i := range gr.Status.Conditions {
		if gr.Status.Conditions[i].Type == conditionType {
			return &gr.Status.Conditions[i]
		}
	}
	return nil
}

// updateCondition will update gatewayRoute's condition. returns whether it's updated.
func updateCondition(gr *appmesh.GatewayRoute, conditionType appmesh.GatewayRouteConditionType, status corev1.ConditionStatus, reason *string, message *string) bool {
	now := metav1.Now()
	existingCondition := getCondition(gr, conditionType)
	if existingCondition == nil {
		newCondition := appmesh.GatewayRouteCondition{
			Type:               conditionType,
			Status:             status,
			LastTransitionTime: &now,
			Reason:             reason,
			Message:            message,
		}
		gr.Status.Conditions = append(gr.Status.Conditions, newCondition)
		return true
	}

	hasChanged := false
	if existingCondition.Status != status {
		existingCondition.Status = status
		existingCondition.LastTransitionTime = &now
		hasChanged = true
	}
	if aws.StringValue(existingCondition.Reason) != aws.StringValue(reason) {
		existingCondition.Reason = reason
		hasChanged = true
	}
	if aws.StringValue(existingCondition.Message) != aws.StringValue(message) {
		existingCondition.Message = message
		hasChanged = true
	}
	return hasChanged
}
