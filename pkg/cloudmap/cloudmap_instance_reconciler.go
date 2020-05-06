package cloudmap

import (
	"context"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	awsservices "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws/services"
	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/cache"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"time"
)

const (
	cloudMapInstanceCacheMaxSize = 2048
	cloudMapInstanceCacheTTL     = 5 * time.Minute
)

var cloudMapBackoff = wait.Backoff{
	Steps:    4,
	Duration: 15 * time.Second,
	Factor:   1.0,
	Jitter:   0.1,
	Cap:      60 * time.Second,
}

type cloudMapInstanceCacheItem struct {
	key             string
	instanceSummary map[string]*cloudMapInstanceInfo
}

type cloudMapInstanceInfo struct {
	podName         string
	namespace       string
	virtualNodeName string
	meshName        string
	healthStatus    string
	labels          map[string]string
}

type CloudMapInstanceReconciler interface {
	ReconcileCloudMapInstances(ctx context.Context, readyPods corev1.PodList,
		notReadyPods corev1.PodList, ignoredPods corev1.PodList, serviceID string,
		namespaceID string, vNode *appmesh.VirtualNode) error
	CleanUpCloudMapInstances(ctx context.Context, serviceID string, namespaceID string,
		cloudMapConfig *appmesh.AWSCloudMapServiceDiscovery) error
}

type CloudMapInstanceResolver struct {
	cloudMapSDK           awsservices.CloudMap
	cloudMapInstanceCache *cache.LRUExpireCache
	log                   logr.Logger
}

func NewCloudMapInstanceResolver(cloudMapSDK awsservices.CloudMap, log logr.Logger) *CloudMapInstanceResolver {
	return &CloudMapInstanceResolver{
		cloudMapSDK:           cloudMapSDK,
		cloudMapInstanceCache: cache.NewLRUExpireCache(cloudMapInstanceCacheMaxSize),
		log:                   log,
	}
}

func (r *CloudMapInstanceResolver) ReconcileCloudMapInstances(ctx context.Context, readyPods corev1.PodList,
	notReadyPods corev1.PodList, ignoredPods corev1.PodList, serviceID string, namespaceID string,
	vNode *appmesh.VirtualNode) error {

	r.log.V(1).Info("CloudMap Reconciler: ", "Ready Pods Count: ", len(readyPods.Items),
		"Unready Pods Count: ", len(notReadyPods.Items), "Ignored Pods Count: ", len(ignoredPods.Items))

	cloudMapConfig := vNode.Spec.ServiceDiscovery.AWSCloudMap
	totalInstancesToBeRegistered := len(readyPods.Items) + len(notReadyPods.Items)
	currentRegisteredInstances, err := r.getRegisteredCloudMapServiceInstances(ctx, cloudMapConfig, serviceID)

	if err != nil {
		return err
	}
	if totalInstancesToBeRegistered == 0 && len(currentRegisteredInstances) != 0 {
		if err := r.deleteAWSCloudMapServiceInstances(ctx, cloudMapConfig, serviceID); err != nil {
			return err
		}
		r.log.Info("No Active Pods matching this VirtualNode")
		return nil
	}

	var instanceId string
	serviceInstanceSummary := make(map[string]*cloudMapInstanceInfo)
	for _, pod := range readyPods.Items {
		instanceInfo := &cloudMapInstanceInfo{}
		instanceInfo.podName = pod.Name
		instanceInfo.namespace = pod.Namespace
		instanceInfo.virtualNodeName = vNode.Name
		instanceInfo.meshName = vNode.Name
		instanceId = pod.Status.PodIP
		instanceInfo.healthStatus = awsservices.InstanceHealthy
		instanceInfo.labels = make(map[string]string)
		for label, v := range pod.Labels {
			instanceInfo.labels[label] = v
		}
		serviceInstanceSummary[instanceId] = instanceInfo
	}

	for _, pod := range notReadyPods.Items {
		instanceInfo := &cloudMapInstanceInfo{}
		instanceInfo.podName = pod.Name
		instanceInfo.namespace = pod.Namespace
		instanceInfo.virtualNodeName = vNode.Name
		instanceInfo.meshName = vNode.Name
		instanceId = pod.Status.PodIP
		instanceInfo.healthStatus = awsservices.InstanceUnHealthy
		instanceInfo.labels = make(map[string]string)
		for label, v := range pod.Labels {
			instanceInfo.labels[label] = v
		}
		serviceInstanceSummary[instanceId] = instanceInfo
	}

	if err := r.updateCloudMapServiceInstances(ctx, cloudMapConfig, serviceInstanceSummary, serviceID, namespaceID); err != nil {
		r.log.Error(err, " Failed to Update/Register instances to CloudMap")
		return err
	}
	return nil
}

