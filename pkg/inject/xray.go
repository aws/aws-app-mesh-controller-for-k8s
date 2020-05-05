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
      "cpu": "{{ .SidecarCPURequests }}",
      "memory": "{{ .SidecarMemoryRequests }}"
    }
  }
}
`

type XrayTemplateVariables struct {
	AWSRegion             string
	SidecarCPURequests    string
	SidecarMemoryRequests string
}

type xrayMutatorConfig struct {
	awsRegion             string
	sidecarCPURequests    string
	sidecarMemoryRequests string
}

func newXrayMutator(mutatorConfig xrayMutatorConfig, enabled bool) *xrayMutator {
	return &xrayMutator{
		mutatorConfig: mutatorConfig,
		enabled:       enabled,
	}
}

type xrayMutator struct {
	mutatorConfig xrayMutatorConfig
	enabled       bool
}

func (m *xrayMutator) mutate(pod *corev1.Pod) error {
	if !m.enabled {
		return nil
	}
	variables := m.buildTemplateVariables(pod)
	xrayDaemonSidecar, err := renderTemplate("xray-daemon", xrayDaemonContainerTemplate, variables)
	if err != nil {
		return err
	}
	container := corev1.Container{}
	err = json.Unmarshal([]byte(xrayDaemonSidecar), &container)
	if err != nil {
		return err
	}
	pod.Spec.Containers = append(pod.Spec.Containers, container)
	return nil
}

func (m *xrayMutator) buildTemplateVariables(pod *corev1.Pod) XrayTemplateVariables {
	return XrayTemplateVariables{
		AWSRegion:             m.mutatorConfig.awsRegion,
		SidecarCPURequests:    getSidecarCPURequest(m.mutatorConfig.sidecarCPURequests, pod),
		SidecarMemoryRequests: getSidecarMemoryRequest(m.mutatorConfig.sidecarMemoryRequests, pod),
	}
}
