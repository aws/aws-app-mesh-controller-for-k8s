package mesh

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-sdk-go/aws"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// getCondition will get pointer to mesh's existing condition.
func getCondition(ms *appmesh.Mesh, conditionType appmesh.MeshConditionType) *appmesh.MeshCondition {
	for i := range ms.Status.Conditions {
		if ms.Status.Conditions[i].Type == conditionType {
			return &ms.Status.Conditions[i]
		}
	}
	return nil
}

// updateCondition will update mesh's condition. returns whether it's updated.
func updateCondition(ms *appmesh.Mesh, conditionType appmesh.MeshConditionType, status corev1.ConditionStatus, reason *string, message *string) bool {
	now := metav1.Now()
	existingCondition := getCondition(ms, conditionType)
	if existingCondition == nil {
		newCondition := appmesh.MeshCondition{
			Type:               conditionType,
			Status:             status,
			LastTransitionTime: &now,
			Reason:             reason,
			Message:            message,
		}
		ms.Status.Conditions = append(ms.Status.Conditions, newCondition)
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
