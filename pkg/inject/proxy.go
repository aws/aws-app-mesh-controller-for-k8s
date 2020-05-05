package inject

import (
	"fmt"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	corev1 "k8s.io/api/core/v1"
	"strings"
)

const (
	defaultProxyEgressPort    = 15001
	defaultProxyIngressPort   = 15000
	defaultProxyUID           = 1337
	defaultEgressIgnoredPorts = "22"
)

type proxyMutatorConfig struct {
	initProxyMutatorConfig
	egressIgnoredIPs string
}

func newProxyMutator(mutatorConfig proxyMutatorConfig, vn *appmesh.VirtualNode) *proxyMutator {
	return &proxyMutator{
		mutatorConfig: mutatorConfig,
		vn:            vn,
	}
}

type proxyMutator struct {
	mutatorConfig proxyMutatorConfig
	vn            *appmesh.VirtualNode
}

func (m *proxyMutator) mutate(pod *corev1.Pod) error {
	proxyConfig := m.buildProxyConfig(pod)
	var mutator PodMutator
	if m.isAppMeshCNIEnabled(pod) {
		mutator = newCNIProxyMutator(proxyConfig)
	} else {
		mutator = newInitProxyMutator(m.mutatorConfig.initProxyMutatorConfig, proxyConfig)
	}
	return mutator.mutate(pod)
}

type proxyConfig struct {
	// Ports that needs to intercepting ingress traffic
	appPorts string
	// IPs that need to be ignored when intercepting traffic
	egressIgnoredIPs string
	// Ports that need to ignored when intercepting traffic
	egressIgnoredPorts string

	// port used by proxy for egress traffic (traffic originating from app container to external services)
	proxyEgressPort int64
	// port used by proxy for incoming traffic
	proxyIngressPort int64
	// UID used by proxy
	proxyUID int64
}

func (m *proxyMutator) buildProxyConfig(pod *corev1.Pod) proxyConfig {
	appPorts := m.getAppPorts(pod)
	egressIgnoredPorts := m.getEgressIgnoredPorts(pod)
	return proxyConfig{
		appPorts:           appPorts,
		egressIgnoredIPs:   m.mutatorConfig.egressIgnoredIPs,
		egressIgnoredPorts: egressIgnoredPorts,
		proxyEgressPort:    defaultProxyEgressPort,
		proxyIngressPort:   defaultProxyIngressPort,
		proxyUID:           defaultProxyUID,
	}
}

func (m *proxyMutator) getAppPorts(pod *corev1.Pod) string {
	if v, ok := pod.ObjectMeta.Annotations[AppMeshPortsAnnotation]; ok {
		return v
	}

	var ports []string
	for _, listener := range m.vn.Spec.Listeners {
		ports = append(ports, fmt.Sprintf("%d", listener.PortMapping.Port))
	}
	if len(ports) == 0 {
		ports = []string{"0"}
	}
	return strings.Join(ports, ",")
}

func (m *proxyMutator) getEgressIgnoredPorts(pod *corev1.Pod) string {
	egressIgnoredPorts := defaultEgressIgnoredPorts
	if v, ok := pod.ObjectMeta.Annotations[AppMeshEgressIgnoredPortsAnnotation]; ok {
		egressIgnoredPorts = v
	}
	return egressIgnoredPorts
}

func (m *proxyMutator) isAppMeshCNIEnabled(pod *corev1.Pod) bool {
	annotations := pod.GetAnnotations()
	if v, ok := annotations[AppMeshCNIAnnotation]; ok {
		return strings.ToLower(v) == "enabled"
	}

	// Fargate platform has appMesh-cni enabled by default
	if v, ok := pod.GetLabels()[FargateProfileLabel]; ok {
		return len(v) > 0
	}
	return false
}
