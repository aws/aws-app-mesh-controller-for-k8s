// Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

package k8s

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Converter for converting k8s object and object list used in watches and list operation
type Converter interface {
	// ConvertObject takes an object and returns the modified object which will be
	// stored in the data store
	ConvertObject(originalObj interface{}) (convertedObj interface{}, err error)
	// ConvertList takes an object and returns the modified list of objects which
	// will be returned to the Simple Pager function to aggregate the list pagination
	// response
	ConvertList(originalList interface{}) (convertedList interface{}, err error)
	// Resource returns the K8s resource name to list/watch
	Resource() string
	// ResourceType returns the k8s object to list/watch
	ResourceType() runtime.Object
}

// Controller Interface implemented by PodController
type Controller interface {
	// StartController starts the controller. Will block the calling routine
	StartController(dataStore cache.Indexer, stopChanel chan struct{})
	// GetDataStore returns the data store once it has synced with the API Server
	GetDataStore() cache.Indexer
}

// CustomController is an Informer which converts Pod Objects and notifies corresponding event handlers via Channels
type CustomController struct {
	// clientSet is the kubernetes client set
	clientSet *kubernetes.Clientset
	// pageLimit is the number of objects returned per page on a list operation
	pageLimit int64
	// namespace to list/watch for
	namespace string
	// converter is the converter implementation that converts the k8s
	// object before storing in the data store
	converter Converter
	// resyncPeriod how often to sync using list with the API Server
	resyncPeriod time.Duration
	// retryOnError whether item should be retried on error. Should remain false in usual use case
	retryOnError bool
	// queue is the Delta FIFO queue
	queue *cache.DeltaFIFO
	// podEventNotificationChan channel will be notified for all pod events
	eventNotificationChan chan<- GenericEvent

	// log for custom controller
	log logr.Logger
	// controller is the K8s Controller
	controller cache.Controller
	// dataStore with the converted k8s object. It should not be directly accessed and used with
	// the exposed APIs
	dataStore cache.Indexer
}

// NewCustomController returns a new podController object
func NewCustomController(clientSet *kubernetes.Clientset, pageLimit int64, namesspace string, converter Converter, resyncPeriod time.Duration,
	retryOnError bool, eventNotificationChan chan<- GenericEvent, log logr.Logger) *CustomController {
	c := &CustomController{
		clientSet:             clientSet,
		pageLimit:             pageLimit,
		namespace:             namesspace,
		converter:             converter,
		resyncPeriod:          resyncPeriod,
		retryOnError:          retryOnError,
		eventNotificationChan: eventNotificationChan,
		log:                   log,
	}
	c.dataStore = cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{NamespaceIndexKey: NamespaceKeyIndexFunc()})
	c.queue = cache.NewDeltaFIFO(cache.MetaNamespaceKeyFunc, c.dataStore)
	return c
}

// StartController starts the custom controller by doing a list and watch on the specified k8s
// resource. The controller would store the converted k8s object in the provided indexer. The
// stop channel should be notified to stop the controller
func (c *CustomController) StartController(ctx context.Context) {
	config := &cache.Config{
		Queue: c.queue,
		ListerWatcher: newListWatcher(c.clientSet.CoreV1().RESTClient(),
			c.converter.Resource(), c.namespace, c.pageLimit, c.converter, c.log),
		ObjectType:       c.converter.ResourceType(),
		FullResyncPeriod: c.resyncPeriod,
		RetryOnError:     c.retryOnError,
		Process: func(obj interface{}) error {
			// from oldest to newest
			for _, d := range obj.(cache.Deltas) {
				// Strip down the pod object and keep only the required details
				convertedObj, err := c.converter.ConvertObject(d.Object)
				if err != nil {
					return err
				}
				switch d.Type {
				case cache.Sync, cache.Added, cache.Updated:
					c.log.V(1).Info("Received Cache event", "event type", d.Type)
					if old, exists, err := c.dataStore.Get(convertedObj); err == nil && exists {
						c.log.V(1).Info("Update event", "pod Ip", convertedObj.(*v1.Pod).Status.PodIP)
						if err := c.dataStore.Update(convertedObj); err != nil {
							return err
						}
						if err := c.notifyChannelOnUpdate(old, convertedObj); err != nil {
							return err
						}
					} else if err == nil && !exists {
						c.log.V(1).Info("Add/Create event", "pod Ip", convertedObj.(*v1.Pod).Status.PodIP)
						if err := c.dataStore.Add(convertedObj); err != nil {
							return err
						}
						if err := c.notifyChannelOnCreate(convertedObj); err != nil {
							return err
						}
					} else {
						return err
					}
				case cache.Deleted:
					c.log.V(1).Info("Delete event", "pod Ip", convertedObj.(*v1.Pod).Status.PodIP)
					if err := c.dataStore.Delete(convertedObj); err != nil {
						return err
					}
					if err := c.notifyChannelOnDelete(convertedObj); err != nil {
						return err
					}
				}
			}
			return nil
		},
	}
	c.controller = cache.New(config)

	// Run the controller
	c.controller.Run(ctx.Done())
}

