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

package custom

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

const (
	// API Server QPS
	DefaultAPIServerQPS   = 10
	DefaultAPIServerBurst = 15
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

// Controller Interface implemented by NewController
type Controller interface {
	// StartController starts the controller. Will block the calling routine
	StartController(dataStore cache.Indexer, stopChanel chan struct{})
	// GetDataStore returns the data store once it has synced with the API Server
	GetDataStore() cache.Indexer
}

// NewController is an Informer which converts Pod Objects and notifies corresponding even handlers via Channels
type NewController struct {
	// ClientSet is the kubernetes client set
	ClientSet *kubernetes.Clientset
	// PageLimit is the number of objects returned per page on a list operation
	PageLimit int64
	// Namespace to list/watch for
	Namespace string
	// Converter is the converter implementation that converts the k8s
	// object before storing in the data store
	Converter Converter
	// ResyncPeriod how often to sync using list with the API Server
	ResyncPeriod time.Duration
	// RetryOnError weather item should be retried on error. Should remain false in usual use case
	RetryOnError bool
	// Queue is the Delta FIFO queue
	Queue *cache.DeltaFIFO
	// CreateEventNotificationChan channel will be notified for all create
	// events for the k8s resource. If we don't want memory usage spikes we should
	// process the events as soon as soon as the channel is notified.
	CreateEventNotificationChan chan event.CreateEvent
	// UpdateEventNotificationChan channel will be notified for all update
	// events for the k8s resource. If we don't want memory usage spikes we should
	// process the events as soon as soon as the channel is notified.
	UpdateEventNotificationChan chan event.UpdateEvent
	// DeleteEventNotificationChan channel will be notified for all delete events for the
	// k8s resource. If we don't want memory usage spikes we should process the events as
	// soon as soon as the channel is notified.
	DeleteEventNotificationChan chan event.DeleteEvent
	// Log for custom controller
	Log logr.Logger
	// Controller is the K8s Controller
	Controller cache.Controller
	// dataStore with the converted k8s object. It should not be directly accessed and used with
	// the exposed APIs
	dataStore cache.Indexer
}

// StartController starts the custom controller by doing a list and watch on the specified k8s
// resource. The controller would store the converted k8s object in the provided indexer. The
// stop channel should be notified to stop the controller
func (c *NewController) StartController(dataStore cache.Indexer, stopChanel chan struct{}) {
	c.dataStore = dataStore

	config := &cache.Config{
		Queue: c.Queue,
		ListerWatcher: newListWatcher(c.ClientSet.CoreV1().RESTClient(),
			c.Converter.Resource(), c.Namespace, c.PageLimit, c.Converter),
		ObjectType:       c.Converter.ResourceType(),
		FullResyncPeriod: c.ResyncPeriod,
		RetryOnError:     c.RetryOnError,
		Process: func(obj interface{}) error {
			// from oldest to newest
			for _, d := range obj.(cache.Deltas) {
				// Strip down the pod object and keep only the required details
				convertedObj, err := c.Converter.ConvertObject(d.Object)
				if err != nil {
					return err
				}
				switch d.Type {
				case cache.Sync, cache.Added, cache.Updated:
					if old, exists, err := c.dataStore.Get(convertedObj); err == nil && exists {
						if err := c.dataStore.Update(convertedObj); err != nil {
							return err
						}
						c.notifyChannelOnUpdate(old, convertedObj)
					} else {
						if err := c.dataStore.Add(convertedObj); err != nil {
							return err
						}
						c.notifyChannelOnCreate(convertedObj)
					}

					if err != nil {
						return err
					}

				case cache.Deleted:
					if err := c.dataStore.Delete(convertedObj); err != nil {
						return err
					}
					c.notifyChannelOnDelete(convertedObj)
				}
			}
			return nil
		},
	}

	c.Log.Info("starting custom controller")
	defer close(stopChanel)
	c.Controller = cache.New(config)

	// Run the controller
	c.Controller.Run(stopChanel)
}

// GetDataStore returns the data store when it has successfully synced with API Server
func (c *NewController) GetDataStore() cache.Indexer {
	for c.Controller == nil || (!c.Controller.HasSynced() && c.Controller.LastSyncResourceVersion() == "") {
		c.Log.Info("waiting for controller to sync")
		time.Sleep(time.Second * 5)
	}
	return c.dataStore
}

// newListWatcher returns a list watcher with a custom list function that converts the
// response for each page using the converter function and returns a general watcher
func newListWatcher(restClient cache.Getter, resource string, namespace string, limit int64,
	converter Converter) *cache.ListWatch {

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

		if err != nil {
			return list, err
		}
		// Strip down the the list before passing the paginated response back to
		// the pager function
		convertedList, err := converter.ConvertList(list)
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

// notifyChannelOnCreate notifies the add event on the appropriate channel
func (c *NewController) notifyChannelOnCreate(obj interface{}) error {
	meta, err := apimeta.Accessor(obj)
	if err != nil {
		return err
	}
	c.CreateEventNotificationChan <- event.CreateEvent{
		Meta:   meta,
		Object: obj.(runtime.Object),
	}
	return nil
}

// notifyChannelOnCreate notifies the add event on the appropriate channel
func (c *NewController) notifyChannelOnUpdate(oldObj, newObj interface{}) error {
	metaOld, err := apimeta.Accessor(oldObj)
	if err != nil {
		return err
	}

	metaNew, err := apimeta.Accessor(newObj)
	if err != nil {
		return err
	}

	c.UpdateEventNotificationChan <- event.UpdateEvent{
		MetaOld:   metaOld,
		ObjectOld: oldObj.(runtime.Object),
		MetaNew:   metaNew,
		ObjectNew: newObj.(runtime.Object),
	}
	return nil
}

// notifyChannelOnDelete notifies the delete event on the appropriate channel
func (c *NewController) notifyChannelOnDelete(obj interface{}) error {
	meta, err := apimeta.Accessor(obj)
	if err != nil {
		return err
	}
	c.DeleteEventNotificationChan <- event.DeleteEvent{
		Meta:   meta,
		Object: obj.(runtime.Object),
	}
	return nil
}
