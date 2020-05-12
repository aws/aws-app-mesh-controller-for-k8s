package cloudmap

import (
	"context"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws/services"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/k8s"
	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// attrAwsInstanceIPV4 is a special attribute expected by CloudMap.
	// See https://github.com/aws/aws-sdk-go/blob/fd304fe4cb2ea1027e7fc7e21062beb768915fcc/service/servicediscovery/api.go#L5161
	attrAwsInstanceIPV4     = "AWS_INSTANCE_IPV4"
	attrAwsInitHealthStatus = "AWS_INIT_HEALTH_STATUS"

	// attrK8sPod is a custom attribute injected by app-mesh controller
	attrK8sPod = "k8s.io/pod"
	// AttrK8sNamespace is a custom attribute injected by app-mesh controller
	attrK8sNamespace = "k8s.io/namespace"
)

type InstancesReconciler interface {
	Reconcile(ctx context.Context, vn *appmesh.VirtualNode, serviceID string, customHealthCheckEnabled bool,
		readyPods []*corev1.Pod, notReadyPods []*corev1.Pod) error
}

func NewDefaultInstancesReconciler(k8sClient client.Client, cloudMapSDK services.CloudMap, instancesCache InstancesCache, instancesHealthProber InstancesHealthProber, log logr.Logger) *defaultInstancesReconciler {
	return &defaultInstancesReconciler{
		k8sClient:             k8sClient,
		cloudMapSDK:           cloudMapSDK,
		instancesCache:        instancesCache,
		instancesHealthProber: instancesHealthProber,
		log:                   log,
	}
}

var _ InstancesReconciler = &defaultInstancesReconciler{}

type defaultInstancesReconciler struct {
	cloudMapSDK           services.CloudMap
	k8sClient             client.Client
	instancesCache        InstancesCache
	instancesHealthProber InstancesHealthProber
	log                   logr.Logger
}

func (r *defaultInstancesReconciler) Reconcile(ctx context.Context, vn *appmesh.VirtualNode, serviceID string, customHealthCheckEnabled bool,
	readyPods []*corev1.Pod, notReadyPods []*corev1.Pod) error {
	instanceProbes := r.buildInstanceProbes(readyPods)
	if customHealthCheckEnabled {
		if err := r.instancesHealthProber.SubmitProbe(ctx, serviceID, instanceProbes); err != nil {
			return err
		}
	} else {
		if err := r.unblockCMHealthyReadinessGateImmediately(ctx, instanceProbes); err != nil {
			return err
		}
	}

	existingInstancesAttrsByID, err := r.instancesCache.ListInstances(ctx, serviceID)
	if err != nil {
		return err
	}
	existingInstancesAttrsByIDForVN := r.filterExistingInstancesAttrsIDForVirtualNode(vn, existingInstancesAttrsByID)
	desiredReadyInstancesAttrsByID := r.buildInstanceAttributesByID(vn, readyPods)
	var desiredNotReadyInstancesAttrsByID map[string]InstanceAttributes
	if customHealthCheckEnabled {
		desiredNotReadyInstancesAttrsByID = r.buildInstanceAttributesByID(vn, notReadyPods)
	} else {
		desiredNotReadyInstancesAttrsByID = nil
	}

	instancesToCreateOrUpdate, instancesToDelete, instancesToUpdateHealthy, instancesToUpdateUnhealthy := r.matchDesiredInstancesAgainstExistingInstances(desiredReadyInstancesAttrsByID, desiredNotReadyInstancesAttrsByID, existingInstancesAttrsByIDForVN)
	r.log.V(1).Info("instances reconcile",
		"instancesToCreateOrUpdate", instancesToCreateOrUpdate,
		"instancesToDelete", instancesToDelete,
		"instancesToUpdateHealthy", instancesToUpdateHealthy,
		"instancesToUpdateUnhealthy", instancesToUpdateUnhealthy,
	)
	if customHealthCheckEnabled {
		for _, instanceID := range instancesToUpdateHealthy {
			if _, err := r.cloudMapSDK.UpdateInstanceCustomHealthStatusWithContext(ctx, &servicediscovery.UpdateInstanceCustomHealthStatusInput{
				ServiceId:  awssdk.String(serviceID),
				InstanceId: awssdk.String(instanceID),
				Status:     awssdk.String(servicediscovery.CustomHealthStatusHealthy),
			}); err != nil {
				return err
			}
		}
		for _, instanceID := range instancesToUpdateUnhealthy {
			if _, err := r.cloudMapSDK.UpdateInstanceCustomHealthStatusWithContext(ctx, &servicediscovery.UpdateInstanceCustomHealthStatusInput{
				ServiceId:  awssdk.String(serviceID),
				InstanceId: awssdk.String(instanceID),
				Status:     awssdk.String(servicediscovery.CustomHealthStatusUnhealthy),
			}); err != nil {
				return err
			}
		}
	}

	for instanceID, attrs := range instancesToCreateOrUpdate {
		if err := r.instancesCache.RegisterInstance(ctx, serviceID, instanceID, attrs); err != nil {
			return err
		}
	}
	for _, instanceID := range instancesToDelete {
		if err := r.instancesCache.DeregisterInstance(ctx, serviceID, instanceID); err != nil {
			return err
		}
	}
	return nil
}

