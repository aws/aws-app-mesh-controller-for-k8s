package cloudmap

import (
	"context"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws/services"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	attrAWSInitHealthStatus = "AWS_INIT_HEALTH_STATUS"
)

// newInstancesReconcileTask constructs new instancesReconcileTask for specific subset of cloudMap service.
func newInstancesReconcileTask(cloudMapSDK services.CloudMap, instancesCache instancesCache, log logr.Logger, done chan struct{}) *instancesReconcileTask {
	return &instancesReconcileTask{
		cloudMapSDK:    cloudMapSDK,
		instancesCache: instancesCache,
		done:           done,

		instancesReconcileRequestChan:   make(chan instancesReconcileRequest),
		instancesWithOngoingOperation:   sets.String{},
		instanceOperationCompletionChan: make(chan instanceOperationResult),
		log:                             log,
	}
}

// instancesReconcileTask representing the work to reconcile instances for specific subset of cloudMap service.
// each instancesReconcileTask should be limited to only work for a single service & subset.
type instancesReconcileTask struct {
	cloudMapSDK    services.CloudMap
	instancesCache instancesCache
	done           chan struct{}

	instancesReconcileRequestChan chan instancesReconcileRequest
	// instances that have on-going operation, we'll skip these instances.
	instancesWithOngoingOperation sets.String
	// chan of instances that completed operation
	instanceOperationCompletionChan chan instanceOperationResult

	log logr.Logger
}

type instanceOperationResult struct {
	instanceID string
	err        error
}

// run starts the instancesReconcileTask
// It terminates when instances have been successfully reconciled according to desired state.
// i.e. no new desiredState and all async operations are completed.
func (t *instancesReconcileTask) Run(ctx context.Context) {
	request := <-t.instancesReconcileRequestChan
	for {
		err := t.reconcile(ctx, request.service, request.subset, request.readyInstanceInfoByID, request.unreadyInstanceInfoByID)
		if err != nil {
			request.resultChan <- err
			close(request.resultChan)
			return
		}
		if len(t.instancesWithOngoingOperation) == 0 {
			close(request.resultChan)
			return
		}
		select {
		case <-ctx.Done():
			request.resultChan <- ctx.Err()
			close(request.resultChan)
			return
		case newRequest := <-t.instancesReconcileRequestChan:
			request.resultChan <- errors.New("cancelled by new desired state")
			close(request.resultChan)
			request = newRequest
		case operationResult := <-t.instanceOperationCompletionChan:
			t.instancesWithOngoingOperation.Delete(operationResult.instanceID)
			if operationResult.err != nil {
				request.resultChan <- operationResult.err
				close(request.resultChan)
				return
			}
		}
	}
}

// reconcile will try to reconcile subset of cloudMap service into desired state.
// returns ready instances that have been reconciled successfully
func (t *instancesReconcileTask) reconcile(ctx context.Context, service serviceSummary, subset serviceSubset,
	desiredReadyInstanceInfoByID map[string]instanceInfo, desiredNotReadyInstanceInfoByID map[string]instanceInfo) error {

	existingInstanceAttrsByID, err := t.listServiceSubsetInstances(ctx, service, subset)
	if err != nil {
		return err
	}

	instancesToCreateOrUpdate, instancesToDelete := t.matchDesiredInstancesAgainstExistingInstances(desiredReadyInstanceInfoByID, desiredNotReadyInstanceInfoByID, existingInstanceAttrsByID)

	t.log.V(1).Info("CloudMap: Register Instances", "InstanceToCreateOrUpdate", instancesToCreateOrUpdate)

	for instanceID, info := range instancesToCreateOrUpdate {
		if t.instancesWithOngoingOperation.Has(instanceID) {
			continue
		}
		t.instancesWithOngoingOperation.Insert(instanceID)
		go func(instanceID string, info instanceInfo) {
			err := t.instancesCache.RegisterInstance(ctx, service.serviceID, instanceID, info.attrs)
			select {
			case t.instanceOperationCompletionChan <- instanceOperationResult{instanceID: instanceID, err: err}:
			case <-t.done:
			}
		}(instanceID, info)
	}

	t.log.V(1).Info("CloudMap: Deregister Instances", "instancesToDelete", instancesToDelete)

	for _, instanceID := range instancesToDelete {
		if t.instancesWithOngoingOperation.Has(instanceID) {
			continue
		}
		t.instancesWithOngoingOperation.Insert(instanceID)
		go func(instanceID string) {
			err := t.instancesCache.DeregisterInstance(ctx, service.serviceID, instanceID)
			select {
			case t.instanceOperationCompletionChan <- instanceOperationResult{instanceID: instanceID, err: err}:
			case <-t.done:
			}
		}(instanceID)
	}

	return nil
}

