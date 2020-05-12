package cloudmap

import (
	"context"
	"fmt"
	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/cache"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"

	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	services "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws/services"
)

const (
	defaultServiceDNSConfigTTL             = 300
	defaultServiceCustomHCFailureThreshold = 1
	defaultNamespaceCacheMaxSize           = 100
	defaultNamespaceCacheTTL               = 2 * time.Minute
	defaultServiceCacheMaxSize             = 1024
	defaultServiceCacheTTL                 = 2 * time.Minute
)

type ResourceManager interface {
	// Reconcile will create/update AppMesh CloudMap Resources
	Reconcile(ctx context.Context, vn *appmesh.VirtualNode) error

	// Cleanup will delete AppMesh CloudMap resources created for VirtualNode.
	Cleanup(ctx context.Context, vn *appmesh.VirtualNode) error
}

func NewDefaultResourceManager(
	k8sClient client.Client,
	cloudMapSDK services.CloudMap,
	virtualNodeEndpointResolver VirtualNodeEndpointResolver,
	instancesReconciler InstancesReconciler,
	log logr.Logger) ResourceManager {

	return &defaultResourceManager{
		k8sClient:                   k8sClient,
		cloudMapSDK:                 cloudMapSDK,
		virtualNodeEndpointResolver: virtualNodeEndpointResolver,
		instancesReconciler:         instancesReconciler,

		namespaceSummaryCache: cache.NewLRUExpireCache(defaultNamespaceCacheMaxSize),
		serviceSummaryCache:   cache.NewLRUExpireCache(defaultServiceCacheMaxSize),
		log:                   log,
	}
}

// defaultResourceManager implements ResourceManager
type defaultResourceManager struct {
	k8sClient                   client.Client
	cloudMapSDK                 services.CloudMap
	virtualNodeEndpointResolver VirtualNodeEndpointResolver
	instancesReconciler         InstancesReconciler
	accountID                   string

	namespaceSummaryCache *cache.LRUExpireCache
	serviceSummaryCache   *cache.LRUExpireCache
	log                   logr.Logger
}

type serviceSummary struct {
	serviceID               string
	healthCheckCustomConfig *servicediscovery.HealthCheckCustomConfig
}

func (m *defaultResourceManager) Reconcile(ctx context.Context, vn *appmesh.VirtualNode) error {
	cloudMapConfig := vn.Spec.ServiceDiscovery.AWSCloudMap
	nsSummary, err := m.findCloudMapNamespace(ctx, cloudMapConfig.NamespaceName)
	if err != nil {
		return err
	}
	if nsSummary == nil {
		return fmt.Errorf("cloudMap namespace not found: %v", cloudMapConfig.NamespaceName)
	}
	svcSummary, err := m.findCloudMapService(ctx, nsSummary, cloudMapConfig.ServiceName)
	if err != nil {
		return err
	}
	if svcSummary == nil {
		svcSummary, err = m.createCloudMapService(ctx, vn, nsSummary, cloudMapConfig.ServiceName)
		if err != nil {
			return err
		}
	}

	if vn.Spec.PodSelector != nil {
		readyPods, notReadyPods, _, err := m.virtualNodeEndpointResolver.Resolve(ctx, vn)
		if err != nil {
			return err
		}
		m.log.V(1).Info("resolved VirtualNode endpoints",
			"readyPods", len(readyPods),
			"notReadyPods", len(notReadyPods),
		)
		if err := m.instancesReconciler.Reconcile(ctx, vn, svcSummary.serviceID, svcSummary.healthCheckCustomConfig != nil, readyPods, notReadyPods); err != nil {
			return err
		}
	}

	return nil
}

func (m *defaultResourceManager) Cleanup(ctx context.Context, vn *appmesh.VirtualNode) error {
	cloudMapConfig := vn.Spec.ServiceDiscovery.AWSCloudMap
	nsSummary, err := m.findCloudMapNamespace(ctx, cloudMapConfig.NamespaceName)
	if err != nil {
		return err
	}
	if nsSummary == nil {
		return nil
	}
	svcSummary, err := m.findCloudMapService(ctx, nsSummary, cloudMapConfig.ServiceName)
	if err != nil {
		return err
	}
	if svcSummary == nil {
		return nil
	}

	if vn.Spec.PodSelector != nil {
		if err := m.instancesReconciler.Reconcile(ctx, vn, svcSummary.serviceID, svcSummary.healthCheckCustomConfig != nil, nil, nil); err != nil {
			return err
		}
	}
	if err := m.deleteCloudMapService(ctx, vn, nsSummary, svcSummary); err != nil {
		return err
	}
	return nil
}

// findCloudMapNamespaceFromAWS will try to find CloudMapNamespace from cache and AWS(if cache miss). returns nil if not found
func (m *defaultResourceManager) findCloudMapNamespace(ctx context.Context, namespaceName string) (*servicediscovery.NamespaceSummary, error) {
	if cachedValue, exists := m.namespaceSummaryCache.Get(namespaceName); exists {
		cacheItem := cachedValue.(*servicediscovery.NamespaceSummary)
		return cacheItem, nil
	}

	nsSummary, err := m.findCloudMapNamespaceFromAWS(ctx, namespaceName)
	if err != nil {
		return nil, err
	}
	if nsSummary != nil {
		m.namespaceSummaryCache.Add(namespaceName, nsSummary, defaultNamespaceCacheTTL)
	}
	return nsSummary, nil
}

