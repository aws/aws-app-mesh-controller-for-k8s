/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/runtime"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/virtualnode"
	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/prometheus/common/log"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"time"

	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	awsservices "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws/services"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/k8s"
)

// CloudMapReconciler reconciles a VirtualNode pod instance to CloudMap Service
type cloudMapReconciler struct {
	k8sClient                   client.Client
	log                         logr.Logger
	cloudMapSDK                 awsservices.CloudMap
	finalizerManager            k8s.FinalizerManager
	cloudMapInstanceCache       cache.Store
	nameSpaceIDCache            cache.Store
	serviceIDCache              cache.Store
	enqueueRequestsForPodEvents handler.EventHandler
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

type cloudmapNamespaceCacheItem struct {
	key   string
	value cloudMapNamespaceSummary
}

type cloudMapNamespaceSummary struct {
	NamespaceID   string
	NamespaceType string
}

type cloudmapServiceCacheItem struct {
	key   string
	value cloudMapServiceSummary
}

type cloudMapServiceSummary struct {
	NamespaceID string
	ServiceID   string
}

var cloudMapBackoff = wait.Backoff{
	Steps:    4,
	Duration: 15 * time.Second,
	Factor:   1.0,
	Jitter:   0.1,
	Cap:      60 * time.Second,
}

func NewCloudMapReconciler(k8sClient client.Client, finalizerManager k8s.FinalizerManager, cloudMapSDK awsservices.CloudMap, log logr.Logger) *cloudMapReconciler {

	return &cloudMapReconciler{
		k8sClient:        k8sClient,
		log:              log,
		cloudMapSDK:      cloudMapSDK,
		finalizerManager: finalizerManager,
		cloudMapInstanceCache: cache.NewTTLStore(func(obj interface{}) (string, error) {
			return obj.(*cloudMapInstanceCacheItem).key, nil
		}, 300*time.Second),
		nameSpaceIDCache: cache.NewTTLStore(func(obj interface{}) (string, error) {
			return obj.(*cloudmapNamespaceCacheItem).key, nil
		}, 60*time.Second),
		serviceIDCache: cache.NewTTLStore(func(obj interface{}) (string, error) {
			return obj.(*cloudmapServiceCacheItem).key, nil
		}, 60*time.Second),
		enqueueRequestsForPodEvents: virtualnode.NewEnqueueRequestsForPodEvents(k8sClient, log),
	}

}

// +kubebuilder:rbac:groups=appmesh.k8s.aws,resources=virtualnodes,verbs=get;list;watch
// +kubebuilder:rbac:groups=appmesh.k8s.aws,resources=virtualnodes/status,verbs=get
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch

func (r *cloudMapReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	return runtime.HandleReconcileError(r.reconcile(req), r.log)
}

func (r *cloudMapReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appmesh.VirtualNode{}).
		Watches(&source.Kind{Type: &corev1.Pod{}}, r.enqueueRequestsForPodEvents).
		Complete(r)
}

func (r *cloudMapReconciler) reconcile(req ctrl.Request) error {
	ctx := context.Background()
	log := r.log.WithValues("Virtualnode", req.NamespacedName)

	log.V(1).Info("In VirtualNode Reconciler", "vNode-Name: ", req.Name, "vNode-NameSpace: ", req.Namespace,
		"vNode-NamespacedName: ", req.NamespacedName)

	vNode := &appmesh.VirtualNode{}
	if err := r.k8sClient.Get(ctx, req.NamespacedName, vNode); err != nil {
		log.Error(err, "vNode")
		return client.IgnoreNotFound(err)
	}

	log.V(1).Info("Found VNode: ", "vNode.creationtimestamp: ", vNode.CreationTimestamp, "awsName: ", vNode.Spec.AWSName,
		"vNode.PodSelector: ", vNode.Spec.PodSelector)

	if !vNode.DeletionTimestamp.IsZero() {
		return r.deleteCloudMapResources(ctx, vNode)
	}
	return r.reconcileVirtualNodeWithCloudMap(ctx, vNode)
}

