package inject

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const envoyTracingConfigVolumeName = "envoy-tracing-config"

// Envoy template variables used by envoys in pod and the envoy in VirtualGateway
// as we use the same envoy image
type EnvoyTemplateVariables struct {
	AWSRegion                string
	MeshName                 string
	VirtualGatewayOrNodeName string
	Preview                  string
	EnableSDS                bool
	SdsUdsPath               string
	LogLevel                 string
	AdminAccessPort          int32
	AdminAccessLogFile       string
	PreStopDelay             string
	PostStartTimeout         int32
	PostStartInterval        int32
	SidecarImageRepository   string
	SidecarImageTag          string
	EnableXrayTracing        bool
	XrayDaemonPort           int32
	XraySamplingRate         string
	EnableJaegerTracing      bool
	JaegerPort               string
	JaegerAddress            string
	EnableDatadogTracing     bool
	DatadogTracerPort        int32
	DatadogTracerAddress     string
	EnableStatsTags          bool
	EnableStatsD             bool
	StatsDPort               int32
	StatsDAddress            string
	StatsDSocketPath         string
	K8sVersion               string
	ControllerVersion        string
	EnableAdminAccessForIpv6 bool
	UseDualStackEndpoint     string
	WaitUntilProxyReady      bool
	UseFipsEndpoint          string
	AwsAccessKeyId           string
	AwsSecretAccessKey       string
	AwsSessionToken          string
}

func updateEnvMapForEnvoy(vars EnvoyTemplateVariables, env map[string]string, vname string) error {
	// add all the controller managed env to the map so
	// 1) we remove duplicates
	// 2) we don't allow overriding controller managed env with pod annotations
	env["APPMESH_VIRTUAL_NODE_NAME"] = vname
	env["AWS_REGION"] = vars.AWSRegion

	// For usage outside traditional EC2 / Fargate IAM based profiles, this is needed to
	// propagate permissions to envoy. This is a rare use-case that's mostly just for testing.
	if len(vars.AwsAccessKeyId) > 0 {
		env["AWS_ACCESS_KEY_ID"] = vars.AwsAccessKeyId
	}
	if len(vars.AwsSecretAccessKey) > 0 {
		env["AWS_SECRET_ACCESS_KEY"] = vars.AwsSecretAccessKey
	}
	if len(vars.AwsSessionToken) > 0 {
		env["AWS_SESSION_TOKEN"] = vars.AwsSessionToken
	}

	env["ENVOY_ADMIN_ACCESS_ENABLE_IPV6"] = strconv.FormatBool(vars.EnableAdminAccessForIpv6)

	env["APPMESH_DUALSTACK_ENDPOINT"] = vars.UseDualStackEndpoint

	env["APPMESH_FIPS_ENDPOINT"] = vars.UseFipsEndpoint
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

		// Override the default sampling rate of 0.05 (5%) for AWS X-Ray tracer
		// The value should be specified as a decimal between 0 and 1.00 (100%)
		samplingRate, ok := env["XRAY_SAMPLING_RATE"]
		if ok {
			// `podAnnotations` contains the sampling rate and gets preference over helm configuration
			// For now delete this value from env so that we can validate before adding again
			delete(env, "XRAY_SAMPLING_RATE")
		} else {
			// `podAnnotations` doesn't contain the sampling rate so get value from helm configuration
			samplingRate = vars.XraySamplingRate
		}

		fixedRate, err := strconv.ParseFloat(samplingRate, 32)
		if err != nil || float64(0) > fixedRate || float64(1) < fixedRate {
			// The value is not a decimal between 0 and 1.00
			return errors.Errorf("tracing.samplingRate should be a decimal between 0 & 1.00, "+
				"but instead got %s %v", samplingRate, err)
		} else {
			fixedRate = math.Round(fixedRate*100) / 100
			env["XRAY_SAMPLING_RATE"] = strconv.FormatFloat(fixedRate, 'f', -1, 32)
		}
	}

	if vars.EnableDatadogTracing {
		// Enables Datadog trace collection using 127.0.0.1:8126
		// as the default Datadog agent endpoint. To enable, set the value to 1
		env["ENABLE_ENVOY_DATADOG_TRACING"] = "1"

		// Specify a port value to override the default Datadog agent port: 8126
		env["DATADOG_TRACER_PORT"] = strconv.Itoa(int(vars.DatadogTracerPort))

		// Specify an IP address or hostname to override the default Datadog agent address: 127.0.0.1
		env["DATADOG_TRACER_ADDRESS"] = vars.DatadogTracerAddress

	}

	if vars.EnableStatsTags {
		env["ENABLE_ENVOY_STATS_TAGS"] = "1"
	}

	if vars.EnableStatsD {
		// Enables DogStatsD stats using 127.0.0.1:8125
		// as the default daemon endpoint. To enable, set the value to 1
		env["ENABLE_ENVOY_DOG_STATSD"] = "1"

		// Specify a port value to override the default DogStatsD daemon port.
		// This value will be overridden if `STATSD_SOCKET_PATH` is specified.
		env["STATSD_PORT"] = strconv.Itoa(int(vars.StatsDPort))

		// Specify an IP address value to override the default DogStatsD daemon IP address
		// Default: 127.0.0.1. This variable can only be used with version 1.15.0 or later
		// of the Envoy image. This value will be overridden if `STATSD_SOCKET_PATH` is specified.
		env["STATSD_ADDRESS"] = vars.StatsDAddress

		// Specify a unix domain socket for DogStatsD daemon. If not specified and if DogStatsD
		// is enabled then defaults to DogStatsD daemon IP address port [default: 127.0.0.1:8125].
		// This variable can only be used with version v1.19.1 or later.
		if statsDSocketPath := strings.TrimSpace(vars.StatsDSocketPath); statsDSocketPath != "" {
			env["STATSD_SOCKET_PATH"] = statsDSocketPath
		}
	}

	if vars.EnableJaegerTracing {
		env["ENABLE_ENVOY_JAEGER_TRACING"] = "1"
		env["JAEGER_TRACER_PORT"] = vars.JaegerPort
		env["JAEGER_TRACER_ADDRESS"] = vars.JaegerAddress
	}

	env["APPMESH_PLATFORM_K8S_VERSION"] = vars.K8sVersion
	env["APPMESH_PLATFORM_APP_MESH_CONTROLLER_VERSION"] = vars.ControllerVersion
	env["APPNET_AGENT_ADMIN_MODE"] = "uds"
	env["APPNET_AGENT_ADMIN_UDS_PATH"] = "/tmp/agent.sock"
	return nil
}

