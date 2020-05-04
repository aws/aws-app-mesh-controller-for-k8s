package inject

import (
	"encoding/json"
	corev1 "k8s.io/api/core/v1"
)

const initContainerTemplate = `
{
  "name": "proxyinit",
  "image": "{{ .ContainerImage }}",
  "securityContext": {
    "capabilities": {
      "add": [
        "NET_ADMIN"
      ]
    }
  },
  "env": [
    {
      "name": "APPMESH_START_ENABLED",
      "value": "1"
    },
    {
      "name": "APPMESH_IGNORE_UID",
      "value": "{{ .ProxyUID }}"
    },
    {
      "name": "APPMESH_ENVOY_INGRESS_PORT",
      "value": "{{ .ProxyIngressPort }}"
    },
    {
      "name": "APPMESH_ENVOY_EGRESS_PORT",
      "value": "{{ .ProxyEgressPort }}"
    },
    {
      "name": "APPMESH_APP_PORTS",
      "value": "{{ .AppPorts }}"
    },
    {
      "name": "APPMESH_EGRESS_IGNORED_IP",
      "value": "{{ .EgressIgnoredIPs }}"
    },
    {
      "name": "APPMESH_EGRESS_IGNORED_PORTS",
      "value": "{{ .EgressIgnoredPorts }}"
    }
  ],
  "resources": {
    "requests": {
      "cpu": "{{ .CPURequests }}",
      "memory": "{{ .MemoryRequests }}"
    }
  }
}
`

type InitContainerTemplateVariables struct {
	AppPorts           string
	EgressIgnoredIPs   string
	EgressIgnoredPorts string
	ProxyEgressPort    int64
	ProxyIngressPort   int64
	ProxyUID           int64
	ContainerImage     string
	CPURequests        string
	MemoryRequests     string
}

type initProxyMutatorConfig struct {
	containerImage string
	cpuRequests    string
	memoryRequests string
}

// newInitProxyMutator constructs new initProxyMutator
func newInitProxyMutator(mutatorConfig initProxyMutatorConfig, proxyConfig proxyConfig) *initProxyMutator {
	return &initProxyMutator{
		mutatorConfig: mutatorConfig,
		proxyConfig:   proxyConfig,
	}
}

// proxy mutator based on init container
type initProxyMutator struct {
	mutatorConfig initProxyMutatorConfig
	proxyConfig   proxyConfig
}

func (m *initProxyMutator) mutate(pod *corev1.Pod) error {
	variables := m.buildTemplateVariables()
	containerJSON, err := renderTemplate("init", initContainerTemplate, variables)
	if err != nil {
		return err
	}
	container := corev1.Container{}
	err = json.Unmarshal([]byte(containerJSON), &container)
	if err != nil {
		return err
	}
	pod.Spec.InitContainers = append(pod.Spec.InitContainers, container)
	return nil
}

func (m *initProxyMutator) buildTemplateVariables() InitContainerTemplateVariables {
	return InitContainerTemplateVariables{
		AppPorts:           m.proxyConfig.appPorts,
		EgressIgnoredIPs:   m.proxyConfig.egressIgnoredIPs,
		EgressIgnoredPorts: m.proxyConfig.egressIgnoredPorts,
		ProxyEgressPort:    m.proxyConfig.proxyEgressPort,
		ProxyIngressPort:   m.proxyConfig.proxyIngressPort,
		ProxyUID:           m.proxyConfig.proxyUID,
		ContainerImage:     m.mutatorConfig.containerImage,
		CPURequests:        m.mutatorConfig.cpuRequests,
		MemoryRequests:     m.mutatorConfig.memoryRequests,
	}
}
