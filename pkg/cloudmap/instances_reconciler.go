package cloudmap

import (
	"context"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws/services"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/runtime"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

const (
	// attrAWSInstanceIPV4 is a special attribute expected by CloudMap.
	// See https://github.com/aws/aws-sdk-go/blob/fd304fe4cb2ea1027e7fc7e21062beb768915fcc/service/servicediscovery/api.go#L5161
	attrAWSInstanceIPV4 = "AWS_INSTANCE_IPV4"

	// attrK8sPod is a custom attribute injected by app-mesh controller
	attrK8sPod = "k8s.io/pod"
	// AttrK8sNamespace is a custom attribute injected by app-mesh controller
	attrK8sNamespace = "k8s.io/namespace"
	// AttrK8sPodRegion is a custom attribute injected by app-mesh controller
	attrK8sPodRegion = "REGION"
	// AttrK8sPodAZ is a custom attribute injected by app-mesh controller
	attrK8sPodAZ = "AVAILABILITY_ZONE"

	attrAppMeshMesh        = "appmesh.k8s.aws/mesh"
	attrAppMeshVirtualNode = "appmesh.k8s.aws/virtualNode"

	// how long to synchronously wait for instances reconcile operation
	defaultInstancesReconcileWaitTimeout = 5 * time.Second
	// how long to requeue a instances reconcile operation
	defaultInstancesReconcileRequeueDuration = 10 * time.Second
	defaultInstancesHealthProbeTimeout       = 30 * time.Minute
)

type InstancesReconciler interface {
	Reconcile(ctx context.Context, ms *appmesh.Mesh, vn *appmesh.VirtualNode, service serviceSummary,
		readyPods []*corev1.Pod, notReadyPods []*corev1.Pod, nodeInfoByName map[string]nodeAttributes) error
}

func NewDefaultInstancesReconciler(k8sClient client.Client, cloudMapSDK services.CloudMap, log logr.Logger, stopChan <-chan struct{}) *defaultInstancesReconciler {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		select {
		case <-stopChan:
			cancel()
		}
	}()

	instancesReconcileReactor := newDefaultInstancesReconcileReactor(ctx, k8sClient, cloudMapSDK, log)
	instancesHealthProber := newDefaultInstancesHealthProber(ctx, k8sClient, cloudMapSDK, log)
	return &defaultInstancesReconciler{
		cloudMapSDK:               cloudMapSDK,
		instancesReconcileReactor: instancesReconcileReactor,
		instancesHealthProber:     instancesHealthProber,
		log:                       log,
	}
}

var _ InstancesReconciler = &defaultInstancesReconciler{}

type defaultInstancesReconciler struct {
	cloudMapSDK               services.CloudMap
	instancesReconcileReactor instancesReconcileReactor
	instancesHealthProber     instancesHealthProber
	log                       logr.Logger
}

func (r *defaultInstancesReconciler) Reconcile(ctx context.Context, ms *appmesh.Mesh, vn *appmesh.VirtualNode, service serviceSummary,
	readyPods []*corev1.Pod, notReadyPods []*corev1.Pod, nodeInfoByName map[string]nodeAttributes) error {

	customHealthCheckEnabled := service.healthCheckCustomConfig != nil
	subset := &virtualNodeServiceSubset{
		ms: ms,
		vn: vn,
	}
	readyInstanceInfoByID := r.buildInstanceInfoByID(ms, vn, readyPods, nodeInfoByName)
	var notReadyInstanceInfoByID map[string]instanceInfo
	if customHealthCheckEnabled {
		notReadyInstanceInfoByID = r.buildInstanceInfoByID(ms, vn, notReadyPods, nodeInfoByName)
	}
	resultChan := r.instancesReconcileReactor.Submit(ctx, service, subset, readyInstanceInfoByID, notReadyInstanceInfoByID)
	select {
	case <-time.After(defaultInstancesReconcileWaitTimeout):
		return runtime.NewRequeueAfterError(nil, defaultInstancesReconcileRequeueDuration)
	case err := <-resultChan:
		if err != nil {
			return runtime.NewRequeueError(err)
		}
	}
	if customHealthCheckEnabled {
		if err := r.reconcileCustomHealthCheck(ctx, service, readyInstanceInfoByID, notReadyInstanceInfoByID); err != nil {
			return err
		}
	}
	if err := r.instancesHealthProber.Submit(ctx, service, subset, readyInstanceInfoByID, defaultInstancesHealthProbeTimeout); err != nil {
		return err
	}
	return nil
}

