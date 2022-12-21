package inject

import (
	"fmt"
	"strconv"
	"strings"

	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)

const envoyContainerName = "envoy"

type envoyMutatorConfig struct {
	accountID                  string
	awsRegion                  string
	preview                    bool
	enableSDS                  bool
	sdsUdsPath                 string
	logLevel                   string
	adminAccessPort            int32
	adminAccessLogFile         string
	preStopDelay               string
	postStartTimeout           int32
	postStartInterval          int32
	readinessProbeInitialDelay int32
	readinessProbePeriod       int32
	sidecarImageRepository     string
	sidecarImageTag            string
	sidecarCPURequests         string
	sidecarMemoryRequests      string
	sidecarCPULimits           string
	sidecarMemoryLimits        string
	enableXrayTracing          bool
	xrayDaemonPort             int32
	xraySamplingRate           string
	enableJaegerTracing        bool
	jaegerPort                 string
	jaegerAddress              string
	enableDatadogTracing       bool
	datadogTracerPort          int32
	datadogTracerAddress       string
	enableStatsTags            bool
	enableStatsD               bool
	statsDPort                 int32
	statsDAddress              string
	statsDSocketPath           string
	waitUntilProxyReady        bool
	controllerVersion          string
	k8sVersion                 string
	useDualStackEndpoint       bool
	enableAdminAccessIPv6      bool
	useFipsEndpoint            bool
	awsAccessKeyId             string
	awsSecretAccessKey         string
	awsSessionToken            string
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

	volumeMounts, err := m.getVolumeMounts(pod)
	if err != nil {
		return err
	}
	variables := m.buildTemplateVariables(pod)

	customEnv, err := m.getCustomEnv(pod)
	if err != nil {
		return err
	}

	container, err := buildEnvoySidecar(variables, customEnv)
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

	// add readiness probe
	container.ReadinessProbe = envoyReadinessProbe(m.mutatorConfig.readinessProbeInitialDelay,
		m.mutatorConfig.readinessProbePeriod, strconv.Itoa(int(m.mutatorConfig.adminAccessPort)))

	m.mutateSecretMounts(pod, &container, secretMounts)
	m.mutateVolumeMounts(pod, &container, volumeMounts)
	if m.mutatorConfig.enableSDS && !isSDSDisabled(pod) {
		mutateSDSMounts(pod, &container, m.mutatorConfig.sdsUdsPath)
	}

	// waitUntilProxyReady requires starting sidecar container first
	if m.mutatorConfig.waitUntilProxyReady {
		pod.Spec.Containers = append([]corev1.Container{container}, pod.Spec.Containers...)
	} else {
		pod.Spec.Containers = append(pod.Spec.Containers, container)
	}

	return nil
}