func (r *CloudMapInstanceResolver) CleanUpCloudMapInstances(ctx context.Context, serviceID string, namespaceID string,
	cloudMapConfig *appmesh.AWSCloudMapServiceDiscovery) error {

	if err := r.deleteAWSCloudMapServiceInstances(ctx, cloudMapConfig, serviceID); err != nil {
		return err
	}
	return nil
}

func (r *CloudMapInstanceResolver) getServiceInstancesFromCloudMap(ctx context.Context,
	cloudMapConfig *appmesh.AWSCloudMapServiceDiscovery, serviceID string) ([]*servicediscovery.InstanceSummary, error) {
	instances := []*servicediscovery.InstanceSummary{}

	input := &servicediscovery.ListInstancesInput{
		ServiceId: awssdk.String(serviceID),
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*awsservices.ListInstancesPagesTimeout)
	defer cancel()

	r.cloudMapSDK.ListInstancesPagesWithContext(ctx, input, func(output *servicediscovery.ListInstancesOutput, lastPage bool) bool {
		for _, i := range output.Instances {
			if _, ok := i.Attributes[awsservices.AttrK8sNamespace]; ok {
				if _, ok := i.Attributes[awsservices.AttrK8sPod]; ok {
					instances = append(instances, i)
				}
			}

		}
		return false
	})
	return instances, nil
}

func (r *CloudMapInstanceResolver) getRegisteredCloudMapServiceInstances(ctx context.Context,
	cloudmapConfig *appmesh.AWSCloudMapServiceDiscovery, serviceID string) (map[string]*cloudMapInstanceInfo, error) {

	var cacheServiceInstanceSummary map[string]*cloudMapInstanceInfo

	cloudMapServiceKey := cloudmapConfig.NamespaceName + "-" + cloudmapConfig.ServiceName
	existingItem, exists := r.cloudMapInstanceCache.Get(cloudMapServiceKey)

	if exists {
		cacheServiceInstanceSummary = existingItem.(map[string]*cloudMapInstanceInfo)
	} else {
		//UpdateCache from CloudMap for this service
		serviceInstances, err := r.getServiceInstancesFromCloudMap(ctx, cloudmapConfig, serviceID)
		if err != nil {
			return cacheServiceInstanceSummary, err
		}

		cacheServiceInstanceSummary = make(map[string]*cloudMapInstanceInfo)
		for _, instance := range serviceInstances {
			registeredInstanceInfo := &cloudMapInstanceInfo{}
			podName := awssdk.StringValue(instance.Attributes[awsservices.AttrK8sPod])
			namespace := awssdk.StringValue(instance.Attributes[awsservices.AttrK8sNamespace])
			instanceID := awssdk.StringValue(instance.Id)
			healthStatus := awssdk.StringValue(instance.Attributes[awsservices.AttrAwsInstanceHealthStatus])

			registeredInstanceInfo.podName = podName
			registeredInstanceInfo.namespace = namespace
			registeredInstanceInfo.healthStatus = healthStatus
			cacheServiceInstanceSummary[instanceID] = registeredInstanceInfo
		}
	}
	return cacheServiceInstanceSummary, nil
}