func (r *defaultInstancesReconciler) reconcileCustomHealthCheck(ctx context.Context, service serviceSummary, readyInstanceInfoByID map[string]instanceInfo, notReadyInstanceInfoByID map[string]instanceInfo) error {
	for instanceID := range readyInstanceInfoByID {
		if _, err := r.cloudMapSDK.UpdateInstanceCustomHealthStatusWithContext(ctx, &servicediscovery.UpdateInstanceCustomHealthStatusInput{
			ServiceId:  aws.String(service.serviceID),
			InstanceId: aws.String(instanceID),
			Status:     aws.String(servicediscovery.CustomHealthStatusHealthy),
		}); err != nil {
			return err
		}
	}
	for instanceID := range notReadyInstanceInfoByID {
		if _, err := r.cloudMapSDK.UpdateInstanceCustomHealthStatusWithContext(ctx, &servicediscovery.UpdateInstanceCustomHealthStatusInput{
			ServiceId:  aws.String(service.serviceID),
			InstanceId: aws.String(instanceID),
			Status:     aws.String(servicediscovery.CustomHealthStatusUnhealthy),
		}); err != nil {
			return err
		}
	}
	return nil
}

// buildInstanceInfoByID build instances info indexed by instanceID
func (r *defaultInstancesReconciler) buildInstanceInfoByID(ms *appmesh.Mesh, vn *appmesh.VirtualNode,
	pods []*corev1.Pod, nodeInfoByName map[string]nodeAttributes) map[string]instanceInfo {
	instanceInfoByID := make(map[string]instanceInfo, len(pods))
	for _, pod := range pods {
		instanceID := r.buildInstanceID(pod)
		instanceAttrs := r.buildInstanceAttributes(ms, vn, pod, nodeInfoByName)
		instanceInfoByID[instanceID] = instanceInfo{
			attrs: instanceAttrs,
			pod:   pod,
		}
	}
	return instanceInfoByID
}

func (r *defaultInstancesReconciler) buildInstanceAttributes(ms *appmesh.Mesh, vn *appmesh.VirtualNode,
	pod *corev1.Pod, nodeInfoByName map[string]nodeAttributes) instanceAttributes {
	attr := make(map[string]string)
	for label, v := range pod.Labels {
		attr[label] = v
	}
	for _, cmAttr := range vn.Spec.ServiceDiscovery.AWSCloudMap.Attributes {
		attr[cmAttr.Key] = cmAttr.Value
	}
	podsNodeName := pod.Spec.NodeName
	attr[attrAWSInstanceIPV4] = pod.Status.PodIP
	attr[attrK8sPod] = pod.Name
	attr[attrK8sNamespace] = pod.Namespace
	attr[attrAppMeshMesh] = aws.StringValue(ms.Spec.AWSName)
	attr[attrAppMeshVirtualNode] = aws.StringValue(vn.Spec.AWSName)
	if nodeInfo, ok := nodeInfoByName[podsNodeName]; ok {
		if nodeInfo.region != "" {
			attr[attrK8sPodRegion] = nodeInfo.region
		}
		if nodeInfo.availabilityZone != "" {
			attr[attrK8sPodAZ] = nodeInfo.availabilityZone
		}
	}
	return attr
}

func (r *defaultInstancesReconciler) buildInstanceID(pod *corev1.Pod) string {
	return pod.Status.PodIP
}
