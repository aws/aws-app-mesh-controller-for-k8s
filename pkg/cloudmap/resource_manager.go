package cloudmap

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/references"
	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/cache"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

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

	nodeRegionLabelKey1           = "failure-domain.beta.kubernetes.io/region"
	nodeRegionLabelKey2           = "topology.kubernetes.io/region"
	nodeAvailabilityZoneLabelKey1 = "failure-domain.beta.kubernetes.io/zone"
	nodeAvailabilityZoneLabelKey2 = "topology.kubernetes.io/zone"

	cloudMapServiceAnnotation = "cloudMapServiceARN"
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
	referencesResolver references.Resolver,
	virtualNodeEndpointResolver VirtualNodeEndpointResolver,
	instancesReconciler InstancesReconciler,
	enableCustomHealthCheck bool,
	log logr.Logger,
	cfg Config) ResourceManager {

	return &defaultResourceManager{
		config:                      cfg,
		k8sClient:                   k8sClient,
		cloudMapSDK:                 cloudMapSDK,
		referencesResolver:          referencesResolver,
		virtualNodeEndpointResolver: virtualNodeEndpointResolver,
		instancesReconciler:         instancesReconciler,
		enableCustomHealthCheck:     enableCustomHealthCheck,
		namespaceSummaryCache:       cache.NewLRUExpireCache(defaultNamespaceCacheMaxSize),
		serviceSummaryCache:         cache.NewLRUExpireCache(defaultServiceCacheMaxSize),
		log:                         log,
	}
}

// defaultResourceManager implements ResourceManager
type defaultResourceManager struct {
	config                      Config
	k8sClient                   client.Client
	cloudMapSDK                 services.CloudMap
	referencesResolver          references.Resolver
	virtualNodeEndpointResolver VirtualNodeEndpointResolver
	instancesReconciler         InstancesReconciler
	enableCustomHealthCheck     bool

	namespaceSummaryCache *cache.LRUExpireCache
	serviceSummaryCache   *cache.LRUExpireCache
	log                   logr.Logger
}

func (m *defaultResourceManager) Reconcile(ctx context.Context, vn *appmesh.VirtualNode) error {
	ms, err := m.findMeshDependency(ctx, vn)
	if err != nil {
		return err
	}
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
		svcSummary, err = m.createCloudMapService(ctx, vn, nsSummary, cloudMapConfig.ServiceName, m.config.CloudMapServiceTTL)
		if err != nil {
			return err
		}
	} else {
		svcSummary, err = m.updateCloudMapService(ctx, svcSummary, vn, nsSummary, cloudMapConfig.ServiceName, m.config.CloudMapServiceTTL)
		if err != nil {
			return err
		}
	}

	if err := m.updateCRDVirtualNode(ctx, vn, svcSummary); err != nil {
		return err
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
		nodeInfoByName := m.getClusterNodeInfo(ctx)
		if err := m.instancesReconciler.Reconcile(ctx, ms, vn, *svcSummary, readyPods, notReadyPods, nodeInfoByName); err != nil {
			return err
		}
	}
	return nil
}

func (m *defaultResourceManager) Cleanup(ctx context.Context, vn *appmesh.VirtualNode) error {
	ms, err := m.findMeshDependency(ctx, vn)
	if err != nil {
		return err
	}
	cloudMapConfig := vn.Spec.ServiceDiscovery.AWSCloudMap
	nsSummary, err := m.findCloudMapNamespace(ctx, cloudMapConfig.NamespaceName)
	if err != nil {
		if !m.isCloudMapServiceCreated(ctx, vn) {
			return nil
		}
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
		if err := m.instancesReconciler.Reconcile(ctx, ms, vn, *svcSummary, nil, nil, nil); err != nil {
			return err
		}
	}
	if err := m.deleteCloudMapService(ctx, vn, nsSummary, svcSummary); err != nil {
		return err
	}
	return nil
}

// findMeshDependency find the Mesh dependency for this virtualNode.
func (m *defaultResourceManager) findMeshDependency(ctx context.Context, vn *appmesh.VirtualNode) (*appmesh.Mesh, error) {
	if vn.Spec.MeshRef == nil {
		return nil, errors.Errorf("meshRef shouldn't be nil, please check webhook setup")
	}
	ms, err := m.referencesResolver.ResolveMeshReference(ctx, *vn.Spec.MeshRef)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to resolve meshRef")
	}
	return ms, nil
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
			serviceARN:              sdkSVCSummary.Arn,
			healthCheckCustomConfig: sdkSVCSummary.HealthCheckCustomConfig,
			DnsConfig:               sdkSVCSummary.DnsConfig,
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

