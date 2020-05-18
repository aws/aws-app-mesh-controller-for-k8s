package cloudmap

import (
	"context"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws/retry"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws/services"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/cache"
	"k8s.io/apimachinery/pkg/util/wait"
	"sync"
	"time"
)

const (
	defaultInstanceAttrsCacheTTL  = 5 * time.Minute
	defaultInstanceAttrsCacheSize = 1024

	defaultOperationPollInterval = 3 * time.Second
	// we want to retry higher times for getOperation than other normal AWS API :D
	defaultOperationPollMaxRetries = 10
)

// instancesCache is an abstraction around cloudMap's asynchronous instances API.
// It is able to take the result of register/deregister operations to provide a more current view.
type instancesCache interface {
	// ListInstances returns all instances associated with cloudMap service.
	ListInstances(ctx context.Context, serviceID string) (map[string]instanceAttributes, error)
	// RegisterInstance register an instance into cloudMap service
	// it blocks until registerInstance operation succeeds or fails.
	RegisterInstance(ctx context.Context, serviceID string, instanceID string, attrs instanceAttributes) error
	// DeregisterInstance deregister an instance from cloudMap service.
	// it blocks until deregisterInstance operation succeeds or fails.
	DeregisterInstance(ctx context.Context, serviceID string, instanceID string) error
}

// newDefaultInstancesCache constructs defaultInstancesCache
func newDefaultInstancesCache(cloudMapSDK services.CloudMap) *defaultInstancesCache {
	return &defaultInstancesCache{
		cloudMapSDK:              cloudMapSDK,
		instancesAttrsCache:      cache.NewLRUExpireCache(defaultInstanceAttrsCacheSize),
		instancesAttrsCacheMutex: sync.Mutex{},
		instancesAttrsCacheTTL:   defaultInstanceAttrsCacheTTL,
		operationPollInterval:    defaultOperationPollInterval,
		operationPollMaxRetries:  defaultOperationPollMaxRetries,
	}
}

var _ instancesCache = &defaultInstancesCache{}

// defaultInstancesCache implements instancesCache
type defaultInstancesCache struct {
	cloudMapSDK              services.CloudMap
	instancesAttrsCache      *cache.LRUExpireCache
	instancesAttrsCacheMutex sync.Mutex
	instancesAttrsCacheTTL   time.Duration

	// interval between each getOperation call
	operationPollInterval time.Duration
	// maximum retries per getOperation call
	operationPollMaxRetries int
}

type instancesAttrsCacheItem struct {
	// the attributes of instances indexed by instance ID.
	instanceAttrsByID map[string]instanceAttributes
	// the last time we modified a instance's attributes
	lastUpdatedTimeByID map[string]time.Time

	mutex sync.RWMutex
}

func (c *defaultInstancesCache) ListInstances(ctx context.Context, serviceID string) (map[string]instanceAttributes, error) {
	c.instancesAttrsCacheMutex.Lock()
	defer c.instancesAttrsCacheMutex.Unlock()

	if cachedValue, exists := c.instancesAttrsCache.Get(serviceID); exists {
		cacheItem := cachedValue.(*instancesAttrsCacheItem)
		cacheItem.mutex.RLock()
		defer cacheItem.mutex.RUnlock()
		return c.cloneInstanceAttributesByID(cacheItem.instanceAttrsByID), nil
	}

	instanceAttrsByID, err := c.listInstancesFromAWS(ctx, serviceID)
	if err != nil {
		return nil, err
	}
	cacheItem := &instancesAttrsCacheItem{
		instanceAttrsByID:   instanceAttrsByID,
		lastUpdatedTimeByID: make(map[string]time.Time),
		mutex:               sync.RWMutex{},
	}
	instanceAttrsByIDClone := c.cloneInstanceAttributesByID(instanceAttrsByID)
	c.instancesAttrsCache.Add(serviceID, cacheItem, c.instancesAttrsCacheTTL)
	return instanceAttrsByIDClone, nil
}

func (c *defaultInstancesCache) RegisterInstance(ctx context.Context, serviceID string, instanceID string, attrs instanceAttributes) error {
	optResp, err := c.cloudMapSDK.RegisterInstanceWithContext(ctx, &servicediscovery.RegisterInstanceInput{
		ServiceId:  aws.String(serviceID),
		InstanceId: aws.String(instanceID),
		Attributes: aws.StringMap(attrs),
	})
	if err != nil {
		return err
	}
	return wait.PollUntil(c.operationPollInterval, func() (done bool, err error) {
		getOptResp, err := c.cloudMapSDK.GetOperationWithContext(ctx, &servicediscovery.GetOperationInput{
			OperationId: optResp.OperationId,
		}, retry.WithMaxRetries(c.operationPollMaxRetries))
		if err != nil {
			return false, err
		}
		switch aws.StringValue(getOptResp.Operation.Status) {
		case servicediscovery.OperationStatusSuccess:
			c.recordSuccessfulRegisterInstanceOperation(serviceID, instanceID, attrs, getOptResp.Operation)
			return true, nil
		case servicediscovery.OperationStatusFail:
			return true, errors.New(aws.StringValue(getOptResp.Operation.ErrorMessage))
		default:
			return false, nil
		}
	}, ctx.Done())
}

