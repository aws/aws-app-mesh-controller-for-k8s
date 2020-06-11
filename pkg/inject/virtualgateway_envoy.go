package inject

import (
	"encoding/json"
	"fmt"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-sdk-go/aws"
	corev1 "k8s.io/api/core/v1"
	"strings"
)

const envoyImageStub = "injector-envoy-image"
const envoyVirtualGatewayEnvMap = `
{
  "APPMESH_VIRTUAL_NODE_NAME": "mesh/{{ .MeshName }}/virtualGateway/{{ .VirtualGatewayName }}",
  "APPMESH_PREVIEW": "{{ .Preview }}",
  "ENVOY_LOG_LEVEL": "{{ .LogLevel }}",
  "AWS_REGION": "{{ .AWSRegion }}"
}
`

type VirtualGatewayEnvoyVariables struct {
	AWSRegion          string
	MeshName           string
	VirtualGatewayName string
	Preview            string
	LogLevel           string
}

type virtualGatwayEnvoyConfig struct {
	accountID    string
	awsRegion    string
	preview      bool
	logLevel     string
	sidecarImage string
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
	envoyEnv, err := renderTemplate("vgenvoy", envoyVirtualGatewayEnvMap, variables)
	if err != nil {
		return err
	}

	newEnvMap := map[string]string{}
	err = json.Unmarshal([]byte(envoyEnv), &newEnvMap)
	if err != nil {
		return err
	}

	//we override the image to latest Envoy so customers do not have to manually manage
	// envoy versions and let controller handle consistency versions across the mesh
	if m.virtualGatewayImageOverride(pod) {
		pod.Spec.Containers[envoyIdx].Image = m.mutatorConfig.sidecarImage
	}

	for idx, env := range pod.Spec.Containers[envoyIdx].Env {
		if val, ok := newEnvMap[env.Name]; ok {
			if val != env.Value {
				pod.Spec.Containers[envoyIdx].Env[idx].Value = val
			}
			delete(newEnvMap, env.Name)
		}
	}

	for name, value := range newEnvMap {
		e := corev1.EnvVar{Name: name,
			Value: value}
		pod.Spec.Containers[envoyIdx].Env = append(pod.Spec.Containers[envoyIdx].Env, e)
	}
	return nil
}

func (m *virtualGatewayEnvoyConfig) buildTemplateVariables(pod *corev1.Pod) VirtualGatewayEnvoyVariables {
	meshName := m.getAugmentedMeshName()
	virtualGatewayName := aws.StringValue(m.vg.Spec.AWSName)
	preview := m.getPreview(pod)

	return VirtualGatewayEnvoyVariables{
		AWSRegion:          m.mutatorConfig.awsRegion,
		MeshName:           meshName,
		VirtualGatewayName: virtualGatewayName,
		Preview:            preview,
		LogLevel:           m.mutatorConfig.logLevel,
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
