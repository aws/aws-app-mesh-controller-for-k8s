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
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/cloudmap"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/runtime"
	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/prometheus/common/log"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/cache"
	"k8s.io/apimachinery/pkg/util/wait"
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
	virtualNodeEndpointResolver cloudmap.VirtualNodeEndpointResolver
	cloudMapInstanceReconciler  cloudmap.CloudMapInstanceReconciler
	nameSpaceIDCache            *cache.LRUExpireCache
	serviceIDCache              *cache.LRUExpireCache
	enqueueRequestsForPodEvents handler.EventHandler
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

const (
	cloudMapNamespaceCacheMaxSize = 100
	cloudMapNamespaceCacheTTL     = 2 * time.Minute
	cloudMapServiceCacheMaxSize   = 1024
	cloudMapServiceCacheTTL       = 2 * time.Minute
)

func NewCloudMapReconciler(k8sClient client.Client, finalizerManager k8s.FinalizerManager,
	cloudMapSDK awsservices.CloudMap, virtualNodeEndpointResolver cloudmap.VirtualNodeEndpointResolver,
	cloudMapInstanceReconciler cloudmap.CloudMapInstanceReconciler, log logr.Logger) *cloudMapReconciler {
	return &cloudMapReconciler{
		k8sClient:                   k8sClient,
		log:                         log,
		cloudMapSDK:                 cloudMapSDK,
		finalizerManager:            finalizerManager,
		virtualNodeEndpointResolver: virtualNodeEndpointResolver,
		cloudMapInstanceReconciler:  cloudMapInstanceReconciler,
		nameSpaceIDCache:            cache.NewLRUExpireCache(cloudMapNamespaceCacheMaxSize),
		serviceIDCache:              cache.NewLRUExpireCache(cloudMapServiceCacheMaxSize),
		enqueueRequestsForPodEvents: cloudmap.NewEnqueueRequestsForPodEvents(k8sClient, log),
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

	vNode := &appmesh.VirtualNode{}
	if err := r.k8sClient.Get(ctx, req.NamespacedName, vNode); err != nil {
		return client.IgnoreNotFound(err)
	}

	if !vNode.DeletionTimestamp.IsZero() {
		return r.deleteCloudMapResources(ctx, vNode)
	}
	return r.reconcileVirtualNodeWithCloudMap(ctx, vNode)
}

func (r *cloudMapReconciler) reconcileVirtualNodeWithCloudMap(ctx context.Context, vNode *appmesh.VirtualNode) error {

	if !r.isCloudMapEnabledForVirtualNode(vNode) {
		r.log.V(1).Info("CloudMap not enabled for", " virtualNode: %s", vNode.Name)
		return nil
	}

	if !r.isPodSelectorDefinedForVirtualNode(vNode) {
		r.log.V(1).Info("PodSelector not defined for", "virtualNode: %s", vNode.Name)
		return nil
	}

	cloudMapConfig := vNode.Spec.ServiceDiscovery.AWSCloudMap
	creatorRequestID := vNode.UID

	if err := r.finalizerManager.AddFinalizers(ctx, vNode, k8s.FinalizerAWSCloudMapResources); err != nil {
		r.log.Error(err, "..while adding cloudmap resource finalizer")
		return err
	}

	serviceSummary, err := r.getOrCreateCloudMapService(ctx, cloudMapConfig, string(creatorRequestID))
	if err != nil {
		r.log.Error(err, "failed to create cloudMap Service for", "vNode: ", vNode.Name)
		return err
	}

	readyPods, notReadyPods, ignoredPods, err := r.virtualNodeEndpointResolver.ResolveCloudMapEndPoints(ctx, vNode)
	if err != nil {
		r.log.Error(err, "failed to get pods for", "vNode: ", vNode.Name)
		return err
	}

	r.log.V(1).Info("EndPoints: ", "Ready Pods Count: ", len(readyPods.Items),
		"Unready Pods Count: ", len(notReadyPods.Items), "Ignored Pods Count: ", len(ignoredPods.Items))

	//Reconcile pod instances with Cloudmap
	if err := r.cloudMapInstanceReconciler.ReconcileCloudMapInstances(ctx, readyPods, notReadyPods, ignoredPods,
		serviceSummary.ServiceID, serviceSummary.NamespaceID, vNode); err != nil {
		log.Error(err, "Error reconciling instances with CloudMap")
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

func (r *cloudMapReconciler) getCloudMapNamespaceFromCache(ctx context.Context,
	key string) (cloudMapNamespaceSummary, error) {
	existingItem, exists := r.nameSpaceIDCache.Get(key)
	var namespaceSummary cloudMapNamespaceSummary
	if exists {
		//return &(existingItem.(*cloudmapNamespaceCacheItem)).value, nil
		namespaceSummary = existingItem.(cloudMapNamespaceSummary)
	}
	return namespaceSummary, nil
}

func (r *cloudMapReconciler) getCloudMapNameSpaceFromAWS(ctx context.Context,
	key string, cloudMapConfig *appmesh.AWSCloudMapServiceDiscovery) (cloudMapNamespaceSummary, error) {
	listNamespacesInput := &servicediscovery.ListNamespacesInput{}
	var namespaceSummary cloudMapNamespaceSummary

	ctx, cancel := context.WithTimeout(ctx, time.Second*awsservices.ListNamespacesPagesTimeout)
	defer cancel()

	err := r.cloudMapSDK.ListNamespacesPagesWithContext(ctx,
		listNamespacesInput,
		func(listNamespacesOutput *servicediscovery.ListNamespacesOutput, lastPage bool) bool {
			for _, ns := range listNamespacesOutput.Namespaces {
				if awssdk.StringValue(ns.Name) == awssdk.StringValue(&cloudMapConfig.NamespaceName) {
					key := key
					namespaceSummary = cloudMapNamespaceSummary{
						NamespaceID:   awssdk.StringValue(ns.Id),
						NamespaceType: awssdk.StringValue(ns.Type),
					}

					r.log.V(4).Info("NameSpace found ", "key: ", key, "namespaceID: ", namespaceSummary.NamespaceID)
					r.nameSpaceIDCache.Add(key, namespaceSummary, cloudMapNamespaceCacheTTL)
					return true
				}
			}
			return false
		},
	)

	if err != nil || namespaceSummary.NamespaceID == "" {
		return cloudMapNamespaceSummary{}, fmt.Errorf("Namespace not found: %s", cloudMapConfig.NamespaceName)
	}

	return namespaceSummary, nil
}

func (r *cloudMapReconciler) getCloudMapNameSpace(ctx context.Context,
	cloudMapConfig *appmesh.AWSCloudMapServiceDiscovery) (cloudMapNamespaceSummary, error) {
	key := r.namespaceCacheKey(cloudMapConfig)
	var namespaceSummary cloudMapNamespaceSummary
	var err error

	if namespaceSummary, _ = r.getCloudMapNamespaceFromCache(ctx, key); namespaceSummary.NamespaceID != "" {
		return namespaceSummary, nil
	}

	r.log.Info("Reach out to CloudMap for Namespace Summary")
	//Namespace info missing in Cache. Reach out to AWS Cloudmap for relevant info.
	namespaceSummary, err = r.getCloudMapNameSpaceFromAWS(ctx, key, cloudMapConfig)
	if err != nil {
		return cloudMapNamespaceSummary{}, err
	}

	return namespaceSummary, nil
}

func (r *cloudMapReconciler) getCloudMapServiceFromCache(ctx context.Context,
	key string) (cloudMapServiceSummary, error) {
	//Get from Cache
	existingItem, exists := r.serviceIDCache.Get(key)

	if exists {
		r.log.Info("vNode: ", "Service in Cache", existingItem.(cloudMapServiceSummary))
		return existingItem.(cloudMapServiceSummary), nil
	}

	r.log.Info("Service Missing in Cache")
	return cloudMapServiceSummary{}, nil
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
	cloudMapConfig *appmesh.AWSCloudMapServiceDiscovery) (*cloudMapServiceSummary, error) {
	key := r.serviceCacheKey(cloudMapConfig)

	if serviceSummary, _ := r.getCloudMapServiceFromCache(ctx, key); serviceSummary.ServiceID != "" {
		r.log.Info("vNode: ", "ServiceSummary from cache: ", serviceSummary)
		return &serviceSummary, nil
	}

	//Service info missing in Cache. Reach out to AWS CloudMap for Service Info.
	namespaceSummary, err := r.getCloudMapNameSpace(ctx, cloudMapConfig)
	if err != nil {
		return nil, err
	}

	if namespaceSummary.NamespaceID == "" {
		return nil, fmt.Errorf("Could not find namespace in cloudmap with name %s", awssdk.StringValue(&cloudMapConfig.NamespaceName))
	}

	cloudmapService, err := r.getCloudMapServiceFromAWS(ctx, namespaceSummary.NamespaceID, awssdk.StringValue(&cloudMapConfig.ServiceName))
	if err != nil {
		return nil, err
	}

	servicekey := key
	value := cloudMapServiceSummary{
		NamespaceID: namespaceSummary.NamespaceID,
		ServiceID:   awssdk.StringValue(cloudmapService.Id),
	}
	r.serviceIDCache.Add(servicekey, value, cloudMapServiceCacheTTL)
	return &value, nil
}

func (r *cloudMapReconciler) getOrCreateCloudMapService(ctx context.Context,
	cloudmapConfig *appmesh.AWSCloudMapServiceDiscovery, creatorRequestID string) (*cloudMapServiceSummary, error) {

	key := r.serviceCacheKey(cloudmapConfig)
	if serviceSummary, _ := r.getCloudMapServiceFromCache(ctx, key); serviceSummary.ServiceID != "" {
		return &serviceSummary, nil
	}

	namespaceSummary, err := r.getCloudMapNameSpace(ctx, cloudmapConfig)
	if err != nil {
		return nil, err
	}

	if namespaceSummary.NamespaceID == "" {
		return nil, fmt.Errorf("Could not find namespace in cloudmap with name %s", awssdk.StringValue(&cloudmapConfig.NamespaceName))
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*awsservices.CreateServiceTimeout)
	defer cancel()

	if namespaceSummary.NamespaceType == servicediscovery.NamespaceTypeDnsPrivate {
		return r.createServiceUnderPrivateDNSNamespace(ctx, cloudmapConfig, creatorRequestID, &namespaceSummary)
	} else if namespaceSummary.NamespaceType == servicediscovery.NamespaceTypeHttp {
		return r.createServiceUnderHTTPNamespace(ctx, cloudmapConfig, creatorRequestID, &namespaceSummary)
	} else {
		return nil, errors.Errorf("Cannot create service under namespace %s with type %s, only namespaces with types %v are supported",
			awssdk.StringValue(&cloudmapConfig.NamespaceName),
			namespaceSummary.NamespaceType,
			[]string{servicediscovery.NamespaceTypeDnsPrivate, servicediscovery.NamespaceTypeHttp},
		)
	}
}

func (r *cloudMapReconciler) createServiceUnderPrivateDNSNamespace(ctx context.Context,
	cloudMapConfig *appmesh.AWSCloudMapServiceDiscovery, creatorRequestID string,
	namespaceSummary *cloudMapNamespaceSummary) (*cloudMapServiceSummary, error) {

	var failureThresholdValue int64 = awsservices.HealthStatusFailureThreshold
	createServiceInput := &servicediscovery.CreateServiceInput{
		CreatorRequestId: awssdk.String(creatorRequestID),
		Name:             &cloudMapConfig.ServiceName,
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
	return r.createAWSCloudMapService(ctx, cloudMapConfig, namespaceSummary, createServiceInput)
}

func (r *cloudMapReconciler) createServiceUnderHTTPNamespace(ctx context.Context, cloudMapConfig *appmesh.AWSCloudMapServiceDiscovery,
	creatorRequestID string, namespaceSummary *cloudMapNamespaceSummary) (*cloudMapServiceSummary, error) {
	createServiceInput := &servicediscovery.CreateServiceInput{
		CreatorRequestId: awssdk.String(creatorRequestID),
		Name:             &cloudMapConfig.ServiceName,
		NamespaceId:      awssdk.String(namespaceSummary.NamespaceID),
	}
	return r.createAWSCloudMapService(ctx, cloudMapConfig, namespaceSummary, createServiceInput)
}

func (r *cloudMapReconciler) createAWSCloudMapService(ctx context.Context,
	cloudMapConfig *appmesh.AWSCloudMapServiceDiscovery, namespaceSummary *cloudMapNamespaceSummary,
	createServiceInput *servicediscovery.CreateServiceInput) (*cloudMapServiceSummary, error) {

	key := r.serviceCacheKey(cloudMapConfig)
	createServiceOutput, err := r.cloudMapSDK.CreateServiceWithContext(ctx, createServiceInput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == servicediscovery.ErrCodeServiceAlreadyExists {
				return r.getCloudMapService(ctx, cloudMapConfig)
			}
		}
		return nil, err
	}

	serviceKey := key
	serviceSummary := cloudMapServiceSummary{
		NamespaceID: namespaceSummary.NamespaceID,
		ServiceID:   awssdk.StringValue(createServiceOutput.Service.Id)}
	r.serviceIDCache.Add(serviceKey, serviceSummary, cloudMapServiceCacheTTL)
	return &serviceSummary, nil
}

func (r *cloudMapReconciler) deleteAWSCloudMapService(ctx context.Context, serviceID string, namespaceID string,
	cloudMapConfig *appmesh.AWSCloudMapServiceDiscovery) error {

	//Deregister instances from CloudMap
	err := r.cloudMapInstanceReconciler.CleanUpCloudMapInstances(ctx, serviceID, namespaceID, cloudMapConfig)
	if err != nil {
		r.log.Error(err, "Couldn't delete registered instances for ", "service: ", cloudMapConfig.ServiceName)
		return err
	}

	//Delete Service. Ideally we should delete it if there are no registered instances but the call will
	//fail if that is the case and we move on. Saves us an additional GET to check the instance count.
	deleteServiceInput := &servicediscovery.DeleteServiceInput{
		Id: awssdk.String(serviceID),
	}

	if err := retry.OnError(cloudMapBackoff, func(err error) bool {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == servicediscovery.ErrCodeResourceInUse {
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
	cloudMapConfig *appmesh.AWSCloudMapServiceDiscovery) error {

	key := r.serviceCacheKey(cloudMapConfig)
	r.serviceIDCache.Remove(key)
	return nil
}

func (r *cloudMapReconciler) deleteCloudMapService(ctx context.Context, vNode *appmesh.VirtualNode,
	cloudMapConfig *appmesh.AWSCloudMapServiceDiscovery) error {
	var serviceSummary *cloudMapServiceSummary
	var err error
	if serviceSummary, err = r.getCloudMapService(ctx, cloudMapConfig); serviceSummary == nil {
		r.log.Error(err, "Service: ", cloudMapConfig.ServiceName, " not found")
		return nil
	}

	if err := r.deleteAWSCloudMapService(ctx, serviceSummary.ServiceID, serviceSummary.NamespaceID, cloudMapConfig); err != nil {
		r.log.Error(err, "Delete from CloudMap failed for: ", "Service: ", cloudMapConfig.ServiceName)
		return err
	}

	if err := r.deleteCloudMapServiceFromCache(ctx, cloudMapConfig); err != nil {
		r.log.Error(err, "Delete from Cache failed for: ", "Service: ", cloudMapConfig.ServiceName)
		return err
	}
	return nil
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

func (r *cloudMapReconciler) serviceCacheKey(cloudMapConfig *appmesh.AWSCloudMapServiceDiscovery) string {
	return awssdk.StringValue(&cloudMapConfig.ServiceName) + "@" + awssdk.StringValue(&cloudMapConfig.NamespaceName)
}

func (r *cloudMapReconciler) namespaceCacheKey(cloudMapConfig *appmesh.AWSCloudMapServiceDiscovery) string {
	return awssdk.StringValue(&cloudMapConfig.NamespaceName)
}

func (r *cloudMapReconciler) serviceInstanceCacheKey(cloudMapConfig *appmesh.AWSCloudMapServiceDiscovery) string {
	return awssdk.StringValue(&cloudMapConfig.NamespaceName) + "-" + awssdk.StringValue(&cloudMapConfig.ServiceName)
}

/* TODO - To be used for Pod Readiness Gate */
/*
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
*/
