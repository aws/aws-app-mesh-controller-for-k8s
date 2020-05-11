package cloudmap

import (
	"context"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws/services"
	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/util/cache"
	"sync"
	"time"
)

const (
	defaultInstanceAttrsCacheTTL        = 5 * time.Minute
	defaultInstanceAttrsCacheSize       = 1024
	defaultOperationPollPeriod          = 3 * time.Second
	defaultOperationPollTimeout         = 15 * time.Minute
	defaultOperationPollEntryChanBuffer = 10
)

type InstanceAttributes map[string]string

// InstancesCache provides interface to asynchronous register/deregister instances call while maintains cache for ListInstances.
type InstancesCache interface {
	// ListInstances list all instances registered for serviceID.
	ListInstances(ctx context.Context, serviceID string) (map[string]InstanceAttributes, error)
	// RegisterInstance register a instance to serviceID.
	RegisterInstance(ctx context.Context, serviceID string, instanceID string, attrs InstanceAttributes) error
	// DeregisterInstance a instance to serviceID.
	DeregisterInstance(ctx context.Context, serviceID string, instanceID string) error
}

func NewDefaultInstancesCache(cloudMapSDK services.CloudMap, log logr.Logger, stopChan <-chan struct{}) *defaultInstancesCache {
	cache := &defaultInstancesCache{
		cloudMapSDK:            cloudMapSDK,
		instancesAttrsCacheTTL: defaultInstanceAttrsCacheTTL,
		instancesAttrsCache:    cache.NewLRUExpireCache(defaultInstanceAttrsCacheSize),

		operationsInProgress:      make(map[instanceWithinServiceID][]operationInfo),
		operationsInProgressMutex: sync.RWMutex{},
		operationPollEntryChan:    make(chan operationPollEntry, defaultOperationPollEntryChanBuffer),
		operationPollPeriod:       defaultOperationPollPeriod,
		operationPollTimeout:      defaultOperationPollTimeout,
		log:                       log,
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		select {
		case <-stopChan:
			cancel()
		}
	}()
	go cache.pollOperationsLoop(ctx)
	return cache
}

var _ InstancesCache = &defaultInstancesCache{}

type defaultInstancesCache struct {
	cloudMapSDK services.CloudMap

	instancesAttrsCacheTTL time.Duration
	instancesAttrsCache    *cache.LRUExpireCache

	// operations in progress indexed by serviceID/instanceID
	operationsInProgress      map[instanceWithinServiceID][]operationInfo
	operationsInProgressMutex sync.RWMutex
	operationPollEntryChan    chan operationPollEntry
	operationPollPeriod       time.Duration
	operationPollTimeout      time.Duration

	log logr.Logger
}

type instancesAttrsCacheItem struct {
	instancesAttrsByID map[string]InstanceAttributes
	mutex              sync.RWMutex
}

// identity for a instance within service.
type instanceWithinServiceID struct {
	serviceID  string
	instanceID string
}

// information about a ongoing operation.
type operationInfo struct {
	operationID   string
	operationType string

	// for OperationTypeRegisterInstance, the attributes will be registered instance attributes.
	instanceAttrs InstanceAttributes
	// first time when this operation is been polled.
	firstPollTime time.Time
}

type operationPollEntry struct {
	instanceWithinServiceID instanceWithinServiceID
	operation               operationInfo
}

func (c *defaultInstancesCache) ListInstances(ctx context.Context, serviceID string) (map[string]InstanceAttributes, error) {
	if cachedValue, exists := c.instancesAttrsCache.Get(serviceID); exists {
		cacheItem := cachedValue.(*instancesAttrsCacheItem)
		cacheItem.mutex.RLock()
		defer cacheItem.mutex.RUnlock()
		return c.cloneInstanceAttributesByID(cacheItem.instancesAttrsByID), nil
	}

	instancesAttrsByID, err := c.listInstancesFromAWS(ctx, serviceID)
	if err != nil {
		return nil, err
	}
	cacheItem := &instancesAttrsCacheItem{
		instancesAttrsByID: instancesAttrsByID,
		mutex:              sync.RWMutex{},
	}
	instanceAttrsByIDClone := c.cloneInstanceAttributesByID(instancesAttrsByID)
	c.instancesAttrsCache.Add(serviceID, cacheItem, c.instancesAttrsCacheTTL)
	return instanceAttrsByIDClone, nil
}

