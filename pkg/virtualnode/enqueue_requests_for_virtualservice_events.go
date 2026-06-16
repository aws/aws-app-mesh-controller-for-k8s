package virtualnode

import (
	"context"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/k8s"
	"github.com/go-logr/logr"
	"k8s.io/client-go/util/workqueue"
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
func (h *enqueueRequestsForVirtualServiceEvents) Create(ctx context.Context, e event.CreateEvent, queue workqueue.TypedRateLimitingInterface[ctrl.Request]) {
	h.enqueueVirtualNodesForMesh(ctx, queue, e.Object.(*appmesh.VirtualService).Spec.MeshRef, e.Object.(*appmesh.VirtualService))
}

// Update is called in response to an update event
func (h *enqueueRequestsForVirtualServiceEvents) Update(ctx context.Context, e event.UpdateEvent, queue workqueue.TypedRateLimitingInterface[ctrl.Request]) {
	// no-op
}

// Delete is called in response to a delete event
func (h *enqueueRequestsForVirtualServiceEvents) Delete(ctx context.Context, e event.DeleteEvent, queue workqueue.TypedRateLimitingInterface[ctrl.Request]) {
	// no-op
}

// Generic is called in response to an event of an unknown type or a synthetic event triggered as a cron or
// external trigger request
func (h *enqueueRequestsForVirtualServiceEvents) Generic(ctx context.Context, e event.GenericEvent, queue workqueue.TypedRateLimitingInterface[ctrl.Request]) {
	// no-op
}

func (h *enqueueRequestsForVirtualServiceEvents) enqueueVirtualNodesForMesh(ctx context.Context, queue workqueue.TypedRateLimitingInterface[ctrl.Request], meshRef *appmesh.MeshReference, vs *appmesh.VirtualService) {
	vnList := &appmesh.VirtualNodeList{}
	if err := h.k8sClient.List(ctx, vnList); err != nil {
		h.log.Error(err, "failed to enqueue virtualNodes for virtual service events", "mesh", meshRef.Name)
		return
	}
	for _, vn := range vnList.Items {
		if vn.Spec.MeshRef == nil || *meshRef != *vn.Spec.MeshRef {
			continue
		}
		for _, bg := range vn.Spec.BackendGroups {
			if bg.Name == "*" {
				if bg.Namespace != nil {
					if *bg.Namespace == vs.Namespace {
						queue.Add(ctrl.Request{NamespacedName: k8s.NamespacedName(&vn)})
					}
				} else {
					if vs.Namespace == vn.Namespace {
						queue.Add(ctrl.Request{NamespacedName: k8s.NamespacedName(&vn)})
					}
				}
			}
		}
	}
}
