package virtualnode

import (
	"context"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/k8s"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
)

func NewEnqueueRequestsForPodEvents(k8sClient client.Client, log logr.Logger) *enqueueRequestsForPodEvents {
	return &enqueueRequestsForPodEvents{
		k8sClient: k8sClient,
		log:       log,
	}
}

var _ handler.EventHandler = (*enqueueRequestsForPodEvents)(nil)

type enqueueRequestsForPodEvents struct {
	k8sClient client.Client
	log       logr.Logger
}

// Create is called in response to an create event
func (h *enqueueRequestsForPodEvents) Create(e event.CreateEvent, queue workqueue.RateLimitingInterface) {
	h.enqueueVirtualNodesForPods(context.Background(), queue, e.Object.(*corev1.Pod))
}

// Update is called in response to an update event
func (h *enqueueRequestsForPodEvents) Update(e event.UpdateEvent, queue workqueue.RateLimitingInterface) {
	// cloudmap reconcile depends Virtualnode Pod Selector labels and if there is an update to a Pod
	// we need to check if there is any change w.r.t VirtualNode it belongs.
	h.enqueueVirtualNodesForPods(context.Background(), queue, e.ObjectNew.(*corev1.Pod))
}

// Delete is called in response to a delete event
func (h *enqueueRequestsForPodEvents) Delete(e event.DeleteEvent, queue workqueue.RateLimitingInterface) {
	//On a VirtualNode delete, we need to clean up corresponding CloudMap Service along with
	//deregistering all the service instances from CloudMap.
	h.enqueueVirtualNodesForPods(context.Background(), queue, e.Object.(*corev1.Pod))
}

// Generic is called in response to an event of an unknown type or a synthetic event triggered as a cron or
// external trigger request
func (h *enqueueRequestsForPodEvents) Generic(e event.GenericEvent, queue workqueue.RateLimitingInterface) {
	// no-op
}

func (h *enqueueRequestsForPodEvents) enqueueVirtualNodesForPods(ctx context.Context, queue workqueue.RateLimitingInterface,
	pod *corev1.Pod) {
	vnList := &appmesh.VirtualNodeList{}
	if err := h.k8sClient.List(ctx, vnList); err != nil {
		h.log.Error(err, "failed to enqueue virtualNodes for pod events",
			"Pod", k8s.NamespacedName(pod))
		return
	}

	for _, vn := range vnList.Items {
		if vn.Spec.PodSelector == nil {
			return
		}
		selector, _ := metav1.LabelSelectorAsSelector(vn.Spec.PodSelector)
		if selector.Matches(labels.Set(pod.Labels)) {
			queue.Add(ctrl.Request{NamespacedName: k8s.NamespacedName(&vn)})
		}
	}
}