func (r *CloudMapInstanceResolver) updateCloudMapServiceInstances(ctx context.Context, cloudMapConfig *appmesh.AWSCloudMapServiceDiscovery,
	currentServiceInstanceSummary map[string]*cloudMapInstanceInfo, serviceID string, namespaceID string) error {

	cacheServiceInstanceSummary, err := r.getRegisteredCloudMapServiceInstances(ctx, cloudMapConfig, serviceID)
	cloudMapServiceKey := r.serviceInstanceCacheKey(cloudMapConfig)

	updatedCacheServiceInstanceSummary := make(map[string]*cloudMapInstanceInfo)
	for instanceId, instanceInfo := range currentServiceInstanceSummary {
		currInstanceInfo := &cloudMapInstanceInfo{}
		currInstanceInfo.podName = instanceInfo.podName
		currInstanceInfo.namespace = instanceInfo.namespace
		currInstanceInfo.meshName = instanceInfo.meshName
		currInstanceInfo.virtualNodeName = instanceInfo.virtualNodeName
		if _, ok := cacheServiceInstanceSummary[instanceId]; ok {
			//Check if there is a change in health status
			if instanceInfo.healthStatus != cacheServiceInstanceSummary[instanceId].healthStatus {
				if err := r.updateCloudMapInstanceHealthStatus(ctx, cloudMapConfig, instanceId, instanceInfo.healthStatus, serviceID); err != nil {
					r.log.Error(err, "Error Updating Instance: ", instanceId, " health Status in CloudMap")
				}
			}
			currInstanceInfo.healthStatus = instanceInfo.healthStatus
			delete(cacheServiceInstanceSummary, instanceId)
		} else {
			//Instance Missing in CloudMap. Register it.
			if err := r.registerCloudMapInstance(ctx, cloudMapConfig, instanceId, instanceInfo, serviceID); err != nil {
				r.log.Error(err, "Error Registering Instance: ", instanceId, " to CloudMap")
				continue
			}
			currInstanceInfo.healthStatus = instanceInfo.healthStatus
		}
		updatedCacheServiceInstanceSummary[instanceId] = currInstanceInfo
	}

	//Delete the instances that are no longer part of the current instance list
	err = r.removeCloudMapServiceInstances(ctx, cloudMapServiceKey, cacheServiceInstanceSummary, cloudMapConfig, serviceID)
	if err != nil {
		return err
	}

	instanceKey := cloudMapServiceKey
	instanceSummary := updatedCacheServiceInstanceSummary
	r.cloudMapInstanceCache.Add(instanceKey, instanceSummary, cloudMapInstanceCacheTTL)
	return nil
}

func (r *CloudMapInstanceResolver) removeCloudMapServiceInstances(ctx context.Context, cloudMapServiceKey string,
	serviceInstancesToBeRemoved map[string]*cloudMapInstanceInfo,
	cloudMapConfig *appmesh.AWSCloudMapServiceDiscovery, serviceID string) error {

	for instanceId, _ := range serviceInstancesToBeRemoved {
		r.log.V(1).Info("De-registering pod: ", "InstanceID: ", instanceId)
		err := r.deregisterCloudMapInstance(ctx, instanceId, cloudMapConfig, serviceID)
		if err != nil {
			//Log an error and continue
			r.log.Error(err, "Couldn't deregister cloudmap resource", "instance: ", instanceId)
		}
	}
	return nil
}

func (r *CloudMapInstanceResolver) updateCloudMapInstanceHealthStatus(ctx context.Context,
	cloudMapConfig *appmesh.AWSCloudMapServiceDiscovery,
	instanceId string, healthStatus string, serviceID string) error {

	r.log.Info("vNode", "Updating Health status of instance: ", *awssdk.String(instanceId),
		" to: ", *awssdk.String(healthStatus))
	updateRequest := &servicediscovery.UpdateInstanceCustomHealthStatusInput{
		InstanceId: awssdk.String(instanceId),
		ServiceId:  awssdk.String(serviceID),
		Status:     awssdk.String(healthStatus),
	}
	if err := retry.OnError(cloudMapBackoff, func(err error) bool {
		if awsErr, ok := err.(awserr.Error); ok &&
			(awsErr.Code() == servicediscovery.ErrCodeServiceNotFound || awsErr.Code() == servicediscovery.ErrCodeInstanceNotFound) {
			r.log.Info("Waiting 15 secs to allow CloudMap to Sync and then retry....")
			return true
		}
		return false
	}, func() error {
		_, err := r.cloudMapSDK.UpdateInstanceCustomHealthStatusWithContext(ctx, updateRequest)
		return err
	}); err != nil {
		return err
	}
	return nil
}

