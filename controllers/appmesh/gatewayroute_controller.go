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

	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/gatewayroute"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/k8s"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/runtime"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
)

// NewGatewayRouteReconciler constructs new gatewayRouteReconciler
func NewGatewayRouteReconciler(
	k8sClient client.Client,
	finalizerManager k8s.FinalizerManager,
	grResManager gatewayroute.ResourceManager,
	log logr.Logger,
	recorder record.EventRecorder) *gatewayRouteReconciler {
	return &gatewayRouteReconciler{
		k8sClient:                              k8sClient,
		finalizerManager:                       finalizerManager,
		grResManager:                           grResManager,
		enqueueRequestsForMeshEvents:           gatewayroute.NewEnqueueRequestsForMeshEvents(k8sClient, log),
		enqueueRequestsForVirtualGatewayEvents: gatewayroute.NewEnqueueRequestsForVirtualGatewayEvents(k8sClient, log),
		log:                                    log,
		recorder:                               recorder,
	}
}

// gatewayRouteReconciler reconciles a GatewayRoute object
type gatewayRouteReconciler struct {
	k8sClient        client.Client
	finalizerManager k8s.FinalizerManager
	grResManager     gatewayroute.ResourceManager

	enqueueRequestsForMeshEvents           handler.EventHandler
	enqueueRequestsForVirtualGatewayEvents handler.EventHandler
	log                                    logr.Logger
	recorder                               record.EventRecorder
}

// +kubebuilder:rbac:groups=appmesh.k8s.aws,resources=gatewayroutes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=appmesh.k8s.aws,resources=gatewayroutes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *gatewayRouteReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return runtime.HandleReconcileError(r.reconcile(ctx, req), r.log)
}

func (r *gatewayRouteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appmesh.GatewayRoute{}).
		Watches(&source.Kind{Type: &appmesh.Mesh{}}, r.enqueueRequestsForMeshEvents).
		Watches(&source.Kind{Type: &appmesh.VirtualGateway{}}, r.enqueueRequestsForVirtualGatewayEvents).
		WithOptions(controller.Options{MaxConcurrentReconciles: 3}).
		Complete(r)
}

func (r *gatewayRouteReconciler) reconcile(ctx context.Context, req ctrl.Request) error {
	gr := &appmesh.GatewayRoute{}
	if err := r.k8sClient.Get(ctx, req.NamespacedName, gr); err != nil {
		return client.IgnoreNotFound(err)
	}
	if !gr.DeletionTimestamp.IsZero() {
		return r.cleanupGatewayRoute(ctx, gr)
	}
	if err := r.reconcileGatewayRoute(ctx, gr); err != nil {
		r.recorder.Event(gr, corev1.EventTypeWarning, "ReconcileError", err.Error())
		return err
	}
	return nil
}

func (r *gatewayRouteReconciler) reconcileGatewayRoute(ctx context.Context, gr *appmesh.GatewayRoute) error {
	if err := r.finalizerManager.AddFinalizers(ctx, gr, k8s.FinalizerAWSAppMeshResources); err != nil {
		return err
	}
	if err := r.grResManager.Reconcile(ctx, gr); err != nil {
		return err
	}
	return nil
}

func (r *gatewayRouteReconciler) cleanupGatewayRoute(ctx context.Context, gr *appmesh.GatewayRoute) error {
	if k8s.HasFinalizer(gr, k8s.FinalizerAWSAppMeshResources) {
		if err := r.grResManager.Cleanup(ctx, gr); err != nil {
			return err
		}
		if err := r.finalizerManager.RemoveFinalizers(ctx, gr, k8s.FinalizerAWSAppMeshResources); err != nil {
			return err
		}
	}
	return nil
}
