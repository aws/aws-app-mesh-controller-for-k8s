package cloudmap

import (
	"context"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
)

type VirtualNodeEndpointResolver interface {
	ResolveCloudMapEndPoints(ctx context.Context,
		vn *appmesh.VirtualNode) (corev1.PodList, corev1.PodList, corev1.PodList, error)
}

type EndpointResolver struct {
	k8sClient client.Client
	log       logr.Logger
}

func NewEndPointResolver(k8sClient client.Client, log logr.Logger) *EndpointResolver {
	return &EndpointResolver{
		k8sClient: k8sClient,
		log:       log,
	}
}

func (e *EndpointResolver) ResolveCloudMapEndPoints(ctx context.Context,
	vNode *appmesh.VirtualNode) (corev1.PodList, corev1.PodList, corev1.PodList, error) {
	var readyPods corev1.PodList
	var notReadyPods corev1.PodList
	var ignoredPods corev1.PodList

	var podsList corev1.PodList
	var listOptions client.ListOptions
	listOptions.LabelSelector, _ = metav1.LabelSelectorAsSelector(vNode.Spec.PodSelector)
	listOptions.Namespace = vNode.Namespace

	if err := e.k8sClient.List(ctx, &podsList, &listOptions); err != nil {
		e.log.Error(err, "Couldn't retrieve pods for VirtualNode")
		return readyPods, notReadyPods, ignoredPods, err
	}

	for _, pod := range podsList.Items {
		if pod.DeletionTimestamp != nil {
			e.log.Info("vNode:", "Pod is being deleted: ", pod.Name)
			ignoredPods.Items = append(ignoredPods.Items, pod)
			continue
		}

		if pod.Status.PodIP == "" {
			ignoredPods.Items = append(ignoredPods.Items, pod)
			e.log.Info("No IP Address assigned to Pod:", pod.Name, "..Skipping for now")
			continue
		}

		if ArePodContainersReady(&pod) {
			readyPods.Items = append(readyPods.Items, pod)
		} else if ShouldPodBeInEndpoints(&pod) {
			notReadyPods.Items = append(notReadyPods.Items, pod)
		} else {
			continue
		}
	}
	return readyPods, notReadyPods, ignoredPods, nil
}
