package inject

import (
	"errors"

	"github.com/spf13/pflag"
)

const (
	flagEnableIAMForServiceAccounts = "enable-iam-for-service-accounts"
	flagEnableECRSecret             = "enable-ecr-secret"
	flagEnableSDS                   = "enable-sds"
	flagSdsUdsPath                  = "sds-uds-path"

	flagSidecarImage               = "sidecar-image"
	flagSidecarCpuRequests         = "sidecar-cpu-requests"
	flagSidecarMemoryRequests      = "sidecar-memory-requests"
	flagSidecarCpuLimits           = "sidecar-cpu-limits"
	flagSidecarMemoryLimits        = "sidecar-memory-limits"
	flagPreview                    = "preview"
	flagLogLevel                   = "sidecar-log-level"
	flagPreStopDelay               = "prestop-delay"
	flagReadinessProbeInitialDelay = "readiness-probe-initial-delay"
	flagReadinessProbePeriod       = "readiness-probe-period"
	flagEnvoyAdminAccessPort       = "envoy-admin-access-port"
	flagEnvoyAdminAccessLogFile    = "envoy-admin-access-log-file"

	flagInitImage  = "init-image"
	flagIgnoredIPs = "ignored-ips"

	flagEnableJaegerTracing  = "enable-jaeger-tracing"
	flagJaegerAddress        = "jaeger-address"
	flagJaegerPort           = "jaeger-port"
	flagEnableDatadogTracing = "enable-datadog-tracing"
	flagDatadogAddress       = "datadog-address"
	flagDatadogPort          = "datadog-port"
	flagEnableXrayTracing    = "enable-xray-tracing"
	flagXrayDaemonPort       = "xray-daemon-port"
	flagEnableStatsTags      = "enable-stats-tags"
	flagEnableStatsD         = "enable-statsd"
	flagStatsDAddress        = "statsd-address"
	flagStatsDPort           = "statsd-port"
	flagXRayImage            = "xray-image"
)

type Config struct {
	// If enabled, an fsGroup: 1337 will be injected in the absence of it within pod securityContext
	// see https://github.com/aws/amazon-eks-pod-identity-webhook/issues/8 for more details
	EnableIAMForServiceAccounts bool
	// If enabled, additional image pull secret(appmesh-ecr-secret) will be injected.
	EnableECRSecret bool
	// If enabled, mTLS support via SDS will be enabled.
	EnableSDS bool
	// Contains the Unix Domain Socket Path for SDS provider.
	SdsUdsPath string

	// Sidecar settings
	SidecarImage               string
	SidecarCpuRequests         string
	SidecarMemoryRequests      string
	SidecarCpuLimits           string
	SidecarMemoryLimits        string
	Preview                    bool
	LogLevel                   string
	PreStopDelay               string
	ReadinessProbeInitialDelay int32
	ReadinessProbePeriod       int32
	EnvoyAdminAcessPort        int32
	EnvoyAdminAccessLogFile    string

	// Init container settings
	InitImage  string
	IgnoredIPs string

	// Observability settings
	EnableJaegerTracing  bool
	JaegerAddress        string
	JaegerPort           string
	EnableDatadogTracing bool
	DatadogAddress       string
	DatadogPort          int32
	EnableXrayTracing    bool
	XrayDaemonPort       int32
	EnableStatsTags      bool
	EnableStatsD         bool
	StatsDAddress        string
	StatsDPort           int32
	XRayImage            string
}

// MultipleTracer checks if more than one tracer is configured.
func multipleTracer(config *Config) bool {
	j := config.EnableJaegerTracing
	d := config.EnableDatadogTracing
	x := config.EnableXrayTracing

	return (j && d) || (d && x) || (j && x)
}

