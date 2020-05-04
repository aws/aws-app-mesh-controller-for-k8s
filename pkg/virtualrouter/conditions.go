package virtualrouter

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-sdk-go/aws"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// getCondition will get pointer to virtualRouter's existing condition.
func getCondition(vr *appmesh.VirtualRouter, conditionType appmesh.VirtualRouterConditionType) *appmesh.VirtualRouterCondition {
	for i := range vr.Status.Conditions {
		if vr.Status.Conditions[i].Type == conditionType {
			return &vr.Status.Conditions[i]
		}
	}
	return nil
}

// updateCondition will update virtualRouter's condition. returns whether it's updated.
func updateCondition(vr *appmesh.VirtualRouter, conditionType appmesh.VirtualRouterConditionType, status corev1.ConditionStatus, reason *string, message *string) bool {
	now := metav1.Now()
	existingCondition := getCondition(vr, conditionType)
	if existingCondition == nil {
		newCondition := appmesh.VirtualRouterCondition{
			Type:               conditionType,
			Status:             status,
			LastTransitionTime: &now,
			Reason:             reason,
			Message:            message,
		}
		vr.Status.Conditions = append(vr.Status.Conditions, newCondition)
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
