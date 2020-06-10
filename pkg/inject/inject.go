package inject

import (
	"context"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/references"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/virtualnode"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

var injectLogger = ctrl.Log.WithName("appmesh_inject")

type PodMutator interface {
	mutate(*corev1.Pod) error
}

type SidecarInjector struct {
	config                 Config
	accountID              string
	awsRegion              string
	k8sClient              client.Client
	referenceResolver      references.Resolver
	vnMembershipDesignator virtualnode.MembershipDesignator
}

func NewSidecarInjector(cfg Config, accountID string, awsRegion string,
	k8sClient client.Client,
	referenceResolver references.Resolver,
	vnMembershipDesignator virtualnode.MembershipDesignator) *SidecarInjector {
	return &SidecarInjector{
		config:                 cfg,
		accountID:              accountID,
		awsRegion:              awsRegion,
		k8sClient:              k8sClient,
		referenceResolver:      referenceResolver,
		vnMembershipDesignator: vnMembershipDesignator,
	}
}

func (m *SidecarInjector) Inject(ctx context.Context, pod *corev1.Pod) error {
	injectMode, err := m.determineSidecarInjectMode(pod)
	if err != nil {
		return errors.Wrap(err, "failed to determine sidecarInject mode")
	}
	if injectMode == sidecarInjectModeDisabled {
		return nil
	}
	vn, err := m.vnMembershipDesignator.Designate(ctx, pod)
	if err != nil {
		return err
	}

	if vn == nil || vn.Spec.MeshRef == nil {
		if injectMode == sidecarInjectModeEnabled {
			return errors.New("sidecarInject enabled but no matching VirtualNode found")
		}
		return nil
	}
	ms, err := m.referenceResolver.ResolveMeshReference(ctx, *vn.Spec.MeshRef)
	if err != nil {
		return err
	}
	return m.injectAppMeshPatches(ms, vn, pod)
}

func (m *SidecarInjector) injectAppMeshPatches(ms *appmesh.Mesh, vn *appmesh.VirtualNode, pod *corev1.Pod) error {
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
			accountID:             m.accountID,
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
		newCloudMapHealthyReadinessGate(vn),
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

type sidecarInjectMode string

const (
	// when enabled, a virtualNode must be found for pod, otherwise, pod will be rejected.
	sidecarInjectModeEnabled = "enabled"
	// when disabled, pod injection will be skipped.
	sidecarInjectModeDisabled = "disabled"
	// when unspecified, if a virtualNode is found for pod, pod will be injected, otherwise, pod will be skipped.
	sidecarInjectModeUnspecified = "unspecified"
)

func (m *SidecarInjector) determineSidecarInjectMode(pod *corev1.Pod) (sidecarInjectMode, error) {
	// The injector webhook uses the namespaceSelector to filter which requests
	// are intercepted. This makes sure all the requests sent to the injector have
	// have sidecar injection enabled based on the label defined by the user.
	// That's why we enable the sidecar inection by default here.
	// Namespace behavior can be overriden by pod level inject annotation
	sidecarInjectAnnotation := sidecarInjectModeEnabled

	if v, ok := pod.ObjectMeta.Annotations[AppMeshSidecarInjectAnnotation]; ok {
		sidecarInjectAnnotation = v
	}

	switch strings.ToLower(sidecarInjectAnnotation) {
	case "enabled":
		return sidecarInjectModeEnabled, nil
	case "disabled":
		return sidecarInjectModeDisabled, nil
	default:
		return sidecarInjectModeUnspecified, nil
	}
}