// findCloudMapNamespaceFromAWS will try to find CloudMapNamespace directly from AWS. returns nil if not found
func (m *defaultResourceManager) findCloudMapNamespaceFromAWS(ctx context.Context, namespaceName string) (*servicediscovery.NamespaceSummary, error) {
	listNamespacesInput := &servicediscovery.ListNamespacesInput{}
	var nsSummary *servicediscovery.NamespaceSummary
	if err := m.cloudMapSDK.ListNamespacesPagesWithContext(ctx, listNamespacesInput,
		func(listNamespacesOutput *servicediscovery.ListNamespacesOutput, lastPage bool) bool {
			for _, ns := range listNamespacesOutput.Namespaces {
				if awssdk.StringValue(ns.Name) == namespaceName {
					nsSummary = ns
					return false
				}
			}
			return true
		},
	); err != nil {
		return nil, err
	}

	return nsSummary, nil
}

func (m *defaultResourceManager) findCloudMapService(ctx context.Context, nsSummary *servicediscovery.NamespaceSummary, serviceName string) (*serviceSummary, error) {
	cacheKey := m.buildCloudMapServiceSummaryCacheKey(nsSummary, serviceName)
	if cachedValue, exists := m.serviceSummaryCache.Get(cacheKey); exists {
		cacheItem := cachedValue.(*serviceSummary)
		return cacheItem, nil
	}

	sdkSVCSummary, err := m.findCloudMapServiceFromAWS(ctx, nsSummary, serviceName)
	if err != nil {
		return nil, err
	}
	if sdkSVCSummary != nil {
		svcSummary := &serviceSummary{
			serviceID:               awssdk.StringValue(sdkSVCSummary.Id),
			healthCheckCustomConfig: sdkSVCSummary.HealthCheckCustomConfig,
		}
		m.serviceSummaryCache.Add(cacheKey, svcSummary, defaultServiceCacheTTL)
		return svcSummary, nil
	}
	return nil, nil
}

func (m *defaultResourceManager) findCloudMapServiceFromAWS(ctx context.Context, nsSummary *servicediscovery.NamespaceSummary, serviceName string) (*servicediscovery.ServiceSummary, error) {
	listServicesInput := &servicediscovery.ListServicesInput{
		Filters: []*servicediscovery.ServiceFilter{
			{
				Name:   awssdk.String(servicediscovery.ServiceFilterNameNamespaceId),
				Values: []*string{nsSummary.Id},
			},
		},
	}

	var sdkSVCSummary *servicediscovery.ServiceSummary
	if err := m.cloudMapSDK.ListServicesPagesWithContext(ctx, listServicesInput,
		func(listServicesOutput *servicediscovery.ListServicesOutput, lastPage bool) bool {
			for _, svc := range listServicesOutput.Services {
				if awssdk.StringValue(svc.Name) == serviceName {
					sdkSVCSummary = svc
					return false
				}
			}
			return true
		},
	); err != nil {
		return nil, err
	}

	return sdkSVCSummary, nil
}

func (m *defaultResourceManager) createCloudMapService(ctx context.Context, vn *appmesh.VirtualNode, nsSummary *servicediscovery.NamespaceSummary, serviceName string) (*serviceSummary, error) {
	switch awssdk.StringValue(nsSummary.Type) {
	case servicediscovery.NamespaceTypeDnsPrivate:
		sdkService, err := m.createCloudMapServiceUnderPrivateDNSNamespace(ctx, vn, nsSummary, serviceName)
		if err != nil {
			return nil, err
		}
		return m.addCloudMapServiceToServiceSummaryCache(nsSummary, sdkService), nil
	case servicediscovery.NamespaceTypeHttp:
		sdkService, err := m.createCloudMapServiceUnderHTTPNamespace(ctx, vn, nsSummary, serviceName)
		if err != nil {
			return nil, err
		}
		return m.addCloudMapServiceToServiceSummaryCache(nsSummary, sdkService), nil
	default:
		return nil, errors.Errorf("unsupported namespace type: %v, use namespace with types %v instead",
			awssdk.StringValue(nsSummary.Type),
			[]string{servicediscovery.NamespaceTypeDnsPrivate, servicediscovery.NamespaceTypeHttp},
		)
	}
}

