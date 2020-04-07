package core

import (
	"context"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/virtualnode"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/webhook"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const apiPathMutatePod = "/mutate-v1-pod"

// NewPodMutator returns a mutator for Pod.
func NewPodMutator(vnMembershipDesignator virtualnode.MembershipDesignator) *podMutator {
	return &podMutator{
		vnMembershipDesignator: vnMembershipDesignator,
	}
}

var _ webhook.Mutator = &podMutator{}

type podMutator struct {
	vnMembershipDesignator virtualnode.MembershipDesignator
}

func (m *podMutator) Prototype(req admission.Request) (runtime.Object, error) {
	return &corev1.Pod{}, nil
}

func (m *podMutator) MutateCreate(ctx context.Context, obj runtime.Object) (runtime.Object, error) {
	pod := obj.(*corev1.Pod)
	vn, err := m.vnMembershipDesignator.Designate(ctx, pod)
	if err != nil {
		return nil, err
	}
	if vn == nil {
		return obj, nil
	}
	if err := m.injectAppMeshPatches(ctx, vn, pod); err != nil {
		return nil, err
	}
	return pod, nil
}

func (m *podMutator) MutateUpdate(ctx context.Context, obj runtime.Object, oldObj runtime.Object) (runtime.Object, error) {
	return obj, nil
}

func (m *podMutator) injectAppMeshPatches(ctx context.Context, vn *appmesh.VirtualNode, pod *corev1.Pod) error {
	// TODO: change this to real implementation of appMesh-Injector.
	annotations := map[string]string{"appmesh.k8s.aws/virtualNode": vn.Name}
	for k, v := range pod.Annotations {
		annotations[k] = v
	}
	pod.Annotations = annotations
	return nil
}

// +kubebuilder:webhook:path=/mutate-v1-pod,mutating=true,failurePolicy=ignore,groups="",resources=pods,verbs=create,versions=v1,name=mpod.appmesh.k8s.aws

func (m *podMutator) SetupWithManager(mgr ctrl.Manager) {
	mgr.GetWebhookServer().Register(apiPathMutatePod, webhook.MutatingWebhookForMutator(m))
}
