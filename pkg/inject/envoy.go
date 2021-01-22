package inject

import (
	"fmt"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"strconv"
	"strings"
)

const envoyTracingConfigVolumeName = "envoy-tracing-config"
const envoyContainerName = "envoy"

type EnvoyTemplateVariables struct {
	AWSRegion                    string
	MeshName                     string
	VirtualNodeName              string
	Preview                      string
	LogLevel                     string
	AdminAccessPort              int32
	AdminAccessLogFile           string
	PreStopDelay                 string
	SidecarImage                 string
	EnvoyTracingConfigVolumeName string
	EnableXrayTracing            bool
	XrayDaemonPort               int32
	EnableJaegerTracing          bool
	EnableDatadogTracing         bool
	DatadogTracerPort            int32
	DatadogTracerAddress         string
	EnableStatsTags              bool
	EnableStatsD                 bool
	StatsDPort                   int32
	StatsDAddress                string
}

type envoyMutatorConfig struct {
	accountID                  string
	awsRegion                  string
	preview                    bool
	logLevel                   string
	adminAccessPort            int32
	adminAccessLogFile         string
	preStopDelay               string
	readinessProbeInitialDelay int32
	readinessProbePeriod       int32
	sidecarImage               string
	sidecarCPURequests         string
	sidecarMemoryRequests      string
	sidecarCPULimits           string
	sidecarMemoryLimits        string
	enableXrayTracing          bool
	xrayDaemonPort             int32
	enableJaegerTracing        bool
	enableDatadogTracing       bool
	datadogTracerPort          int32
	datadogTracerAddress       string
	enableStatsTags            bool
	enableStatsD               bool
	statsDPort                 int32
	statsDAddress              string
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
	if ok, _ := containsEnvoyContainer(pod); ok {
		return nil
	}
	secretMounts, err := m.getSecretMounts(pod)
	if err != nil {
		return err
	}
	variables := m.buildTemplateVariables(pod)
	container := buildEnvoySidecar(variables)

	// add resource requests and limits
	container.Resources, err = sidecarResources(getSidecarCPURequest(m.mutatorConfig.sidecarCPURequests, pod),
		getSidecarMemoryRequest(m.mutatorConfig.sidecarMemoryRequests, pod),
		getSidecarCPULimit(m.mutatorConfig.sidecarCPULimits, pod),
		getSidecarMemoryLimit(m.mutatorConfig.sidecarMemoryLimits, pod))
	if err != nil {
		return err
	}

	// add readiness probe
	container.ReadinessProbe = envoyReadinessProbe(m.mutatorConfig.readinessProbeInitialDelay,
		m.mutatorConfig.readinessProbePeriod, strconv.Itoa(int(m.mutatorConfig.adminAccessPort)))

	m.mutateSecretMounts(pod, &container, secretMounts)
	pod.Spec.Containers = append(pod.Spec.Containers, container)
	return nil
}

func (m *envoyMutator) buildTemplateVariables(pod *corev1.Pod) EnvoyTemplateVariables {
	meshName := m.getAugmentedMeshName()
	virtualNodeName := aws.StringValue(m.vn.Spec.AWSName)
	preview := m.getPreview(pod)

	return EnvoyTemplateVariables{
		AWSRegion:                    m.mutatorConfig.awsRegion,
		MeshName:                     meshName,
		VirtualNodeName:              virtualNodeName,
		Preview:                      preview,
		LogLevel:                     m.mutatorConfig.logLevel,
		AdminAccessPort:              m.mutatorConfig.adminAccessPort,
		AdminAccessLogFile:           m.mutatorConfig.adminAccessLogFile,
		PreStopDelay:                 m.mutatorConfig.preStopDelay,
		SidecarImage:                 m.mutatorConfig.sidecarImage,
		EnvoyTracingConfigVolumeName: envoyTracingConfigVolumeName,
		EnableXrayTracing:            m.mutatorConfig.enableXrayTracing,
		XrayDaemonPort:               m.mutatorConfig.xrayDaemonPort,
		EnableJaegerTracing:          m.mutatorConfig.enableJaegerTracing,
		EnableDatadogTracing:         m.mutatorConfig.enableDatadogTracing,
		DatadogTracerPort:            m.mutatorConfig.datadogTracerPort,
		DatadogTracerAddress:         m.mutatorConfig.datadogTracerAddress,
		EnableStatsTags:              m.mutatorConfig.enableStatsTags,
		EnableStatsD:                 m.mutatorConfig.enableStatsD,
		StatsDPort:                   m.mutatorConfig.statsDPort,
		StatsDAddress:                m.mutatorConfig.statsDAddress,
	}
}

func (m *envoyMutator) getAugmentedMeshName() string {
	meshName := aws.StringValue(m.ms.Spec.AWSName)
	if m.ms.Spec.MeshOwner != nil && aws.StringValue(m.ms.Spec.MeshOwner) != m.mutatorConfig.accountID {
		return fmt.Sprintf("%v@%v", meshName, aws.StringValue(m.ms.Spec.MeshOwner))
	}
	return meshName
}

func (m *envoyMutator) getPreview(pod *corev1.Pod) string {
	preview := m.mutatorConfig.preview
	if v, ok := pod.ObjectMeta.Annotations[AppMeshPreviewAnnotation]; ok {
		preview = strings.ToLower(v) == "enabled"
	}
	if preview {
		return "1"
	}
	return "0"
}

func (m *envoyMutator) mutateSecretMounts(pod *corev1.Pod, envoyContainer *corev1.Container, secretMounts map[string]string) {
	for secretName, mountPath := range secretMounts {
		volume := corev1.Volume{
			Name: secretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: secretName,
				},
			},
		}
		volumeMount := corev1.VolumeMount{
			Name:      secretName,
			MountPath: mountPath,
			ReadOnly:  true,
		}
		envoyContainer.VolumeMounts = append(envoyContainer.VolumeMounts, volumeMount)
		pod.Spec.Volumes = append(pod.Spec.Volumes, volume)
	}
}

func (m *envoyMutator) getSecretMounts(pod *corev1.Pod) (map[string]string, error) {
	secretMounts := make(map[string]string)
	if v, ok := pod.ObjectMeta.Annotations[AppMeshSecretMountsAnnotation]; ok {
		for _, segment := range strings.Split(v, ",") {
			pair := strings.Split(segment, ":")
			if len(pair) != 2 { // secretName:mountPath
				return nil, errors.Errorf("malformed annotation %s, expected format: %s", AppMeshSecretMountsAnnotation, "secretName:mountPath")
			}
			secretName := strings.TrimSpace(pair[0])
			mountPath := strings.TrimSpace(pair[1])
			secretMounts[secretName] = mountPath
		}
	}
	return secretMounts, nil
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