// GetDataStore returns the data store when it has successfully synced with API Server
func (c *CustomController) GetDataStore() cache.Indexer {
	// Custom data store, it should not be accessed directly as the cache could be out of sync
	// on startup. Must be accessed from the pod controller's data store instead
	// TODO: we should refactor this in the future, as this approach will make controllers to run without having pod synced.
	// (It thus blocks when pod information is accessed)
	for c.controller == nil || (!c.controller.HasSynced() && c.controller.LastSyncResourceVersion() == "") {
		c.log.Info("waiting for controller to sync")
		time.Sleep(time.Second * 5)
	}
	return c.dataStore
}

// newListWatcher returns a list watcher with a custom list function that converts the
// response for each page using the converter function and returns a general watcher
func newListWatcher(restClient cache.Getter, resource string, namespace string, limit int64,
	converter Converter, log logr.Logger) *cache.ListWatch {
	log.V(1).Info("Initializing List Watcher")
	listFunc := func(options metav1.ListOptions) (runtime.Object, error) {
		ctx := context.Background()

		list, err := restClient.Get().
			Namespace(namespace).
			Resource(resource).
			// This needs to be done because just setting the limit using option's
			// Limit is being overridden and the response is returned without pagination.
			VersionedParams(&metav1.ListOptions{
				Limit:    limit,
				Continue: options.Continue,
			}, metav1.ParameterCodec).
			Do(ctx).
			Get()

		printList(list, "Original List", log)
		if err != nil {
			return list, err
		}
		// Strip down the the list before passing the paginated response back to
		// the pager function
		convertedList, err := converter.ConvertList(list)

		printList(convertedList, "Converted List", log)
		return convertedList.(runtime.Object), err
	}

	// We don't need to modify the watcher, we will strip down the k8s object in the ProcessFunc
	// before storing the object in the data store.
	watchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		ctx := context.Background()
		options.Watch = true

		return restClient.Get().
			Namespace(namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec).
			Watch(ctx)
	}
	return &cache.ListWatch{ListFunc: listFunc, WatchFunc: watchFunc}
}

func printList(list interface{}, listType string, log logr.Logger) {
	podList, ok := list.(*v1.PodList)
	if !ok {
		log.V(1).Info("Error converting in printList function")
	}
	ip := make([]string, 0, len(podList.Items))
	for _, pod := range podList.Items {
		ip = append(ip, pod.Status.PodIP)
	}
	log.V(1).Info("List of PodIPs", listType, ip)
}

// notifyChannelOnCreate notifies the add event on the appropriate channel
func (c *CustomController) notifyChannelOnCreate(obj interface{}) error {
	c.eventNotificationChan <- GenericEvent{
		EventType: CREATE,
		Object:    obj.(controllerutil.Object),
	}
	return nil
}

// notifyChannelOnCreate notifies the add event on the appropriate channel
func (c *CustomController) notifyChannelOnUpdate(oldObj, newObj interface{}) error {
	c.eventNotificationChan <- GenericEvent{
		EventType: UPDATE,
		OldObject: oldObj.(controllerutil.Object),
		Object:    newObj.(controllerutil.Object),
	}
	return nil
}

// notifyChannelOnDelete notifies the delete event on the appropriate channel
func (c *CustomController) notifyChannelOnDelete(obj interface{}) error {
	c.eventNotificationChan <- GenericEvent{
		EventType: DELETE,
		OldObject: obj.(controllerutil.Object),
	}
	return nil
}
