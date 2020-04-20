package appmeshinject

import (
	"encoding/json"
	corev1 "k8s.io/api/core/v1"
)

const xrayDaemonContainerTemplate = `
{
  "name": "xray-daemon",
  "image": "amazon/aws-xray-daemon",
  "securityContext": {
    "runAsUser": 1337
  },
  "ports": [
    {
      "containerPort": 2000,
      "name": "xray",
      "protocol": "UDP"
    }
  ],
  "env": [
    {
      "name": "AWS_REGION",
      "value": "{{ .Region }}"
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

type XrayMutator struct {
}

func (m *XrayMutator) mutate(pod *corev1.Pod) error {
	if !config.InjectXraySidecar {
		return nil
	}
	newConfig := updateConfigFromPodAnnotations(config, pod)
	xrayDaemonSidecar, err := renderTemplate("xray-daemon", xrayDaemonContainerTemplate, newConfig)
	if err != nil {
		return err
	}
	var container corev1.Container
	err = json.Unmarshal([]byte(xrayDaemonSidecar), &container)
	if err != nil {
		return err
	}
	pod.Spec.Containers = append(pod.Spec.Containers, container)
	return nil
}
