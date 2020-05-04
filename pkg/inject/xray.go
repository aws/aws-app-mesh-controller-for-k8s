package inject

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
      "value": "{{ .AWSRegion }}"
    }
  ],
  "resources": {
    "requests": {
      "cpu": "{{ .XraySidecarCpu }}",
      "memory": "{{ .XraySidecarMemory }}"
    }
  }
}
`

type XrayMutator struct {
	config *Config
}

type XrayMeta struct {
	AWSRegion         string
	XraySidecarCpu    string
	XraySidecarMemory string
}

func NewXrayMutator(Config *Config) *XrayMutator {
	return &XrayMutator{config: Config}
}

func (m *XrayMutator) Meta(pod *corev1.Pod) *XrayMeta {
	return &XrayMeta{
		AWSRegion:         m.config.Region,
		XraySidecarCpu:    GetSidecarCpu(m.config, pod),
		XraySidecarMemory: GetSidecarMemory(m.config, pod),
	}
}

func (m *XrayMutator) mutate(pod *corev1.Pod) error {
	if !m.config.EnableXrayTracing {
		return nil
	}
	xrayMeta := m.Meta(pod)
	xrayDaemonSidecar, err := renderTemplate("xray-daemon", xrayDaemonContainerTemplate, xrayMeta)
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