func (m *defaultResourceManager) deleteCloudMapService(ctx context.Context, vn *appmesh.VirtualNode, nsSummary *servicediscovery.NamespaceSummary, svcSummary *serviceSummary) error {
	getServiceInput := &servicediscovery.GetServiceInput{Id: awssdk.String(svcSummary.serviceID)}
	getServiceOutput, err := m.cloudMapSDK.GetServiceWithContext(ctx, getServiceInput)
	if err != nil {
		return errors.Wrapf(err, "failed to get cloudMap service")
	}
	if !m.isCloudMapServiceOwnedByVirtualNode(ctx, getServiceOutput.Service, vn) {
		m.log.V(1).Info("skip cloudMap service deletion since it's not owned",
			"namespaceName", awssdk.StringValue(nsSummary.Name),
			"namespaceID", awssdk.StringValue(nsSummary.Id),
			"serviceName", awssdk.StringValue(getServiceOutput.Service.Name),
			"serviceID", awssdk.StringValue(getServiceOutput.Service.Id),
		)
		return nil
	}

	deleteServiceInput := &servicediscovery.DeleteServiceInput{
		Id: awssdk.String(svcSummary.serviceID),
	}

	deleteServiceBackoff := wait.Backoff{
		Steps:    4,
		Duration: 15 * time.Second,
		Factor:   1.0,
		Jitter:   0.1,
		Cap:      60 * time.Second,
	}
	// Delete Service. Ideally we should delete it if there are no registered instances but the call will
	// fail if that is the case and we move on. Saves us an additional GET to check the instance count.
	if err := retry.OnError(deleteServiceBackoff, func(err error) bool {
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
	m.removeCloudMapServiceFromServiceSummaryCache(nsSummary, getServiceOutput.Service)
	return nil
}

func (m *defaultResourceManager) createCloudMapServiceUnderPrivateDNSNamespace(ctx context.Context, vn *appmesh.VirtualNode,
	nsSummary *servicediscovery.NamespaceSummary, serviceName string) (*servicediscovery.Service, error) {
	creatorRequestID := string(vn.UID)
	createServiceInput := &servicediscovery.CreateServiceInput{
		CreatorRequestId: awssdk.String(creatorRequestID),
		NamespaceId:      nsSummary.Id,
		Name:             awssdk.String(serviceName),
		DnsConfig: &servicediscovery.DnsConfig{
			RoutingPolicy: awssdk.String(servicediscovery.RoutingPolicyMultivalue),
			DnsRecords: []*servicediscovery.DnsRecord{
				{
					Type: awssdk.String(servicediscovery.RecordTypeA),
					TTL:  awssdk.Int64(defaultServiceDNSConfigTTL),
				},
			},
		},
		HealthCheckCustomConfig: &servicediscovery.HealthCheckCustomConfig{
			FailureThreshold: awssdk.Int64(defaultServiceCustomHCFailureThreshold),
		},
	}
	resp, err := m.cloudMapSDK.CreateServiceWithContext(ctx, createServiceInput)
	if err != nil {
		return nil, err
	}
	return resp.Service, nil
}

func (m *defaultResourceManager) createCloudMapServiceUnderHTTPNamespace(ctx context.Context, vn *appmesh.VirtualNode,
	nsSummary *servicediscovery.NamespaceSummary, serviceName string) (*servicediscovery.Service, error) {
	creatorRequestID := string(vn.UID)
	createServiceInput := &servicediscovery.CreateServiceInput{
		CreatorRequestId: awssdk.String(creatorRequestID),
		NamespaceId:      nsSummary.Id,
		Name:             awssdk.String(serviceName),
		HealthCheckCustomConfig: &servicediscovery.HealthCheckCustomConfig{
			FailureThreshold: awssdk.Int64(defaultServiceCustomHCFailureThreshold),
		},
	}
	resp, err := m.cloudMapSDK.CreateServiceWithContext(ctx, createServiceInput)
	if err != nil {
		return nil, err
	}
	return resp.Service, nil
}

func (m *defaultResourceManager) addCloudMapServiceToServiceSummaryCache(nsSummary *servicediscovery.NamespaceSummary, service *servicediscovery.Service) *serviceSummary {
	cacheKey := m.buildCloudMapServiceSummaryCacheKey(nsSummary, awssdk.StringValue(service.Name))
	svcSummary := &serviceSummary{
		serviceID:               awssdk.StringValue(service.Id),
		healthCheckCustomConfig: service.HealthCheckCustomConfig,
	}
	m.serviceSummaryCache.Add(cacheKey, svcSummary, defaultServiceCacheTTL)
	return svcSummary
}

func (m *defaultResourceManager) removeCloudMapServiceFromServiceSummaryCache(nsSummary *servicediscovery.NamespaceSummary, service *servicediscovery.Service) {
	cacheKey := m.buildCloudMapServiceSummaryCacheKey(nsSummary, awssdk.StringValue(service.Name))
	m.serviceSummaryCache.Remove(cacheKey)
}

// isCloudMapServiceOwnedByVirtualNode checks whether an CloudMap ervice is owned by VirtualNode.
// if it's owned, VirtualNode deletion is responsible for deleting the CloudMap Service
func (m *defaultResourceManager) isCloudMapServiceOwnedByVirtualNode(ctx context.Context, svc *servicediscovery.Service, vn *appmesh.VirtualNode) bool {
	return awssdk.StringValue(svc.CreatorRequestId) == string(vn.UID)
}

func (m *defaultResourceManager) buildCloudMapServiceSummaryCacheKey(nsSummary *servicediscovery.NamespaceSummary, serviceName string) string {
	return fmt.Sprintf("%s/%s", awssdk.StringValue(nsSummary.Id), serviceName)
}