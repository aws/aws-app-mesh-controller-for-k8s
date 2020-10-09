package inject

import (
	"encoding/json"
	corev1 "k8s.io/api/core/v1"
)

// Creating a template to avoid relying on an extra ConfigMap
const datadogEnvoyConfigTemplate = `
tracing:
  http:
    name: envoy.tracers.datadog
    config:
      collector_cluster: datadog_agent
      service_name: envoy
static_resources:
  clusters:
  - name: datadog_agent
    connect_timeout: 1s
    type: STRICT_DNS
    lb_policy: ROUND_ROBIN
    load_assignment:
      cluster_name: datadog_agent
      endpoints:
      - lb_endpoints:
        - endpoint:
           address:
            socket_address:
             address: {{ .DatadogAddress }}
             port_value: {{ .DatadogPort }}
`

const datadogInitContainerTemplate = `
{
  "command": [
    "sh",
    "-c",
    "cat <<EOF >> /tmp/envoy/envoyconf.yaml{{ .EnvoyConfig }}EOF\n\ncat /tmp/envoy/envoyconf.yaml\n"
  ],
  "image": "busybox",
  "imagePullPolicy": "IfNotPresent",
  "name": "inject-datadog-config",
  "volumeMounts": [
    {
      "mountPath": "/tmp/envoy",
      "name": "{{ .EnvoyTracingConfigVolumeName }}"
    }
  ],
  "resources": {
    "limits": {
      "cpu": "100m",
      "memory": "64Mi"
    },
    "requests": {
      "cpu": "10m",
      "memory": "32Mi"
    }
  }
}
`

type DatadogEnvoyConfigTemplateVariables struct {
	DatadogAddress string
	DatadogPort    string
}

type DatadogInitContainerTemplateVariables struct {
	EnvoyConfig                  string
	EnvoyTracingConfigVolumeName string
}

type datadogMutatorConfig struct {
	datadogAddress string
	datadogPort    string
}

func newDatadogMutator(mutatorConfig datadogMutatorConfig, enabled bool) *datadogMutator {
	return &datadogMutator{
		mutatorConfig: mutatorConfig,
		enabled:       enabled,
	}
}

type datadogMutator struct {
	mutatorConfig datadogMutatorConfig
	enabled       bool
}

func (m *datadogMutator) mutate(pod *corev1.Pod) error {
	if !m.enabled {
		return nil
	}
	if containsEnvoyTracingConfigVolume(pod) {
		return nil
	}
	variables, err := m.buildInitContainerTemplateVariables()
	if err != nil {
		return err
	}
	initContainer, err := renderTemplate("datadog-init-container", datadogInitContainerTemplate, variables)
	if err != nil {
		return err
	}
	container := corev1.Container{}
	err = json.Unmarshal([]byte(initContainer), &container)
	if err != nil {
		return err
	}
	pod.Spec.InitContainers = append(pod.Spec.InitContainers, container)
	volume := corev1.Volume{Name: envoyTracingConfigVolumeName, VolumeSource: corev1.VolumeSource{
		EmptyDir: &corev1.EmptyDirVolumeSource{},
	}}
	pod.Spec.Volumes = append(pod.Spec.Volumes, volume)
	return nil
}

func (m *datadogMutator) buildEnvoyConfigTemplateVariables() DatadogEnvoyConfigTemplateVariables {
	return DatadogEnvoyConfigTemplateVariables{
		DatadogAddress: m.mutatorConfig.datadogAddress,
		DatadogPort:    m.mutatorConfig.datadogPort,
	}
}

func (m *datadogMutator) buildInitContainerTemplateVariables() (DatadogInitContainerTemplateVariables, error) {
	envoyConfigVariables := m.buildEnvoyConfigTemplateVariables()
	envoyConfig, err := renderTemplate("datadog-envoy-config", datadogEnvoyConfigTemplate, envoyConfigVariables)
	if err != nil {
		return DatadogInitContainerTemplateVariables{}, err
	}
	envoyConfig, err = escapeYaml(envoyConfig)
	if err != nil {
		return DatadogInitContainerTemplateVariables{}, err
	}
	return DatadogInitContainerTemplateVariables{
		EnvoyConfig:                  envoyConfig,
		EnvoyTracingConfigVolumeName: envoyTracingConfigVolumeName,
	}, nil
}