func (m *defaultResourceManager) createCloudMapService(ctx context.Context, vn *appmesh.VirtualNode, nsSummary *servicediscovery.NamespaceSummary, serviceName string,
	cloudMapTTL int64) (*serviceSummary, error) {
	switch awssdk.StringValue(nsSummary.Type) {
	case servicediscovery.NamespaceTypeDnsPrivate:
		sdkService, err := m.createCloudMapServiceUnderPrivateDNSNamespace(ctx, vn, nsSummary, serviceName, cloudMapTTL)
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

func (m *defaultResourceManager) updateCloudMapService(ctx context.Context, svcSummary *serviceSummary, vn *appmesh.VirtualNode, nsSummary *servicediscovery.NamespaceSummary, serviceName string,
	cloudMapTTL int64) (*serviceSummary, error) {
	if awssdk.StringValue(nsSummary.Type) == servicediscovery.NamespaceTypeDnsPrivate {
		actualsvcSummary := *svcSummary
		desiredsvcSummary := BuildCloudMapServiceSummary(ctx, svcSummary, cloudMapTTL)
		opts := cmp.Options{
			cmpopts.EquateEmpty(),
			cmp.AllowUnexported(serviceSummary{}),
		}
		// return if no change to service summary
		if cmp.Equal(*desiredsvcSummary, actualsvcSummary, opts) {
			return svcSummary, nil
		}
		diff := cmp.Diff(*desiredsvcSummary, actualsvcSummary, opts)
		m.log.V(1).Info("cloudMap service changed",
			"actualDnsConfig", actualsvcSummary,
			"desiredDnsConfig", *desiredsvcSummary,
			"diff", diff,
		)
		m.log.V(1).Info("Sending cloudMap service update request")
		operationId, err := m.updateCloudMapServiceUnderPrivateDNSNamespace(ctx, vn, nsSummary, serviceName, cloudMapTTL, svcSummary)
		if err != nil {
			return nil, err
		}
		// wait for update to succeed or timeout and udpate on next reconcile loop
		operationInput := &servicediscovery.GetOperationInput{
			OperationId: operationId,
		}
		status := AwaitOperationSuccess(30, 1, func() string {
			operationOutput, _ := m.cloudMapSDK.GetOperation(operationInput)
			return *operationOutput.Operation.Status
		})
		switch status {
		case SUCCESS:
			return m.updateCloudMapServiceInServiceSummaryCache(nsSummary, desiredsvcSummary, serviceName), nil
		default:
			return nil, fmt.Errorf("CloudMap Service Update Request status: %v", status)
		}
	}
	// if namespace type is not set to DnsPrivate then return without updates
	return svcSummary, nil
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
	nsSummary *servicediscovery.NamespaceSummary, serviceName string, cloudMapTTL int64) (*servicediscovery.Service, error) {
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
					TTL:  awssdk.Int64(cloudMapTTL),
				},
			},
		},
	}
	if m.enableCustomHealthCheck {
		createServiceInput.HealthCheckCustomConfig = &servicediscovery.HealthCheckCustomConfig{
			FailureThreshold: awssdk.Int64(defaultServiceCustomHCFailureThreshold),
		}
	}

	resp, err := m.cloudMapSDK.CreateServiceWithContext(ctx, createServiceInput)
	if err != nil {
		return nil, err
	}
	return resp.Service, nil
}

func (m *defaultResourceManager) updateCloudMapServiceUnderPrivateDNSNamespace(ctx context.Context, vn *appmesh.VirtualNode,
	nsSummary *servicediscovery.NamespaceSummary, serviceName string, cloudMapTTL int64, svcSummary *serviceSummary) (*string, error) {
	updateServiceInput := &servicediscovery.UpdateServiceInput{
		Id: awssdk.String(svcSummary.serviceID),
		Service: &servicediscovery.ServiceChange{
			DnsConfig: &servicediscovery.DnsConfigChange{
				DnsRecords: []*servicediscovery.DnsRecord{
					{
						Type: awssdk.String(servicediscovery.RecordTypeA),
						TTL:  awssdk.Int64(cloudMapTTL),
					},
				},
			},
		},
	}
	if m.enableCustomHealthCheck {
		updateServiceInput.Service.HealthCheckConfig = &servicediscovery.HealthCheckConfig{
			FailureThreshold: awssdk.Int64(defaultServiceCustomHCFailureThreshold),
		}
	}
	UpdateServiceOutput, err := m.cloudMapSDK.UpdateServiceWithContext(ctx, updateServiceInput)
	if err != nil {
		return nil, err
	}
	return UpdateServiceOutput.OperationId, nil
}