func (c *defaultInstancesCache) RegisterInstance(ctx context.Context, serviceID string, instanceID string, attrs InstanceAttributes) error {
	instanceWithinServiceID := instanceWithinServiceID{
		serviceID:  serviceID,
		instanceID: instanceID,
	}
	if c.hasInprogressRegisterInstanceOperation(instanceWithinServiceID, attrs) {
		return nil
	}
	resp, err := c.cloudMapSDK.RegisterInstanceWithContext(ctx, &servicediscovery.RegisterInstanceInput{
		ServiceId:  awssdk.String(serviceID),
		InstanceId: awssdk.String(instanceID),
		Attributes: awssdk.StringMap(attrs),
	})
	if err != nil {
		return err
	}
	c.operationPollEntryChan <- operationPollEntry{
		instanceWithinServiceID: instanceWithinServiceID,
		operation: operationInfo{
			operationID:   awssdk.StringValue(resp.OperationId),
			operationType: servicediscovery.OperationTypeRegisterInstance,
			instanceAttrs: attrs,
			firstPollTime: time.Time{},
		},
	}
	return nil
}

func (c *defaultInstancesCache) DeregisterInstance(ctx context.Context, serviceID string, instanceID string) error {
	instanceWithinServiceID := instanceWithinServiceID{
		serviceID:  serviceID,
		instanceID: instanceID,
	}
	if c.hasInprogressDeregisterInstanceOperation(instanceWithinServiceID) {
		return nil
	}
	resp, err := c.cloudMapSDK.DeregisterInstanceWithContext(ctx, &servicediscovery.DeregisterInstanceInput{
		ServiceId:  awssdk.String(serviceID),
		InstanceId: awssdk.String(instanceID),
	})
	if err != nil {
		return err
	}
	c.operationPollEntryChan <- operationPollEntry{
		instanceWithinServiceID: instanceWithinServiceID,
		operation: operationInfo{
			operationID:   awssdk.StringValue(resp.OperationId),
			operationType: servicediscovery.OperationTypeDeregisterInstance,
			instanceAttrs: nil,
			firstPollTime: time.Time{},
		},
	}
	return nil
}

func (c *defaultInstancesCache) pollOperationsLoop(ctx context.Context) {
	for {
		var timer <-chan time.Time
		if len(c.operationsInProgress) > 0 {
			timer = time.After(c.operationPollPeriod)
		}
		select {
		case <-ctx.Done():
			return
		case operationPollEntry := <-c.operationPollEntryChan:
			c.addOperationToPoll(operationPollEntry)
		case <-timer:
			c.pollOperations(ctx)
		}
	}
}

func (c *defaultInstancesCache) addOperationToPoll(entry operationPollEntry) {
	c.operationsInProgressMutex.Lock()
	defer c.operationsInProgressMutex.Unlock()

	for _, existingOperation := range c.operationsInProgress[entry.instanceWithinServiceID] {
		if existingOperation.operationID == entry.operation.operationID {
			c.log.Info("skipping existing operation",
				"serviceID", entry.instanceWithinServiceID.serviceID,
				"instanceID", entry.instanceWithinServiceID.instanceID,
				"operationID", entry.operation.operationID)
			return
		}
	}
	c.operationsInProgress[entry.instanceWithinServiceID] = append(c.operationsInProgress[entry.instanceWithinServiceID], entry.operation)
}

func (c *defaultInstancesCache) hasInprogressRegisterInstanceOperation(instanceWithinServiceID instanceWithinServiceID, attrs InstanceAttributes) bool {
	c.operationsInProgressMutex.RLock()
	defer c.operationsInProgressMutex.RUnlock()
	for _, existingOperation := range c.operationsInProgress[instanceWithinServiceID] {
		if existingOperation.operationType == servicediscovery.OperationTypeRegisterInstance && cmp.Equal(existingOperation.instanceAttrs, attrs) {
			return true
		}
	}
	return false
}

func (c *defaultInstancesCache) hasInprogressDeregisterInstanceOperation(instanceWithinServiceID instanceWithinServiceID) bool {
	c.operationsInProgressMutex.RLock()
	defer c.operationsInProgressMutex.RUnlock()
	for _, existingOperation := range c.operationsInProgress[instanceWithinServiceID] {
		if existingOperation.operationType == servicediscovery.OperationTypeDeregisterInstance {
			return true
		}
	}
	return false
}