func (m *envoyMutator) buildTemplateVariables(pod *corev1.Pod) EnvoyTemplateVariables {
	meshName := m.getAugmentedMeshName()
	virtualNodeName := aws.StringValue(m.vn.Spec.AWSName)
	preview := m.getPreview(pod)
	useDualStackEndpoint := m.getUseDualStackEndpoint(m.mutatorConfig.useDualStackEndpoint)
	useFipsEndpoint := m.getUseFipsEndpoint(m.mutatorConfig.useFipsEndpoint)
	sdsEnabled := m.mutatorConfig.enableSDS
	if m.mutatorConfig.enableSDS && isSDSDisabled(pod) {
		sdsEnabled = false
	}

	return EnvoyTemplateVariables{
		AWSRegion:                m.mutatorConfig.awsRegion,
		MeshName:                 meshName,
		VirtualGatewayOrNodeName: virtualNodeName,
		Preview:                  preview,
		EnableSDS:                sdsEnabled,
		SdsUdsPath:               m.mutatorConfig.sdsUdsPath,
		LogLevel:                 m.mutatorConfig.logLevel,
		AdminAccessPort:          m.mutatorConfig.adminAccessPort,
		AdminAccessLogFile:       m.mutatorConfig.adminAccessLogFile,
		PreStopDelay:             m.mutatorConfig.preStopDelay,
		PostStartTimeout:         m.mutatorConfig.postStartTimeout,
		PostStartInterval:        m.mutatorConfig.postStartInterval,
		SidecarImageRepository:   m.mutatorConfig.sidecarImageRepository,
		SidecarImageTag:          m.mutatorConfig.sidecarImageTag,
		EnableXrayTracing:        m.mutatorConfig.enableXrayTracing,
		XrayDaemonPort:           m.mutatorConfig.xrayDaemonPort,
		XraySamplingRate:         m.mutatorConfig.xraySamplingRate,
		EnableJaegerTracing:      m.mutatorConfig.enableJaegerTracing,
		JaegerPort:               m.mutatorConfig.jaegerPort,
		JaegerAddress:            m.mutatorConfig.jaegerAddress,
		EnableDatadogTracing:     m.mutatorConfig.enableDatadogTracing,
		DatadogTracerPort:        m.mutatorConfig.datadogTracerPort,
		DatadogTracerAddress:     m.mutatorConfig.datadogTracerAddress,
		EnableStatsTags:          m.mutatorConfig.enableStatsTags,
		EnableStatsD:             m.mutatorConfig.enableStatsD,
		StatsDPort:               m.mutatorConfig.statsDPort,
		StatsDAddress:            m.mutatorConfig.statsDAddress,
		StatsDSocketPath:         m.mutatorConfig.statsDSocketPath,
		ControllerVersion:        m.mutatorConfig.controllerVersion,
		K8sVersion:               m.mutatorConfig.k8sVersion,
		UseDualStackEndpoint:     useDualStackEndpoint,
		EnableAdminAccessForIpv6: m.mutatorConfig.enableAdminAccessIPv6,
		WaitUntilProxyReady:      m.mutatorConfig.waitUntilProxyReady,
		UseFipsEndpoint:          useFipsEndpoint,
		AwsAccessKeyId:           m.mutatorConfig.awsAccessKeyId,
		AwsSecretAccessKey:       m.mutatorConfig.awsSecretAccessKey,
		AwsSessionToken:          m.mutatorConfig.awsSessionToken,
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

func (m *envoyMutator) getCustomEnv(pod *corev1.Pod) (map[string]string, error) {
	customEnv := make(map[string]string)
	if v, ok := pod.ObjectMeta.Annotations[AppMeshEnvAnnotation]; ok {
		for _, segment := range strings.Split(v, ",") {
			pair := strings.Split(segment, "=")
			if len(pair) != 2 { // EnvVariableKey=EnvVariableValue
				return nil, errors.Errorf("malformed annotation %s, expected format: %s", AppMeshEnvAnnotation, "EnvVariableKey=EnvVariableValue")
			}
			envKey := strings.TrimSpace(pair[0])
			envVal := strings.TrimSpace(pair[1])
			customEnv[envKey] = envVal
		}
	}
	return customEnv, nil
}

func (m *envoyMutator) mutateVolumeMounts(pod *corev1.Pod, envoyContainer *corev1.Container, volumeMounts map[string]string) {
	for volumeName, mountPath := range volumeMounts {
		volumeMount := corev1.VolumeMount{
			Name:      volumeName,
			MountPath: mountPath,
			ReadOnly:  true,
		}
		envoyContainer.VolumeMounts = append(envoyContainer.VolumeMounts, volumeMount)
	}
}

func (m *envoyMutator) getVolumeMounts(pod *corev1.Pod) (map[string]string, error) {
	volumeMounts := make(map[string]string)
	if v, ok := pod.ObjectMeta.Annotations[AppMeshVolumeMountsAnnotation]; ok {
		for _, segment := range strings.Split(v, ",") {
			pair := strings.Split(segment, ":")
			if len(pair) != 2 { // volumeName:mountPath
				return nil, errors.Errorf("malformed annotation %s, expected format: %s", AppMeshVolumeMountsAnnotation, "volumeName:mountPath")
			}
			volumeName := strings.TrimSpace(pair[0])
			mountPath := strings.TrimSpace(pair[1])
			volumeMounts[volumeName] = mountPath
		}
	}
	return volumeMounts, nil
}

func (m *envoyMutator) getUseDualStackEndpoint(useDualStackEndpoint bool) string {
	if useDualStackEndpoint {
		return "1"
	} else {
		return "0"
	}
}

func (m *envoyMutator) getUseFipsEndpoint(useFipsEndpoint bool) string {
	if useFipsEndpoint {
		return "1"
	} else {
		return "0"
	}
}
