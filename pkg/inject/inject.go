package inject

import (
	"errors"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	ecrSecret = "appmesh-ecr-secret"
	// We don't want to make this configurable since users shouldn't rely on this
	// feature to set a fsGroup for them. This feature is just to protect innocent
	// users that are not aware of the limitation of iam-for-service-accounts:
	// https://github.com/aws/amazon-eks-pod-identity-webhook/issues/8
	// Users should set fsGroup on the pod spec directly if a specific fsGroup is desired.
	defaultFSGroup int64 = 1337
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
	if m.config.EnableIAMForServiceAccounts && (pod.Spec.SecurityContext == nil || pod.Spec.SecurityContext.FSGroup == nil) {
		dfsgroup := defaultFSGroup
		if pod.Spec.SecurityContext == nil {
			pod.Spec.SecurityContext = new(corev1.PodSecurityContext)
		}
		pod.Spec.SecurityContext.FSGroup = &dfsgroup
	}
	// Has image pull secret
	if m.config.EcrSecret {
		ecrS := corev1.LocalObjectReference{Name: ecrSecret}
		pod.Spec.ImagePullSecrets = append(pod.Spec.ImagePullSecrets, ecrS)
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
	}

	for _, mutator := range mutators {
		err := mutator.mutate(pod)
		if err != nil {
			return err
		}
	}
	return nil
}