func (r *defaultInstancesReconciler) matchDesiredInstancesAgainstExistingInstances(
	desiredReadyInstancesAttrsByID map[string]InstanceAttributes,
	desiredNotReadyInstancesAttrsByID map[string]InstanceAttributes,
	existingInstancesAttrsByID map[string]InstanceAttributes) (map[string]InstanceAttributes, []string, []string, []string) {

	instancesToCreateOrUpdate := make(map[string]InstanceAttributes)
	var instancesToUpdateHealthy []string
	var instancesToUpdateUnhealthy []string

	for instanceID, desiredAttrs := range desiredReadyInstancesAttrsByID {
		if existingAttrs, exists := existingInstancesAttrsByID[instanceID]; exists {
			if !cmp.Equal(desiredAttrs, existingAttrs, ignoreAttrAwsInitHealthStatus()) {
				if existingInitHealthStatus, ok := existingAttrs[attrAwsInitHealthStatus]; ok {
					desiredAttrs[attrAwsInitHealthStatus] = existingInitHealthStatus
				} else {
					desiredAttrs[attrAwsInitHealthStatus] = servicediscovery.CustomHealthStatusHealthy
				}
				instancesToCreateOrUpdate[instanceID] = desiredAttrs
			}
			instancesToUpdateHealthy = append(instancesToUpdateHealthy, instanceID)
		} else {
			desiredAttrs[attrAwsInitHealthStatus] = servicediscovery.CustomHealthStatusHealthy
			instancesToCreateOrUpdate[instanceID] = desiredAttrs
		}
	}

	for instanceID, desiredAttrs := range desiredNotReadyInstancesAttrsByID {
		if existingAttrs, exists := existingInstancesAttrsByID[instanceID]; exists {
			if !cmp.Equal(desiredAttrs, existingAttrs, ignoreAttrAwsInitHealthStatus()) {
				if existingInitHealthStatus, ok := existingAttrs[attrAwsInitHealthStatus]; ok {
					desiredAttrs[attrAwsInitHealthStatus] = existingInitHealthStatus
				} else {
					desiredAttrs[attrAwsInitHealthStatus] = servicediscovery.CustomHealthStatusUnhealthy
				}
				instancesToCreateOrUpdate[instanceID] = desiredAttrs
			}
			instancesToUpdateUnhealthy = append(instancesToUpdateUnhealthy, instanceID)
		}
	}
	desiredInstanceIDs := sets.StringKeySet(desiredReadyInstancesAttrsByID).Union(sets.StringKeySet(desiredNotReadyInstancesAttrsByID))
	existingInstanceIDs := sets.StringKeySet(existingInstancesAttrsByID)
	instancesToDelete := existingInstanceIDs.Difference(desiredInstanceIDs).List()
	return instancesToCreateOrUpdate, instancesToDelete, instancesToUpdateHealthy, instancesToUpdateUnhealthy
}

