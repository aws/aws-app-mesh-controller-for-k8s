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
func (h *enqueueRequestsForBackendGroupEvents) Create(e event.CreateEvent, queue workqueue.RateLimitingInterface) {
	bg := e.Object.(*appmesh.BackendGroup)
	h.enqueueVirtualNodesForMesh(context.Background(), queue, bg.Spec.MeshRef, bg)
}

// Update is called in response to an update event
func (h *enqueueRequestsForBackendGroupEvents) Update(e event.UpdateEvent, queue workqueue.RateLimitingInterface) {
	bgOld := e.ObjectOld.(*appmesh.BackendGroup)
	bgNew := e.ObjectNew.(*appmesh.BackendGroup)
	if !reflect.DeepEqual(bgOld.Spec.VirtualServices, bgNew.Spec.VirtualServices) {
		h.enqueueVirtualNodesForMesh(context.Background(), queue, bgNew.Spec.MeshRef, bgNew)
	}
}

// Delete is called in response to a delete event
func (h *enqueueRequestsForBackendGroupEvents) Delete(e event.DeleteEvent, queue workqueue.RateLimitingInterface) {
	// no-op
}

// Generic is called in response to an event of an unknown type or a synthetic event triggered as a cron or
// external trigger request
func (h *enqueueRequestsForBackendGroupEvents) Generic(e event.GenericEvent, queue workqueue.RateLimitingInterface) {
	// no-op
}

func (h *enqueueRequestsForBackendGroupEvents) enqueueVirtualNodesForMesh(ctx context.Context, queue workqueue.RateLimitingInterface, meshRef *appmesh.MeshReference,
	backendGroup *appmesh.BackendGroup) {
	vnList := &appmesh.VirtualNodeList{}
	if err := h.k8sClient.List(ctx, vnList); err != nil {
		h.log.Error(err, "failed to enqueue virtualNodes for backend group events",
			"mesh", meshRef.Name)
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
