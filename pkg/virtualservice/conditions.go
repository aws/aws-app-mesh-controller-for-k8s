package virtualservice

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-sdk-go/aws"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// getCondition will get pointer to virtualService's existing condition.
func getCondition(vs *appmesh.VirtualService, conditionType appmesh.VirtualServiceConditionType) *appmesh.VirtualServiceCondition {
	for i := range vs.Status.Conditions {
		if vs.Status.Conditions[i].Type == conditionType {
			return &vs.Status.Conditions[i]
		}
	}
	return nil
}

// updateCondition will update virtualService's condition. returns whether it's updated.
func updateCondition(vs *appmesh.VirtualService, conditionType appmesh.VirtualServiceConditionType, status corev1.ConditionStatus, reason *string, message *string) bool {
	now := metav1.Now()
	existingCondition := getCondition(vs, conditionType)
	if existingCondition == nil {
		newCondition := appmesh.VirtualServiceCondition{
			Type:               conditionType,
			Status:             status,
			LastTransitionTime: &now,
			Reason:             reason,
			Message:            message,
		}
		vs.Status.Conditions = append(vs.Status.Conditions, newCondition)
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
