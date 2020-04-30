package appmeshinject

import (
	"errors"
	"flag"
)

type Config struct {
	// Injetion Settings
	InjectDefault bool

	// If enabled, an fsGroup: 1337 will be injected in the absence of it within pod securityContext
	// see https://github.com/aws/amazon-eks-pod-identity-webhook/issues/8 for more details
	EnableIAMForServiceAccounts bool

	// Sidecar settings
	SidecarImage  string
	SidecarCpu    string
	SidecarMemory string
	//MeshName      string
	Region    string
	Preview   bool
	LogLevel  string
	EcrSecret bool

	// Init container settings
	InitImage          string
	IgnoredIPs         string
	EgressIgnoredPorts string

	// Observability settings
	InjectXraySidecar           bool
	EnableStatsTags             bool
	EnableStatsD                bool
	InjectStatsDExporterSidecar bool
	EnableJaegerTracing         bool
	JaegerAddress               string
	JaegerPort                  string
	EnableDatadogTracing        bool
	DatadogAddress              string
	DatadogPort                 string
}

// MultipleTracer checks if more than one tracer is configured.
func MultipleTracer(config *Config) bool {
	j := config.EnableJaegerTracing
	d := config.EnableDatadogTracing
	x := config.InjectXraySidecar

	return (j && d) || (d && x) || (j && x)
}

func (cfg *Config) BindFlags() {
	flag.BoolVar(&cfg.InjectDefault, "inject-default", true,
		`If enabled, sidecars will be injected in the absence of the corresponding pod annotation`)
	flag.BoolVar(&cfg.EnableIAMForServiceAccounts, "enable-iam-for-service-accounts", true,
		`If enabled, an fsGroup: 1337 will be injected in the absence of it within pod securityContext`)
	flag.StringVar(&cfg.Region, "region" /*os.Getenv("APPMESH_REGION")*/, "",
		"AWS App Mesh region")
	flag.BoolVar(&cfg.Preview, "preview", false,
		"Enable preview channel")
	flag.StringVar(&cfg.LogLevel, "log-level" /*os.Getenv("APPMESH_LOG_LEVEL")*/, "info",
		"AWS App Mesh envoy log level")
	flag.BoolVar(&cfg.EcrSecret, "ecr-secret", false,
		"Inject AWS app mesh pull secrets")
	flag.StringVar(&cfg.SidecarImage, "sidecar-image", "840364872350.dkr.ecr.us-west-2.amazonaws.com/aws-appmesh-envoy:v1.12.3.0-prod",
		"Envoy sidecar container image.")
	flag.StringVar(&cfg.SidecarCpu, "sidecar-cpu-requests", "10m",
		"Envoy sidecar CPU resources requests.")
	flag.StringVar(&cfg.SidecarMemory, "sidecar-memory-requests", "32Mi",
		"Envoy sidecar memory resources requests.")
	flag.StringVar(&cfg.InitImage, "init-image", "111345817488.dkr.ecr.us-west-2.amazonaws.com/aws-appmesh-proxy-route-manager:v2",
		"Init container image.")
	flag.StringVar(&cfg.IgnoredIPs, "ignored-ips", "169.254.169.254",
		"Init container ignored IPs.")
	flag.BoolVar(&cfg.EnableJaegerTracing, "enable-jaeger-tracing", false,
		"Enable Envoy Jaeger tracing")
	flag.StringVar(&cfg.JaegerAddress, "jaeger-address", "appmesh-jaeger.appmesh-system",
		"Jaeger address")
	flag.StringVar(&cfg.JaegerPort, "jaeger-port", "9411",
		"Jaeger port")
	flag.BoolVar(&cfg.EnableDatadogTracing, "enable-datadog-tracing", false,
		"Enable Envoy Datadog tracing")
	flag.StringVar(&cfg.DatadogAddress, "datadog-address", "datadog.appmesh-system",
		"Datadog Agent address")
	flag.StringVar(&cfg.DatadogPort, "datadog-port", "8126",
		"Datadog Agent tracing port")
	flag.BoolVar(&cfg.InjectXraySidecar, "inject-xray-sidecar", false,
		"Enable Envoy X-Ray tracing integration and injects xray-daemon as sidecar")
	flag.BoolVar(&cfg.EnableStatsTags, "enable-stats-tags", false,
		"Enable Envoy to tag stats")
	flag.BoolVar(&cfg.EnableStatsD, "enable-statsd", false,
		"If enabled, Envoy will send DogStatsD metrics to 127.0.0.1:8125")
	flag.BoolVar(&cfg.InjectStatsDExporterSidecar, "inject-statsd-exporter-sidecar", false,
		"This fs is deprecated and does nothing")
}

func (cfg *Config) BindEnv() error {
	return nil
}

func (cfg *Config) Validate() error {
	if MultipleTracer(cfg) {
		return errors.New("Envoy only supports a single tracer instance. Please choose between Jaeger, Datadog or X-Ray.")
	}
	return nil
}
