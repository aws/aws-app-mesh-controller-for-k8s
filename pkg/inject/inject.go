package inject

import (
	"errors"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

var injectLogger = ctrl.Log.WithName("appmesh_inject")

type PodMutator interface {
	mutate(*corev1.Pod) error
}

type SidecarInjector struct {
	config *Config
}

func NewSidecarInjector(cfg *Config, region string) *SidecarInjector {
	if len(cfg.Region) == 0 {
		cfg.Region = region
	}
	return &SidecarInjector{
		config: cfg,
	}
}

func (m *SidecarInjector) InjectAppMeshPatches(ms *appmesh.Mesh, vn *appmesh.VirtualNode, pod *corev1.Pod) error {
	if !ShouldInject(m.config, pod) {
		injectLogger.Info("Not injecting sidecars for pod ", pod.Name)
		return nil
	}
	if MultipleTracer(m.config) {
		return errors.New("Unable to apply patches with multiple tracers. Please choose between Jaeger, Datadog or X-Ray.")
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
		NewEnvoyMutator(m.config, ms, vn),
		NewXrayMutator(m.config),
		NewDatadogMutator(m.config),
		NewJaegerMutator(m.config),
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
