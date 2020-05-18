package cloudmap

import (
	"context"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws/services"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sync"
)

const (
	defaultInstancesReconcileReactorRequestChanBuffer = 10
)

// instancesReconcileReactor manages the asynchronous execution for instances reconcile.
type instancesReconcileReactor interface {
	// Submit submits a instances reconcile request, it will asynchronously drive cloudMap service's subset to match desiredState.
	Submit(ctx context.Context, service serviceSummary, subset serviceSubset, readyInstanceInfoByID map[string]instanceInfo, unreadyInstanceInfoByID map[string]instanceInfo) <-chan error
}

// newDefaultInstancesReconcileReactor constructs new defaultInstancesReconcileReactor
func newDefaultInstancesReconcileReactor(ctx context.Context, k8sClient client.Client, cloudMapSDK services.CloudMap, log logr.Logger) *defaultInstancesReconcileReactor {
	instancesCache := newDefaultInstancesCache(cloudMapSDK)
	reactor := &defaultInstancesReconcileReactor{
		cloudMapSDK:                       cloudMapSDK,
		instancesCache:                    instancesCache,
		reconcileRequestChan:              make(chan instancesReconcileRequest, defaultInstancesReconcileReactorRequestChanBuffer),
		reconcileTaskByServiceSubset:      make(map[serviceSubsetID]*instancesReconcileTask),
		reconcileTaskByServiceSubsetMutex: sync.RWMutex{},
		log:                               log,
	}

	go reactor.reactorLoop(ctx)
	return reactor
}

var _ instancesReconcileReactor = &defaultInstancesReconcileReactor{}

type defaultInstancesReconcileReactor struct {
	cloudMapSDK    services.CloudMap
	instancesCache instancesCache

	// channel to receive reconcile requests
	reconcileRequestChan chan instancesReconcileRequest

	// reconcileTask by serviceSubsetID
	reconcileTaskByServiceSubset map[serviceSubsetID]*instancesReconcileTask
	// protects reconcileTaskByServiceSubset
	reconcileTaskByServiceSubsetMutex sync.RWMutex

	log logr.Logger
}

type instancesReconcileRequest struct {
	service                 serviceSummary
	subset                  serviceSubset
	readyInstanceInfoByID   map[string]instanceInfo
	unreadyInstanceInfoByID map[string]instanceInfo
	resultChan              chan<- error
}

func (r *defaultInstancesReconcileReactor) Submit(ctx context.Context, service serviceSummary, subset serviceSubset, readyInstanceInfoByID map[string]instanceInfo, unreadyInstanceInfoByID map[string]instanceInfo) <-chan error {
	resultChan := make(chan error, 1)
	reconcileRequest := instancesReconcileRequest{
		service:                 service,
		subset:                  subset,
		readyInstanceInfoByID:   readyInstanceInfoByID,
		unreadyInstanceInfoByID: unreadyInstanceInfoByID,
		resultChan:              resultChan,
	}

	select {
	case <-ctx.Done():
		resultChan <- ctx.Err()
		close(resultChan)
	case r.reconcileRequestChan <- reconcileRequest:
	}

	return resultChan
}

func (r *defaultInstancesReconcileReactor) reactorLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case request := <-r.reconcileRequestChan:
			r.dispatch(ctx, request)
		}
	}
}

// dispatch cloudMap service subset reconcile task into existing task or new task.
// note: dispatch should run in a single goroutine.
func (r *defaultInstancesReconcileReactor) dispatch(ctx context.Context, request instancesReconcileRequest) {
	serviceSubsetID := serviceSubsetID{serviceID: request.service.serviceID, subsetID: request.subset.SubsetID()}
	r.reconcileTaskByServiceSubsetMutex.RLock()
	reconcileTask, ok := r.reconcileTaskByServiceSubset[serviceSubsetID]
	r.reconcileTaskByServiceSubsetMutex.RUnlock()

	if ok {
		select {
		case <-reconcileTask.done: // existing task already quited.
			r.dispatchToNewTask(ctx, serviceSubsetID, request)
		case reconcileTask.instancesReconcileRequestChan <- request: // existing task accepted our new request.
		}
	} else {
		r.dispatchToNewTask(ctx, serviceSubsetID, request)
	}
}

// dispatch new cloudMap service subset reconcile task to run on a new task.
func (r *defaultInstancesReconcileReactor) dispatchToNewTask(ctx context.Context, serviceSubsetID serviceSubsetID, request instancesReconcileRequest) {
	reconcileTask := newInstancesReconcileTask(r.cloudMapSDK, r.instancesCache, r.log, make(chan struct{}))
	r.reconcileTaskByServiceSubsetMutex.Lock()
	r.reconcileTaskByServiceSubset[serviceSubsetID] = reconcileTask
	r.reconcileTaskByServiceSubsetMutex.Unlock()

	go func() {
		reconcileTask.Run(ctx)
		r.reconcileTaskByServiceSubsetMutex.Lock()
		delete(r.reconcileTaskByServiceSubset, serviceSubsetID)
		r.reconcileTaskByServiceSubsetMutex.Unlock()
		close(reconcileTask.done)
	}()
	reconcileTask.instancesReconcileRequestChan <- request
}
