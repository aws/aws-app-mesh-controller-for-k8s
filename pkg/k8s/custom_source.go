package k8s

import (
	"fmt"
	"sync"

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

type EventType int

const (
	CREATE EventType = 1
	DELETE EventType = 2
	UPDATE EventType = 3
)

// NotificationChannel monitors channels of type Create/Update/Delete
type NotificationChannel struct {
	// once ensures the event distribution goroutine will be performed only once
	once sync.Once

	// stop is to end ongoing goroutine, and close the Create channel
	stop <-chan struct{}

	EventType

	Create <-chan event.CreateEvent

	Delete <-chan event.DeleteEvent

	Update <-chan event.UpdateEvent

	// dest is the destination channels of the Create event handlers
	destCreate []chan event.CreateEvent

	// dest is the destination channels of the Delete event handlers
	destDelete []chan event.DeleteEvent

	// dest is the destination channels of the Update event handlers
	destUpdate []chan event.UpdateEvent

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
	_ = fmt.Errorf("Event Type: %v", cs.EventType)

	// Source should have been specified by the user.
	if cs.EventType == CREATE && cs.Create == nil {
		return fmt.Errorf("must specify either Create, Delete or Update Channel")
	} else if cs.EventType == DELETE && cs.Delete == nil {
		return fmt.Errorf("must specify either Create, Delete or Update Channel")
	} else if cs.EventType == UPDATE && cs.Update == nil {
		return fmt.Errorf("must specify either Create, Delete or Update Channel")
	} else if cs.EventType != CREATE && cs.EventType != DELETE && cs.EventType != UPDATE {
		return fmt.Errorf("Invalid Type: %v", cs.EventType)
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
		// Distribute Events to corresponding EventHandler / Queue pairs Watching this source
		if cs.EventType == CREATE {
			go cs.syncLoopCreate()
		} else if cs.EventType == DELETE {
			go cs.syncLoopDelete()
		} else if cs.EventType == UPDATE {
			go cs.syncLoopUpdate()
		} else {
			_ = fmt.Errorf("Invalid Event Type %v", cs.EventType)
			return
		}
	})

	if cs.EventType == CREATE {
		dstCreate := make(chan event.CreateEvent, cs.DestBufferSize)

		go func() {
			for evt := range dstCreate {
				handler.Create(evt, queue)
			}
		}()

		cs.destLock.Lock()
		defer cs.destLock.Unlock()
		cs.destCreate = append(cs.destCreate, dstCreate)
	} else if cs.EventType == DELETE {
		dstDelete := make(chan event.DeleteEvent, cs.DestBufferSize)

		go func() {
			for evt := range dstDelete {
				handler.Delete(evt, queue)
			}
		}()

		cs.destLock.Lock()
		defer cs.destLock.Unlock()

		cs.destDelete = append(cs.destDelete, dstDelete)
	} else if cs.EventType == UPDATE {
		dstUpdate := make(chan event.UpdateEvent, cs.DestBufferSize)

		go func() {
			for evt := range dstUpdate {
				handler.Update(evt, queue)
			}
		}()

		cs.destLock.Lock()
		defer cs.destLock.Unlock()

		cs.destUpdate = append(cs.destUpdate, dstUpdate)
	}
	return nil
}

func (cs *NotificationChannel) doStopCreate() {
	cs.destLock.Lock()
	defer cs.destLock.Unlock()

	for _, dst := range cs.destCreate {
		close(dst)
	}
}

func (cs *NotificationChannel) doStopDelete() {
	cs.destLock.Lock()
	defer cs.destLock.Unlock()

	for _, dst := range cs.destDelete {
		close(dst)
	}
}

func (cs *NotificationChannel) doStopUpdate() {
	cs.destLock.Lock()
	defer cs.destLock.Unlock()

	for _, dst := range cs.destUpdate {
		close(dst)
	}
}

func (cs *NotificationChannel) distributeCreate(evt event.CreateEvent) {
	cs.destLock.Lock()
	defer cs.destLock.Unlock()

	for _, dst := range cs.destCreate {
		// We cannot make it under goroutine here, or we'll meet the
		// race condition of writing message to closed channels.
		// To avoid blocking, the dest channels are expected to be of
		// proper buffer size. If we still see it blocked, then
		// the controller is thought to be in an abnormal state.
		dst <- evt
	}
}

func (cs *NotificationChannel) distributeDelete(evt event.DeleteEvent) {
	cs.destLock.Lock()
	defer cs.destLock.Unlock()

	for _, dst := range cs.destDelete {
		// We cannot make it under goroutine here, or we'll meet the
		// race condition of writing message to closed channels.
		// To avoid blocking, the dest channels are expected to be of
		// proper buffer size. If we still see it blocked, then
		// the controller is thought to be in an abnormal state.
		dst <- evt
	}
}

func (cs *NotificationChannel) distributeUpdate(evt event.UpdateEvent) {
	cs.destLock.Lock()
	defer cs.destLock.Unlock()

	for _, dst := range cs.destUpdate {
		// We cannot make it under goroutine here, or we'll meet the
		// race condition of writing message to closed channels.
		// To avoid blocking, the dest channels are expected to be of
		// proper buffer size. If we still see it blocked, then
		// the controller is thought to be in an abnormal state.
		dst <- evt
	}
}

func (cs *NotificationChannel) syncLoopCreate() {
	for {
		select {
		case <-cs.stop:
			cs.doStopCreate()
			return
		case evt := <-cs.Create:
			cs.distributeCreate(evt)
		}
	}
}

func (cs *NotificationChannel) syncLoopDelete() {
	for {
		select {
		case <-cs.stop:
			cs.doStopDelete()
			return
		case evt := <-cs.Delete:
			cs.distributeDelete(evt)
		}
	}
}

func (cs *NotificationChannel) syncLoopUpdate() {
	for {
		select {
		case <-cs.stop:
			cs.doStopUpdate()
			return
		case evt := <-cs.Update:
			cs.distributeUpdate(evt)
		}
	}
}
