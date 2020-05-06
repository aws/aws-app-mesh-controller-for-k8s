package cloudmap

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog"
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

func PodToInstanceID(pod *corev1.Pod) string {
	if pod.Status.PodIP == "" {
		return ""
	}
	return pod.Status.PodIP
}

func IsCloudMapEnabledForVirtualNode(vNode *appmesh.VirtualNode) bool {
	if vNode.Spec.ServiceDiscovery == nil || vNode.Spec.ServiceDiscovery.AWSCloudMap == nil {
		return false
	}
	if vNode.Spec.ServiceDiscovery.AWSCloudMap.NamespaceName == "" ||
		vNode.Spec.ServiceDiscovery.AWSCloudMap.ServiceName == "" {
		klog.Errorf("CloudMap NamespaceName or ServiceName is null")
		return false
	}
	return true
}

func IsPodSelectorDefinedForVirtualNode(vNode *appmesh.VirtualNode) bool {
	if vNode.Spec.PodSelector == nil {
		return false
	}
	return true
}