func (c *defaultInstancesCache) pollOperations(ctx context.Context) {
	operationsContinuePoll := make(map[instanceWithinServiceID][]operationInfo)
	var operationsContinuePollMutex sync.Mutex
	var wg sync.WaitGroup
	c.operationsInProgressMutex.RLock()
	for id, operations := range c.operationsInProgress {
		for _, operation := range operations {
			wg.Add(1)
			go func(instanceWithinServiceID instanceWithinServiceID, operation operationInfo) {
				defer wg.Done()
				continuePoll, err := c.pollOperation(ctx, instanceWithinServiceID, &operation)
				if err != nil {
					c.log.Error(err, "failed to poll operation",
						"serviceID", instanceWithinServiceID.serviceID,
						"instanceID", instanceWithinServiceID.instanceID,
						"operationID", operation.operationID)
					operationsContinuePollMutex.Lock()
					operationsContinuePoll[instanceWithinServiceID] = append(operationsContinuePoll[instanceWithinServiceID], operation)
					operationsContinuePollMutex.Unlock()
				} else if continuePoll {
					operationsContinuePollMutex.Lock()
					operationsContinuePoll[instanceWithinServiceID] = append(operationsContinuePoll[instanceWithinServiceID], operation)
					operationsContinuePollMutex.Unlock()
				}
			}(id, operation)
		}
	}
	c.operationsInProgressMutex.RUnlock()
	wg.Wait()
	c.operationsInProgressMutex.Lock()
	c.operationsInProgress = operationsContinuePoll
	c.operationsInProgressMutex.Unlock()
}

func (c *defaultInstancesCache) pollOperation(ctx context.Context, instanceWithinServiceID instanceWithinServiceID, operation *operationInfo) (bool, error) {
	if operation.firstPollTime.IsZero() {
		operation.firstPollTime = time.Now()
	}

	if time.Since(operation.firstPollTime) > c.operationPollTimeout {
		c.log.Error(nil, "timeout poll operation",
			"serviceID", instanceWithinServiceID.serviceID,
			"instanceID", instanceWithinServiceID.instanceID,
			"operationID", operation.operationID)
		return false, nil
	}

	resp, err := c.cloudMapSDK.GetOperationWithContext(ctx, &servicediscovery.GetOperationInput{
		OperationId: awssdk.String(operation.operationID),
	})
	if err != nil {
		return false, err
	}
	switch awssdk.StringValue(resp.Operation.Status) {
	case servicediscovery.OperationStatusSuccess:
		c.recordSuccessOperation(ctx, instanceWithinServiceID, *operation, resp.Operation)
		return false, nil
	case servicediscovery.OperationStatusFail:
		return false, nil
	default:
		return true, nil
	}
}

func (c *defaultInstancesCache) recordSuccessOperation(ctx context.Context, instanceWithinServiceID instanceWithinServiceID, operation operationInfo, sdkOperation *servicediscovery.Operation) {
	cachedValue, exists := c.instancesAttrsCache.Get(instanceWithinServiceID.serviceID)
	if !exists {
		return
	}

	cacheItem := cachedValue.(*instancesAttrsCacheItem)
	cacheItem.mutex.RLock()
	defer cacheItem.mutex.RUnlock()
	switch operation.operationType {
	case servicediscovery.OperationTypeRegisterInstance:
		cacheItem.instancesAttrsByID[instanceWithinServiceID.instanceID] = operation.instanceAttrs
	case servicediscovery.OperationTypeDeregisterInstance:
		delete(cacheItem.instancesAttrsByID, instanceWithinServiceID.instanceID)
	}
}

func (c *defaultInstancesCache) listInstancesFromAWS(ctx context.Context, serviceID string) (map[string]InstanceAttributes, error) {
	input := &servicediscovery.ListInstancesInput{
		ServiceId: awssdk.String(serviceID),
	}
	instancesAttrsByID := make(map[string]InstanceAttributes)
	if err := c.cloudMapSDK.ListInstancesPagesWithContext(ctx, input, func(output *servicediscovery.ListInstancesOutput, b bool) bool {
		for _, instance := range output.Instances {
			instancesAttrsByID[awssdk.StringValue(instance.Id)] = awssdk.StringValueMap(instance.Attributes)
		}
		return true
	}); err != nil {
		return nil, err
	}
	return instancesAttrsByID, nil
}

// cloneInstanceAttributesByID make a copy of instancesAttrsByID.
// we return a copy in ListInstances to avoid race if the map is read concurrently with writes from registerInstance/deregisterInstance.
func (c *defaultInstancesCache) cloneInstanceAttributesByID(instancesAttrsByID map[string]InstanceAttributes) map[string]InstanceAttributes {
	instancesAttrsByIDClone := make(map[string]InstanceAttributes, len(instancesAttrsByID))
	for id, attrs := range instancesAttrsByID {
		instancesAttrsByIDClone[id] = attrs
	}
	return instancesAttrsByIDClone
}
