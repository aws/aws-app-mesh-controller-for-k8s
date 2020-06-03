package virtualgateway

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-sdk-go/aws"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// getCondition will get pointer to virtualGateway's existing condition.
func getCondition(vg *appmesh.VirtualGateway, conditionType appmesh.VirtualGatewayConditionType) *appmesh.VirtualGatewayCondition {
	for i := range vg.Status.Conditions {
		if vg.Status.Conditions[i].Type == conditionType {
			return &vg.Status.Conditions[i]
		}
	}
	return nil
}

// updateCondition will update virtualGateway's condition. returns whether it's updated.
func updateCondition(vg *appmesh.VirtualGateway, conditionType appmesh.VirtualGatewayConditionType, status corev1.ConditionStatus, reason *string, message *string) bool {
	now := metav1.Now()
	existingCondition := getCondition(vg, conditionType)
	if existingCondition == nil {
		newCondition := appmesh.VirtualGatewayCondition{
			Type:               conditionType,
			Status:             status,
			LastTransitionTime: &now,
			Reason:             reason,
			Message:            message,
		}
		vg.Status.Conditions = append(vg.Status.Conditions, newCondition)
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
