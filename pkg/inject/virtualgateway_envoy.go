package inject

import (
	"fmt"
	"strconv"
	"strings"

	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-sdk-go/aws"
	corev1 "k8s.io/api/core/v1"
)

type virtualGatwayEnvoyConfig struct {
	accountID                  string
	awsRegion                  string
	preview                    bool
	enableSDS                  bool
	sdsUdsPath                 string
	logLevel                   string
	adminAccessPort            int32
	adminAccessLogFile         string
	sidecarImageRepository     string
	sidecarImageTag            string
	readinessProbeInitialDelay int32
	readinessProbePeriod       int32
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
	controllerVersion          string
	k8sVersion                 string
	useDualStackEndpoint       bool
	enableAdminAccessIPv6      bool
	useFipsEndpoint            bool
	awsAccessKeyId             string
	awsSecretAccessKey         string
	awsSessionToken            string
}

// newVirtualGatewayEnvoyConfig constructs new newVirtualGatewayEnvoyConfig
func newVirtualGatewayEnvoyConfig(mutatorConfig virtualGatwayEnvoyConfig, ms *appmesh.Mesh, vg *appmesh.VirtualGateway) *virtualGatewayEnvoyConfig {
	return &virtualGatewayEnvoyConfig{
		ms:            ms,
		mutatorConfig: mutatorConfig,
		vg:            vg,
	}
}

var _ PodMutator = &virtualGatewayEnvoyConfig{}

// mutator adding a virtualgateway config to envoy pod
type virtualGatewayEnvoyConfig struct {
	vg            *appmesh.VirtualGateway
	ms            *appmesh.Mesh
	mutatorConfig virtualGatwayEnvoyConfig
}

func (m *virtualGatewayEnvoyConfig) mutate(pod *corev1.Pod) error {
	ok, envoyIdx := containsEnvoyContainer(pod)
	if !ok {
		return nil
	}

	variables := m.buildTemplateVariables(pod)
	envoy := pod.Spec.Containers[envoyIdx]

	vg := fmt.Sprintf("mesh/%s/virtualGateway/%s", variables.MeshName, variables.VirtualGatewayOrNodeName)

	envMap := map[string]string{}
	if err := updateEnvMapForEnvoy(variables, envMap, vg); err != nil {
		return err
	}

	//we override the image to latest Envoy so customers do not have to manually manage
	// envoy versions and let controller handle consistency versions across the mesh
	if m.virtualGatewayImageOverride(pod) {
		envoy.Image = fmt.Sprintf("%s:%s", m.mutatorConfig.sidecarImageRepository, m.mutatorConfig.sidecarImageTag)
	}

	for idx, env := range pod.Spec.Containers[envoyIdx].Env {
		if val, ok := envMap[env.Name]; ok {
			if val != env.Value {
				envoy.Env[idx].Value = val
			}
			delete(envMap, env.Name)
		}
	}

	env := getEnvoyEnv(envMap)
	envoy.Env = append(envoy.Env, env...)

	// customer can bring their own envoy image/spec for virtual gateway so we will only set readiness probe if not already set
	if envoy.ReadinessProbe == nil {
		envoy.ReadinessProbe = envoyReadinessProbe(m.mutatorConfig.readinessProbeInitialDelay,
			m.mutatorConfig.readinessProbePeriod, strconv.Itoa(int(m.mutatorConfig.adminAccessPort)))
	}

	if m.mutatorConfig.enableSDS && !isSDSDisabled(pod) {
		mutateSDSMounts(pod, &envoy, m.mutatorConfig.sdsUdsPath)
	}
	pod.Spec.Containers[envoyIdx] = envoy
	return nil
}

func (m *virtualGatewayEnvoyConfig) buildTemplateVariables(pod *corev1.Pod) EnvoyTemplateVariables {
	meshName := m.getAugmentedMeshName()
	virtualGatewayName := aws.StringValue(m.vg.Spec.AWSName)
	preview := m.getPreview(pod)
	useDualStackEndpoint := m.getUseDualStackEndpoint(m.mutatorConfig.useDualStackEndpoint)
	sdsEnabled := m.mutatorConfig.enableSDS
	useFipsEndpoint := m.getUseFipsEndpoint(m.mutatorConfig.useFipsEndpoint)
	if m.mutatorConfig.enableSDS && isSDSDisabled(pod) {
		sdsEnabled = false
	}

	return EnvoyTemplateVariables{
		AWSRegion:                m.mutatorConfig.awsRegion,
		MeshName:                 meshName,
		VirtualGatewayOrNodeName: virtualGatewayName,
		Preview:                  preview,
		EnableSDS:                sdsEnabled,
		SdsUdsPath:               m.mutatorConfig.sdsUdsPath,
		LogLevel:                 m.mutatorConfig.logLevel,
		AdminAccessPort:          m.mutatorConfig.adminAccessPort,
		AdminAccessLogFile:       m.mutatorConfig.adminAccessLogFile,
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
		UseFipsEndpoint:          useFipsEndpoint,
		AwsAccessKeyId:           m.mutatorConfig.awsAccessKeyId,
		AwsSecretAccessKey:       m.mutatorConfig.awsSecretAccessKey,
		AwsSessionToken:          m.mutatorConfig.awsSessionToken,
	}
}

func (m *virtualGatewayEnvoyConfig) getPreview(pod *corev1.Pod) string {
	preview := m.mutatorConfig.preview
	if v, ok := pod.ObjectMeta.Annotations[AppMeshPreviewAnnotation]; ok {
		preview = strings.ToLower(v) == "enabled"
	}
	if preview {
		return "1"
	}
	return "0"
}

func (m *virtualGatewayEnvoyConfig) getAugmentedMeshName() string {
	meshName := aws.StringValue(m.ms.Spec.AWSName)
	if m.ms.Spec.MeshOwner != nil && aws.StringValue(m.ms.Spec.MeshOwner) != m.mutatorConfig.accountID {
		return fmt.Sprintf("%v@%v", meshName, aws.StringValue(m.ms.Spec.MeshOwner))
	}
	return meshName
}

const (
	// when enabled, a virtual gateway image will not be overriden
	gatewayImageSkipOverrideModeEnabled = "enabled"
	// when disabled, a virtual gateway image will be overriden. This is also the default behavior
	gatewayImageSkipOverrideModeDisabled = "disabled"
)

func (m *virtualGatewayEnvoyConfig) virtualGatewayImageOverride(pod *corev1.Pod) bool {

	var imageOverrideAnnotation string
	if v, ok := pod.ObjectMeta.Annotations[AppMeshGatewaySkipImageOverride]; ok {
		imageOverrideAnnotation = v
	}

	switch strings.ToLower(imageOverrideAnnotation) {
	case gatewayImageSkipOverrideModeEnabled:
		return false
	case gatewayImageSkipOverrideModeDisabled:
		return true
	default:
		return true
	}
}

func (m *virtualGatewayEnvoyConfig) getUseDualStackEndpoint(useDualStackEndpoint bool) string {
	if useDualStackEndpoint {
		return "1"
	} else {
		return "0"
	}
}

func (m *virtualGatewayEnvoyConfig) getUseFipsEndpoint(useFipsEndpoint bool) string {
	if useFipsEndpoint {
		return "1"
	} else {
		return "0"
	}
}
