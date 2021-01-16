package k8s

import (
	"github.com/aws/aws-sdk-go/aws"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
)

const (
	ConditionAWSCloudMapHealthy = "conditions.appmesh.k8s.aws/aws-cloudmap-healthy"
	NodeNameSpec                = "nodeName"
	Namespace                   = "namespace"
)

// GetPodCondition will get pointer to Pod's existing condition.
func GetPodCondition(pod *corev1.Pod, conditionType corev1.PodConditionType) *corev1.PodCondition {
	for i := range pod.Status.Conditions {
		if pod.Status.Conditions[i].Type == conditionType {
			return &pod.Status.Conditions[i]
		}
	}
	return nil
}

// NamespaceIndexer returns indexer to index in data store using namespace
func NamespaceIndexer() cache.Indexers {
	indexer := map[string]cache.IndexFunc{}
	indexer[Namespace] = func(obj interface{}) (strings []string, err error) {
		return []string{obj.(*corev1.Pod).Namespace}, nil
	}
	return indexer
}

// NodeNameIndexer returns indexer to index in the data store using node name
func NodeNameIndexer() cache.Indexers {
	indexer := map[string]cache.IndexFunc{}
	indexer[NodeNameSpec] = func(obj interface{}) (strings []string, err error) {
		return []string{obj.(*corev1.Pod).Spec.NodeName}, nil
	}
	return indexer
}

// NSKeyIndexer is the key function to index the pod object using namespace/name
func NSKeyIndexer(obj interface{}) (string, error) {
	pod := obj.(*corev1.Pod)
	return types.NamespacedName{
		Namespace: pod.Namespace,
		Name:      pod.Name,
	}.String(), nil
}

// UpdatePodCondition will update Pod's condition. returns whether it's updated.
func UpdatePodCondition(pod *corev1.Pod, conditionType corev1.PodConditionType, status corev1.ConditionStatus, reason *string, message *string) bool {
	existingCondition := GetPodCondition(pod, conditionType)
	if existingCondition == nil {
		newCondition := corev1.PodCondition{
			Type:               conditionType,
			Status:             status,
			LastProbeTime:      metav1.Now(),
			LastTransitionTime: metav1.Now(),
			Reason:             aws.StringValue(reason),
			Message:            aws.StringValue(message),
		}
		pod.Status.Conditions = append(pod.Status.Conditions, newCondition)
		return true
	}

	if existingCondition.Status != status {
		existingCondition.Status = status
		existingCondition.LastTransitionTime = metav1.Now()
	}
	if existingCondition.Reason != aws.StringValue(reason) {
		existingCondition.Reason = aws.StringValue(reason)
	}
	if existingCondition.Message != aws.StringValue(message) {
		existingCondition.Message = aws.StringValue(message)
	}
	existingCondition.LastProbeTime = metav1.Now()
	return true
}
