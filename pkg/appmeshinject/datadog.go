package appmeshinject

import (
	corev1 "k8s.io/api/core/v1"
)

// Creating a template to avoid relying on an extra ConfigMap
const datadogTemplate = `
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
    type: strict_dns
    lb_policy: round_robin
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

const injectDatadogTemplate = `
{
  "command": [
    "sh",
    "-c",
    "cat <<EOF >> /tmp/envoy/envoyconf.yaml{{ .Config }}EOF\n\ncat /tmp/envoy/envoyconf.yaml\n"
  ],
  "image": "busybox",
  "imagePullPolicy": "IfNotPresent",
  "name": "inject-datadog-config",
  "volumeMounts": [
    {
      "mountPath": "/tmp/envoy",
      "name": "config"
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

type DatadogMutator struct {
}

func (d *DatadogMutator) mutate(pod *corev1.Pod) error {
	if !config.EnableDatadogTracing {
		return nil
	}
	init, err := renderInitContainer("datadog", datadogTemplate, injectDatadogTemplate, config)
	if err != nil {
		return err
	}
	pod.Spec.InitContainers = append(pod.Spec.InitContainers, *init)
	pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{Name: tracingConfigVolumeName})
	return nil
}
