package inject

import (
	"encoding/json"
	corev1 "k8s.io/api/core/v1"
)

const xrayDaemonContainerName = "xray-daemon"
const xrayDaemonContainerTemplate = `
{
  "name": "xray-daemon",
  "image": "{{ .XRayImage }}",
  "securityContext": {
    "runAsUser": 1337
  },
  "ports": [
    {
      "containerPort": {{ .XrayDaemonPort }},
      "name": "xray",
      "protocol": "UDP"
    }
  ],
  "env": [
    {
      "name": "AWS_REGION",
      "value": "{{ .AWSRegion }}"
    }
  ]
}
`

type XrayTemplateVariables struct {
	AWSRegion      string
	XRayImage      string
	XrayDaemonPort int32
}

type xrayMutatorConfig struct {
	awsRegion             string
	sidecarCPURequests    string
	sidecarMemoryRequests string
	sidecarCPULimits      string
	sidecarMemoryLimits   string
	xRayImage             string
	xRayDaemonPort        int32
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
	if containsXRAYDaemonContainer(pod) {
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

	// add resource requests and limits
	container.Resources, err = sidecarResources(getSidecarCPURequest(m.mutatorConfig.sidecarCPURequests, pod),
		getSidecarMemoryRequest(m.mutatorConfig.sidecarMemoryRequests, pod),
		getSidecarCPULimit(m.mutatorConfig.sidecarCPULimits, pod),
		getSidecarMemoryLimit(m.mutatorConfig.sidecarMemoryLimits, pod))
	if err != nil {
		return err
	}

	pod.Spec.Containers = append(pod.Spec.Containers, container)
	return nil
}

func (m *xrayMutator) buildTemplateVariables(pod *corev1.Pod) XrayTemplateVariables {
	return XrayTemplateVariables{
		AWSRegion:      m.mutatorConfig.awsRegion,
		XRayImage:      m.mutatorConfig.xRayImage,
		XrayDaemonPort: m.mutatorConfig.xRayDaemonPort,
	}
}

func containsXRAYDaemonContainer(pod *corev1.Pod) bool {
	for _, container := range pod.Spec.Containers {
		if container.Name == xrayDaemonContainerName {
			return true
		}
	}
	return false
}
