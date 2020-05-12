package inject

import (
	"encoding/json"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-sdk-go/aws"
	corev1 "k8s.io/api/core/v1"
	"strings"
)

const envoyTracingConfigVolumeName = "envoy-tracing-config"
const envoyContainerName = "envoy"

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
      "value": "{{ .Preview }}"
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
      "value": "{{ .AWSRegion }}"
    }{{ if .EnableXrayTracing }},
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
      "name": "{{ .EnvoyTracingConfigVolumeName }}"
    }
  ]{{ end }},
  "resources": {
    "requests": {
      "cpu": "{{ .SidecarCPURequests }}",
      "memory": "{{ .SidecarMemoryRequests }}"
    }
  }
}
`

type EnvoyTemplateVariables struct {
	AWSRegion                    string
	MeshName                     string
	VirtualNodeName              string
	Preview                      string
	LogLevel                     string
	SidecarImage                 string
	SidecarCPURequests           string
	SidecarMemoryRequests        string
	EnvoyTracingConfigVolumeName string
	EnableXrayTracing            bool
	EnableJaegerTracing          bool
	EnableDatadogTracing         bool
	EnableStatsTags              bool
	EnableStatsD                 bool
}

type envoyMutatorConfig struct {
	awsRegion             string
	preview               bool
	logLevel              string
	sidecarImage          string
	sidecarCPURequests    string
	sidecarMemoryRequests string
	enableXrayTracing     bool
	enableJaegerTracing   bool
	enableDatadogTracing  bool
	enableStatsTags       bool
	enableStatsD          bool
}

func newEnvoyMutator(mutatorConfig envoyMutatorConfig, ms *appmesh.Mesh, vn *appmesh.VirtualNode) *envoyMutator {
	return &envoyMutator{
		vn:            vn,
		ms:            ms,
		mutatorConfig: mutatorConfig,
	}
}

type envoyMutator struct {
	vn            *appmesh.VirtualNode
	ms            *appmesh.Mesh
	mutatorConfig envoyMutatorConfig
}

func (m *envoyMutator) mutate(pod *corev1.Pod) error {
	if containsEnvoyContainer(pod) {
		return nil
	}
	variables := m.buildTemplateVariables(pod)
	envoySidecar, err := renderTemplate("envoy", envoyContainerTemplate, variables)
	if err != nil {
		return err
	}
	container := corev1.Container{}
	err = json.Unmarshal([]byte(envoySidecar), &container)
	if err != nil {
		return err
	}
	pod.Spec.Containers = append(pod.Spec.Containers, container)
	return nil
}

func (m *envoyMutator) buildTemplateVariables(pod *corev1.Pod) EnvoyTemplateVariables {
	meshName := aws.StringValue(m.ms.Spec.AWSName)
	virtualNodeName := aws.StringValue(m.vn.Spec.AWSName)
	preview := m.getPreview(pod)

	return EnvoyTemplateVariables{
		AWSRegion:                    m.mutatorConfig.awsRegion,
		MeshName:                     meshName,
		VirtualNodeName:              virtualNodeName,
		Preview:                      preview,
		LogLevel:                     m.mutatorConfig.logLevel,
		SidecarImage:                 m.mutatorConfig.sidecarImage,
		SidecarCPURequests:           getSidecarCPURequest(m.mutatorConfig.sidecarCPURequests, pod),
		SidecarMemoryRequests:        getSidecarMemoryRequest(m.mutatorConfig.sidecarMemoryRequests, pod),
		EnvoyTracingConfigVolumeName: envoyTracingConfigVolumeName,
		EnableXrayTracing:            m.mutatorConfig.enableXrayTracing,
		EnableJaegerTracing:          m.mutatorConfig.enableJaegerTracing,
		EnableDatadogTracing:         m.mutatorConfig.enableDatadogTracing,
		EnableStatsTags:              m.mutatorConfig.enableStatsTags,
		EnableStatsD:                 m.mutatorConfig.enableStatsD,
	}
}

func (m *envoyMutator) getPreview(pod *corev1.Pod) string {
	preview := m.mutatorConfig.preview
	if v, ok := pod.ObjectMeta.Annotations[AppMeshPreviewAnnotation]; ok {
		preview = strings.ToLower(v) == "true"
	}
	if preview {
		return "1"
	}
	return "0"
}

// containsEnvoyContainer checks whether pod already contains "envoy" container
func containsEnvoyContainer(pod *corev1.Pod) bool {
	for _, container := range pod.Spec.Containers {
		if container.Name == envoyContainerName {
			return true
		}
	}
	return false
}

// containsEnvoyTracingConfigVolume checks whether pod already contains "envoy-tracing-config" volume
func containsEnvoyTracingConfigVolume(pod *corev1.Pod) bool {
	for _, volume := range pod.Spec.Volumes {
		if volume.Name == envoyTracingConfigVolumeName {
			return true
		}
	}
	return false
}