func (c *defaultInstancesCache) DeregisterInstance(ctx context.Context, serviceID string, instanceID string) error {
	optResp, err := c.cloudMapSDK.DeregisterInstanceWithContext(ctx, &servicediscovery.DeregisterInstanceInput{
		ServiceId:  aws.String(serviceID),
		InstanceId: aws.String(instanceID),
	})
	if err != nil {
		return err
	}

	return wait.PollUntil(c.operationPollInterval, func() (done bool, err error) {
		getOptResp, err := c.cloudMapSDK.GetOperationWithContext(ctx, &servicediscovery.GetOperationInput{
			OperationId: optResp.OperationId,
		}, retry.WithMaxRetries(c.operationPollMaxRetries))
		if err != nil {
			return false, err
		}
		switch aws.StringValue(getOptResp.Operation.Status) {
		case servicediscovery.OperationStatusSuccess:
			c.recordSuccessfulDeregisterInstanceOperation(serviceID, instanceID, getOptResp.Operation)
			return true, nil
		case servicediscovery.OperationStatusFail:
			return true, errors.New(aws.StringValue(getOptResp.Operation.ErrorMessage))
		default:
			return false, nil
		}
	}, ctx.Done())
}

func (c *defaultInstancesCache) listInstancesFromAWS(ctx context.Context, serviceID string) (map[string]instanceAttributes, error) {
	input := &servicediscovery.ListInstancesInput{
		ServiceId: aws.String(serviceID),
	}
	instanceAttrsByID := make(map[string]instanceAttributes)
	if err := c.cloudMapSDK.ListInstancesPagesWithContext(ctx, input, func(output *servicediscovery.ListInstancesOutput, b bool) bool {
		for _, instance := range output.Instances {
			instanceAttrsByID[aws.StringValue(instance.Id)] = aws.StringValueMap(instance.Attributes)
		}
		return true
	}); err != nil {
		return nil, err
	}
	return instanceAttrsByID, nil
}

func (c *defaultInstancesCache) recordSuccessfulRegisterInstanceOperation(serviceID string, instanceID string, attrs instanceAttributes, operation *servicediscovery.Operation) {
	cachedValue, exists := c.instancesAttrsCache.Get(serviceID)
	if !exists {
		return
	}
	cacheItem := cachedValue.(*instancesAttrsCacheItem)
	cacheItem.mutex.Lock()
	defer cacheItem.mutex.Unlock()
	if operation.UpdateDate.Before(cacheItem.lastUpdatedTimeByID[instanceID]) {
		return
	}
	cacheItem.lastUpdatedTimeByID[instanceID] = *operation.UpdateDate
	cacheItem.instanceAttrsByID[instanceID] = attrs
}

func (c *defaultInstancesCache) recordSuccessfulDeregisterInstanceOperation(serviceID string, instanceID string, operation *servicediscovery.Operation) {
	cachedValue, exists := c.instancesAttrsCache.Get(serviceID)
	if !exists {
		return
	}
	cacheItem := cachedValue.(*instancesAttrsCacheItem)
	cacheItem.mutex.Lock()
	defer cacheItem.mutex.Unlock()
	if operation.UpdateDate.Before(cacheItem.lastUpdatedTimeByID[instanceID]) {
		return
	}
	cacheItem.lastUpdatedTimeByID[instanceID] = *operation.UpdateDate
	delete(cacheItem.instanceAttrsByID, instanceID)
}

// cloneInstanceAttributesByID make a copy of instanceAttrsByID.
// we return a copy in ListInstances to avoid race if the map is read concurrently with writes from registerInstance/deregisterInstance.
func (c *defaultInstancesCache) cloneInstanceAttributesByID(instanceAttrsByID map[string]instanceAttributes) map[string]instanceAttributes {
	instanceAttrsByIDClone := make(map[string]instanceAttributes, len(instanceAttrsByID))
	for instanceID, attrs := range instanceAttrsByID {
		instanceAttrsByIDClone[instanceID] = attrs
	}
	return instanceAttrsByIDClone
}
