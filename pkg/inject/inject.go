package inject

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

var injectLogger = ctrl.Log.WithName("appmesh_inject")

type PodMutator interface {
	mutate(*corev1.Pod) error
}

type SidecarInjector struct {
	config    Config
	awsRegion string
}

func NewSidecarInjector(cfg Config, awsRegion string) *SidecarInjector {
	return &SidecarInjector{
		config:    cfg,
		awsRegion: awsRegion,
	}
}

func (m *SidecarInjector) InjectAppMeshPatches(ms *appmesh.Mesh, vn *appmesh.VirtualNode, pod *corev1.Pod) error {
	if !shouldInject(m.config.EnableSidecarInjectorWebhook, pod) {
		injectLogger.Info("Not injecting sidecars for pod ", pod.Name)
		return nil
	}

	// List out all the mutators in sequence
	var mutators = []PodMutator{
		newProxyMutator(proxyMutatorConfig{
			egressIgnoredIPs: m.config.IgnoredIPs,
			initProxyMutatorConfig: initProxyMutatorConfig{
				containerImage: m.config.InitImage,
				cpuRequests:    m.config.SidecarCpu,
				memoryRequests: m.config.SidecarMemory,
			},
		}, vn),
		newEnvoyMutator(envoyMutatorConfig{
			awsRegion:             m.awsRegion,
			preview:               m.config.Preview,
			logLevel:              m.config.LogLevel,
			sidecarImage:          m.config.SidecarImage,
			sidecarCPURequests:    m.config.SidecarCpu,
			sidecarMemoryRequests: m.config.SidecarMemory,
			enableXrayTracing:     m.config.EnableXrayTracing,
			enableJaegerTracing:   m.config.EnableJaegerTracing,
			enableDatadogTracing:  m.config.EnableDatadogTracing,
			enableStatsTags:       m.config.EnableStatsTags,
			enableStatsD:          m.config.EnableStatsD,
		}, ms, vn),
		newXrayMutator(xrayMutatorConfig{
			awsRegion:             m.awsRegion,
			sidecarCPURequests:    m.config.SidecarCpu,
			sidecarMemoryRequests: m.config.SidecarMemory,
		}, m.config.EnableXrayTracing),
		newDatadogMutator(datadogMutatorConfig{
			datadogAddress: m.config.DatadogAddress,
			datadogPort:    m.config.DatadogPort,
		}, m.config.EnableDatadogTracing),
		newJaegerMutator(jaegerMutatorConfig{
			jaegerAddress: m.config.JaegerAddress,
			jaegerPort:    m.config.JaegerPort,
		}, m.config.EnableJaegerTracing),
		newIAMForServiceAccountsMutator(m.config.EnableIAMForServiceAccounts),
		newECRSecretMutator(m.config.EnableECRSecret),
	}

	for _, mutator := range mutators {
		err := mutator.mutate(pod)
		if err != nil {
			return err
		}
	}
	return nil
}
