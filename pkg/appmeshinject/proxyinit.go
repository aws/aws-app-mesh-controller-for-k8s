package appmeshinject

import (
	"encoding/json"
	corev1 "k8s.io/api/core/v1"
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
}

func (m *ProxyinitMutator) mutate(pod *corev1.Pod) error {
	if isAppMeshCNIEnabled(pod) {
		return nil
	}
	meta := struct {
		Config
		Ports string
	}{
		Config: updateConfigFromPodAnnotations(config, pod),
		Ports:  getPortsFromContainers(pod.Spec.Containers),
	}

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
