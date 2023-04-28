package mesh

import (
	"context"
	"fmt"
	"time"

	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/k8s"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type MeshTest struct {
	Meshes map[string]*appmesh.Mesh
}

func (m *MeshTest) Create(ctx context.Context, f *framework.Framework, mesh *appmesh.Mesh) error {
	err := f.K8sClient.Create(ctx, mesh)
	if err != nil {
		return err
	}
	_, err = f.MeshManager.WaitUntilMeshActive(ctx, mesh)
	if err != nil {
		return err
	}
	m.Meshes[mesh.Name] = mesh
	return nil
}

func (m *MeshTest) Get(ctx context.Context, f *framework.Framework, mesh *appmesh.Mesh) (*appmesh.Mesh, error) {
	observedMesh := &appmesh.Mesh{}
	if err := f.K8sClient.Get(ctx, k8s.NamespacedName(mesh), observedMesh); err != nil {
		return nil, err
	}
	return observedMesh, nil
}

func (m *MeshTest) HasDeletionTimestamp(mesh *appmesh.Mesh) bool {
	if mesh.ObjectMeta.DeletionTimestamp != nil {
		return true
	}
	return false
}

func (m *MeshTest) WaitForDeletionTimestamp(ctx context.Context, f *framework.Framework, mesh *appmesh.Mesh) bool {
	retries := 5

	// check for deletion timestamp on the object with retries
	for i := 0; i < retries; i++ {
		time.Sleep(100 * time.Millisecond)

		ms, err := m.Get(ctx, f, mesh)
		if err != nil {
			continue
		}

		if m.HasDeletionTimestamp(ms) {
			return true
		}
	}
	return false
}

func (m *MeshTest) Update(ctx context.Context, f *framework.Framework, newMesh *appmesh.Mesh, mesh *appmesh.Mesh) error {
	err := f.K8sClient.Patch(ctx, newMesh, client.MergeFrom(mesh))
	if err != nil {
		return err
	}
	_, err = f.MeshManager.WaitUntilMeshActive(ctx, newMesh)
	if err != nil {
		return err
	}

	return nil
}

func (m *MeshTest) Cleanup(ctx context.Context, f *framework.Framework) {
	var deletionErrors []error

	for _, mesh := range m.Meshes {
		By(fmt.Sprintf("Delete mesh %s", mesh.Name), func() {
			if err := f.K8sClient.Delete(ctx, mesh,
				client.PropagationPolicy(metav1.DeletePropagationForeground), client.GracePeriodSeconds(0)); err != nil {

				if apierrs.IsNotFound(err) {
					f.Logger.Info("Mesh already deleted",
						zap.String("mesh", mesh.Name))
					return
				}

				f.Logger.Error("Failed to delete mesh",
					zap.String("mesh", mesh.Name),
					zap.Error(err))
				deletionErrors = append(deletionErrors, err)
				return
			}

			By(fmt.Sprintf("Wait for mesh to be deleted: %s", mesh.Name), func() {
				if err := f.MeshManager.WaitUntilMeshDeleted(ctx, mesh); err != nil {
					f.Logger.Error("failed to wait mesh deletion",
						zap.String("mesh", mesh.Name),
						zap.Error(err))
					deletionErrors = append(deletionErrors, err)
				}
			})
		})
		delete(m.Meshes, mesh.Name)
	}

	for _, err := range deletionErrors {
		f.Logger.Error("Mesh clean up failed", zap.Error(err))
	}
	Expect(len(deletionErrors)).To(BeZero())
}

func (m *MeshTest) CheckInAWS(ctx context.Context, f *framework.Framework, mesh *appmesh.Mesh) error {
	return f.MeshManager.CheckMeshInAWS(ctx, mesh)
}
