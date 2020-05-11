package core

import (
	"context"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/inject"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/webhook"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const apiPathMutatePod = "/mutate-v1-pod"

// NewPodMutator returns a mutator for Pod.
func NewPodMutator(injector *inject.SidecarInjector) *podMutator {
	return &podMutator{
		sidecarInjector: injector,
	}
}

var _ webhook.Mutator = &podMutator{}

type podMutator struct {
	sidecarInjector *inject.SidecarInjector
}

func (m *podMutator) Prototype(req admission.Request) (runtime.Object, error) {
	return &corev1.Pod{}, nil
}

func (m *podMutator) MutateCreate(ctx context.Context, obj runtime.Object) (runtime.Object, error) {
	pod := obj.(*corev1.Pod)
	if err := m.sidecarInjector.Inject(ctx, pod); err != nil {
		return nil, err
	}
	return pod, nil
}

func (m *podMutator) MutateUpdate(ctx context.Context, obj runtime.Object, oldObj runtime.Object) (runtime.Object, error) {
	return obj, nil
}

// +kubebuilder:webhook:path=/mutate-v1-pod,mutating=true,failurePolicy=ignore,groups="",resources=pods,verbs=create,versions=v1,name=mpod.appmesh.k8s.aws

func (m *podMutator) SetupWithManager(mgr ctrl.Manager) {
	mgr.GetWebhookServer().Register(apiPathMutatePod, webhook.MutatingWebhookForMutator(m))
}
