package inject

import (
	"encoding/json"
	corev1 "k8s.io/api/core/v1"
)

const jaegerEnvoyConfigTemplate = `
tracing:
 http:
  name: envoy.tracers.zipkin
  typed_config:
   "@type": type.googleapis.com/envoy.config.trace.v3.ZipkinConfig
   collector_cluster: jaeger
   collector_endpoint: "/api/v2/spans"
   collector_endpoint_version: HTTP_JSON
   shared_span_context: false
static_resources:
  clusters:
  - name: jaeger
    connect_timeout: 1s
    type: STRICT_DNS
    lb_policy: ROUND_ROBIN
    load_assignment:
      cluster_name: jaeger
      endpoints:
      - lb_endpoints:
        - endpoint:
           address:
            socket_address:
             address: {{ .JaegerAddress }}
             port_value: {{ .JaegerPort }}
`

const jaegerInitContainerTemplate = `
{
  "command": [
    "sh",
    "-c",
    "cat <<EOF >> /tmp/envoy/envoyconf.yaml{{ .EnvoyConfig }}EOF\n\ncat /tmp/envoy/envoyconf.yaml\n"
  ],
  "image": "public.ecr.aws/docker/library/busybox",
  "imagePullPolicy": "IfNotPresent",
  "name": "inject-jaeger-config",
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

type JaegerEnvoyConfigTemplateVariables struct {
	JaegerAddress string
	JaegerPort    string
}

type JaegerInitContainerTemplateVariables struct {
	EnvoyConfig                  string
	EnvoyTracingConfigVolumeName string
}

type jaegerMutatorConfig struct {
	jaegerAddress string
	jaegerPort    string
}

func newJaegerMutator(mutatorConfig jaegerMutatorConfig, enabled bool) *jaegerMutator {
	return &jaegerMutator{
		mutatorConfig: mutatorConfig,
		enabled:       enabled,
	}
}

type jaegerMutator struct {
	mutatorConfig jaegerMutatorConfig
	enabled       bool
}

func (m *jaegerMutator) mutate(pod *corev1.Pod) error {
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
	initContainer, err := renderTemplate("jaeger-init-container", jaegerInitContainerTemplate, variables)
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

func (m *jaegerMutator) buildEnvoyConfigTemplateVariables() JaegerEnvoyConfigTemplateVariables {
	return JaegerEnvoyConfigTemplateVariables{
		JaegerAddress: m.mutatorConfig.jaegerAddress,
		JaegerPort:    m.mutatorConfig.jaegerPort,
	}
}

func (m *jaegerMutator) buildInitContainerTemplateVariables() (JaegerInitContainerTemplateVariables, error) {
	envoyConfigVariables := m.buildEnvoyConfigTemplateVariables()
	envoyConfig, err := renderTemplate("jaeger-envoy-config", jaegerEnvoyConfigTemplate, envoyConfigVariables)
	if err != nil {
		return JaegerInitContainerTemplateVariables{}, err
	}
	envoyConfig, err = escapeYaml(envoyConfig)
	if err != nil {
		return JaegerInitContainerTemplateVariables{}, err
	}
	return JaegerInitContainerTemplateVariables{
		EnvoyConfig:                  envoyConfig,
		EnvoyTracingConfigVolumeName: envoyTracingConfigVolumeName,
	}, nil
}