func buildEnvoySidecar(vars EnvoyTemplateVariables, env map[string]string) (corev1.Container, error) {

	envoy := corev1.Container{
		Name:  "envoy",
		Image: fmt.Sprintf("%s:%s", vars.SidecarImageRepository, vars.SidecarImageTag),
		SecurityContext: &corev1.SecurityContext{
			RunAsUser: aws.Int64(1337),
		},
		Ports: []corev1.ContainerPort{
			{
				Name:          "stats",
				ContainerPort: vars.AdminAccessPort,
				Protocol:      "TCP",
			},
		},
		Lifecycle: &corev1.Lifecycle{
			PostStart: nil,
			PreStop: &corev1.LifecycleHandler{
				Exec: &corev1.ExecAction{Command: []string{
					"sh", "-c", fmt.Sprintf("sleep %s", vars.PreStopDelay),
				}},
			},
		},
	}

	if vars.WaitUntilProxyReady {
		envoy.Lifecycle.PostStart = &corev1.LifecycleHandler{
			Exec: &corev1.ExecAction{Command: []string{
				// use bash regex and rematch to parse and check envoy version is >= 1.22.2.1
				"sh", "-c", fmt.Sprintf("if [[ $(/usr/bin/envoy --version) =~ ([0-9]+)\\.([0-9]+)\\.([0-9]+)-appmesh\\.([0-9]+) && "+
					"${BASH_REMATCH[1]} -ge 2 || (${BASH_REMATCH[1]} -ge 1 && ${BASH_REMATCH[2]} -gt 22) || (${BASH_REMATCH[1]} -ge 1 && "+
					"${BASH_REMATCH[2]} -ge 22 && ${BASH_REMATCH[3]} -gt 2) || (${BASH_REMATCH[1]} -ge 1 && ${BASH_REMATCH[2]} -ge 22 && "+
					"${BASH_REMATCH[3]} -ge 2 && ${BASH_REMATCH[4]} -gt 0) ]]; then APPNET_AGENT_POLL_ENVOY_READINESS_TIMEOUT_S=%d "+
					"APPNET_AGENT_POLL_ENVOY_READINESS_INTERVAL_S=%d /usr/bin/agent -envoyReadiness; else echo 'WaitUntilProxyReady "+
					"is not supported in Envoy version < 1.22.2.1'; fi", vars.PostStartTimeout, vars.PostStartInterval),
			}},
		}
	}

	vname := fmt.Sprintf("mesh/%s/virtualNode/%s", vars.MeshName, vars.VirtualGatewayOrNodeName)
	if err := updateEnvMapForEnvoy(vars, env, vname); err != nil {
		return envoy, err
	}
	envoy.Env = getEnvoyEnv(env)
	return envoy, nil

}

