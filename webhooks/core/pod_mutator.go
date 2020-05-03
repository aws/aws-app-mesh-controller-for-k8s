package core

import (
	"context"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	appmeshinject "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/inject"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/references"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/virtualnode"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/webhook"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const apiPathMutatePod = "/mutate-v1-pod"

// NewPodMutator returns a mutator for Pod.
func NewPodMutator(referenceResolver references.Resolver, vnMembershipDesignator virtualnode.MembershipDesignator, injector *appmeshinject.SidecarInjector) *podMutator {
	return &podMutator{
		referenceResolver:      referenceResolver,
		vnMembershipDesignator: vnMembershipDesignator,
		sidecarInjector:        injector,
	}
}

var _ webhook.Mutator = &podMutator{}

type podMutator struct {
	referenceResolver      references.Resolver
	vnMembershipDesignator virtualnode.MembershipDesignator
	sidecarInjector        *appmeshinject.SidecarInjector
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
	if vn == nil || vn.Spec.MeshRef == nil {
		return obj, nil
	}
	ms, err := m.referenceResolver.ResolveMeshReference(ctx, *vn.Spec.MeshRef)
	if err != nil {
		return nil, err
	}
	if err := m.injectAppMeshPatches(ctx, ms, vn, pod); err != nil {
		return nil, err
	}
	return pod, nil
}

func (m *podMutator) MutateUpdate(ctx context.Context, obj runtime.Object, oldObj runtime.Object) (runtime.Object, error) {
	return obj, nil
}

func (m *podMutator) injectAppMeshPatches(ctx context.Context, ms *appmesh.Mesh, vn *appmesh.VirtualNode, pod *corev1.Pod) error {
	return m.sidecarInjector.InjectAppMeshPatches(ms, vn, pod)
}

// +kubebuilder:webhook:path=/mutate-v1-pod,mutating=true,failurePolicy=ignore,groups="",resources=pods,verbs=create,versions=v1,name=mpod.appmesh.k8s.aws

func (m *podMutator) SetupWithManager(mgr ctrl.Manager) {
	mgr.GetWebhookServer().Register(apiPathMutatePod, webhook.MutatingWebhookForMutator(m))
}
