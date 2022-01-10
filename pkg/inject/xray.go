package inject

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"strings"

	"github.com/aws/aws-sdk-go/aws/arn"
)

const xrayDaemonContainerName = "xray-daemon"
const xrayDaemonContainerTemplate = `
{
  "name": "xray-daemon",
  "image": "{{ .XRayImage }}",
  "securityContext": {
    "runAsUser": 1337
  },
  "ports": [
    {
      "containerPort": {{ .XrayDaemonPort }},
      "name": "xray",
      "protocol": "UDP"
    }
  ],
  "env": [
    {
      "name": "AWS_REGION",
      "value": "{{ .AWSRegion }}"
    }
  ]
}
`

type XrayTemplateVariables struct {
	AWSRegion      string
	XRayImage      string
	XrayDaemonPort int32
}

type xrayMutatorConfig struct {
	awsRegion             string
	sidecarCPURequests    string
	sidecarMemoryRequests string
	sidecarCPULimits      string
	sidecarMemoryLimits   string
	xRayImage             string
	xRayDaemonPort        int32
	xRayLogLevel          string
	xRayConfigRoleArn     string
}

func newXrayMutator(mutatorConfig xrayMutatorConfig, enabled bool) *xrayMutator {
	return &xrayMutator{
		mutatorConfig: mutatorConfig,
		enabled:       enabled,
	}
}

type xrayMutator struct {
	mutatorConfig xrayMutatorConfig
	enabled       bool
}

func (m *xrayMutator) mutate(pod *corev1.Pod) error {
	if !m.enabled {
		return nil
	}
	if containsXRAYDaemonContainer(pod) {
		return nil
	}

	err := m.checkConfig()
	if err != nil {
		return err
	}

	xrayDaemonConfigMount, err := m.getXrayDaemonConfigMount(pod)
	if err != nil {
		return err
	}

	variables := m.buildTemplateVariables(pod)
	xrayDaemonSidecar, err := renderTemplate("xray-daemon", xrayDaemonContainerTemplate, variables)
	if err != nil {
		return err
	}

	container := corev1.Container{}
	err = json.Unmarshal([]byte(xrayDaemonSidecar), &container)
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

	// Add xray daemon configurations as args. If config file is mounted then it will override all other configurations. See
	// https://docs.aws.amazon.com/xray/latest/devguide/xray-daemon-configuration.html#xray-daemon-configuration-commandline
	err = m.buildXrayDaemonArgs(pod, &container, xrayDaemonConfigMount)
	if err != nil {
		return err
	}

	pod.Spec.Containers = append(pod.Spec.Containers, container)
	return nil
}

func (m *xrayMutator) getXrayDaemonConfigMount(pod *corev1.Pod) (map[string]string, error) {
	xrayDaemonConfigMount := make(map[string]string)
	if v, ok := pod.ObjectMeta.Annotations[AppMeshXrayAgentConfigAnnotation]; ok {
		if strings.Contains(v, ",") {
			return nil, errors.Errorf("provide only one config mount for annotation \"%s: %v\"", AppMeshXrayAgentConfigAnnotation, v)
		}
		pair := strings.Split(v, ":")
		if len(pair) != 2 { // volumeName:mountPath
			return nil, errors.Errorf("malformed annotation \"%s\", expected format: %s", AppMeshXrayAgentConfigAnnotation, "volumeName:mountPath")
		}
		volumeName := strings.TrimSpace(pair[0])
		mountPath := strings.TrimSpace(pair[1])
		xrayDaemonConfigMount[volumeName] = mountPath
	}
	return xrayDaemonConfigMount, nil
}

func (m *xrayMutator) buildXrayDaemonArgs(pod *corev1.Pod, xrayDaemonContainer *corev1.Container, xrayDaemonConfigMount map[string]string) error {
	// For-loop will loop only once as we expect only a single xrayDaemonConfigMount to be provided.
	// If more than 1 is supplied we error out in `getXrayDaemonConfigMount` func itself.
	for volumeName, mountPath := range xrayDaemonConfigMount {
		volumeMount := corev1.VolumeMount{
			Name:      volumeName,
			MountPath: mountPath,
			ReadOnly:  true,
		}
		xrayDaemonContainer.VolumeMounts = append(xrayDaemonContainer.VolumeMounts, volumeMount)
		// Override the xray daemon's entrypoint command do load config from xray-daemon.yaml
		xrayDaemonContainer.Command = []string{"/xray", "--config", fmt.Sprintf("%v/xray-daemon.yaml", strings.TrimRight(mountPath, " /"))}
		// We return here since file config will override all other cli arguments
		return nil
	}
	// Check for other arguments such as role-arn or log-level
	if m.mutatorConfig.xRayLogLevel != "" {
		switch m.mutatorConfig.xRayLogLevel {
		case "dev":
			fallthrough
		case "debug":
			fallthrough
		case "info":
			fallthrough
		case "warn":
			fallthrough
		case "error":
			fallthrough
		case "prod":
			xrayDaemonContainer.Args = append(xrayDaemonContainer.Args, "--log-level", m.mutatorConfig.xRayLogLevel)
		default:
			return errors.Errorf("tracing.logLevel: \"%v\" is not valid."+
				" Set one of dev, debug, info, prod, warn, error", m.mutatorConfig.xRayLogLevel)
		}
	}
	if m.mutatorConfig.xRayConfigRoleArn != "" {
		if arn.IsARN(m.mutatorConfig.xRayConfigRoleArn) {
			xrayDaemonContainer.Args = append(xrayDaemonContainer.Args, "--role-arn", m.mutatorConfig.xRayConfigRoleArn)
		} else {
			return errors.Errorf("tracing.role: \"%v\" is not a valid `--role-arn`."+
				" Please refer to AWS X-Ray Documentation for more information", m.mutatorConfig.xRayConfigRoleArn)
		}
	}
	return nil
}

func (m *xrayMutator) buildTemplateVariables(pod *corev1.Pod) XrayTemplateVariables {
	return XrayTemplateVariables{
		AWSRegion:      m.mutatorConfig.awsRegion,
		XRayImage:      m.mutatorConfig.xRayImage,
		XrayDaemonPort: m.mutatorConfig.xRayDaemonPort,
	}
}

func (m *xrayMutator) checkConfig() error {
	var missingConfig []string

	if m.mutatorConfig.awsRegion == "" {
		missingConfig = append(missingConfig, "AWSRegion")
	}
	if m.mutatorConfig.xRayImage == "" {
		missingConfig = append(missingConfig, "xRayImage")
	}

	if m.mutatorConfig.xRayDaemonPort == 0 {
		missingConfig = append(missingConfig, "xRayDaemonPort")
	}

	if len(missingConfig) > 0 {
		return errors.Errorf("Missing configuration parameters: %s", strings.Join(missingConfig, ","))
	}

	return nil
}

func containsXRAYDaemonContainer(pod *corev1.Pod) bool {
	for _, container := range pod.Spec.Containers {
		if container.Name == xrayDaemonContainerName {
			return true
		}
	}
	return false
}
