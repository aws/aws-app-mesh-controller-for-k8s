package cloudmap

import (
	"context"

	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/k8s"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type VirtualNodeEndpointResolver interface {
	Resolve(ctx context.Context, vn *appmesh.VirtualNode) ([]*corev1.Pod, []*corev1.Pod, []*corev1.Pod, error)
}

func NewDefaultVirtualNodeEndpointResolver(PodsWrapper k8s.PodsWrapper, log logr.Logger) *defaultVirtualNodeEndpointResolver {
	return &defaultVirtualNodeEndpointResolver{
		PodsWrapper: PodsWrapper,
		log:         log,
	}
}

var _ VirtualNodeEndpointResolver = &defaultVirtualNodeEndpointResolver{}

type defaultVirtualNodeEndpointResolver struct {
	PodsWrapper k8s.PodsWrapper
	log         logr.Logger
}

func (e *defaultVirtualNodeEndpointResolver) Resolve(ctx context.Context, vNode *appmesh.VirtualNode) ([]*corev1.Pod, []*corev1.Pod, []*corev1.Pod, error) {
	var podsList *corev1.PodList
	var err error
	var listOptions client.ListOptions
	listOptions.LabelSelector, _ = metav1.LabelSelectorAsSelector(vNode.Spec.PodSelector)
	listOptions.Namespace = vNode.Namespace

	if podsList, err = e.PodsWrapper.ListPodsWithMatchingLabels(listOptions); err != nil {
		return nil, nil, nil, err
	}

	var readyPods []*corev1.Pod
	var notReadyPods []*corev1.Pod
	var ignoredPods []*corev1.Pod
	for i := range podsList.Items {
		pod := &podsList.Items[i]
		if !pod.DeletionTimestamp.IsZero() {
			ignoredPods = append(ignoredPods, pod)
			continue
		}

		if pod.Status.PodIP == "" {
			ignoredPods = append(ignoredPods, pod)
			continue
		}

		if ArePodContainersReady(pod) {
			readyPods = append(readyPods, pod)
		} else if ShouldPodBeInEndpoints(pod) {
			notReadyPods = append(notReadyPods, pod)
		} else {
			ignoredPods = append(ignoredPods, pod)
		}
	}
	return readyPods, notReadyPods, ignoredPods, nil
}