func (r *cloudMapReconciler) reconcileVirtualNodeWithCloudMap(ctx context.Context, vNode *appmesh.VirtualNode) error {

	if !r.isCloudMapEnabledForVirtualNode(vNode) {
		return errors.Errorf("CloudMap not enabled for virtualNode: %s", vNode.Name)
	}

	if !r.isPodSelectorDefinedForVirtualNode(vNode) {
		return errors.Errorf("PodSelector not defined for virtualNode: %s", vNode.Name)
	}

	cloudmapConfig := vNode.Spec.ServiceDiscovery.AWSCloudMap
	creatorRequestID := vNode.UID

	var podsList corev1.PodList
	var listOptions client.ListOptions
	listOptions.LabelSelector, _ = metav1.LabelSelectorAsSelector(vNode.Spec.PodSelector)
	listOptions.Namespace = vNode.Namespace

	if err := r.k8sClient.List(ctx, &podsList, &listOptions); err != nil {
		r.log.Error(err, "Couldn't retrieve pods for VirtualNode")
		return err
	}

	r.log.Info("Pods found matching this VNode: ", "Total Pods: ", len(podsList.Items))
	if err := r.finalizerManager.AddFinalizers(ctx, vNode, k8s.FinalizerAWSCloudMapResources); err != nil {
		r.log.Error(err, "..while adding cloudmap resource finalizer")
		return err
	}

	_, err := r.createCloudMapService(ctx, cloudmapConfig, string(creatorRequestID))
	if err != nil {
		r.log.Error(err, "failed to create cloudMap Service for", "vNode: ", vNode.Name)
		return err
	}

	serviceInstanceSummary := make(map[string]*cloudMapInstanceInfo)
	for _, pod := range podsList.Items {
		r.log.Info("Processing Pod: ", "Pod-Name:", pod.Name)

		if pod.DeletionTimestamp != nil {
			r.log.Info("vNode:", "Pod is being deleted: ", pod.Name)
			continue
		}

		instanceId := r.podToInstanceID(&pod)
		if instanceId == "" {
			r.log.Info("No IP Address assigned to Pod:", pod.Name, "..Skipping for now")
			continue
		}

		instanceInfo := &cloudMapInstanceInfo{}
		r.log.Info("vNode: ", "Instance ID of Pod:", instanceId)
		if r.isPodReady(&pod) {
			instanceInfo.healthStatus = awsservices.InstanceHealthy
		} else if r.shouldPodBeRegisteredWithCloudMapService(&pod) {
			instanceInfo.healthStatus = awsservices.InstanceUnHealthy
		} else {
			continue
		}
		instanceInfo.podName = pod.Name
		instanceInfo.namespace = pod.Namespace
		instanceInfo.virtualNodeName = vNode.Name
		instanceInfo.meshName = vNode.Name
		instanceInfo.labels = make(map[string]string)
		for label, v := range pod.Labels {
			instanceInfo.labels[label] = v
		}
		serviceInstanceSummary[instanceId] = instanceInfo
	}

	if err := r.updateCloudMapServiceInstances(ctx, cloudmapConfig, serviceInstanceSummary, string(creatorRequestID)); err != nil {
		r.log.Error(err, " Failed to Update/Register instances to CloudMap")
		return err
	}
	return nil
}

func (r *cloudMapReconciler) deleteCloudMapResources(ctx context.Context, vNode *appmesh.VirtualNode) error {
	if k8s.HasFinalizer(vNode, k8s.FinalizerAWSCloudMapResources) {
		cloudmapConfig := vNode.Spec.ServiceDiscovery.AWSCloudMap
		if err := r.deleteCloudMapService(ctx, vNode, cloudmapConfig); err != nil {
			return err
		}
		if err := r.finalizerManager.RemoveFinalizers(ctx, vNode, k8s.FinalizerAWSCloudMapResources); err != nil {
			return err
		}
	}
	return nil
}

