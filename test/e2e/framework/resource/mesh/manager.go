package mesh

import (
	"context"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/e2e/framework/k8s"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/e2e/framework/utils"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Manager interface {
	WaitUntilMeshActive(ctx context.Context, mesh *appmesh.Mesh) (*appmesh.Mesh, error)
	WaitUntilMeshDeleted(ctx context.Context, mesh *appmesh.Mesh) error
}

func NewManager(k8sClient client.Client) Manager {
	return &defaultManager{k8sClient: k8sClient}
}

type defaultManager struct {
	k8sClient client.Client
}

func (m *defaultManager) WaitUntilMeshActive(ctx context.Context, mesh *appmesh.Mesh) (*appmesh.Mesh, error) {
	observedMesh := &appmesh.Mesh{}
	return observedMesh, wait.PollImmediateUntil(utils.PollIntervalShort, func() (bool, error) {
		if err := m.k8sClient.Get(ctx, k8s.NamespacedName(mesh), observedMesh); err != nil {
			return false, err
		}

		for _, condition := range observedMesh.Status.Conditions {
			if condition.Type == appmesh.MeshActive && condition.Status == corev1.ConditionTrue {
				return true, nil
			}
		}

		return false, nil
	}, ctx.Done())
}

func (m *defaultManager) WaitUntilMeshDeleted(ctx context.Context, mesh *appmesh.Mesh) error {
	observedMesh := &appmesh.Mesh{}
	return wait.PollImmediateUntil(utils.PollIntervalShort, func() (bool, error) {
		if err := m.k8sClient.Get(ctx, k8s.NamespacedName(mesh), observedMesh); err != nil {
			if apierrs.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	}, ctx.Done())
}
