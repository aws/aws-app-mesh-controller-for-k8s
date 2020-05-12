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
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/k8s"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/references"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/runtime"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/virtualservice"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
)

// NewVirtualServiceReconciler constructs new virtualServiceReconciler
func NewVirtualServiceReconciler(k8sClient client.Client, finalizerManager k8s.FinalizerManager, referencesIndexer references.ObjectReferenceIndexer, vsResManager virtualservice.ResourceManager, log logr.Logger) *virtualServiceReconciler {
	return &virtualServiceReconciler{
		k8sClient:                             k8sClient,
		finalizerManager:                      finalizerManager,
		referencesIndexer:                     referencesIndexer,
		vsResManager:                          vsResManager,
		enqueueRequestsForMeshEvents:          virtualservice.NewEnqueueRequestsForMeshEvents(k8sClient, log),
		enqueueRequestsForVirtualNodeEvents:   virtualservice.NewEnqueueRequestsForVirtualNodeEvents(referencesIndexer, log),
		enqueueRequestsForVirtualRouterEvents: virtualservice.NewEnqueueRequestsForVirtualRouterEvents(referencesIndexer, log),
		log:                                   log,
	}
}

// virtualServiceReconciler reconciles a VirtualService object
type virtualServiceReconciler struct {
	k8sClient         client.Client
	finalizerManager  k8s.FinalizerManager
	referencesIndexer references.ObjectReferenceIndexer
	vsResManager      virtualservice.ResourceManager

	enqueueRequestsForMeshEvents          handler.EventHandler
	enqueueRequestsForVirtualNodeEvents   handler.EventHandler
	enqueueRequestsForVirtualRouterEvents handler.EventHandler
	log                                   logr.Logger
}

// +kubebuilder:rbac:groups=appmesh.k8s.aws,resources=virtualservices,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=appmesh.k8s.aws,resources=virtualservices/status,verbs=get;update;patch

func (r *virtualServiceReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	return runtime.HandleReconcileError(r.reconcile(req), r.log)
}

func (r *virtualServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := r.referencesIndexer.Setup(&appmesh.VirtualService{}, map[string]references.ObjectReferenceIndexFunc{
		virtualservice.ReferenceKindVirtualNode:   virtualservice.VirtualNodeReferenceIndexFunc,
		virtualservice.ReferenceKindVirtualRouter: virtualservice.VirtualRouterReferenceIndexFunc,
	}); err != nil {
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&appmesh.VirtualService{}).
		Watches(&source.Kind{Type: &appmesh.Mesh{}}, r.enqueueRequestsForMeshEvents).
		Watches(&source.Kind{Type: &appmesh.VirtualNode{}}, r.enqueueRequestsForVirtualNodeEvents).
		Watches(&source.Kind{Type: &appmesh.VirtualRouter{}}, r.enqueueRequestsForVirtualRouterEvents).
		Complete(r)
}

func (r *virtualServiceReconciler) reconcile(req ctrl.Request) error {
	ctx := context.Background()
	vs := &appmesh.VirtualService{}
	if err := r.k8sClient.Get(ctx, req.NamespacedName, vs); err != nil {
		return client.IgnoreNotFound(err)
	}
	if !vs.DeletionTimestamp.IsZero() {
		return r.cleanupVirtualService(ctx, vs)
	}
	return r.reconcileVirtualService(ctx, vs)
}

func (r *virtualServiceReconciler) reconcileVirtualService(ctx context.Context, vs *appmesh.VirtualService) error {
	if err := r.finalizerManager.AddFinalizers(ctx, vs, k8s.FinalizerAWSAppMeshResources); err != nil {
		return err
	}
	if err := r.vsResManager.Reconcile(ctx, vs); err != nil {
		return err
	}
	return nil
}

func (r *virtualServiceReconciler) cleanupVirtualService(ctx context.Context, vs *appmesh.VirtualService) error {
	if k8s.HasFinalizer(vs, k8s.FinalizerAWSAppMeshResources) {
		if err := r.vsResManager.Cleanup(ctx, vs); err != nil {
			return err
		}
		if err := r.finalizerManager.RemoveFinalizers(ctx, vs, k8s.FinalizerAWSAppMeshResources); err != nil {
			return err
		}
	}
	return nil
}