func (r *cloudMapReconciler) getRegisteredCloudMapServiceInstances(ctx context.Context,
	cloudmapConfig *appmesh.AWSCloudMapServiceDiscovery) (map[string]*cloudMapInstanceInfo, error) {

	var cacheServiceInstanceSummary map[string]*cloudMapInstanceInfo

	cloudMapServiceKey := cloudmapConfig.NamespaceName + "-" + cloudmapConfig.ServiceName
	existingItem, exists, _ := r.cloudMapInstanceCache.Get(&cloudMapInstanceCacheItem{
		key: cloudMapServiceKey,
	})

	if exists {
		//Service Present but instance might be missing. Update the cache and register the instance.
		cacheServiceInstanceSummary = existingItem.(*cloudMapInstanceCacheItem).instanceSummary
	} else {
		//UpdateCache from CloudMap for this service
		serviceInstances, err := r.getServiceInstancesFromCloudMap(ctx, cloudmapConfig)
		if err != nil {
			return cacheServiceInstanceSummary, err
		}

		cacheServiceInstanceSummary = make(map[string]*cloudMapInstanceInfo)
		for _, instance := range serviceInstances {
			registeredInstanceInfo := &cloudMapInstanceInfo{}
			podName := awssdk.StringValue(instance.Attributes[awsservices.AttrK8sPod])
			namespace := awssdk.StringValue(instance.Attributes[awsservices.AttrK8sNamespace])
			instanceID := awssdk.StringValue(instance.Id)
			serviceName := awssdk.StringValue(instance.Attributes["app"])
			healthStatus := awssdk.StringValue(instance.Attributes[awsservices.AttrAwsInstanceHealthStatus])

			r.log.Info("Pod: %s is currently registered with the service: %s", podName, serviceName)
			registeredInstanceInfo.podName = podName
			registeredInstanceInfo.namespace = namespace
			registeredInstanceInfo.healthStatus = healthStatus

			cacheServiceInstanceSummary[instanceID] = registeredInstanceInfo
		}
	}
	return cacheServiceInstanceSummary, nil
}

func (r *cloudMapReconciler) updateCloudMapServiceInstances(ctx context.Context, cloudmapConfig *appmesh.AWSCloudMapServiceDiscovery,
	currentServiceInstanceSummary map[string]*cloudMapInstanceInfo, creatorRequestID string) error {

	cacheServiceInstanceSummary, err := r.getRegisteredCloudMapServiceInstances(ctx, cloudmapConfig)
	cloudMapServiceKey := r.serviceInstanceCacheKey(cloudmapConfig)

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
				if err := r.updateCloudMapInstanceHealthStatus(ctx, cloudmapConfig, instanceId, instanceInfo.healthStatus); err != nil {
					r.log.Error(err, "Error Updating Instance: ", instanceId, " Health Status in CloudMap")
				}
			}
			currInstanceInfo.healthStatus = instanceInfo.healthStatus
			delete(cacheServiceInstanceSummary, instanceId)
		} else {
			//Instance Missing in CloudMap. Register it.
			if err := r.registerCloudMapInstance(ctx, cloudmapConfig, instanceId, instanceInfo, creatorRequestID); err != nil {
				r.log.Error(err, "Error Registering Instance: ", instanceId, " to CloudMap")
				continue
			}
			currInstanceInfo.healthStatus = instanceInfo.healthStatus
		}
		updatedCacheServiceInstanceSummary[instanceId] = currInstanceInfo
	}

	//Delete the instances that are no longer part of the current instance list
	err = r.removeCloudMapServiceInstances(ctx, cloudMapServiceKey, cacheServiceInstanceSummary, cloudmapConfig)
	if err != nil {
		return err
	}

	serviceItem := &cloudMapInstanceCacheItem{
		key:             cloudMapServiceKey,
		instanceSummary: updatedCacheServiceInstanceSummary,
	}
	_ = r.cloudMapInstanceCache.Add(serviceItem)
	return nil
}

func (r *cloudMapReconciler) removeCloudMapServiceInstances(ctx context.Context, cloudMapServiceKey string,
	serviceInstancesToBeRemoved map[string]*cloudMapInstanceInfo, cloudmapConfig *appmesh.AWSCloudMapServiceDiscovery) error {

	for instanceId, _ := range serviceInstancesToBeRemoved {
		r.log.Info("De-registering pod: ", "InstanceID: ", instanceId)
		err := r.deregisterCloudMapInstance(ctx, instanceId, cloudmapConfig)
		if err != nil {
			//Log an error and continue
			r.log.Error(err, "Couldn't deregister instance: ", instanceId)
		}
	}
	return nil
}