func (cfg *Config) BindFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&cfg.EnableIAMForServiceAccounts, flagEnableIAMForServiceAccounts, true,
		`If enabled, a fsGroup: 1337 will be injected in the absence of it within pod securityContext`)
	fs.BoolVar(&cfg.EnableECRSecret, flagEnableECRSecret, false,
		"If enabled, 'appmesh-ecr-secret' secret will be injected in the absence of it within pod imagePullSecrets")
	fs.BoolVar(&cfg.EnableSDS, flagEnableSDS, false,
		"If enabled, mTLS support via SDS will be enabled")
	fs.StringVar(&cfg.SdsUdsPath, flagSdsUdsPath, "/run/spire/sockets/agent.sock",
		"Unix Domain Socket path for SDS provider")
	fs.StringVar(&cfg.SidecarImage, flagSidecarImage, "840364872350.dkr.ecr.us-west-2.amazonaws.com/aws-appmesh-envoy:v1.15.1.0-prod",
		"Envoy sidecar container image.")
	fs.StringVar(&cfg.SidecarCpuRequests, flagSidecarCpuRequests, "10m",
		"Sidecar CPU resources requests.")
	fs.StringVar(&cfg.SidecarMemoryRequests, flagSidecarMemoryRequests, "32Mi",
		"Sidecar memory resources requests.")
	fs.StringVar(&cfg.SidecarCpuLimits, flagSidecarCpuLimits, "",
		"Sidecar CPU resources limits.")
	fs.StringVar(&cfg.SidecarMemoryLimits, flagSidecarMemoryLimits, "",
		"Sidecar memory resources limits.")
	fs.BoolVar(&cfg.Preview, flagPreview, false,
		"Enable preview channel")
	fs.StringVar(&cfg.LogLevel, flagLogLevel, "info",
		"AWS App Mesh envoy log level")
	fs.Int32Var(&cfg.EnvoyAdminAcessPort, flagEnvoyAdminAccessPort, 9901,
		"AWS App Mesh envoy admin access port")
	fs.StringVar(&cfg.EnvoyAdminAccessLogFile, flagEnvoyAdminAccessLogFile, "/tmp/envoy_admin_access.log",
		"AWS App Mesh envoy access log path")
	fs.StringVar(&cfg.PreStopDelay, flagPreStopDelay, "20",
		"AWS App Mesh envoy preStop hook sleep duration")
	fs.Int32Var(&cfg.ReadinessProbeInitialDelay, flagReadinessProbeInitialDelay, 1,
		"Number of seconds after Envoy has started before readiness probes are initiated")
	fs.Int32Var(&cfg.ReadinessProbePeriod, flagReadinessProbePeriod, 10,
		"How often (in seconds) to perform the readiness probe on Envoy container")
	fs.StringVar(&cfg.InitImage, flagInitImage, "840364872350.dkr.ecr.us-west-2.amazonaws.com/aws-appmesh-proxy-route-manager:v3-prod",
		"Init container image.")
	fs.StringVar(&cfg.IgnoredIPs, flagIgnoredIPs, "169.254.169.254",
		"Init container ignored IPs.")
	fs.BoolVar(&cfg.EnableJaegerTracing, flagEnableJaegerTracing, false,
		"Enable Envoy Jaeger tracing")
	fs.StringVar(&cfg.JaegerAddress, flagJaegerAddress, "appmesh-jaeger.appmesh-system",
		"Jaeger address")
	fs.StringVar(&cfg.JaegerPort, flagJaegerPort, "9411",
		"Jaeger port")
	fs.BoolVar(&cfg.EnableDatadogTracing, flagEnableDatadogTracing, false,
		"Enable Envoy Datadog tracing")
	fs.StringVar(&cfg.DatadogAddress, flagDatadogAddress, "datadog.appmesh-system",
		"Datadog Agent address")
	fs.Int32Var(&cfg.DatadogPort, flagDatadogPort, 8126,
		"Datadog Agent tracing port")
	fs.BoolVar(&cfg.EnableXrayTracing, flagEnableXrayTracing, false,
		"Enable Envoy X-Ray tracing integration and injects xray-daemon as sidecar")
	fs.Int32Var(&cfg.XrayDaemonPort, flagXrayDaemonPort, 2000,
		"Datadog Agent tracing port")
	fs.StringVar(&cfg.XRayImage, flagXRayImage, "amazon/aws-xray-daemon",
		"X-Ray daemon container image")
	fs.BoolVar(&cfg.EnableStatsTags, flagEnableStatsTags, false,
		"Enable Envoy to tag stats")
	fs.BoolVar(&cfg.EnableStatsD, flagEnableStatsD, false,
		"If enabled, Envoy will send DogStatsD metrics to 127.0.0.1:8125")
	fs.StringVar(&cfg.StatsDAddress, flagStatsDAddress, "127.0.0.1",
		"Datadog Agent address")
	fs.Int32Var(&cfg.StatsDPort, flagStatsDPort, 8125,
		"Datadog Agent tracing port")
}

func (cfg *Config) BindEnv() error {
	return nil
}

func (cfg *Config) Validate() error {
	if multipleTracer(cfg) {
		return errors.New("Envoy only supports a single tracer instance. Please choose between Jaeger, Datadog or X-Ray.")
	}
	return nil
}
