/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/cloudmap"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/k8s"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/runtime"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// CloudMapReconciler reconciles a VirtualNode pod instance to CloudMap Service
type cloudMapReconciler struct {
	k8sClient                   client.Client
	log                         logr.Logger
	finalizerManager            k8s.FinalizerManager
	cloudMapResourceManager     cloudmap.ResourceManager
	enqueueRequestsForPodEvents handler.EventHandler
}

func NewCloudMapReconciler(k8sClient client.Client, finalizerManager k8s.FinalizerManager,
	cloudMapResourceManager cloudmap.ResourceManager, log logr.Logger) *cloudMapReconciler {
	return &cloudMapReconciler{
		k8sClient:                   k8sClient,
		log:                         log,
		finalizerManager:            finalizerManager,
		cloudMapResourceManager:     cloudMapResourceManager,
		enqueueRequestsForPodEvents: cloudmap.NewEnqueueRequestsForPodEvents(k8sClient, log),
	}
}

// +kubebuilder:rbac:groups=appmesh.k8s.aws,resources=virtualnodes,verbs=get;list;watch
// +kubebuilder:rbac:groups=appmesh.k8s.aws,resources=virtualnodes/status,verbs=get
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=pods/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=nodes,verbs=get;list;watch

func (r *cloudMapReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	return runtime.HandleReconcileError(r.reconcile(req), r.log)
}

func (r *cloudMapReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("cloudMap").
		For(&appmesh.VirtualNode{}).
		Watches(&source.Kind{Type: &corev1.Pod{}}, r.enqueueRequestsForPodEvents).
		WithOptions(controller.Options{MaxConcurrentReconciles: 3}).
		Complete(r)
}

func (r *cloudMapReconciler) reconcile(req ctrl.Request) error {
	ctx := context.Background()

	vNode := &appmesh.VirtualNode{}
	if err := r.k8sClient.Get(ctx, req.NamespacedName, vNode); err != nil {
		return client.IgnoreNotFound(err)
	}

	if !vNode.DeletionTimestamp.IsZero() {
		return r.cleanupCloudMapResources(ctx, vNode)
	}
	return r.reconcileVirtualNodeWithCloudMap(ctx, vNode)
}

func (r *cloudMapReconciler) reconcileVirtualNodeWithCloudMap(ctx context.Context, vNode *appmesh.VirtualNode) error {
	if vNode.Spec.ServiceDiscovery == nil || vNode.Spec.ServiceDiscovery.AWSCloudMap == nil {
		return nil
	}
	if err := r.finalizerManager.AddFinalizers(ctx, vNode, k8s.FinalizerAWSCloudMapResources); err != nil {
		return err
	}
	if err := r.cloudMapResourceManager.Reconcile(ctx, vNode); err != nil {
		return err
	}
	return nil
}

func (r *cloudMapReconciler) cleanupCloudMapResources(ctx context.Context, vNode *appmesh.VirtualNode) error {
	if k8s.HasFinalizer(vNode, k8s.FinalizerAWSCloudMapResources) {
		if vNode.Spec.ServiceDiscovery != nil && vNode.Spec.ServiceDiscovery.AWSCloudMap != nil {
			if err := r.cloudMapResourceManager.Cleanup(ctx, vNode); err != nil {
				return err
			}
		}
		if err := r.finalizerManager.RemoveFinalizers(ctx, vNode, k8s.FinalizerAWSCloudMapResources); err != nil {
			return err
		}
	}
	return nil
}