func (r *defaultInstancesReconciler) filterExistingInstancesAttrsIDForVirtualNode(vn *appmesh.VirtualNode, existingInstancesAttrsByID map[string]InstanceAttributes) map[string]InstanceAttributes {
	labelSelector, _ := metav1.LabelSelectorAsSelector(vn.Spec.PodSelector)
	existingInstancesAttrsIDForVirtualNode := make(map[string]InstanceAttributes, len(existingInstancesAttrsByID))
	for instanceID, attrs := range existingInstancesAttrsByID {
		if attrs[attrK8sNamespace] == vn.Namespace && labelSelector.Matches(labels.Set(attrs)) {
			existingInstancesAttrsIDForVirtualNode[instanceID] = attrs
		}
	}
	return existingInstancesAttrsIDForVirtualNode
}

// unblockCMHealthyReadinessGateImmediately will unblock readinessGate immediately.
// this is for **backwards-compatibility** for old cloudMapService we created, which don't have custom healthCheck(when using controller < v1.0.0).
func (r *defaultInstancesReconciler) unblockCMHealthyReadinessGateImmediately(ctx context.Context, instances []InstanceProbe) error {
	instancesToUnblock := filterInstancesBlockedByCMHealthyReadinessGate(instances)
	for _, instance := range instancesToUnblock {
		oldPod := instance.pod.DeepCopy()
		if updated := k8s.UpdatePodCondition(instance.pod, k8s.ConditionAWSCloudMapHealthy, corev1.ConditionTrue, nil, nil); updated {
			if err := r.k8sClient.Status().Patch(ctx, instance.pod, client.MergeFrom(oldPod)); err != nil {
				return err
			}
		}
	}
	return nil
}

// buildInstanceAttributesByID build instances attributes indexed by instanceID
func (r *defaultInstancesReconciler) buildInstanceAttributesByID(vn *appmesh.VirtualNode, pods []*corev1.Pod) map[string]InstanceAttributes {
	instanceAttrsByID := make(map[string]InstanceAttributes, len(pods))
	for _, pod := range pods {
		instanceID := r.buildInstanceID(pod)
		instanceAttrs := r.buildInstanceAttributes(vn, pod)
		instanceAttrsByID[instanceID] = instanceAttrs
	}
	return instanceAttrsByID
}

func (r *defaultInstancesReconciler) buildInstanceProbes(pods []*corev1.Pod) []InstanceProbe {
	var instanceProbes []InstanceProbe
	for _, pod := range pods {
		instanceID := r.buildInstanceID(pod)
		instanceProbes = append(instanceProbes, InstanceProbe{
			instanceID: instanceID,
			pod:        pod,
		})
	}
	return instanceProbes
}

func (r *defaultInstancesReconciler) buildInstanceAttributes(vn *appmesh.VirtualNode, pod *corev1.Pod) InstanceAttributes {
	attr := make(map[string]string)
	for label, v := range pod.Labels {
		attr[label] = v
	}
	for _, cmAttr := range vn.Spec.ServiceDiscovery.AWSCloudMap.Attributes {
		attr[cmAttr.Key] = cmAttr.Value
	}
	attr[attrAwsInstanceIPV4] = pod.Status.PodIP
	attr[attrK8sPod] = pod.Name
	attr[attrK8sNamespace] = pod.Namespace
	return attr
}

func (r *defaultInstancesReconciler) buildInstanceID(pod *corev1.Pod) string {
	return pod.Status.PodIP
}

func ignoreAttrAwsInitHealthStatus() cmp.Option {
	return cmpopts.IgnoreMapEntries(func(key string, _ string) bool {
		return key == attrAwsInitHealthStatus
	})
}
