package k8s

import (
	"context"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	FinalizerMeshMembers          = "finalizers.appmesh.k8s.aws/mesh-members"
	FinalizerAWSAppMeshResources  = "finalizers.appmesh.k8s.aws/aws-appmesh-resources"
	FinalizerAWSCloudMapResources = "finalizers.appmesh.k8s.aws/aws-cloudmap-resources"
)

type APIObject interface {
	metav1.Object
	runtime.Object
}

type FinalizerManager interface {
	AddFinalizers(ctx context.Context, obj APIObject, finalizers ...string) error
	RemoveFinalizers(ctx context.Context, obj APIObject, finalizers ...string) error
}

func NewDefaultFinalizerManager(k8sClient client.Client, log logr.Logger) FinalizerManager {
	return &defaultFinalizerManager{
		k8sClient: k8sClient,
		log:       log,
	}
}

type defaultFinalizerManager struct {
	k8sClient client.Client
	log       logr.Logger
}

func (m *defaultFinalizerManager) AddFinalizers(ctx context.Context, obj APIObject, finalizers ...string) error {
	oldObj := obj.DeepCopyObject()
	needsUpdate := false
	for _, finalizer := range finalizers {
		if !HasFinalizer(obj, finalizer) {
			controllerutil.AddFinalizer(obj, finalizer)
			needsUpdate = true
		}
	}
	if !needsUpdate {
		return nil
	}
	return m.k8sClient.Patch(ctx, obj, client.MergeFrom(oldObj))
}

func (m *defaultFinalizerManager) RemoveFinalizers(ctx context.Context, obj APIObject, finalizers ...string) error {
	oldObj := obj.DeepCopyObject()
	needsUpdate := false
	for _, finalizer := range finalizers {
		if HasFinalizer(obj, finalizer) {
			controllerutil.RemoveFinalizer(obj, finalizer)
			needsUpdate = true
		}
	}
	if !needsUpdate {
		return nil
	}
	return m.k8sClient.Patch(ctx, obj, client.MergeFrom(oldObj))
}

// HasFinalizer tests whether k8s object has specified finalizer
func HasFinalizer(obj metav1.Object, finalizer string) bool {
	f := obj.GetFinalizers()
	for _, e := range f {
		if e == finalizer {
			return true
		}
	}
	return false
}
