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
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/runtime"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/virtualgateway"
	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
)

// NewVirtualGatewayReconciler constructs new virtualGatewayReconciler
func NewVirtualGatewayReconciler(
	k8sClient client.Client,
	finalizerManager k8s.FinalizerManager,
	vgMembersFinalizer virtualgateway.MembersFinalizer,
	vgResManager virtualgateway.ResourceManager,
	log logr.Logger) *virtualGatewayReconciler {
	return &virtualGatewayReconciler{
		k8sClient:                    k8sClient,
		finalizerManager:             finalizerManager,
		vgMembersFinalizer:           vgMembersFinalizer,
		vgResManager:                 vgResManager,
		enqueueRequestsForMeshEvents: virtualgateway.NewEnqueueRequestsForMeshEvents(k8sClient, log),
		log:                          log,
	}
}

// virtualGatewayReconciler reconciles a VirtualGateway object
type virtualGatewayReconciler struct {
	k8sClient          client.Client
	finalizerManager   k8s.FinalizerManager
	vgMembersFinalizer virtualgateway.MembersFinalizer
	vgResManager       virtualgateway.ResourceManager

	enqueueRequestsForMeshEvents handler.EventHandler
	log                          logr.Logger
}

//// +kubebuilder:rbac:groups=appmesh.k8s.aws,resources=virtualgateways,verbs=get;list;watch;create;update;patch;delete
//// +kubebuilder:rbac:groups=appmesh.k8s.aws,resources=virtualgateways/status,verbs=get;update;patch

func (r *virtualGatewayReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	return runtime.HandleReconcileError(r.reconcile(req), r.log)
}

func (r *virtualGatewayReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appmesh.VirtualGateway{}).
		Watches(&source.Kind{Type: &appmesh.Mesh{}}, r.enqueueRequestsForMeshEvents).
		WithOptions(controller.Options{MaxConcurrentReconciles: 3}).
		Complete(r)
}

func (r *virtualGatewayReconciler) reconcile(req ctrl.Request) error {
	ctx := context.Background()
	vg := &appmesh.VirtualGateway{}
	if err := r.k8sClient.Get(ctx, req.NamespacedName, vg); err != nil {
		return client.IgnoreNotFound(err)
	}
	if !vg.DeletionTimestamp.IsZero() {
		return r.cleanupVirtualGateway(ctx, vg)
	}
	return r.reconcileVirtualGateway(ctx, vg)
}

func (r *virtualGatewayReconciler) reconcileVirtualGateway(ctx context.Context, vg *appmesh.VirtualGateway) error {
	if err := r.finalizerManager.AddFinalizers(ctx, vg, k8s.FinalizerVirtualGatewayMembers, k8s.FinalizerAWSAppMeshResources); err != nil {
		return err
	}
	if err := r.vgResManager.Reconcile(ctx, vg); err != nil {
		return err
	}
	return nil
}

func (r *virtualGatewayReconciler) cleanupVirtualGateway(ctx context.Context, vg *appmesh.VirtualGateway) error {
	if k8s.HasFinalizer(vg, k8s.FinalizerVirtualGatewayMembers) {
		if err := r.vgMembersFinalizer.Finalize(ctx, vg); err != nil {
			return err
		}
		if err := r.finalizerManager.RemoveFinalizers(ctx, vg, k8s.FinalizerVirtualGatewayMembers); err != nil {
			return err
		}
	}

	if k8s.HasFinalizer(vg, k8s.FinalizerAWSAppMeshResources) {
		if err := r.vgResManager.Cleanup(ctx, vg); err != nil {
			return err
		}
		if err := r.finalizerManager.RemoveFinalizers(ctx, vg, k8s.FinalizerAWSAppMeshResources); err != nil {
			return err
		}
	}
	return nil
}
