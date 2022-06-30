package virtualnode

import (
	"context"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/k8s"
	"github.com/go-logr/logr"
	"k8s.io/client-go/util/workqueue"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
)

func NewEnqueueRequestsForVirtualServiceEvents(client client.Client, log logr.Logger) *enqueueRequestsForVirtualServiceEvents {
	return &enqueueRequestsForVirtualServiceEvents{
		k8sClient: client,
		log:       log,
	}
}

var _ handler.EventHandler = (*enqueueRequestsForVirtualServiceEvents)(nil)

type enqueueRequestsForVirtualServiceEvents struct {
	k8sClient client.Client
	log       logr.Logger
}

// Create is called in response to a create event
func (h *enqueueRequestsForVirtualServiceEvents) Create(e event.CreateEvent, queue workqueue.RateLimitingInterface) {
	h.enqueueVirtualNodeReconciles(context.Background(), queue)
}

// Update is called in response to an update event
func (h *enqueueRequestsForVirtualServiceEvents) Update(e event.UpdateEvent, queue workqueue.RateLimitingInterface) {
	if reflect.DeepEqual(e.ObjectOld.GetLabels(), e.ObjectNew.GetLabels()) {
		h.enqueueVirtualNodeReconciles(context.Background(), queue)
	}
}

// Delete is called in response to a delete event
func (h *enqueueRequestsForVirtualServiceEvents) Delete(e event.DeleteEvent, queue workqueue.RateLimitingInterface) {
	// no-op
}

// Generic is called in response to an event of an unknown type or a synthetic event triggered as a cron or
// external trigger request
func (h *enqueueRequestsForVirtualServiceEvents) Generic(e event.GenericEvent, queue workqueue.RateLimitingInterface) {
	// no-op
}

func (h *enqueueRequestsForVirtualServiceEvents) enqueueVirtualNodeReconciles(ctx context.Context, queue workqueue.RateLimitingInterface) {
	vnList := &appmesh.VirtualNodeList{}
	if err := h.k8sClient.List(ctx, vnList); err != nil {
		h.log.Error(err, "failed to enqueue virtual node reconciles")
		return
	}
	for _, vn := range vnList.Items {
		queue.Add(ctrl.Request{NamespacedName: k8s.NamespacedName(&vn)})
	}
}
