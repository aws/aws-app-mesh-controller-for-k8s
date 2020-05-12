package virtualservice

import (
	"context"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/k8s"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/references"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/virtualrouter"
	"github.com/go-logr/logr"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
)

func NewEnqueueRequestsForVirtualRouterEvents(referencesIndexer references.ObjectReferenceIndexer, log logr.Logger) *enqueueRequestsForVirtualRouterEvents {
	return &enqueueRequestsForVirtualRouterEvents{
		referencesIndexer: referencesIndexer,
		log:               log,
	}
}

var _ handler.EventHandler = (*enqueueRequestsForVirtualRouterEvents)(nil)

type enqueueRequestsForVirtualRouterEvents struct {
	referencesIndexer references.ObjectReferenceIndexer
	log               logr.Logger
}

// Create is called in response to an create event
func (h *enqueueRequestsForVirtualRouterEvents) Create(e event.CreateEvent, queue workqueue.RateLimitingInterface) {
	// no-op
}

// Update is called in response to an update event
func (h *enqueueRequestsForVirtualRouterEvents) Update(e event.UpdateEvent, queue workqueue.RateLimitingInterface) {
	// VirtualService reconcile depends on virtualRouter is active or not.
	// so we only need to trigger VirtualService reconcile if virtualRouter's active status changed.
	vrOld := e.ObjectOld.(*appmesh.VirtualRouter)
	vrNew := e.ObjectNew.(*appmesh.VirtualRouter)

	if virtualrouter.IsVirtualRouterActive(vrOld) != virtualrouter.IsVirtualRouterActive(vrNew) {
		h.enqueueVirtualServicesForVirtualRouter(context.Background(), queue, vrNew)
	}
}

// Delete is called in response to a delete event
func (h *enqueueRequestsForVirtualRouterEvents) Delete(e event.DeleteEvent, queue workqueue.RateLimitingInterface) {
	// no-op
}

// Generic is called in response to an event of an unknown type or a synthetic event triggered as a cron or
// external trigger request
func (h *enqueueRequestsForVirtualRouterEvents) Generic(e event.GenericEvent, queue workqueue.RateLimitingInterface) {
	// no-op
}

func (h *enqueueRequestsForVirtualRouterEvents) enqueueVirtualServicesForVirtualRouter(ctx context.Context, queue workqueue.RateLimitingInterface, vr *appmesh.VirtualRouter) {
	vsList := &appmesh.VirtualServiceList{}
	if err := h.referencesIndexer.Fetch(ctx, vsList, ReferenceKindVirtualRouter, k8s.NamespacedName(vr)); err != nil {
		h.log.Error(err, "failed to enqueue virtualServices for virtualRouter events",
			"virtualRouter", k8s.NamespacedName(vr))
		return
	}
	for _, vs := range vsList.Items {
		queue.Add(ctrl.Request{NamespacedName: k8s.NamespacedName(&vs)})
	}
}
