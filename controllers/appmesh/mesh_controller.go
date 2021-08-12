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
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/mesh"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/runtime"
	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
)

func NewMeshReconciler(
	k8sClient client.Client,
	finalizerManager k8s.FinalizerManager,
	meshMembersFinalizer mesh.MembersFinalizer,
	meshResManager mesh.ResourceManager,
	log logr.Logger) *meshReconciler {
	return &meshReconciler{
		k8sClient:            k8sClient,
		finalizerManager:     finalizerManager,
		meshMembersFinalizer: meshMembersFinalizer,
		meshResManager:       meshResManager,
		log:                  log,
	}
}

// meshReconciler reconciles a Mesh object
type meshReconciler struct {
	k8sClient            client.Client
	finalizerManager     k8s.FinalizerManager
	meshMembersFinalizer mesh.MembersFinalizer
	meshResManager       mesh.ResourceManager
	log                  logr.Logger
}

// +kubebuilder:rbac:groups=appmesh.k8s.aws,resources=meshes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=appmesh.k8s.aws,resources=meshes/status,verbs=get;update;patch

func (r *meshReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return runtime.HandleReconcileError(r.reconcile(ctx, req), r.log)
}

func (r *meshReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appmesh.Mesh{}).
		Complete(r)
}

func (r *meshReconciler) reconcile(ctx context.Context, req ctrl.Request) error {
	ms := &appmesh.Mesh{}
	if err := r.k8sClient.Get(ctx, req.NamespacedName, ms); err != nil {
		return client.IgnoreNotFound(err)
	}
	if !ms.DeletionTimestamp.IsZero() {
		return r.cleanupMesh(ctx, ms)
	}
	return r.reconcileMesh(ctx, ms)
}

func (r *meshReconciler) reconcileMesh(ctx context.Context, ms *appmesh.Mesh) error {
	if err := r.finalizerManager.AddFinalizers(ctx, ms, k8s.FinalizerMeshMembers, k8s.FinalizerAWSAppMeshResources); err != nil {
		return err
	}
	if err := r.meshResManager.Reconcile(ctx, ms); err != nil {
		return err
	}
	return nil
}

func (r *meshReconciler) cleanupMesh(ctx context.Context, ms *appmesh.Mesh) error {
	if k8s.HasFinalizer(ms, k8s.FinalizerMeshMembers) {
		if err := r.meshMembersFinalizer.Finalize(ctx, ms); err != nil {
			return err
		}
		if err := r.finalizerManager.RemoveFinalizers(ctx, ms, k8s.FinalizerMeshMembers); err != nil {
			return err
		}
	}

	if k8s.HasFinalizer(ms, k8s.FinalizerAWSAppMeshResources) {
		if err := r.meshResManager.Cleanup(ctx, ms); err != nil {
			return err
		}
		if err := r.finalizerManager.RemoveFinalizers(ctx, ms, k8s.FinalizerAWSAppMeshResources); err != nil {
			return err
		}
	}
	return nil
}
