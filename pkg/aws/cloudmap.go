package aws

import (
	"context"
	"fmt"
	"time"

	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/appmesh"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog"
)

const (
	CreateServiceTimeout       = 10
	DeregisterInstanceTimeout  = 10
	GetServiceTimeout          = 10
	ListInstancesPagesTimeout  = 10
	ListNamespacesPagesTimeout = 10
	ListServicesPagesTimeout   = 10
	RegisterInstanceTimeout    = 10

	//AttrAwsInstanceIPV4 is a special attribute expected by CloudMap.
	//See https://github.com/aws/aws-sdk-go/blob/fd304fe4cb2ea1027e7fc7e21062beb768915fcc/service/servicediscovery/api.go#L5161
	AttrAwsInstanceIPV4 = "AWS_INSTANCE_IPV4"
	//AttrK8sPod is a custom attribute injected by app-mesh controller
	AttrK8sPod = "k8s.io/pod"
	//AttrK8sNamespace is a custom attribute injected by app-mesh controller
	AttrK8sNamespace = "k8s.io/namespace"
)

//CloudMapAPI is wrapper util to invoke CloudMap API
type CloudMapAPI interface {
	CloudMapCreateService(context.Context, *appmesh.AwsCloudMapServiceDiscovery, string) (*CloudMapServiceSummary, error)
	CloudMapGetService(context.Context, string) (*CloudMapServiceSummary, error)
	RegisterInstance(context.Context, string, *corev1.Pod, *appmesh.AwsCloudMapServiceDiscovery) error
	DeregisterInstance(context.Context, string, *appmesh.AwsCloudMapServiceDiscovery) error
	ListInstances(context.Context, *appmesh.AwsCloudMapServiceDiscovery) ([]*servicediscovery.InstanceSummary, error)
}

//CloudMapCreateService calls AWS ServiceDiscovery CreateService API
func (c *Cloud) CloudMapCreateService(ctx context.Context, cloudmapConfig *appmesh.AwsCloudMapServiceDiscovery, creatorRequestID string) (*CloudMapServiceSummary, error) {
	key := c.serviceCacheKey(cloudmapConfig)

	existingItem, exists, _ := c.serviceIDCache.Get(&cloudmapServiceCacheItem{
		key: key,
	})
	if exists && existingItem != nil {
		return &(existingItem.(*cloudmapServiceCacheItem)).value, nil
	}

	namespaceSummary, err := c.getNamespace(ctx, cloudmapConfig)
	if err != nil {
		return nil, err
	}

	if namespaceSummary == nil {
		return nil, fmt.Errorf("Could not find namespace in cloudmap with name %s", awssdk.StringValue(cloudmapConfig.NamespaceName))
	}

	//Only DNS namespaces provide support for applications to resolve service names
	//to endpoints. HTTP namespaces on the other hand require discover-instances API
	//to be called to resolve names to endpoints. In App Mesh application uses standard
	//DNS resolution before Envoy intercepts the traffic and uses discover-instances via
	//App Mesh's EDS endpoint. Hence, both DNS and discover-instances resolution need to
	//work. Today only private dns namespaces support this, hence this check.
	//It is possible to use HTTP namespaces when Envoy supports DNS filter
	//https://github.com/envoyproxy/envoy/issues/6748
	//TODO add support for HTTP namespace
	if namespaceSummary.NamespaceType != servicediscovery.NamespaceTypeDnsPrivate {
		return nil, fmt.Errorf("Cannot create service under namespace %s with type %s, only namespaces with type %s are supported",
			awssdk.StringValue(cloudmapConfig.NamespaceName),
			namespaceSummary.NamespaceType,
			servicediscovery.NamespaceTypeDnsPrivate,
		)
	}

	return c.createServiceUnderPrivateDNSNamespace(ctx, cloudmapConfig, creatorRequestID, namespaceSummary)
}

func (c *Cloud) createServiceUnderPrivateDNSNamespace(ctx context.Context, cloudmapConfig *appmesh.AwsCloudMapServiceDiscovery, creatorRequestID string, namespaceSummary *CloudMapNamespaceSummary) (*CloudMapServiceSummary, error) {
	key := c.serviceCacheKey(cloudmapConfig)
	createServiceInput := &servicediscovery.CreateServiceInput{
		CreatorRequestId: awssdk.String(creatorRequestID),
		Name:             cloudmapConfig.ServiceName,
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
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*CreateServiceTimeout)
	defer cancel()

	createServiceOutput, err := c.cloudmap.CreateServiceWithContext(ctx, createServiceInput)
	if err != nil {
		//ignore already exists error
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == servicediscovery.ErrCodeServiceAlreadyExists {
				return c.getService(ctx, cloudmapConfig)
			}
		}
		return nil, err
	}

	serviceItem := &cloudmapServiceCacheItem{
		key: key,
		value: CloudMapServiceSummary{
			NamespaceID: namespaceSummary.NamespaceID,
			ServiceID:   awssdk.StringValue(createServiceOutput.Service.Id),
		},
	}
	_ = c.serviceIDCache.Add(serviceItem)
	return &serviceItem.value, nil
}

