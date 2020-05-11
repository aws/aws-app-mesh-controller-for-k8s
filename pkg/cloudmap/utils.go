package cloudmap

import (
	corev1 "k8s.io/api/core/v1"
)

func ArePodContainersReady(pod *corev1.Pod) bool {
	conditions := (&pod.Status).Conditions
	for i := range conditions {
		if conditions[i].Type == corev1.ContainersReady && conditions[i].Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func ShouldPodBeInEndpoints(pod *corev1.Pod) bool {
	switch pod.Spec.RestartPolicy {
	case corev1.RestartPolicyNever:
		return pod.Status.Phase != corev1.PodFailed && pod.Status.Phase != corev1.PodSucceeded
	case corev1.RestartPolicyOnFailure:
		return pod.Status.Phase != corev1.PodSucceeded
	default:
		return true
	}
}
