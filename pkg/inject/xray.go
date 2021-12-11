package inject

import (
	"encoding/json"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"strings"
)

const xrayDaemonContainerName = "xray-daemon"
const xrayDaemonContainerTemplate = `
{
  "name": "xray-daemon",
  "image": "{{ .XRayImage }}",
  "command": "-l {{ .XrayLogLevel }}",
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
	XrayLogLevel   string
}

type xrayMutatorConfig struct {
	awsRegion             string
	sidecarCPURequests    string
	sidecarMemoryRequests string
	sidecarCPULimits      string
	sidecarMemoryLimits   string
	xRayImage             string
	xRayDaemonPort        int32
	xRayLogLevel          string
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

	err := m.checkConfig()
	if err != nil {
		return err
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
		XrayLogLevel:   m.mutatorConfig.xRayLogLevel,
	}
}

func (m *xrayMutator) checkConfig() error {
	var missingConfig []string

	if m.mutatorConfig.awsRegion == "" {
		missingConfig = append(missingConfig, "AWSRegion")
	}
	if m.mutatorConfig.xRayImage == "" {
		missingConfig = append(missingConfig, "xRayImage")
	}

	if m.mutatorConfig.xRayDaemonPort == 0 {
		missingConfig = append(missingConfig, "xRayDaemonPort")
	}

	if m.mutatorConfig.xRayLogLevel == "" {
		missingConfig = append(missingConfig, "xRayLogLevel")
	}

	if len(missingConfig) > 0 {
		return errors.Errorf("Missing configuration parameters: %s", strings.Join(missingConfig, ","))
	}

	return nil
}

func containsXRAYDaemonContainer(pod *corev1.Pod) bool {
	for _, container := range pod.Spec.Containers {
		if container.Name == xrayDaemonContainerName {
			return true
		}
	}
	return false
}
