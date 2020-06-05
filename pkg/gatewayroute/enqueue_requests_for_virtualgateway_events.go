package gatewayroute

import (
	"context"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/k8s"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/virtualgateway"
	"github.com/go-logr/logr"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
)

func NewEnqueueRequestsForVirtualGatewayEvents(k8sClient client.Client, log logr.Logger) *enqueueRequestsForVirtualGatewayEvents {
	return &enqueueRequestsForVirtualGatewayEvents{
		k8sClient: k8sClient,
		log:       log,
	}
}

var _ handler.EventHandler = (*enqueueRequestsForVirtualGatewayEvents)(nil)

type enqueueRequestsForVirtualGatewayEvents struct {
	k8sClient client.Client
	log       logr.Logger
}

// Create is called in response to an create event
func (h *enqueueRequestsForVirtualGatewayEvents) Create(e event.CreateEvent, queue workqueue.RateLimitingInterface) {
	// no-op
}

// Update is called in response to an update event
func (h *enqueueRequestsForVirtualGatewayEvents) Update(e event.UpdateEvent, queue workqueue.RateLimitingInterface) {
	// gatewayRoute reconcile depends on virtualGateway is active or not.
	// so we only need to trigger gatewayRoute reconcile if virtualGateway's active status changed.
	vgOld := e.ObjectOld.(*appmesh.VirtualGateway)
	vgNew := e.ObjectNew.(*appmesh.VirtualGateway)

	if virtualgateway.IsVirtualGatewayActive(vgOld) != virtualgateway.IsVirtualGatewayActive(vgNew) {
		h.enqueueGatewayRoutesForVirtualGateway(context.Background(), queue, vgNew)
	}
}

// Delete is called in response to a delete event
func (h *enqueueRequestsForVirtualGatewayEvents) Delete(e event.DeleteEvent, queue workqueue.RateLimitingInterface) {
	// no-op
}

// Generic is called in response to an event of an unknown type or a synthetic event triggered as a cron or
// external trigger request
func (h *enqueueRequestsForVirtualGatewayEvents) Generic(e event.GenericEvent, queue workqueue.RateLimitingInterface) {
	// no-op
}

func (h *enqueueRequestsForVirtualGatewayEvents) enqueueGatewayRoutesForVirtualGateway(ctx context.Context, queue workqueue.RateLimitingInterface, vg *appmesh.VirtualGateway) {
	grList := &appmesh.GatewayRouteList{}
	if err := h.k8sClient.List(ctx, grList); err != nil {
		h.log.Error(err, "failed to enqueue gatewayRoutes for virtualGateway events",
			"virtualGateway", k8s.NamespacedName(vg))
		return
	}
	for _, gr := range grList.Items {
		if gr.Spec.VirtualGatewayRef == nil || !virtualgateway.IsVirtualGatewayReferenced(vg, *gr.Spec.VirtualGatewayRef) {
			continue
		}
		queue.Add(ctrl.Request{NamespacedName: k8s.NamespacedName(&gr)})
	}
}
