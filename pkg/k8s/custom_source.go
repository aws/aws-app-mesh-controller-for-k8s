package k8s

import (
	"fmt"
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
)

const (
	// defaultBufferSize is the default number of event notifications that can be buffered.
	defaultBufferSize = 1024
)

// Source is a source of events
type Source interface {
	// Start is internal and should be called only by the Controller to register an EventHandler with the Informer
	// to enqueue reconcile.Requests.
	Start(handler.EventHandler, workqueue.RateLimitingInterface, ...predicate.Predicate) error
}

var _ Source = &NotificationChannel{}

// EventType identifies type of PodEvent (CREATE/UPDATE/DELETE)
type EventType int

const (
	CREATE EventType = 1
	DELETE EventType = 2
	UPDATE EventType = 3
)

// PodEvent is a wrapper for Create/Update/Delete Events
type PodEvent struct {
	EventType

	// Meta is the ObjectMeta of the Kubernetes Type that was created
	Meta metav1.Object

	// Object is the object from the event
	Object runtime.Object

	// MetaOld is the ObjectMeta of the Kubernetes Type that was updated (before the update)
	MetaOld metav1.Object

	// ObjectOld is the object from the event
	ObjectOld runtime.Object

	// MetaNew is the ObjectMeta of the Kubernetes Type that was updated (after the update)
	MetaNew metav1.Object

	// ObjectNew is the object from the event
	ObjectNew runtime.Object
}

// NotificationChannel monitors channels of type Create/Update/Delete
type NotificationChannel struct {
	// once ensures the event distribution goroutine will be performed only once
	once sync.Once

	// stop is to end ongoing goroutine, and close the Create channel
	stop <-chan struct{}

	Source <-chan PodEvent

	// dest is the destination channels of the Pod event handlers
	dest []chan PodEvent

	// DestBufferSize is the specified buffer size of dest channels.
	// Default to 1024 if not specified.
	DestBufferSize int

	// destLock is to ensure the destination channels are safely added/removed
	destLock sync.Mutex
}

var _ inject.Stoppable = &NotificationChannel{}

// InjectStopChannel is internal should be called only by the Controller.
// It is used to inject the stop channel initialized by the ControllerManager.
func (cs *NotificationChannel) InjectStopChannel(stop <-chan struct{}) error {
	if cs.stop == nil {
		cs.stop = stop
	}

	return nil
}

func (cs *NotificationChannel) String() string {
	return fmt.Sprintf("channel source: %p", cs)
}

// Start implements Source and should only be called by the Controller.
func (cs *NotificationChannel) Start(
	handler handler.EventHandler,
	queue workqueue.RateLimitingInterface,
	prct ...predicate.Predicate) error {
	// Source should have been specified by the user.
	if cs.Source == nil {
		return fmt.Errorf("must specify CustomChannle.Source")
	}

	// stop should have been injected before Start was called
	if cs.stop == nil {
		return fmt.Errorf("must call InjectStop on Channel before calling Start")
	}

	// use default value if DestBufferSize not specified
	if cs.DestBufferSize == 0 {
		cs.DestBufferSize = defaultBufferSize
	}

	cs.once.Do(func() {
		// Distribute GenericEvents to all EventHandler / Queue pairs Watching this source
		go cs.syncLoop()
	})

	dst := make(chan PodEvent, cs.DestBufferSize)
	go func() {
		for evt := range dst {
			switch evt.EventType {
			case CREATE:
				handler.Create(event.CreateEvent{Meta: evt.Meta, Object: evt.Object}, queue)
			case DELETE:
				handler.Delete(event.DeleteEvent{Meta: evt.Meta, Object: evt.Object}, queue)
			case UPDATE:
				handler.Update(event.UpdateEvent{MetaOld: evt.MetaOld, ObjectOld: evt.ObjectOld, MetaNew: evt.MetaNew, ObjectNew: evt.ObjectNew}, queue)
			default:
				_ = fmt.Errorf("Invalid Type %T", evt.EventType)
			}
		}
	}()

	cs.destLock.Lock()
	defer cs.destLock.Unlock()

	cs.dest = append(cs.dest, dst)

	return nil
}

func (cs *NotificationChannel) doStop() {
	cs.destLock.Lock()
	defer cs.destLock.Unlock()

	for _, dst := range cs.dest {
		close(dst)
	}
}

func (cs *NotificationChannel) distribute(evt PodEvent) {
	cs.destLock.Lock()
	defer cs.destLock.Unlock()

	for _, dst := range cs.dest {
		// We cannot make it under goroutine here, or we'll meet the
		// race condition of writing message to closed channels.
		// To avoid blocking, the dest channels are expected to be of
		// proper buffer size. If we still see it blocked, then
		// the controller is thought to be in an abnormal state.
		dst <- evt
	}
}

func (cs *NotificationChannel) syncLoop() {
	for {
		select {
		case <-cs.stop:
			cs.doStop()
			return
		case evt := <-cs.Source:
			cs.distribute(evt)
		}
	}
}