func (t *instancesReconcileTask) matchDesiredInstancesAgainstExistingInstances(
	desiredReadyInstanceInfoByID map[string]instanceInfo,
	desiredNotReadyInstanceInfoByID map[string]instanceInfo,
	existingInstanceAttrsByID map[string]instanceAttributes) (map[string]instanceInfo, []string) {

	instancesToCreateOrUpdate := make(map[string]instanceInfo)

	for instanceID, desiredInfo := range desiredReadyInstanceInfoByID {
		if existingAttrs, exists := existingInstanceAttrsByID[instanceID]; exists {
			if !cmp.Equal(desiredInfo.attrs, existingAttrs, ignoreAttrAWSInitHealthStatus()) {
				if existingInitHealthStatus, ok := existingAttrs[attrAWSInitHealthStatus]; ok {
					desiredInfo.attrs[attrAWSInitHealthStatus] = existingInitHealthStatus
				} else {
					desiredInfo.attrs[attrAWSInitHealthStatus] = servicediscovery.CustomHealthStatusHealthy
				}
				instancesToCreateOrUpdate[instanceID] = desiredInfo
			}
		} else {
			desiredInfo.attrs[attrAWSInitHealthStatus] = servicediscovery.CustomHealthStatusHealthy
			instancesToCreateOrUpdate[instanceID] = desiredInfo
		}
	}

	for instanceID, desiredInfo := range desiredNotReadyInstanceInfoByID {
		if existingAttrs, exists := existingInstanceAttrsByID[instanceID]; exists {
			if !cmp.Equal(desiredInfo.attrs, existingAttrs, ignoreAttrAWSInitHealthStatus()) {
				if existingInitHealthStatus, ok := existingAttrs[attrAWSInitHealthStatus]; ok {
					desiredInfo.attrs[attrAWSInitHealthStatus] = existingInitHealthStatus
				} else {
					desiredInfo.attrs[attrAWSInitHealthStatus] = servicediscovery.CustomHealthStatusUnhealthy
				}
				instancesToCreateOrUpdate[instanceID] = desiredInfo
			}
		}
	}

	desiredInstanceIDs := sets.StringKeySet(desiredReadyInstanceInfoByID).Union(sets.StringKeySet(desiredNotReadyInstanceInfoByID))
	existingInstanceIDs := sets.StringKeySet(existingInstanceAttrsByID)
	instancesToDelete := existingInstanceIDs.Difference(desiredInstanceIDs).List()
	return instancesToCreateOrUpdate, instancesToDelete
}

// listServiceSubsetInstances returns instances that should belong to subset of cloudMap service.
func (t *instancesReconcileTask) listServiceSubsetInstances(ctx context.Context, service serviceSummary, subset serviceSubset) (map[string]instanceAttributes, error) {
	instanceAttrsByID, err := t.instancesCache.ListInstances(ctx, service.serviceID)
	if err != nil {
		return nil, err
	}
	subsetInstanceAttrsByID := make(map[string]instanceAttributes)
	for instanceID, attrs := range instanceAttrsByID {
		if subset.Contains(instanceID, attrs) {
			subsetInstanceAttrsByID[instanceID] = attrs
		}
	}
	return subsetInstanceAttrsByID, nil
}

func ignoreAttrAWSInitHealthStatus() cmp.Option {
	return cmpopts.IgnoreMapEntries(func(key string, _ string) bool {
		return key == attrAWSInitHealthStatus
	})
}
