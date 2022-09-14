package inject

import (
	"fmt"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/k8s"
	corev1 "k8s.io/api/core/v1"
)

// newCloudMapHealthyReadinessGate constructs new cloudMapHealthyReadinessGate
func newCloudMapHealthyReadinessGate(vn *appmesh.VirtualNode) *cloudMapHealthyReadinessGate {
	return &cloudMapHealthyReadinessGate{
		vn: vn,
	}
}

var _ PodMutator = &cloudMapHealthyReadinessGate{}

// mutator adding a healthy readiness gate for pods selected by VirtualNode with cloudMap serviceDiscovery.
type cloudMapHealthyReadinessGate struct {
	vn *appmesh.VirtualNode
}

func (m *cloudMapHealthyReadinessGate) mutate(pod *corev1.Pod) error {
	if m.vn.Spec.ServiceDiscovery == nil || m.vn.Spec.ServiceDiscovery.AWSCloudMap == nil {
		return nil
	}
	fmt.Printf("pod status: %s", pod.Status.String())
	containsAWSCloudMapHealthyReadinessGate := false
	for _, item := range pod.Spec.ReadinessGates {
		if item.ConditionType == k8s.ConditionAWSCloudMapHealthy {
			containsAWSCloudMapHealthyReadinessGate = true
			fmt.Printf("gate for %s is ready\n", pod.Name)
			break
		}
	}
	if !containsAWSCloudMapHealthyReadinessGate {
		pod.Spec.ReadinessGates = append(pod.Spec.ReadinessGates, corev1.PodReadinessGate{
			ConditionType: k8s.ConditionAWSCloudMapHealthy,
		})
	}
	return nil
}