//CloudMapGetService calls AWS ServiceDiscovery GetService API
func (c *Cloud) CloudMapGetService(ctx context.Context, serviceID string) (*CloudMapServiceSummary, error) {
	getServiceInput := &servicediscovery.GetServiceInput{
		Id: awssdk.String(serviceID),
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*GetServiceTimeout)
	defer cancel()

	getServiceOutput, err := c.cloudmap.GetServiceWithContext(ctx, getServiceInput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == servicediscovery.ErrCodeServiceNotFound {
				return nil, nil
			}
		}
		return nil, err
	}

	return &CloudMapServiceSummary{
		NamespaceID: awssdk.StringValue(getServiceOutput.Service.NamespaceId),
		ServiceID:   awssdk.StringValue(getServiceOutput.Service.Id),
	}, nil
}

// RegisterInstance calls AWS ServiceDiscovery RegisterInstance API
func (c *Cloud) RegisterInstance(ctx context.Context, instanceID string, pod *corev1.Pod, cloudmapConfig *appmesh.AwsCloudMapServiceDiscovery) error {
	if pod.Status.Phase != corev1.PodRunning {
		klog.V(4).Infof("Pod is in %s phase, skipping", pod.Status.Phase)
		return nil
	}

	serviceSummary, err := c.getService(ctx, cloudmapConfig)
	if err != nil {
		klog.V(4).Infof("Could not find service in cloudmap with name %s in namespace %s, err=%v",
			awssdk.StringValue(cloudmapConfig.ServiceName),
			awssdk.StringValue(cloudmapConfig.NamespaceName),
			err)
		return nil
	}

	attr := make(map[string]*string)
	for k, v := range pod.Labels {
		attr[k] = awssdk.String(v)
	}
	attr[AttrAwsInstanceIPV4] = awssdk.String(pod.Status.PodIP)
	attr[AttrK8sPod] = awssdk.String(pod.Name)
	attr[AttrK8sNamespace] = awssdk.String(pod.Namespace)
	//copy the attributes specified on virtual-node
	for _, a := range cloudmapConfig.Attributes {
		attr[awssdk.StringValue(a.Key)] = a.Value
	}

	input := &servicediscovery.RegisterInstanceInput{
		ServiceId:        awssdk.String(serviceSummary.ServiceID),
		InstanceId:       awssdk.String(instanceID),
		CreatorRequestId: awssdk.String(pod.Name),
		Attributes:       attr,
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*RegisterInstanceTimeout)
	defer cancel()

	_, err = c.cloudmap.RegisterInstanceWithContext(ctx, input)
	//ignore duplicate-request
	if aerr, ok := err.(awserr.Error); ok {
		if aerr.Code() == servicediscovery.ErrCodeDuplicateRequest {
			return nil
		}
		return err
	}
	return nil
}

