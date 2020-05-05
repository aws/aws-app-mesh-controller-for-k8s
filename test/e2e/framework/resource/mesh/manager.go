package mesh

import (
	"context"
	appmeshv1beta1 "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/apis/appmesh/v1beta1"
	meshclientset "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/client/clientset/versioned"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/e2e/framework/utils"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

type Manager interface {
	WaitUntilMeshActive(ctx context.Context, mesh *appmeshv1beta1.Mesh) (*appmeshv1beta1.Mesh, error)
	WaitUntilMeshDeleted(ctx context.Context, mesh *appmeshv1beta1.Mesh) error
}

func NewManager(cs meshclientset.Interface) Manager {
	return &defaultManager{cs: cs}
}

type defaultManager struct {
	cs meshclientset.Interface
}

func (m *defaultManager) WaitUntilMeshActive(ctx context.Context, mesh *appmeshv1beta1.Mesh) (*appmeshv1beta1.Mesh, error) {
	var (
		observedMesh *appmeshv1beta1.Mesh
		err          error
	)
	return observedMesh, wait.PollImmediateUntil(utils.PollIntervalShort, func() (bool, error) {
		observedMesh, err = m.cs.AppmeshV1beta1().Meshes().Get(mesh.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		for _, condition := range observedMesh.Status.Conditions {
			if condition.Type == appmeshv1beta1.MeshActive && condition.Status == corev1.ConditionTrue {
				return true, nil
			}
		}

		return false, nil
	}, ctx.Done())
}

func (m *defaultManager) WaitUntilMeshDeleted(ctx context.Context, mesh *appmeshv1beta1.Mesh) error {
	return wait.PollImmediateUntil(utils.PollIntervalShort, func() (bool, error) {
		if _, err := m.cs.AppmeshV1beta1().Meshes().Get(mesh.Name, metav1.GetOptions{}); err != nil {
			if apierrs.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	}, ctx.Done())
}
