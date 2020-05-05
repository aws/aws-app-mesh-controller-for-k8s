package inject

import (
	"fmt"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/algorithm"
	corev1 "k8s.io/api/core/v1"
)

// newCNIProxyMutator constructs new cniProxyMutator
func newCNIProxyMutator(proxyConfig proxyConfig) *cniProxyMutator {
	return &cniProxyMutator{
		proxyConfig: proxyConfig,
	}
}

// proxy mutator based on AppMeshCNI
type cniProxyMutator struct {
	proxyConfig proxyConfig
}

func (m *cniProxyMutator) mutate(pod *corev1.Pod) error {
	cniAnnotations := map[string]string{
		AppMeshPortsAnnotation:              m.proxyConfig.appPorts,
		AppMeshEgressIgnoredIPsAnnotation:   m.proxyConfig.egressIgnoredIPs,
		AppMeshEgressIgnoredPortsAnnotation: m.proxyConfig.egressIgnoredPorts,
		AppMeshProxyEgressPortAnnotation:    fmt.Sprintf("%d", m.proxyConfig.proxyEgressPort),
		AppMeshProxyIngressPortAnnotation:   fmt.Sprintf("%d", m.proxyConfig.proxyIngressPort),
		AppMeshIgnoredUIDAnnotation:         fmt.Sprintf("%d", m.proxyConfig.proxyUID),
		AppMeshSidecarInjectAnnotation:      "enabled",
	}
	annotations := algorithm.MergeStringMap(cniAnnotations, pod.GetAnnotations())
	pod.SetAnnotations(annotations)
	return nil
}
