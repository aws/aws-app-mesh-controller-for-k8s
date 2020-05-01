package inject

import (
	corev1 "k8s.io/api/core/v1"
)

const jaegerTemplate = `
tracing:
 http:
  name: envoy.zipkin
  typed_config:
   "@type": type.googleapis.com/envoy.config.trace.v2.ZipkinConfig
   collector_cluster: jaeger
   collector_endpoint: "/api/v1/spans"
   shared_span_context: false
static_resources:
  clusters:
  - name: jaeger
    connect_timeout: 1s
    type: strict_dns
    lb_policy: round_robin
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

const injectJaegerTemplate = `
{
  "command": [
    "sh",
    "-c",
    "cat <<EOF >> /tmp/envoy/envoyconf.yaml{{ .EnvoyConfig }}EOF\n\ncat /tmp/envoy/envoyconf.yaml\n"
  ],
  "image": "busybox",
  "imagePullPolicy": "IfNotPresent",
  "name": "inject-jaeger-config",
  "volumeMounts": [
    {
      "mountPath": "/tmp/envoy",
      "name": "envoy-tracing-config"
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

type JaegerMutator struct {
	config *Config
}

type JaegerMeta struct {
	JaegerAddress string
	JaegerPort    string
}

func NewJaegerMutator(Config *Config) *JaegerMutator {
	return &JaegerMutator{config: Config}
}

func (j *JaegerMutator) Meta(pod *corev1.Pod) *JaegerMeta {
	return &JaegerMeta{
		JaegerAddress: j.config.JaegerAddress,
		JaegerPort:    j.config.JaegerPort,
	}
}

func (j *JaegerMutator) mutate(pod *corev1.Pod) error {
	if !j.config.EnableJaegerTracing {
		return nil
	}
	jaegerMeta := j.Meta(pod)
	init, err := renderInitContainer("jaeger", jaegerTemplate, injectJaegerTemplate, jaegerMeta)
	if err != nil {
		return err
	}
	pod.Spec.InitContainers = append(pod.Spec.InitContainers, *init)
	pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{Name: tracingConfigVolumeName})
	return nil
}
