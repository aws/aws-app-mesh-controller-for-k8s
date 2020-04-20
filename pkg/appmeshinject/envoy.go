package appmeshinject

import (
	"encoding/json"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/k8s"
	corev1 "k8s.io/api/core/v1"
)

const envoyContainerTemplate = `
{
  "name": "envoy",
  "image": "{{ .SidecarImage }}",
  "securityContext": {
    "runAsUser": 1337
  },
  "ports": [
    {
      "containerPort": 9901,
      "name": "stats",
      "protocol": "TCP"
    }
  ],
  "env": [
    {
      "name": "APPMESH_VIRTUAL_NODE_NAME",
      "value": "mesh/{{ .MeshName }}/virtualNode/{{ .VirtualNodeName }}"
    },
    {
      "name": "APPMESH_PREVIEW",
      "value": "{{ .AppMeshPreview }}"
    },
    {
      "name": "ENVOY_LOG_LEVEL",
      "value": "{{ .LogLevel }}"
    }{{ if or .EnableJaegerTracing .EnableDatadogTracing }},
    {
      "name": "ENVOY_STATS_CONFIG_FILE",
      "value": "/tmp/envoy/envoyconf.yaml"
    }{{ end }},
    {
      "name": "AWS_REGION",
      "value": "{{ .Region }}"
    }{{ if .InjectXraySidecar }},
    {
      "name": "ENABLE_ENVOY_XRAY_TRACING",
      "value": "1"
    }{{ end }}{{ if .EnableStatsTags }},
    {
      "name": "ENABLE_ENVOY_STATS_TAGS",
      "value": "1"
    }{{ end }}{{ if .EnableStatsD }},
    {
      "name": "ENABLE_ENVOY_DOG_STATSD",
      "value": "1"
    }{{ end }}
  ]{{ if or .EnableJaegerTracing .EnableDatadogTracing }},
  "volumeMounts": [
    {
      "mountPath": "/tmp/envoy",
      "name": "envoy-tracing-config"
    }
  ]{{ end }},
  "resources": {
    "requests": {
      "cpu": "{{ .SidecarCpu }}",
      "memory": "{{ .SidecarMemory }}"
    }
  }
}
`

type EnvoyMutator struct {
	vn appmesh.VirtualNode
}

func (m *EnvoyMutator) mutate(pod *corev1.Pod) error {
	meshName := m.vn.Spec.MeshRef.Name
	virtualNodeName := k8s.NamespacedName(&m.vn).String()
	sidecarMeta := struct {
		Config
		MeshName        string
		VirtualNodeName string
		AppMeshPreview  string
	}{
		Config:          updateConfigFromPodAnnotations(config, pod),
		MeshName:        meshName,
		VirtualNodeName: virtualNodeName,
		AppMeshPreview:  "0",
	}

	if v, ok := pod.ObjectMeta.Annotations[AppMeshPreviewAnnotation]; ok {
		if v == "true" {
			sidecarMeta.AppMeshPreview = "1"
		}
	} else {
		if config.Preview {
			sidecarMeta.AppMeshPreview = "1"
		}
	}

	envoySidecar, err := renderTemplate("envoy", envoyContainerTemplate, sidecarMeta)
	if err != nil {
		return err
	}
	var container corev1.Container
	err = json.Unmarshal([]byte(envoySidecar), &container)
	if err != nil {
		return err
	}
	pod.Spec.Containers = append(pod.Spec.Containers, container)
	pod.Annotations[AppMeshVirtualNodeNameAnnotation] = virtualNodeName
	return nil
}
