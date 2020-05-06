package cloudmap

import (
	"context"
	"fmt"
	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/prometheus/common/log"
	"k8s.io/apimachinery/pkg/util/cache"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"

	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	services "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws/services"
)

type ResourceManager interface {
	// Reconcile will create/update AppMesh VirtualNode to match vn.spec, and update vn.status
	Reconcile(ctx context.Context, vn *appmesh.VirtualNode) error

	// Cleanup will delete AppMesh VirtualNode created for vn.
	Cleanup(ctx context.Context, vn *appmesh.VirtualNode) error
}

func NewCloudMapResourceManager(
	k8sClient client.Client,
	cloudMapSDK services.CloudMap,
	virtualNodeEndpointResolver VirtualNodeEndpointResolver,
	cloudMapInstanceReconciler CloudMapInstanceReconciler,
	accountID string,
	log logr.Logger) ResourceManager {

	return &cloudMapResourceManager{
		k8sClient:                   k8sClient,
		cloudMapSDK:                 cloudMapSDK,
		virtualNodeEndpointResolver: virtualNodeEndpointResolver,
		cloudMapInstanceReconciler:  cloudMapInstanceReconciler,
		namespaceIDCache:            cache.NewLRUExpireCache(cloudMapNamespaceCacheMaxSize),
		serviceIDCache:              cache.NewLRUExpireCache(cloudMapServiceCacheMaxSize),
		accountID:                   accountID,
		log:                         log,
	}
}

// cloudMapResourceManager implements ResourceManager
type cloudMapResourceManager struct {
	k8sClient                   client.Client
	cloudMapSDK                 services.CloudMap
	virtualNodeEndpointResolver VirtualNodeEndpointResolver
	cloudMapInstanceReconciler  CloudMapInstanceReconciler
	namespaceIDCache            *cache.LRUExpireCache
	serviceIDCache              *cache.LRUExpireCache
	accountID                   string
	log                         logr.Logger
}

type cloudMapNamespaceSummary struct {
	NamespaceID   string
	NamespaceType string
}

type cloudMapServiceSummary struct {
	NamespaceID string
	ServiceID   string
}

const (
	cloudMapNamespaceCacheMaxSize = 100
	cloudMapNamespaceCacheTTL     = 2 * time.Minute
	cloudMapServiceCacheMaxSize   = 1024
	cloudMapServiceCacheTTL       = 2 * time.Minute
)

func (m *cloudMapResourceManager) Reconcile(ctx context.Context, vNode *appmesh.VirtualNode) error {

	cloudMapConfig := vNode.Spec.ServiceDiscovery.AWSCloudMap
	creatorRequestID := vNode.UID

	serviceSummary, err := m.getOrCreateCloudMapService(ctx, cloudMapConfig, string(creatorRequestID))
	if err != nil {
		m.log.Error(err, "CloudMap RM: failed to create cloudMap Service for", "vNode: ", vNode.Name)
		return err
	}

	readyPods, notReadyPods, ignoredPods, err := m.virtualNodeEndpointResolver.ResolveCloudMapEndPoints(ctx, vNode)
	if err != nil {
		m.log.Error(err, "CloudMap RM: failed to get pods for", "vNode: ", vNode.Name)
		return err
	}

	//Reconcile pod instances with Cloudmap
	if err := m.cloudMapInstanceReconciler.ReconcileCloudMapInstances(ctx, readyPods, notReadyPods, ignoredPods,
		serviceSummary.ServiceID, serviceSummary.NamespaceID, vNode); err != nil {
		log.Error(err, "CloudMap RM: Error reconciling instances with CloudMap")
		return err
	}
	return nil
}

func (m *cloudMapResourceManager) Cleanup(ctx context.Context, vNode *appmesh.VirtualNode) error {
	cloudMapConfig := vNode.Spec.ServiceDiscovery.AWSCloudMap
	if err := m.deleteCloudMapService(ctx, vNode, cloudMapConfig); err != nil {
		return err
	}
	return nil
}

