package inject

import (
	"context"
	"strings"

	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/references"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/virtualgateway"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/virtualnode"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/webhook"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var injectLogger = ctrl.Log.WithName("appmesh_inject")

type PodMutator interface {
	mutate(*corev1.Pod) error
}

type SidecarInjector struct {
	config                 Config
	accountID              string
	awsRegion              string
	controllerVersion      string
	k8sVersion             string
	k8sClient              client.Client
	referenceResolver      references.Resolver
	vgMembershipDesignator virtualgateway.MembershipDesignator
	vnMembershipDesignator virtualnode.MembershipDesignator
}

func NewSidecarInjector(cfg Config, accountID string, awsRegion string, controllerVersion string, k8sVersion string,
	k8sClient client.Client,
	referenceResolver references.Resolver,
	vnMembershipDesignator virtualnode.MembershipDesignator,
	vgMembershipDesignator virtualgateway.MembershipDesignator) *SidecarInjector {
	return &SidecarInjector{
		config:                 cfg,
		accountID:              accountID,
		awsRegion:              awsRegion,
		controllerVersion:      controllerVersion,
		k8sVersion:             k8sVersion,
		k8sClient:              k8sClient,
		referenceResolver:      referenceResolver,
		vgMembershipDesignator: vgMembershipDesignator,
		vnMembershipDesignator: vnMembershipDesignator,
	}
}

func (m *SidecarInjector) Inject(ctx context.Context, pod *corev1.Pod) error {
	injectMode, err := m.determineSidecarInjectMode(ctx, pod)
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

	vg, err := m.vgMembershipDesignator.DesignateForPod(ctx, pod)
	if err != nil {
		return err
	}

	if vn != nil && vg != nil {
		return errors.Errorf("sidecarInject enabled for both virtualNode %s and virtualGateway %s on pod %s. Please use podSelector on one", vn.Name, vg.Name, pod.Name)
	}

	if (vn == nil || vn.Spec.MeshRef == nil) && (vg == nil || vg.Spec.MeshRef == nil) {
		if injectMode == sidecarInjectModeEnabled {
			return errors.New("sidecarInject enabled but no matching VirtualNode or VirtualGateway found")
		}
		return nil
	}

	var msRef *appmesh.MeshReference
	if vn != nil {
		msRef = vn.Spec.MeshRef
	} else if vg != nil {
		msRef = vg.Spec.MeshRef
	} else {
		return errors.New("No matching VirtualNode or VirtualGateway found to resolve Mesh reference")
	}

	ms, err := m.referenceResolver.ResolveMeshReference(ctx, *msRef)
	if err != nil {
		return err
	}
	return m.injectAppMeshPatches(ms, vn, vg, pod)
}

func (m *SidecarInjector) injectAppMeshPatches(ms *appmesh.Mesh, vn *appmesh.VirtualNode, vg *appmesh.VirtualGateway, pod *corev1.Pod) error {
	// List out all the mutators in sequence
	var mutators []PodMutator

	if vn != nil {
		mutators = []PodMutator{
			newProxyMutator(proxyMutatorConfig{
				egressIgnoredIPs: m.config.IgnoredIPs,
				initProxyMutatorConfig: initProxyMutatorConfig{
					containerImage: m.config.InitImage,
					cpuRequests:    m.config.SidecarCpuRequests,
					memoryRequests: m.config.SidecarMemoryRequests,
					cpuLimits:      m.config.SidecarCpuLimits,
					memoryLimits:   m.config.SidecarMemoryLimits,
				},
			}, vn),
			newEnvoyMutator(envoyMutatorConfig{
				accountID:                  m.accountID,
				awsRegion:                  m.awsRegion,
				preview:                    m.config.Preview,
				enableSDS:                  m.config.EnableSDS,
				sdsUdsPath:                 m.config.SdsUdsPath,
				logLevel:                   m.config.LogLevel,
				adminAccessPort:            m.config.EnvoyAdminAcessPort,
				adminAccessLogFile:         m.config.EnvoyAdminAccessLogFile,
				preStopDelay:               m.config.PreStopDelay,
				readinessProbeInitialDelay: m.config.ReadinessProbeInitialDelay,
				readinessProbePeriod:       m.config.ReadinessProbePeriod,
				sidecarImageRepository:     m.config.SidecarImageRepository,
				sidecarImageTag:            m.config.SidecarImageTag,
				sidecarCPURequests:         m.config.SidecarCpuRequests,
				sidecarMemoryRequests:      m.config.SidecarMemoryRequests,
				sidecarCPULimits:           m.config.SidecarCpuLimits,
				sidecarMemoryLimits:        m.config.SidecarMemoryLimits,
				enableXrayTracing:          m.config.EnableXrayTracing,
				xrayDaemonPort:             m.config.XrayDaemonPort,
				xraySamplingRate:           m.config.XraySamplingRate,
				enableJaegerTracing:        m.config.EnableJaegerTracing,
				jaegerPort:                 m.config.JaegerPort,
				jaegerAddress:              m.config.JaegerAddress,
				enableDatadogTracing:       m.config.EnableDatadogTracing,
				datadogTracerPort:          m.config.DatadogPort,
				datadogTracerAddress:       m.config.DatadogAddress,
				enableStatsTags:            m.config.EnableStatsTags,
				enableStatsD:               m.config.EnableStatsD,
				statsDPort:                 m.config.StatsDPort,
				statsDAddress:              m.config.StatsDAddress,
				statsDSocketPath:           m.config.StatsDSocketPath,
				waitUntilProxyReady:        m.config.WaitUntilProxyReady,
				controllerVersion:          m.controllerVersion,
				k8sVersion:                 m.k8sVersion,
				useDualStackEndpoint:       m.config.DualStackEndpoint,
				enableAdminAccessIPv6:      m.config.EnvoyAdminAccessEnableIPv6,
				postStartTimeout:           m.config.PostStartTimeout,
				postStartInterval:          m.config.PostStartInterval,
				useFipsEndpoint:            m.config.FipsEndpoint,
				awsAccessKeyId:             m.config.EnvoyAwsAccessKeyId,
				awsSecretAccessKey:         m.config.EnvoyAwsSecretAccessKey,
				awsSessionToken:            m.config.EnvoyAwsSessionToken,
			}, ms, vn),
			newXrayMutator(xrayMutatorConfig{
				awsRegion:             m.awsRegion,
				sidecarCPURequests:    m.config.SidecarCpuRequests,
				sidecarMemoryRequests: m.config.SidecarMemoryRequests,
				sidecarCPULimits:      m.config.SidecarCpuLimits,
				sidecarMemoryLimits:   m.config.SidecarMemoryLimits,
				xRayImage:             m.config.XRayImage,
				xRayDaemonPort:        m.config.XrayDaemonPort,
				xRayLogLevel:          m.config.XrayLogLevel,
				xRayConfigRoleArn:     m.config.XrayConfigRoleArn,
			}, m.config.EnableXrayTracing),
			newJaegerMutator(jaegerMutatorConfig{
				jaegerAddress: m.config.JaegerAddress,
				jaegerPort:    m.config.JaegerPort,
			}, m.config.EnableJaegerTracing),
			newCloudMapHealthyReadinessGate(vn),
			newIAMForServiceAccountsMutator(m.config.EnableIAMForServiceAccounts),
			newECRSecretMutator(m.config.EnableECRSecret),
		}
	} else if vg != nil {
		mutators = []PodMutator{newVirtualGatewayEnvoyConfig(virtualGatwayEnvoyConfig{
			accountID:                  m.accountID,
			awsRegion:                  m.awsRegion,
			preview:                    m.config.Preview,
			enableSDS:                  m.config.EnableSDS,
			sdsUdsPath:                 m.config.SdsUdsPath,
			logLevel:                   m.config.LogLevel,
			adminAccessPort:            m.config.EnvoyAdminAcessPort,
			adminAccessLogFile:         m.config.EnvoyAdminAccessLogFile,
			sidecarImageRepository:     m.config.SidecarImageRepository,
			sidecarImageTag:            m.config.SidecarImageTag,
			readinessProbeInitialDelay: m.config.ReadinessProbeInitialDelay,
			readinessProbePeriod:       m.config.ReadinessProbePeriod,
			enableXrayTracing:          m.config.EnableXrayTracing,
			xrayDaemonPort:             m.config.XrayDaemonPort,
			xraySamplingRate:           m.config.XraySamplingRate,
			enableJaegerTracing:        m.config.EnableJaegerTracing,
			jaegerPort:                 m.config.JaegerPort,
			jaegerAddress:              m.config.JaegerAddress,
			enableDatadogTracing:       m.config.EnableDatadogTracing,
			datadogTracerPort:          m.config.DatadogPort,
			datadogTracerAddress:       m.config.DatadogAddress,
			enableStatsTags:            m.config.EnableStatsTags,
			enableStatsD:               m.config.EnableStatsD,
			statsDPort:                 m.config.StatsDPort,
			statsDAddress:              m.config.StatsDAddress,
			statsDSocketPath:           m.config.StatsDSocketPath,
			controllerVersion:          m.controllerVersion,
			k8sVersion:                 m.k8sVersion,
			useDualStackEndpoint:       m.config.DualStackEndpoint,
			enableAdminAccessIPv6:      m.config.EnvoyAdminAccessEnableIPv6,
			useFipsEndpoint:            m.config.FipsEndpoint,
			awsAccessKeyId:             m.config.EnvoyAwsAccessKeyId,
			awsSecretAccessKey:         m.config.EnvoyAwsSecretAccessKey,
			awsSessionToken:            m.config.EnvoyAwsSessionToken,
		}, ms, vg),
			newXrayMutator(xrayMutatorConfig{
				awsRegion:             m.awsRegion,
				sidecarCPURequests:    m.config.SidecarCpuRequests,
				sidecarMemoryRequests: m.config.SidecarMemoryRequests,
				sidecarCPULimits:      m.config.SidecarCpuLimits,
				sidecarMemoryLimits:   m.config.SidecarMemoryLimits,
				xRayImage:             m.config.XRayImage,
				xRayDaemonPort:        m.config.XrayDaemonPort,
				xRayLogLevel:          m.config.XrayLogLevel,
				xRayConfigRoleArn:     m.config.XrayConfigRoleArn,
			}, m.config.EnableXrayTracing),
			newJaegerMutator(jaegerMutatorConfig{
				jaegerAddress: m.config.JaegerAddress,
				jaegerPort:    m.config.JaegerPort,
			}, m.config.EnableJaegerTracing),
		}
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

func (m *SidecarInjector) determineSidecarInjectMode(ctx context.Context, pod *corev1.Pod) (sidecarInjectMode, error) {
	// The injector webhook uses the namespaceSelector to filter which requests
	// are intercepted. This makes sure all the requests sent to the injector have
	// sidecar injection label `appmesh.k8s.aws/sidecarInjectorWebhook` specified with valid values: enabled and disabled

	// appmesh.k8s.aws/sidecarInjectorWebhook: disabled
	// The sidecar injector will not inject the sidecar into pods by default. Add the `appmesh.k8s.aws/sidecarInjectorWebhook` annotation
	// with value `enabled` to the pod template spec to override the default and enable injection

	// appmesh.k8s.aws/sidecarInjectorWebhook: enabled
	// The sidecar injector will inject the sidecar into pods by default. Add the `appmesh.k8s.aws/sidecarInjectorWebhook` annotation
	// with value `disabled` to the pod template spec to override the default and disable injection.

	var namespaceDefaultInjectionMode string

	// see https://github.com/kubernetes/kubernetes/issues/88282 and https://github.com/kubernetes/kubernetes/issues/76680
	req := webhook.ContextGetAdmissionRequest(ctx)
	objectNS := &corev1.Namespace{}
	if err := m.k8sClient.Get(ctx, types.NamespacedName{Name: req.Namespace}, objectNS); err != nil {
		return sidecarInjectModeUnspecified, err
	}
	if v, ok := objectNS.ObjectMeta.Labels[AppMeshSidecarInjectAnnotation]; ok {
		namespaceDefaultInjectionMode = v
	}

	sidecarInjectAnnotation := namespaceDefaultInjectionMode

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
