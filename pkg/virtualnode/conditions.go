package virtualnode

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-sdk-go/aws"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// getCondition will get pointer to virtualNode's existing condition.
func getCondition(vn *appmesh.VirtualNode, conditionType appmesh.VirtualNodeConditionType) *appmesh.VirtualNodeCondition {
	for i := range vn.Status.Conditions {
		if vn.Status.Conditions[i].Type == conditionType {
			return &vn.Status.Conditions[i]
		}
	}
	return nil
}

// updateCondition will update virtualNode's condition. returns whether it's updated.
func updateCondition(vn *appmesh.VirtualNode, conditionType appmesh.VirtualNodeConditionType, status corev1.ConditionStatus, reason *string, message *string) bool {
	now := metav1.Now()
	existingCondition := getCondition(vn, conditionType)
	if existingCondition == nil {
		newCondition := appmesh.VirtualNodeCondition{
			Type:               conditionType,
			Status:             status,
			LastTransitionTime: &now,
			Reason:             reason,
			Message:            message,
		}
		vn.Status.Conditions = append(vn.Status.Conditions, newCondition)
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