func (m *cloudMapResourceManager) getCloudMapNamespaceFromCache(ctx context.Context,
	key string) (cloudMapNamespaceSummary, error) {
	existingItem, exists := m.namespaceIDCache.Get(key)
	var namespaceSummary cloudMapNamespaceSummary
	if exists {
		//return &(existingItem.(*cloudmapNamespaceCacheItem)).value, nil
		namespaceSummary = existingItem.(cloudMapNamespaceSummary)
	}
	return namespaceSummary, nil
}

func (m *cloudMapResourceManager) getCloudMapNameSpaceFromAWS(ctx context.Context,
	key string, cloudMapConfig *appmesh.AWSCloudMapServiceDiscovery) (cloudMapNamespaceSummary, error) {
	listNamespacesInput := &servicediscovery.ListNamespacesInput{}
	var namespaceSummary cloudMapNamespaceSummary

	ctx, cancel := context.WithTimeout(ctx, time.Second*services.ListNamespacesPagesTimeout)
	defer cancel()

	err := m.cloudMapSDK.ListNamespacesPagesWithContext(ctx,
		listNamespacesInput,
		func(listNamespacesOutput *servicediscovery.ListNamespacesOutput, lastPage bool) bool {
			for _, ns := range listNamespacesOutput.Namespaces {
				if awssdk.StringValue(ns.Name) == awssdk.StringValue(&cloudMapConfig.NamespaceName) {
					key := key
					namespaceSummary = cloudMapNamespaceSummary{
						NamespaceID:   awssdk.StringValue(ns.Id),
						NamespaceType: awssdk.StringValue(ns.Type),
					}

					m.log.V(4).Info("NameSpace found ", "key: ", key, "namespaceID: ", namespaceSummary.NamespaceID)
					m.namespaceIDCache.Add(key, namespaceSummary, cloudMapNamespaceCacheTTL)
					return false
				}
			}
			return true
		},
	)

	if err != nil || namespaceSummary.NamespaceID == "" {
		return cloudMapNamespaceSummary{}, fmt.Errorf("Namespace not found: %s", cloudMapConfig.NamespaceName)
	}

	return namespaceSummary, nil
}

func (m *cloudMapResourceManager) getCloudMapNamespace(ctx context.Context,
	cloudMapConfig *appmesh.AWSCloudMapServiceDiscovery) (cloudMapNamespaceSummary, error) {
	key := m.namespaceCacheKey(cloudMapConfig)
	var namespaceSummary cloudMapNamespaceSummary
	var err error

	if namespaceSummary, _ = m.getCloudMapNamespaceFromCache(ctx, key); namespaceSummary.NamespaceID != "" {
		return namespaceSummary, nil
	}

	//Namespace info missing in Cache. Reach out to AWS Cloudmap for relevant info.
	namespaceSummary, err = m.getCloudMapNameSpaceFromAWS(ctx, key, cloudMapConfig)
	if err != nil {
		return cloudMapNamespaceSummary{}, err
	}

	return namespaceSummary, nil
}

func (m *cloudMapResourceManager) getCloudMapServiceFromCache(ctx context.Context,
	key string) (cloudMapServiceSummary, error) {
	//Get from Cache
	existingItem, exists := m.serviceIDCache.Get(key)

	if exists {
		m.log.Info("vNode: ", "Service in Cache", existingItem.(cloudMapServiceSummary))
		return existingItem.(cloudMapServiceSummary), nil
	}

	m.log.Info("Service Missing in Cache")
	return cloudMapServiceSummary{}, nil
}

