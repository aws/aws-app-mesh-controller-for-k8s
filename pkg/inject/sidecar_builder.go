package inject

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"strconv"
)

func buildEnvoySidecar(vars EnvoyTemplateVariables) corev1.Container {

	envoy := corev1.Container{
		Name:  "envoy",
		Image: vars.SidecarImage,
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
			PreStop: &corev1.Handler{
				Exec: &corev1.ExecAction{Command: []string{
					"sh", "-c", fmt.Sprintf("sleep %s", vars.PreStopDelay),
				}},
			},
		},
	}

	vn := fmt.Sprintf("mesh/%s/virtualNode/%s", vars.MeshName, vars.VirtualNodeName)
	env := []corev1.EnvVar{
		{
			Name:  "APPMESH_VIRTUAL_NODE_NAME",
			Value: vn,
		},
		{
			// Set the value to 1 to connect to the App Mesh Preview Channel endpoint.
			// See https://docs.aws.amazon.com/app-mesh/latest/userguide/preview.html
			Name:  "APPMESH_PREVIEW",
			Value: vars.Preview,
		},
		{
			// Specifies the log level for the Envoy container
			// Valid values: trace, debug, info, warning, error, critical, off
			Name:  "ENVOY_LOG_LEVEL",
			Value: vars.LogLevel,
		},
		{
			Name:  "AWS_REGION",
			Value: vars.AWSRegion,
		},
	}

	if vars.AdminAccessPort != 0 {
		port_env := corev1.EnvVar{
			// Specify a custom admin port for Envoy to listen on
			// Default: 9901
			Name:  "ENVOY_ADMIN_ACCESS_PORT",
			Value: strconv.Itoa(int(vars.AdminAccessPort)),
		}
		env = append(env, port_env)
	}

	if vars.AdminAccessLogFile != "" {
		log_env := corev1.EnvVar{
			// Specify a custom path to write Envoy access logs to
			// Default: /tmp/envoy_admin_access.log
			Name:  "ENVOY_ADMIN_ACCESS_LOG_FILE",
			Value: vars.AdminAccessLogFile,
		}
		env = append(env, log_env)
	}

	if vars.EnableXrayTracing {
		xray_env := []corev1.EnvVar{
			{
				// Enables X-Ray tracing using 127.0.0.1:2000 as the default daemon endpoint
				// To enable, set the value to 1
				Name:  "ENABLE_ENVOY_XRAY_TRACING",
				Value: "1",
			},
			{
				// Specify a port value to override the default X-Ray daemon port: 2000
				Name:  "XRAY_DAEMON_PORT",
				Value: strconv.Itoa(int(vars.XrayDaemonPort)),
			},
		}
		env = append(env, xray_env...)

	}

	if vars.EnableDatadogTracing {
		dd_env := []corev1.EnvVar{
			{
				// Enables Datadog trace collection using 127.0.0.1:8126
				// as the default Datadog agent endpoint. To enable, set the value to 1
				Name:  "ENABLE_ENVOY_DATADOG_TRACING",
				Value: "1",
			},
			{
				// Specify a port value to override the default Datadog agent port: 8126
				Name:  "DATADOG_TRACER_PORT",
				Value: strconv.Itoa(int(vars.DatadogTracerPort)),
			},
		}

		// Specify an IP address or hostname to override the default Datadog agent address: 127.0.0.1
		if vars.DatadogTracerAddress == "ref:status.hostIP" {
			dd_env = append(dd_env, refHostIP("DATADOG_TRACER_ADDRESS"))

		} else {
			dd_addr := corev1.EnvVar{
				Name:  "DATADOG_TRACER_ADDRESS",
				Value: vars.DatadogTracerAddress,
			}
			dd_env = append(dd_env, dd_addr)
		}
		env = append(env, dd_env...)
	}

	if vars.EnableStatsTags {
		stats_tags_env := corev1.EnvVar{
			Name:  "ENABLE_ENVOY_STATS_TAGS",
			Value: "1",
		}
		env = append(env, stats_tags_env)
	}

	if vars.EnableStatsD {
		sd_env := []corev1.EnvVar{
			{
				// Enables DogStatsD stats using 127.0.0.1:8125
				// as the default daemon endpoint. To enable, set the value to 1
				Name:  "ENABLE_ENVOY_DOG_STATSD",
				Value: "1",
			},
			{
				// Specify a port value to override the default DogStatsD daemon port
				Name:  "STATSD_PORT",
				Value: strconv.Itoa(int(vars.StatsDPort)),
			},
		}

		// Specify an IP address value to override the default DogStatsD daemon IP address
		// Default: 127.0.0.1. This variable can only be used with version 1.15.0 or later
		// of the Envoy image
		if vars.StatsDAddress == "ref:status.hostIP" {
			sd_env = append(sd_env, refHostIP("STATSD_ADDRESS"))

		} else {
			sd_addr := corev1.EnvVar{
				Name:  "STATSD_ADDRESS",
				Value: vars.StatsDAddress,
			}
			sd_env = append(sd_env, sd_addr)
		}
		env = append(env, sd_env...)
	}

	if vars.EnableJaegerTracing {
		jaeger_env := corev1.EnvVar{
			// Specify a file path in the Envoy container file system.
			// See https://www.envoyproxy.io/docs/envoy/latest/api-v2/config/trace/v2/http_tracer.proto
			Name:  "ENVOY_TRACING_CFG_FILE",
			Value: "/tmp/envoy/envoyconf.yaml",
		}
		env = append(env, jaeger_env)

		vol_mount := []corev1.VolumeMount{
			{
				Name:      vars.EnvoyTracingConfigVolumeName,
				MountPath: "/tmp/envoy",
			},
		}
		envoy.VolumeMounts = vol_mount
	}

	envoy.Env = env
	return envoy

}

func envoyReadinessProbe(initialDelaySeconds int32, periodSeconds int32, adminAccessPort string) *corev1.Probe {
	envoyReadinessCommand := "curl -s http://localhost:" + adminAccessPort + "/server_info | grep state | grep -q LIVE"
	return &corev1.Probe{
		Handler: corev1.Handler{

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