func (r *CloudMapInstanceResolver) registerCloudMapInstance(ctx context.Context,
	cloudmapConfig *appmesh.AWSCloudMapServiceDiscovery,
	instanceId string, instanceInfo *cloudMapInstanceInfo, serviceID string) error {

	attr := make(map[string]*string)
	for label, v := range instanceInfo.labels {
		attr[label] = awssdk.String(v)
	}
	attr[awsservices.AttrAwsInstanceIPV4] = awssdk.String(instanceId)
	attr[awsservices.AttrK8sPod] = awssdk.String(instanceInfo.podName)
	attr[awsservices.AttrK8sNamespace] = awssdk.String(instanceInfo.namespace)
	if instanceInfo.healthStatus == awsservices.InstanceHealthy {
		attr[awsservices.AttrAwsInstanceHealthStatus] = awssdk.String(awsservices.InstanceHealthy)
	} else {
		attr[awsservices.AttrAwsInstanceHealthStatus] = awssdk.String(awsservices.InstanceUnHealthy)
	}

	input := &servicediscovery.RegisterInstanceInput{
		ServiceId:        awssdk.String(serviceID),
		InstanceId:       awssdk.String(instanceId),
		CreatorRequestId: awssdk.String(instanceInfo.podName),
		Attributes:       attr,
	}

	r.log.V(1).Info("Registering Instance ", " ID: ", instanceId, " against Service: ", serviceID)
	_, err := r.cloudMapSDK.RegisterInstanceWithContext(ctx, input)
	if aerr, ok := err.(awserr.Error); ok {
		if aerr.Code() == servicediscovery.ErrCodeDuplicateRequest {
			return nil
		}
		return err
	}
	return nil
}

func (r *CloudMapInstanceResolver) deregisterCloudMapInstance(ctx context.Context, instanceId string,
	cloudMapConfig *appmesh.AWSCloudMapServiceDiscovery, serviceID string) error {

	input := &servicediscovery.DeregisterInstanceInput{
		ServiceId:  awssdk.String(serviceID),
		InstanceId: awssdk.String(instanceId),
	}

	_, err := r.cloudMapSDK.DeregisterInstanceWithContext(ctx, input)
	//ignore duplicate-request
	if aerr, ok := err.(awserr.Error); ok {
		if aerr.Code() == servicediscovery.ErrCodeDuplicateRequest ||
			aerr.Code() == servicediscovery.ErrCodeInstanceNotFound {
			return nil
		}
		return err
	}
	return nil
}

func (r *CloudMapInstanceResolver) deleteAWSCloudMapServiceInstances(ctx context.Context,
	cloudMapConfig *appmesh.AWSCloudMapServiceDiscovery, serviceID string) error {
	cloudMapServiceKey := cloudMapConfig.NamespaceName + "-" + cloudMapConfig.ServiceName
	registeredServiceInstanceSummary, err := r.getRegisteredCloudMapServiceInstances(ctx, cloudMapConfig, serviceID)
	if err != nil {
		r.log.Error(err, "Couldn't get registered set of instances for", "service: ", cloudMapConfig.ServiceName)
		return err
	}

	err = r.removeCloudMapServiceInstances(ctx, cloudMapServiceKey, registeredServiceInstanceSummary, cloudMapConfig, serviceID)
	if err != nil {
		return err
	}
	return nil
}

func (r *CloudMapInstanceResolver) serviceInstanceCacheKey(cloudMapConfig *appmesh.AWSCloudMapServiceDiscovery) string {
	return awssdk.StringValue(&cloudMapConfig.NamespaceName) + "-" + awssdk.StringValue(&cloudMapConfig.ServiceName)
}