func (m *cloudMapResourceManager) getCloudMapServiceFromAWS(ctx context.Context, namespaceID string,
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
	ctx, cancel := context.WithTimeout(ctx, time.Second*services.ListServicesPagesTimeout)
	defer cancel()

	err := m.cloudMapSDK.ListServicesPagesWithContext(ctx,
		listServicesInput,
		func(listServicesOutput *servicediscovery.ListServicesOutput, hasMore bool) bool {
			for _, svc := range listServicesOutput.Services {
				m.log.Info("vNode: ", "service ID: ", svc.Id)
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

func (m *cloudMapResourceManager) getCloudMapService(ctx context.Context,
	cloudMapConfig *appmesh.AWSCloudMapServiceDiscovery) (*cloudMapServiceSummary, error) {
	key := m.serviceCacheKey(cloudMapConfig)

	if serviceSummary, _ := m.getCloudMapServiceFromCache(ctx, key); serviceSummary.ServiceID != "" {
		m.log.Info("vNode: ", "ServiceSummary from cache: ", serviceSummary)
		return &serviceSummary, nil
	}

	//Service info missing in Cache. Reach out to AWS CloudMap for Service Info.
	namespaceSummary, err := m.getCloudMapNamespace(ctx, cloudMapConfig)
	if err != nil {
		return nil, err
	}

	if namespaceSummary.NamespaceID == "" {
		return nil, fmt.Errorf("Could not find namespace in cloudmap with name %s", awssdk.StringValue(&cloudMapConfig.NamespaceName))
	}

	cloudmapService, err := m.getCloudMapServiceFromAWS(ctx, namespaceSummary.NamespaceID, awssdk.StringValue(&cloudMapConfig.ServiceName))
	if err != nil {
		return nil, err
	}

	servicekey := key
	value := cloudMapServiceSummary{
		NamespaceID: namespaceSummary.NamespaceID,
		ServiceID:   awssdk.StringValue(cloudmapService.Id),
	}
	m.serviceIDCache.Add(servicekey, value, cloudMapServiceCacheTTL)
	return &value, nil
}

func (m *cloudMapResourceManager) getOrCreateCloudMapService(ctx context.Context,
	cloudmapConfig *appmesh.AWSCloudMapServiceDiscovery, creatorRequestID string) (*cloudMapServiceSummary, error) {

	key := m.serviceCacheKey(cloudmapConfig)
	if serviceSummary, _ := m.getCloudMapServiceFromCache(ctx, key); serviceSummary.ServiceID != "" {
		return &serviceSummary, nil
	}

	namespaceSummary, err := m.getCloudMapNamespace(ctx, cloudmapConfig)
	if err != nil {
		return nil, err
	}

	if namespaceSummary.NamespaceID == "" {
		return nil, fmt.Errorf("Could not find namespace in cloudmap with name %s", awssdk.StringValue(&cloudmapConfig.NamespaceName))
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*services.CreateServiceTimeout)
	defer cancel()

	if namespaceSummary.NamespaceType == servicediscovery.NamespaceTypeDnsPrivate {
		return m.createServiceUnderPrivateDNSNamespace(ctx, cloudmapConfig, creatorRequestID, &namespaceSummary)
	} else if namespaceSummary.NamespaceType == servicediscovery.NamespaceTypeHttp {
		return m.createServiceUnderHTTPNamespace(ctx, cloudmapConfig, creatorRequestID, &namespaceSummary)
	} else {
		return nil, errors.Errorf("Cannot create service under namespace %s with type %s, only namespaces with types %v are supported",
			awssdk.StringValue(&cloudmapConfig.NamespaceName),
			namespaceSummary.NamespaceType,
			[]string{servicediscovery.NamespaceTypeDnsPrivate, servicediscovery.NamespaceTypeHttp},
		)
	}
}

func (m *cloudMapResourceManager) createServiceUnderPrivateDNSNamespace(ctx context.Context,
	cloudMapConfig *appmesh.AWSCloudMapServiceDiscovery, creatorRequestID string,
	namespaceSummary *cloudMapNamespaceSummary) (*cloudMapServiceSummary, error) {

	var failureThresholdValue int64 = services.HealthStatusFailureThreshold
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
	return m.createAWSCloudMapService(ctx, cloudMapConfig, namespaceSummary, createServiceInput)
}

func (m *cloudMapResourceManager) createServiceUnderHTTPNamespace(ctx context.Context, cloudMapConfig *appmesh.AWSCloudMapServiceDiscovery,
	creatorRequestID string, namespaceSummary *cloudMapNamespaceSummary) (*cloudMapServiceSummary, error) {
	createServiceInput := &servicediscovery.CreateServiceInput{
		CreatorRequestId: awssdk.String(creatorRequestID),
		Name:             &cloudMapConfig.ServiceName,
		NamespaceId:      awssdk.String(namespaceSummary.NamespaceID),
	}
	return m.createAWSCloudMapService(ctx, cloudMapConfig, namespaceSummary, createServiceInput)
}

func (m *cloudMapResourceManager) createAWSCloudMapService(ctx context.Context,
	cloudMapConfig *appmesh.AWSCloudMapServiceDiscovery, namespaceSummary *cloudMapNamespaceSummary,
	createServiceInput *servicediscovery.CreateServiceInput) (*cloudMapServiceSummary, error) {

	key := m.serviceCacheKey(cloudMapConfig)
	createServiceOutput, err := m.cloudMapSDK.CreateServiceWithContext(ctx, createServiceInput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == servicediscovery.ErrCodeServiceAlreadyExists {
				return m.getCloudMapService(ctx, cloudMapConfig)
			}
		}
		return nil, err
	}

	serviceKey := key
	serviceSummary := cloudMapServiceSummary{
		NamespaceID: namespaceSummary.NamespaceID,
		ServiceID:   awssdk.StringValue(createServiceOutput.Service.Id)}
	m.serviceIDCache.Add(serviceKey, serviceSummary, cloudMapServiceCacheTTL)
	return &serviceSummary, nil
}

func (m *cloudMapResourceManager) deleteAWSCloudMapService(ctx context.Context, serviceID string, namespaceID string,
	cloudMapConfig *appmesh.AWSCloudMapServiceDiscovery) error {

	//Deregister instances from CloudMap
	err := m.cloudMapInstanceReconciler.CleanUpCloudMapInstances(ctx, serviceID, namespaceID, cloudMapConfig)
	if err != nil {
		m.log.Error(err, "Couldn't delete registered instances for ", "service: ", cloudMapConfig.ServiceName)
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
		_, err := m.cloudMapSDK.DeleteServiceWithContext(ctx, deleteServiceInput)
		return err
	}); err != nil {
		return err
	}

	return nil
}

func (m *cloudMapResourceManager) deleteCloudMapServiceFromCache(ctx context.Context,
	cloudMapConfig *appmesh.AWSCloudMapServiceDiscovery) error {

	key := m.serviceCacheKey(cloudMapConfig)
	m.serviceIDCache.Remove(key)
	return nil
}

func (m *cloudMapResourceManager) deleteCloudMapService(ctx context.Context, vNode *appmesh.VirtualNode,
	cloudMapConfig *appmesh.AWSCloudMapServiceDiscovery) error {
	var serviceSummary *cloudMapServiceSummary
	var err error
	if serviceSummary, err = m.getCloudMapService(ctx, cloudMapConfig); serviceSummary == nil {
		m.log.Error(err, "Service: ", cloudMapConfig.ServiceName, " not found")
		return nil
	}

	if err := m.deleteAWSCloudMapService(ctx, serviceSummary.ServiceID, serviceSummary.NamespaceID, cloudMapConfig); err != nil {
		m.log.Error(err, "Delete from CloudMap failed for: ", "Service: ", cloudMapConfig.ServiceName)
		return err
	}

	if err := m.deleteCloudMapServiceFromCache(ctx, cloudMapConfig); err != nil {
		m.log.Error(err, "Delete from Cache failed for: ", "Service: ", cloudMapConfig.ServiceName)
		return err
	}
	return nil
}

func (m *cloudMapResourceManager) isCloudMapEnabledForVirtualNode(vNode *appmesh.VirtualNode) bool {
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

func (m *cloudMapResourceManager) isPodSelectorDefinedForVirtualNode(vNode *appmesh.VirtualNode) bool {
	if vNode.Spec.PodSelector == nil {
		return false
	}
	return true
}

func (m *cloudMapResourceManager) serviceCacheKey(cloudMapConfig *appmesh.AWSCloudMapServiceDiscovery) string {
	return awssdk.StringValue(&cloudMapConfig.ServiceName) + "@" + awssdk.StringValue(&cloudMapConfig.NamespaceName)
}

func (m *cloudMapResourceManager) namespaceCacheKey(cloudMapConfig *appmesh.AWSCloudMapServiceDiscovery) string {
	return awssdk.StringValue(&cloudMapConfig.NamespaceName)
}

func (m *cloudMapResourceManager) serviceInstanceCacheKey(cloudMapConfig *appmesh.AWSCloudMapServiceDiscovery) string {
	return awssdk.StringValue(&cloudMapConfig.NamespaceName) + "-" + awssdk.StringValue(&cloudMapConfig.ServiceName)
}
