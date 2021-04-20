package inject

import (
	"fmt"
	"strconv"
	"strings"

	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-sdk-go/aws"
	corev1 "k8s.io/api/core/v1"
)

const envoyImageStub = "injector-envoy-image"

type VirtualGatewayEnvoyVariables struct {
	AWSRegion                    string
	MeshName                     string
	VirtualGatewayName           string
	Preview                      string
	EnableSDS                    bool
	SdsUdsPath                   string
	LogLevel                     string
	AdminAccessPort              int32
	AdminAccessLogFile           string
	EnvoyTracingConfigVolumeName string
	EnableXrayTracing            bool
	XrayDaemonPort               int32
	EnableJaegerTracing          bool
	JaegerPort                   string
	JaegerAddress                string
}

type virtualGatwayEnvoyConfig struct {
	accountID                  string
	awsRegion                  string
	preview                    bool
	enableSDS                  bool
	sdsUdsPath                 string
	logLevel                   string
	adminAccessPort            int32
	adminAccessLogFile         string
	sidecarImage               string
	readinessProbeInitialDelay int32
	readinessProbePeriod       int32
	enableXrayTracing          bool
	xrayDaemonPort             int32
	enableJaegerTracing        bool
	jaegerPort                 int32
	jaegerAddress              string
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

	newEnvMap := m.getEnvMapForVirtualGatewayEnvoy(variables)

	//we override the image to latest Envoy so customers do not have to manually manage
	// envoy versions and let controller handle consistency versions across the mesh
	if m.virtualGatewayImageOverride(pod) {
		envoy.Image = m.mutatorConfig.sidecarImage
	}

	for idx, env := range pod.Spec.Containers[envoyIdx].Env {
		if val, ok := newEnvMap[env.Name]; ok {
			if val != env.Value {
				envoy.Env[idx].Value = val
			}
			delete(newEnvMap, env.Name)
		}
	}

	for name, value := range newEnvMap {
		e := corev1.EnvVar{Name: name,
			Value: value}
		envoy.Env = append(envoy.Env, e)
	}

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

func (m *virtualGatewayEnvoyConfig) getEnvMapForVirtualGatewayEnvoy(vars VirtualGatewayEnvoyVariables) map[string]string {
	env := map[string]string{}
	vg := fmt.Sprintf("mesh/%s/virtualGateway/%s", vars.MeshName, vars.VirtualGatewayName)

	env["APPMESH_VIRTUAL_NODE_NAME"] = vg
	env["AWS_REGION"] = vars.AWSRegion

	// Set the value to 1 to connect to the App Mesh Preview Channel endpoint.
	// See https://docs.aws.amazon.com/app-mesh/latest/userguide/preview.html
	env["APPMESH_PREVIEW"] = vars.Preview

	// Specifies the log level for the Envoy container
	// Valid values: trace, debug, info, warning, error, critical, off
	env["ENVOY_LOG_LEVEL"] = vars.LogLevel

	if vars.EnableSDS {
		env["APPMESH_SDS_SOCKET_PATH"] = vars.SdsUdsPath
	}

	if vars.AdminAccessPort != 0 {
		// Specify a custom admin port for Envoy to listen on
		// Default: 9901
		env["ENVOY_ADMIN_ACCESS_PORT"] = strconv.Itoa(int(vars.AdminAccessPort))
	}

	if vars.AdminAccessLogFile != "" {
		// Specify a custom path to write Envoy access logs to
		// Default: /tmp/envoy_admin_access.log
		env["ENVOY_ADMIN_ACCESS_LOG_FILE"] = vars.AdminAccessLogFile
	}

	if vars.EnableXrayTracing {

		// Enables X-Ray tracing using 127.0.0.1:2000 as the default daemon endpoint
		// To enable, set the value to 1
		env["ENABLE_ENVOY_XRAY_TRACING"] = "1"

		// Specify a port value to override the default X-Ray daemon port: 2000
		env["XRAY_DAEMON_PORT"] = strconv.Itoa(int(vars.XrayDaemonPort))

	}

	if vars.EnableJaegerTracing {
		env["ENABLE_ENVOY_JAEGER_TRACING"] = "1"
		env["JAEGER_TRACER_PORT"] = vars.JaegerPort
		env["JAEGER_TRACER_ADDRESS"] = vars.JaegerAddress
	}
	return env
}

func (m *virtualGatewayEnvoyConfig) buildTemplateVariables(pod *corev1.Pod) VirtualGatewayEnvoyVariables {
	meshName := m.getAugmentedMeshName()
	virtualGatewayName := aws.StringValue(m.vg.Spec.AWSName)
	preview := m.getPreview(pod)
	sdsEnabled := m.mutatorConfig.enableSDS
	if m.mutatorConfig.enableSDS && isSDSDisabled(pod) {
		sdsEnabled = false
	}

	return VirtualGatewayEnvoyVariables{
		AWSRegion:                    m.mutatorConfig.awsRegion,
		MeshName:                     meshName,
		VirtualGatewayName:           virtualGatewayName,
		Preview:                      preview,
		EnableSDS:                    sdsEnabled,
		SdsUdsPath:                   m.mutatorConfig.sdsUdsPath,
		LogLevel:                     m.mutatorConfig.logLevel,
		AdminAccessPort:              m.mutatorConfig.adminAccessPort,
		AdminAccessLogFile:           m.mutatorConfig.adminAccessLogFile,
		EnvoyTracingConfigVolumeName: envoyTracingConfigVolumeName,
		EnableXrayTracing:            m.mutatorConfig.enableXrayTracing,
		XrayDaemonPort:               m.mutatorConfig.xrayDaemonPort,
		EnableJaegerTracing:          m.mutatorConfig.enableJaegerTracing,
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