// DeregisterInstance calls AWS ServiceDiscovery DeregisterInstance API
func (c *Cloud) DeregisterInstance(ctx context.Context, instanceID string, cloudmapConfig *appmesh.AwsCloudMapServiceDiscovery) error {
	serviceSummary, err := c.getService(ctx, cloudmapConfig)
	if err != nil || serviceSummary == nil {
		return nil
	}

	input := &servicediscovery.DeregisterInstanceInput{
		ServiceId:  awssdk.String(serviceSummary.ServiceID),
		InstanceId: awssdk.String(instanceID),
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*DeregisterInstanceTimeout)
	defer cancel()

	_, err = c.cloudmap.DeregisterInstanceWithContext(ctx, input)
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

// ListInstances calls AWS ServiceDiscovery ListInstances API
func (c *Cloud) ListInstances(ctx context.Context, cloudmapConfig *appmesh.AwsCloudMapServiceDiscovery) ([]*servicediscovery.InstanceSummary, error) {
	instances := []*servicediscovery.InstanceSummary{}

	serviceSummary, err := c.getService(ctx, cloudmapConfig)
	if err != nil || serviceSummary == nil {
		return instances, nil
	}

	//NOTE: not calling discover-instances API because it is not paginated
	input := &servicediscovery.ListInstancesInput{
		ServiceId: awssdk.String(serviceSummary.ServiceID),
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*ListInstancesPagesTimeout)
	defer cancel()

	c.cloudmap.ListInstancesPagesWithContext(ctx, input, func(output *servicediscovery.ListInstancesOutput, lastPage bool) bool {
		for _, i := range output.Instances {
			if _, ok := i.Attributes[AttrK8sNamespace]; ok {
				if _, ok := i.Attributes[AttrK8sPod]; ok {
					instances = append(instances, i)
				}
			}

		}
		return false
	})

	return instances, nil
}

func (c *Cloud) getNamespace(ctx context.Context, cloudmapConfig *appmesh.AwsCloudMapServiceDiscovery) (*CloudMapNamespaceSummary, error) {
	key := c.namespaceCacheKey(cloudmapConfig)

	existingItem, exists, _ := c.namespaceIDCache.Get(&cloudmapNamespaceCacheItem{
		key: key,
	})
	if exists {
		return &(existingItem.(*cloudmapNamespaceCacheItem)).value, nil
	}

	listNamespacesInput := &servicediscovery.ListNamespacesInput{}
	var namespaceItem *cloudmapNamespaceCacheItem

	ctx, cancel := context.WithTimeout(ctx, time.Second*ListNamespacesPagesTimeout)
	defer cancel()

	err := c.cloudmap.ListNamespacesPagesWithContext(ctx,
		listNamespacesInput,
		func(listNamespacesOutput *servicediscovery.ListNamespacesOutput, lastPage bool) bool {
			for _, ns := range listNamespacesOutput.Namespaces {
				if awssdk.StringValue(ns.Name) == awssdk.StringValue(cloudmapConfig.NamespaceName) {
					namespaceItem = &cloudmapNamespaceCacheItem{
						key: key,
						value: CloudMapNamespaceSummary{
							NamespaceID:   awssdk.StringValue(ns.Id),
							NamespaceType: awssdk.StringValue(ns.Type),
						},
					}
					c.namespaceIDCache.Add(namespaceItem)
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
		klog.Infof("No namespace found with name %s", awssdk.StringValue(cloudmapConfig.NamespaceName))
		return nil, nil
	}

	return &namespaceItem.value, err
}

func (c *Cloud) getService(ctx context.Context, cloudmapConfig *appmesh.AwsCloudMapServiceDiscovery) (*CloudMapServiceSummary, error) {
	key := c.serviceCacheKey(cloudmapConfig)

	existingItem, exists, _ := c.serviceIDCache.Get(&cloudmapServiceCacheItem{
		key: key,
	})
	if exists {
		return &(existingItem.(*cloudmapServiceCacheItem)).value, nil
	}

	namespaceSummary, err := c.getNamespace(ctx, cloudmapConfig)
	if err != nil {
		return nil, err
	}

	if namespaceSummary == nil {
		return nil, fmt.Errorf("Could not find namespace in cloudmap with name %s", awssdk.StringValue(cloudmapConfig.NamespaceName))
	}

	cloudmapService, err := c.getServiceFromCloudMap(ctx, namespaceSummary.NamespaceID, awssdk.StringValue(cloudmapConfig.ServiceName))
	if err != nil {
		return nil, err
	}

	serviceItem := &cloudmapServiceCacheItem{
		key: key,
		value: CloudMapServiceSummary{
			NamespaceID: namespaceSummary.NamespaceID,
			ServiceID:   awssdk.StringValue(cloudmapService.Id),
		},
	}
	c.serviceIDCache.Add(serviceItem)
	return &serviceItem.value, nil
}

func (c *Cloud) getServiceFromCloudMap(ctx context.Context, namespaceID string, serviceName string) (*servicediscovery.ServiceSummary, error) {

	listServicesInput := &servicediscovery.ListServicesInput{
		Filters: []*servicediscovery.ServiceFilter{
			&servicediscovery.ServiceFilter{
				Name:   awssdk.String(servicediscovery.ServiceFilterNameNamespaceId),
				Values: []*string{awssdk.String(namespaceID)},
			},
		},
	}

	var svcSummary *servicediscovery.ServiceSummary

	ctx, cancel := context.WithTimeout(ctx, time.Second*ListServicesPagesTimeout)
	defer cancel()

	err := c.cloudmap.ListServicesPagesWithContext(ctx,
		listServicesInput,
		func(listServicesOutput *servicediscovery.ListServicesOutput, hasMore bool) bool {
			for _, svc := range listServicesOutput.Services {
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

func (c *Cloud) serviceCacheKey(cloudmapConfig *appmesh.AwsCloudMapServiceDiscovery) string {
	return awssdk.StringValue(cloudmapConfig.ServiceName) + "@" + awssdk.StringValue(cloudmapConfig.NamespaceName)
}

func (c *Cloud) namespaceCacheKey(cloudmapConfig *appmesh.AwsCloudMapServiceDiscovery) string {
	return awssdk.StringValue(cloudmapConfig.NamespaceName)
}