func (m *defaultResourceManager) createCloudMapServiceUnderHTTPNamespace(ctx context.Context, vn *appmesh.VirtualNode,
	nsSummary *servicediscovery.NamespaceSummary, serviceName string) (*servicediscovery.Service, error) {
	creatorRequestID := string(vn.UID)
	createServiceInput := &servicediscovery.CreateServiceInput{
		CreatorRequestId: awssdk.String(creatorRequestID),
		NamespaceId:      nsSummary.Id,
		Name:             awssdk.String(serviceName),
	}
	if m.enableCustomHealthCheck {
		createServiceInput.HealthCheckCustomConfig = &servicediscovery.HealthCheckCustomConfig{
			FailureThreshold: awssdk.Int64(defaultServiceCustomHCFailureThreshold),
		}
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
		serviceARN:              service.Arn,
		healthCheckCustomConfig: service.HealthCheckCustomConfig,
		DnsConfig:               service.DnsConfig,
	}
	m.serviceSummaryCache.Add(cacheKey, svcSummary, defaultServiceCacheTTL)
	return svcSummary
}

func (m *defaultResourceManager) updateCloudMapServiceInServiceSummaryCache(nsSummary *servicediscovery.NamespaceSummary, desiredsvcSummary *serviceSummary, serviceName string) *serviceSummary {
	cacheKey := m.buildCloudMapServiceSummaryCacheKey(nsSummary, serviceName)
	m.serviceSummaryCache.Remove(cacheKey)
	m.serviceSummaryCache.Add(cacheKey, desiredsvcSummary, defaultServiceCacheTTL)
	return desiredsvcSummary
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

func BuildCloudMapServiceSummary(ctx context.Context, svcSummary *serviceSummary, cloudMapTTL int64) *serviceSummary {
	return &serviceSummary{
		serviceID:               awssdk.StringValue(&svcSummary.serviceID),
		serviceARN:              svcSummary.serviceARN,
		healthCheckCustomConfig: svcSummary.healthCheckCustomConfig,
		DnsConfig: &servicediscovery.DnsConfig{
			DnsRecords: []*servicediscovery.DnsRecord{
				{
					Type: awssdk.String(servicediscovery.RecordTypeA),
					TTL:  awssdk.Int64(cloudMapTTL),
				},
			},
			NamespaceId:   svcSummary.DnsConfig.NamespaceId,
			RoutingPolicy: svcSummary.DnsConfig.RoutingPolicy,
		},
	}
}

func (m *defaultResourceManager) getClusterNodeInfo(ctx context.Context) map[string]nodeAttributes {
	nodeList := &corev1.NodeList{}
	if err := m.k8sClient.List(ctx, nodeList); err != nil {
		return nil
	}

	m.log.V(1).Info("Listed Nodes", "count", len(nodeList.Items))
	nodeInfoByName := make(map[string]nodeAttributes, len(nodeList.Items))
	for i := range nodeList.Items {
		var nodeRegion string
		var nodeAvailabilityZone string
		node := nodeList.Items[i]
		for label, value := range node.Labels {
			if label == nodeRegionLabelKey1 || label == nodeRegionLabelKey2 {
				nodeRegion = value
			} else if label == nodeAvailabilityZoneLabelKey1 || label == nodeAvailabilityZoneLabelKey2 {
				nodeAvailabilityZone = value
			}
		}
		nodeAttrs := nodeAttributes{
			region:           nodeRegion,
			availabilityZone: nodeAvailabilityZone,
		}
		nodeInfoByName[node.Name] = nodeAttrs
	}
	return nodeInfoByName
}

func (m *defaultResourceManager) isCloudMapServiceCreated(ctx context.Context, vn *appmesh.VirtualNode) bool {
	oldVN := vn.DeepCopy()
	vnAnnotations := oldVN.Annotations

	for key, _ := range vnAnnotations {
		if key == cloudMapServiceAnnotation {
			return true
		}
	}
	return false
}

func (m *defaultResourceManager) updateCRDVirtualNode(ctx context.Context, vn *appmesh.VirtualNode, svcSummary *serviceSummary) error {
	if svcSummary.serviceARN == nil {
		return nil
	}
	oldVN := vn.DeepCopy()
	vnAnnotations := oldVN.Annotations

	if vn.Annotations == nil {
		vn.Annotations = make(map[string]string)
	}
	for key, _ := range vnAnnotations {
		if key == cloudMapServiceAnnotation {
			return nil
		}
	}
	vn.Annotations[cloudMapServiceAnnotation] = *svcSummary.serviceARN
	return m.k8sClient.Patch(ctx, vn, client.MergeFrom(oldVN))
}
