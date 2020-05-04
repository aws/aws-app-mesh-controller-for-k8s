package inject

import (
	"encoding/json"
	"fmt"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	corev1 "k8s.io/api/core/v1"
	"strings"
)

const initContainerTemplate = `
{
  "name": "proxyinit",
  "image": "{{ .InitImage }}",
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
      "value": "1337"
    },
    {
      "name": "APPMESH_ENVOY_INGRESS_PORT",
      "value": "15000"
    },
    {
      "name": "APPMESH_ENVOY_EGRESS_PORT",
      "value": "15001"
    },
    {
      "name": "APPMESH_APP_PORTS",
      "value": "{{ .Ports }}"
    },
    {
      "name": "APPMESH_EGRESS_IGNORED_IP",
      "value": "{{ .IgnoredIPs }}"
    },
    {
      "name": "APPMESH_EGRESS_IGNORED_PORTS",
      "value": "{{ .EgressIgnoredPorts }}"
    }
  ],
  "resources": {
    "requests": {
      "cpu": "{{ .SidecarCpu }}",
      "memory": "{{ .SidecarMemory }}"
    }
  }
}
`

type ProxyinitMutator struct {
	vn     *appmesh.VirtualNode
	config *Config
}

type ProxyInitMeta struct {
	InitImage          string
	Ports              string
	IgnoredIPs         string
	EgressIgnoredPorts string
	SidecarCpu         string
	SidecarMemory      string
}

func NewProxyInitMutator(Config *Config, vn *appmesh.VirtualNode) *ProxyinitMutator {
	return &ProxyinitMutator{
		vn:     vn,
		config: Config,
	}
}

func (m *ProxyinitMutator) Meta(pod *corev1.Pod) *ProxyInitMeta {
	var ports []string
	for _, listener := range m.vn.Spec.Listeners {
		ports = append(ports, fmt.Sprintf("%v", listener.PortMapping.Port))
	}
	if len(ports) == 0 {
		ports = []string{"0"}
	}
	return &ProxyInitMeta{
		InitImage:          m.config.InitImage,
		Ports:              strings.Join(ports, ","),
		IgnoredIPs:         m.config.IgnoredIPs,
		EgressIgnoredPorts: GetEgressIgnoredPorts(pod),
		SidecarCpu:         GetSidecarCpu(m.config, pod),
		SidecarMemory:      GetSidecarMemory(m.config, pod),
	}
}

func (m *ProxyinitMutator) mutate(pod *corev1.Pod) error {
	if IsAppMeshCNIEnabled(pod) {
		return nil
	}
	meta := m.Meta(pod)
	proxyInit, err := renderTemplate("init", initContainerTemplate, meta)
	if err != nil {
		return err
	}
	var container corev1.Container
	err = json.Unmarshal([]byte(proxyInit), &container)
	if err != nil {
		return err
	}
	pod.Spec.InitContainers = append(pod.Spec.InitContainers, container)
	return nil
}