func getEnvoyEnv(env map[string]string) []corev1.EnvVar {

	ev := []corev1.EnvVar{}
	for key, val := range env {

		switch k := key; k {
		case "STATSD_ADDRESS", "DATADOG_TRACER_ADDRESS":
			if val == "ref:status.hostIP" {
				ev = append(ev, refHostIP(key))
			} else {
				ev = append(ev, envVar(key, val))
			}
		default:
			ev = append(ev, envVar(key, val))
		}

	}
	ev = append(ev, refPodUid("APPMESH_PLATFORM_K8S_POD_UID"))
	return ev
}

func envoyReadinessProbe(initialDelaySeconds int32, periodSeconds int32, adminAccessPort string) *corev1.Probe {
	envoyReadinessCommand := "curl -s http://localhost:" + adminAccessPort + "/server_info | grep state | grep -q LIVE"
	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{

			// server_info returns the following struct:
			// {
			//	"version": "...",
			//	"state": "...",
			//	"uptime_current_epoch": "{...}",
			//	"uptime_all_epochs": "{...}",
			//	"hot_restart_version": "...",
			//      "command_line_options": "{...}"
			//  }
			// server_info->state supports the following states: LIVE, DRAINING, PRE_INITIALIZING, and INITIALIZING
			// LIVE: Server is live and serving traffic
			// DRAINING: Server is draining listeners in response to external health checks failing
			// PRE_INITIALIZING: Server has not yet completed cluster manager initialization
			// INITIALIZING: Server is running the cluster manager initialization callbacks
			Exec: &corev1.ExecAction{Command: []string{
				"sh", "-c", envoyReadinessCommand,
			}},
		},

		// Number of seconds after the container has started before readiness probes are initiated
		InitialDelaySeconds: initialDelaySeconds,

		// Number of seconds after which the probe times out
		// This is a call to the local Envoy endpoint. 1 second is more than enough for timeout
		TimeoutSeconds: 1,

		// How often (in seconds) to perform the probe
		PeriodSeconds: periodSeconds,

		// Minimum consecutive successes for the probe to be considered successful after having failed
		// If Envoy shows LIVE status once, we're good to call it a success
		SuccessThreshold: 1,

		// Minimum consecutive failures for the probe to be considered failed after having succeeded
		// Keeping the failure threshold to 3 to not fail preemptively
		FailureThreshold: 3,
	}
}

func sidecarResources(cpuRequest, memoryRequest, cpuLimit, memoryLimit string) (corev1.ResourceRequirements, error) {
	resources := corev1.ResourceRequirements{}

	if cpuRequest != "" || memoryRequest != "" {
		requests := corev1.ResourceList{}

		if cpuRequest != "" {
			cr, err := resource.ParseQuantity(cpuRequest)
			if err != nil {
				return resources, err
			}
			requests["cpu"] = cr
		}

		if memoryRequest != "" {
			mr, err := resource.ParseQuantity(memoryRequest)
			if err != nil {
				return resources, err
			}
			requests["memory"] = mr
		}

		resources.Requests = requests

	}

	if cpuLimit != "" || memoryLimit != "" {
		limits := corev1.ResourceList{}

		if cpuLimit != "" {
			cl, err := resource.ParseQuantity(cpuLimit)
			if err != nil {
				return resources, err
			}
			limits["cpu"] = cl
		}

		if memoryLimit != "" {
			ml, err := resource.ParseQuantity(memoryLimit)
			if err != nil {
				return resources, err
			}
			limits["memory"] = ml
		}

		resources.Limits = limits

	}

	return resources, nil
}

// refHostIP is to use the k8s downward API and render the host IP
// this is useful in cases where the tracing agent is running as a daemonset
func refHostIP(envName string) corev1.EnvVar {
	return corev1.EnvVar{
		Name:  envName,
		Value: "",
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "status.hostIP",
			},
		},
	}
}

func envVar(envName, envVal string) corev1.EnvVar {
	return corev1.EnvVar{
		Name:  envName,
		Value: envVal,
	}
}

// refPodUid is to use the k8s downward API and render pod uid
// this info will be used to help App Mesh team to identify
// the platform Envoy is running on
func refPodUid(envName string) corev1.EnvVar {
	return corev1.EnvVar{
		Name:  envName,
		Value: "",
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "metadata.uid",
			},
		},
	}
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
