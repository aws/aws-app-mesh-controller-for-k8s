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

func NewEnqueueRequestsForBackendGroupEvents(client client.Client, log logr.Logger) *enqueueRequestsForBackendGroupEvents {
	return &enqueueRequestsForBackendGroupEvents{
		k8sClient: client,
		log:       log,
	}
}

var _ handler.EventHandler = (*enqueueRequestsForBackendGroupEvents)(nil)

type enqueueRequestsForBackendGroupEvents struct {
	k8sClient client.Client
	log       logr.Logger
}

// Create is called in response to a create event
func (h *enqueueRequestsForBackendGroupEvents) Create(ctx context.Context, e event.CreateEvent, queue workqueue.TypedRateLimitingInterface[ctrl.Request]) {
	bg := e.Object.(*appmesh.BackendGroup)
	h.enqueueVirtualNodesForMesh(ctx, queue, bg.Spec.MeshRef, bg)
}

// Update is called in response to an update event
func (h *enqueueRequestsForBackendGroupEvents) Update(ctx context.Context, e event.UpdateEvent, queue workqueue.TypedRateLimitingInterface[ctrl.Request]) {
	bgOld := e.ObjectOld.(*appmesh.BackendGroup)
	bgNew := e.ObjectNew.(*appmesh.BackendGroup)
	if !reflect.DeepEqual(bgOld.Spec.VirtualServices, bgNew.Spec.VirtualServices) {
		h.enqueueVirtualNodesForMesh(ctx, queue, bgNew.Spec.MeshRef, bgNew)
	}
}

// Delete is called in response to a delete event
func (h *enqueueRequestsForBackendGroupEvents) Delete(ctx context.Context, e event.DeleteEvent, queue workqueue.TypedRateLimitingInterface[ctrl.Request]) {
	// no-op
}

// Generic is called in response to an event of an unknown type or a synthetic event triggered as a cron or
// external trigger request
func (h *enqueueRequestsForBackendGroupEvents) Generic(ctx context.Context, e event.GenericEvent, queue workqueue.TypedRateLimitingInterface[ctrl.Request]) {
	// no-op
}

func (h *enqueueRequestsForBackendGroupEvents) enqueueVirtualNodesForMesh(ctx context.Context, queue workqueue.TypedRateLimitingInterface[ctrl.Request], meshRef *appmesh.MeshReference, backendGroup *appmesh.BackendGroup) {
	vnList := &appmesh.VirtualNodeList{}
	if err := h.k8sClient.List(ctx, vnList); err != nil {
		h.log.Error(err, "failed to enqueue virtualNodes for backend group events", "mesh", meshRef.Name)
		return
	}
	for _, vn := range vnList.Items {
		if vn.Spec.MeshRef == nil || *meshRef != *vn.Spec.MeshRef {
			continue
		}
		for _, bg := range vn.Spec.BackendGroups {
			if bg.Name == backendGroup.Name {
				if bg.Namespace != nil {
					if *bg.Namespace == backendGroup.Namespace {
						queue.Add(ctrl.Request{NamespacedName: k8s.NamespacedName(&vn)})
					}
				} else {
					if backendGroup.Namespace == vn.Namespace {
						queue.Add(ctrl.Request{NamespacedName: k8s.NamespacedName(&vn)})
					}
				}
			}
		}
	}
}