func (r *cloudMapReconciler) updateCloudMapInstanceHealthStatus(ctx context.Context,
	cloudmapConfig *appmesh.AWSCloudMapServiceDiscovery,
	instanceId string, healthStatus string) error {

	//Check if we have the service in CloudMap
	serviceSummary, err := r.getCloudMapService(ctx, cloudmapConfig)
	if err != nil {
		r.log.Error(err, "Couldn't update health status for instance: ", instanceId, " Service: ",
			cloudmapConfig.ServiceName, " not found")
		return err
	}

	r.log.Info("vNode", "Updating Health status of Service", *awssdk.String(serviceSummary.ServiceID),
		"instance: ", *awssdk.String(instanceId), " to: ", *awssdk.String(healthStatus))
	updateRequest := &servicediscovery.UpdateInstanceCustomHealthStatusInput{
		InstanceId: awssdk.String(instanceId),
		ServiceId:  awssdk.String(serviceSummary.ServiceID),
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

func (r *cloudMapReconciler) registerCloudMapInstance(ctx context.Context,
	cloudmapConfig *appmesh.AWSCloudMapServiceDiscovery,
	instanceId string, instanceInfo *cloudMapInstanceInfo, creatorRequestID string) error {

	//Check if we have the service in CloudMap
	serviceSummary, err := r.getCloudMapService(ctx, cloudmapConfig)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == servicediscovery.ErrCodeServiceNotFound {
			//Service is missing in CloudMap. Go ahead and create it.
			if serviceSummary, err = r.createCloudMapService(ctx, cloudmapConfig, creatorRequestID); err != nil {
				r.log.Error(err, "Failed to create CloudMap service: ", cloudmapConfig.ServiceName)
				return err
			}
		}
	}

	if serviceSummary == nil {
		r.log.Error(err, "Service missing in CloudMap: ", cloudmapConfig.ServiceName)
		return nil
	}

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

	//copy the attributes specified on virtual-node
	/*TODO
	for _, a := range cloudmapConfig.Attributes {
		attr[awssdk.StringValue(&a.Key)] = &a.Value
	}
	*/

	input := &servicediscovery.RegisterInstanceInput{
		ServiceId:        awssdk.String(serviceSummary.ServiceID),
		InstanceId:       awssdk.String(instanceId),
		CreatorRequestId: awssdk.String(instanceInfo.podName),
		Attributes:       attr,
	}

	r.log.Info("Registering Instance ", " ID: ", instanceId, " against Service: ", serviceSummary.ServiceID)
	_, err = r.cloudMapSDK.RegisterInstanceWithContext(ctx, input)
	if aerr, ok := err.(awserr.Error); ok {
		if aerr.Code() == servicediscovery.ErrCodeDuplicateRequest {
			return nil
		}
		return err
	}
	r.log.Info("Successfully Registered Instance ", " ID: ", instanceId, " against Service: ", serviceSummary.ServiceID)
	return nil
}

func (r *cloudMapReconciler) deregisterCloudMapInstance(ctx context.Context, instanceId string,
	cloudmapConfig *appmesh.AWSCloudMapServiceDiscovery) error {

	//Check if we have the service in CloudMap
	serviceSummary, err := r.getCloudMapService(ctx, cloudmapConfig)
	if err != nil {
		r.log.Error(err, "Couldn't deregister instance: ", instanceId, " Service: ", cloudmapConfig.ServiceName, " not found")
	}

	input := &servicediscovery.DeregisterInstanceInput{
		ServiceId:  awssdk.String(serviceSummary.ServiceID),
		InstanceId: awssdk.String(instanceId),
	}

	_, err = r.cloudMapSDK.DeregisterInstanceWithContext(ctx, input)
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

func (r *cloudMapReconciler) getCloudMapNameSpaceFromCache(ctx context.Context,
	key string) (*cloudMapNamespaceSummary, error) {
	existingItem, exists, _ := r.nameSpaceIDCache.Get(&cloudmapNamespaceCacheItem{
		key: key,
	})
	if exists {
		return &(existingItem.(*cloudmapNamespaceCacheItem)).value, nil
	}
	return nil, nil
}

func (r *cloudMapReconciler) getCloudMapNameSpaceFromAWS(ctx context.Context,
	key string, cloudmapConfig *appmesh.AWSCloudMapServiceDiscovery) (*cloudMapNamespaceSummary, error) {
	listNamespacesInput := &servicediscovery.ListNamespacesInput{}
	var namespaceItem *cloudmapNamespaceCacheItem

	ctx, cancel := context.WithTimeout(ctx, time.Second*awsservices.ListNamespacesPagesTimeout)
	defer cancel()

	err := r.cloudMapSDK.ListNamespacesPagesWithContext(ctx,
		listNamespacesInput,
		func(listNamespacesOutput *servicediscovery.ListNamespacesOutput, lastPage bool) bool {
			for _, ns := range listNamespacesOutput.Namespaces {
				if awssdk.StringValue(ns.Name) == awssdk.StringValue(&cloudmapConfig.NamespaceName) {
					namespaceItem = &cloudmapNamespaceCacheItem{
						key: key,
						value: cloudMapNamespaceSummary{
							NamespaceID:   awssdk.StringValue(ns.Id),
							NamespaceType: awssdk.StringValue(ns.Type),
						},
					}
					r.nameSpaceIDCache.Add(namespaceItem)
					return true
				}
			}
			return false
		},
	)

	if err != nil {
		return nil, err
	}

	if namespaceItem == nil {
		klog.Infof("No namespace found with name %s", awssdk.StringValue(&cloudmapConfig.NamespaceName))
		return nil, nil
	}
	return &namespaceItem.value, err
}

func (r *cloudMapReconciler) getCloudMapNameSpace(ctx context.Context,
	cloudmapConfig *appmesh.AWSCloudMapServiceDiscovery) (*cloudMapNamespaceSummary, error) {
	key := r.namespaceCacheKey(cloudmapConfig)

	if namespaceSummary, _ := r.getCloudMapNameSpaceFromCache(ctx, key); namespaceSummary != nil {
		return namespaceSummary, nil
	}

	//Namespace info missing in Cache. Reach out to AWS Cloudmap for relevant info.
	namespaceItem, err := r.getCloudMapNameSpaceFromAWS(ctx, key, cloudmapConfig)
	if err != nil {
		return nil, err
	}
	return namespaceItem, err
}

func (r *cloudMapReconciler) getCloudMapServiceFromCache(ctx context.Context,
	key string) (*cloudMapServiceSummary, error) {
	//Get from Cache
	existingItem, exists, _ := r.serviceIDCache.Get(&cloudmapServiceCacheItem{
		key: key,
	})
	if exists {
		r.log.Info("vNode: ", "Service in Cache", (existingItem.(*cloudmapServiceCacheItem)).value)
		return &(existingItem.(*cloudmapServiceCacheItem)).value, nil
	}

	r.log.Info("Service Missing in Cache")
	return nil, nil
}

func (r *cloudMapReconciler) getCloudMapServiceFromAWS(ctx context.Context, namespaceID string,
	serviceName string) (*servicediscovery.ServiceSummary, error) {
	listServicesInput := &servicediscovery.ListServicesInput{
		Filters: []*servicediscovery.ServiceFilter{
			&servicediscovery.ServiceFilter{
				Name:   awssdk.String(servicediscovery.ServiceFilterNameNamespaceId),
				Values: []*string{awssdk.String(namespaceID)},
			},
		},
	}

	var svcSummary *servicediscovery.ServiceSummary

	r.log.Info("vNode: ", "NameSpace ID: ", namespaceID, "serviceName: ", serviceName)

	ctx, cancel := context.WithTimeout(ctx, time.Second*awsservices.ListServicesPagesTimeout)
	defer cancel()

	err := r.cloudMapSDK.ListServicesPagesWithContext(ctx,
		listServicesInput,
		func(listServicesOutput *servicediscovery.ListServicesOutput, hasMore bool) bool {
			for _, svc := range listServicesOutput.Services {
				r.log.Info("vNode: ", "service ID: ", svc.Id)
				if awssdk.StringValue(svc.Name) == serviceName {
					svcSummary = svc
					return true
				}
			}
			return false
		},
	)

	if err != nil {
		return nil, err
	}

	if svcSummary == nil {
		return nil, fmt.Errorf("Could not find service with name %s in namespace %s", serviceName, namespaceID)
	}
	return svcSummary, err
}

func (r *cloudMapReconciler) getCloudMapService(ctx context.Context,
	cloudmapConfig *appmesh.AWSCloudMapServiceDiscovery) (*cloudMapServiceSummary, error) {
	key := r.serviceCacheKey(cloudmapConfig)

	if serviceSummary, _ := r.getCloudMapServiceFromCache(ctx, key); serviceSummary != nil {
		r.log.Info("vNode: ", "ServiceSummary from cache: ", serviceSummary)
		return serviceSummary, nil
	}

	r.log.Info("vNode: ", "Service missing in Cache: ", cloudmapConfig.ServiceName)
	//Service info missing in Cache. Reach out to AWS CloudMap for Service Info.
	namespaceSummary, err := r.getCloudMapNameSpace(ctx, cloudmapConfig)
	if err != nil {
		return nil, err
	}

	r.log.Info("vNode: ", "NameSpace ID...", namespaceSummary.NamespaceID)
	if namespaceSummary == nil {
		return nil, fmt.Errorf("Could not find namespace in cloudmap with name %s", awssdk.StringValue(&cloudmapConfig.NamespaceName))
	}

	r.log.Info("vNode: ", "Reaching out to CloudMap for...", cloudmapConfig.ServiceName)
	cloudmapService, err := r.getCloudMapServiceFromAWS(ctx, namespaceSummary.NamespaceID, awssdk.StringValue(&cloudmapConfig.ServiceName))
	if err != nil {
		return nil, err
	}

	serviceItem := &cloudmapServiceCacheItem{
		key: key,
		value: cloudMapServiceSummary{
			NamespaceID: namespaceSummary.NamespaceID,
			ServiceID:   awssdk.StringValue(cloudmapService.Id),
		},
	}
	r.serviceIDCache.Add(serviceItem)
	return &serviceItem.value, nil
}

func (r *cloudMapReconciler) createCloudMapService(ctx context.Context,
	cloudmapConfig *appmesh.AWSCloudMapServiceDiscovery, creatorRequestID string) (*cloudMapServiceSummary, error) {

	key := r.serviceCacheKey(cloudmapConfig)
	if serviceSummary, _ := r.getCloudMapServiceFromCache(ctx, key); serviceSummary != nil {
		return serviceSummary, nil
	}

	namespaceSummary, err := r.getCloudMapNameSpace(ctx, cloudmapConfig)
	if err != nil {
		return nil, err
	}

	if namespaceSummary == nil {
		return nil, fmt.Errorf("Could not find namespace in cloudmap with name %s", awssdk.StringValue(&cloudmapConfig.NamespaceName))
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*awsservices.CreateServiceTimeout)
	defer cancel()

	if namespaceSummary.NamespaceType == servicediscovery.NamespaceTypeDnsPrivate {
		return r.createServiceUnderPrivateDNSNamespace(ctx, cloudmapConfig, creatorRequestID, namespaceSummary)
	} else if namespaceSummary.NamespaceType == servicediscovery.NamespaceTypeHttp {
		return r.createServiceUnderHTTPNamespace(ctx, cloudmapConfig, creatorRequestID, namespaceSummary)
	} else {
		return nil, errors.Errorf("Cannot create service under namespace %s with type %s, only namespaces with types %v are supported",
			awssdk.StringValue(&cloudmapConfig.NamespaceName),
			namespaceSummary.NamespaceType,
			[]string{servicediscovery.NamespaceTypeDnsPrivate, servicediscovery.NamespaceTypeHttp},
		)
	}
}

func (r *cloudMapReconciler) createServiceUnderPrivateDNSNamespace(ctx context.Context,
	cloudmapConfig *appmesh.AWSCloudMapServiceDiscovery, creatorRequestID string,
	namespaceSummary *cloudMapNamespaceSummary) (*cloudMapServiceSummary, error) {

	var failureThresholdValue int64 = awsservices.HealthStatusFailureThreshold
	createServiceInput := &servicediscovery.CreateServiceInput{
		CreatorRequestId: awssdk.String(creatorRequestID),
		Name:             &cloudmapConfig.ServiceName,
		DnsConfig: &servicediscovery.DnsConfig{
			NamespaceId:   awssdk.String(namespaceSummary.NamespaceID),
			RoutingPolicy: awssdk.String(servicediscovery.RoutingPolicyMultivalue),
			DnsRecords: []*servicediscovery.DnsRecord{
				&servicediscovery.DnsRecord{
					Type: awssdk.String(servicediscovery.RecordTypeA),
					TTL:  awssdk.Int64(300),
				},
			},
		},
		HealthCheckCustomConfig: &servicediscovery.HealthCheckCustomConfig{
			FailureThreshold: &failureThresholdValue,
		},
	}
	return r.createAWSCloudMapService(ctx, cloudmapConfig, namespaceSummary, createServiceInput)
}

func (r *cloudMapReconciler) createServiceUnderHTTPNamespace(ctx context.Context, cloudmapConfig *appmesh.AWSCloudMapServiceDiscovery,
	creatorRequestID string, namespaceSummary *cloudMapNamespaceSummary) (*cloudMapServiceSummary, error) {
	createServiceInput := &servicediscovery.CreateServiceInput{
		CreatorRequestId: awssdk.String(creatorRequestID),
		Name:             &cloudmapConfig.ServiceName,
		NamespaceId:      awssdk.String(namespaceSummary.NamespaceID),
	}
	return r.createAWSCloudMapService(ctx, cloudmapConfig, namespaceSummary, createServiceInput)
}

func (r *cloudMapReconciler) createAWSCloudMapService(ctx context.Context,
	cloudmapConfig *appmesh.AWSCloudMapServiceDiscovery, namespaceSummary *cloudMapNamespaceSummary,
	createServiceInput *servicediscovery.CreateServiceInput) (*cloudMapServiceSummary, error) {

	key := r.serviceCacheKey(cloudmapConfig)
	createServiceOutput, err := r.cloudMapSDK.CreateServiceWithContext(ctx, createServiceInput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == servicediscovery.ErrCodeServiceAlreadyExists {
				return r.getCloudMapService(ctx, cloudmapConfig)
			}
		}
		return nil, err
	}

	serviceItem := &cloudmapServiceCacheItem{
		key: key,
		value: cloudMapServiceSummary{
			NamespaceID: namespaceSummary.NamespaceID,
			ServiceID:   awssdk.StringValue(createServiceOutput.Service.Id),
		},
	}
	_ = r.serviceIDCache.Add(serviceItem)
	return &serviceItem.value, nil
}

func (r *cloudMapReconciler) deleteAWSCloudMapService(ctx context.Context, serviceID string,
	cloudmapConfig *appmesh.AWSCloudMapServiceDiscovery) error {

	//Deregister instances from CloudMap
	cloudMapServiceKey := cloudmapConfig.NamespaceName + "-" + cloudmapConfig.ServiceName
	cacheServiceInstanceSummary, err := r.getRegisteredCloudMapServiceInstances(ctx, cloudmapConfig)
	if err != nil {
		r.log.Error(err, "Couldn't get registered set of instances for service: ", cloudmapConfig.ServiceName)
		return err
	}

	err = r.removeCloudMapServiceInstances(ctx, cloudMapServiceKey, cacheServiceInstanceSummary, cloudmapConfig)
	if err != nil {
		return err
	}

	//Delete Service. Ideally we should delete it if there are no registered instances but the call will
	//fail if that is the case and we move on. Saves us an additional GET to check the instance count.
	deleteServiceInput := &servicediscovery.DeleteServiceInput{
		Id: awssdk.String(serviceID),
	}

	if err := retry.OnError(cloudMapBackoff, func(err error) bool {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == servicediscovery.ErrCodeResourceInUse {
			log.Info("Waiting 15 secs and retrying to allow cloudMap to sync....")
			return true
		}
		return false
	}, func() error {
		_, err := r.cloudMapSDK.DeleteServiceWithContext(ctx, deleteServiceInput)
		return err
	}); err != nil {
		return err
	}

	return nil
}

func (r *cloudMapReconciler) deleteCloudMapServiceFromCache(ctx context.Context,
	cloudmapConfig *appmesh.AWSCloudMapServiceDiscovery) error {

	key := r.serviceCacheKey(cloudmapConfig)
	serviceSummary, _ := r.getCloudMapServiceFromCache(ctx, key)

	if serviceSummary == nil {
		return nil
	}
	serviceItem := &cloudmapServiceCacheItem{
		key: key,
		value: cloudMapServiceSummary{
			NamespaceID: serviceSummary.NamespaceID,
			ServiceID:   serviceSummary.ServiceID,
		},
	}

	//Delete from Cache
	if err := r.serviceIDCache.Delete(serviceItem); err != nil {
		return err
	}
	return nil
}

func (r *cloudMapReconciler) deleteCloudMapService(ctx context.Context, vNode *appmesh.VirtualNode,
	cloudmapConfig *appmesh.AWSCloudMapServiceDiscovery) error {
	var serviceSummary *cloudMapServiceSummary
	var err error
	if serviceSummary, err = r.getCloudMapService(ctx, cloudmapConfig); serviceSummary == nil {
		r.log.Error(err, "Service: ", cloudmapConfig.ServiceName, " not found")
		return nil
	}

	r.log.Info("vNode: ", "serviceSummary: ", serviceSummary)
	if err := r.deleteAWSCloudMapService(ctx, serviceSummary.ServiceID, cloudmapConfig); err != nil {
		return err
	}

	if err := r.deleteCloudMapServiceFromCache(ctx, cloudmapConfig); err != nil {
		r.log.Error(err, "Delete from Cache failed for: ", cloudmapConfig.ServiceName)
		return err
	}
	return nil
}

func (r *cloudMapReconciler) getServiceInstancesFromCloudMap(ctx context.Context,
	cloudmapConfig *appmesh.AWSCloudMapServiceDiscovery) ([]*servicediscovery.InstanceSummary, error) {
	instances := []*servicediscovery.InstanceSummary{}

	serviceSummary, err := r.getCloudMapService(ctx, cloudmapConfig)
	if err != nil || serviceSummary == nil {
		return instances, err
	}
	input := &servicediscovery.ListInstancesInput{
		ServiceId: awssdk.String(serviceSummary.ServiceID),
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

func (r *cloudMapReconciler) podToInstanceID(pod *corev1.Pod) string {
	if pod.Status.PodIP == "" {
		return ""
	}
	return pod.Status.PodIP
}

func (r *cloudMapReconciler) isCloudMapEnabledForVirtualNode(vNode *appmesh.VirtualNode) bool {
	if vNode.Spec.ServiceDiscovery == nil || vNode.Spec.ServiceDiscovery.AWSCloudMap == nil {
		return false
	}
	if vNode.Spec.ServiceDiscovery.AWSCloudMap.NamespaceName == "" ||
		vNode.Spec.ServiceDiscovery.AWSCloudMap.ServiceName == "" {
		klog.Errorf("CloudMap NamespaceName or ServiceName is null")
		return false
	}
	return true
}

func (r *cloudMapReconciler) isPodSelectorDefinedForVirtualNode(vNode *appmesh.VirtualNode) bool {
	if vNode.Spec.PodSelector == nil {
		return false
	}
	return true
}

func (r *cloudMapReconciler) isPodReady(pod *corev1.Pod) bool {

	conditions := (&pod.Status).Conditions
	for i := range conditions {
		if conditions[i].Type == corev1.PodReady && conditions[i].Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func (r *cloudMapReconciler) shouldPodBeRegisteredWithCloudMapService(pod *corev1.Pod) bool {
	switch pod.Spec.RestartPolicy {
	case corev1.RestartPolicyNever:
		return pod.Status.Phase != corev1.PodFailed && pod.Status.Phase != corev1.PodSucceeded
	case corev1.RestartPolicyOnFailure:
		return pod.Status.Phase != corev1.PodSucceeded
	default:
		return true
	}
}

func (r *cloudMapReconciler) serviceCacheKey(cloudmapConfig *appmesh.AWSCloudMapServiceDiscovery) string {
	return awssdk.StringValue(&cloudmapConfig.ServiceName) + "@" + awssdk.StringValue(&cloudmapConfig.NamespaceName)
}

func (r *cloudMapReconciler) namespaceCacheKey(cloudmapConfig *appmesh.AWSCloudMapServiceDiscovery) string {
	return awssdk.StringValue(&cloudmapConfig.NamespaceName)
}

func (r *cloudMapReconciler) serviceInstanceCacheKey(cloudmapConfig *appmesh.AWSCloudMapServiceDiscovery) string {
	return awssdk.StringValue(&cloudmapConfig.NamespaceName) + "-" + awssdk.StringValue(&cloudmapConfig.ServiceName)
}

/* TODO - To be used for Pod Readiness Gate */
func (r *cloudMapReconciler) updateCloudMapPodReadinessConditionForPod(ctx context.Context,
	instanceInfo *cloudMapInstanceInfo) error {
	pod := &corev1.Pod{}
	if err := r.k8sClient.Get(ctx, k8s.NamespacedName(pod), pod); err != nil {
		r.log.Error(err, "Unable to fetch Pod: ", instanceInfo.podName, " under namespace: ", instanceInfo.namespace)
		return err
	}

	podCloudMapCondition := corev1.PodCondition{
		Type:               "cloudmap",
		Status:             "true",
		LastProbeTime:      metav1.Time{},
		LastTransitionTime: metav1.Time{},
		Reason:             "cloudMapInstanceRegistered",
		Message:            "Instance registered successfully with CloudMap",
	}
	pod.Status.Conditions = append(pod.Status.Conditions, podCloudMapCondition)
	err := r.k8sClient.Status().Update(ctx, pod)
	if err != nil {
		return err
	}
	return nil
}
