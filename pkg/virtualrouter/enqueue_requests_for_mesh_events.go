package virtualrouter

import (
	"context"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/k8s"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/mesh"
	"github.com/go-logr/logr"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
)

func NewEnqueueRequestsForMeshEvents(k8sClient client.Client, log logr.Logger) *enqueueRequestsForMeshEvents {
	return &enqueueRequestsForMeshEvents{
		k8sClient: k8sClient,
		log:       log,
	}
}

var _ handler.EventHandler = (*enqueueRequestsForMeshEvents)(nil)

type enqueueRequestsForMeshEvents struct {
	k8sClient client.Client
	log       logr.Logger
}

// Create is called in response to an create event
func (h *enqueueRequestsForMeshEvents) Create(e event.CreateEvent, queue workqueue.RateLimitingInterface) {
	// no-op
}

// Update is called in response to an update event
func (h *enqueueRequestsForMeshEvents) Update(e event.UpdateEvent, queue workqueue.RateLimitingInterface) {
	// virtualNode reconcile depends on mesh is active or not.
	// so we only need to trigger virtualNode reconcile if mesh's active status changed.
	msOld := e.ObjectOld.(*appmesh.Mesh)
	msNew := e.ObjectNew.(*appmesh.Mesh)

	if mesh.IsMeshActive(msOld) != mesh.IsMeshActive(msNew) {
		h.enqueueVirtualRoutersForMesh(context.Background(), queue, msNew)
	}
}

// Delete is called in response to a delete event
func (h *enqueueRequestsForMeshEvents) Delete(e event.DeleteEvent, queue workqueue.RateLimitingInterface) {
	// no-op
}

// Generic is called in response to an event of an unknown type or a synthetic event triggered as a cron or
// external trigger request
func (h *enqueueRequestsForMeshEvents) Generic(e event.GenericEvent, queue workqueue.RateLimitingInterface) {
	// no-op
}

func (h *enqueueRequestsForMeshEvents) enqueueVirtualRoutersForMesh(ctx context.Context, queue workqueue.RateLimitingInterface, ms *appmesh.Mesh) {
	vrList := &appmesh.VirtualRouterList{}
	if err := h.k8sClient.List(ctx, vrList); err != nil {
		h.log.Error(err, "failed to enqueue virtualRouters for mesh events",
			"mesh", k8s.NamespacedName(ms))
		return
	}
	for _, vr := range vrList.Items {
		if vr.Spec.MeshRef == nil || !mesh.IsMeshReferenced(ms, *vr.Spec.MeshRef) {
			continue
		}
		queue.Add(ctrl.Request{NamespacedName: k8s.NamespacedName(&vr)})
	}
}
