package deployment

import (
	"context"

	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/k8s"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/utils"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Manager interface {
	WaitUntilDeploymentReady(ctx context.Context, dp *appsv1.Deployment) (*appsv1.Deployment, error)
	WaitUntilDeploymentDeleted(ctx context.Context, dp *appsv1.Deployment) error
}

func NewManager(k8sClient client.Client) Manager {
	return &defaultManager{
		k8sClient: k8sClient,
	}
}

type defaultManager struct {
	k8sClient client.Client
}

func (m *defaultManager) WaitUntilDeploymentReady(ctx context.Context, dp *appsv1.Deployment) (*appsv1.Deployment, error) {
	observedDP := &appsv1.Deployment{}
	return observedDP, wait.PollImmediateUntil(utils.PollIntervalShort, func() (bool, error) {
		err := m.k8sClient.Get(ctx, k8s.NamespacedName(dp), observedDP)
		if err != nil {
			if e, ok := err.(*errors.StatusError); ok && e.ErrStatus.Code == 404 {
				return false, nil
			}

			return false, err
		}

		if observedDP.Status.UpdatedReplicas == (*dp.Spec.Replicas) &&
			observedDP.Status.Replicas == (*dp.Spec.Replicas) &&
			observedDP.Status.AvailableReplicas == (*dp.Spec.Replicas) &&
			observedDP.Status.ObservedGeneration >= dp.Generation {
			return true, nil
		}
		return false, nil
	}, ctx.Done())
}

func (m *defaultManager) WaitUntilDeploymentDeleted(ctx context.Context, dp *appsv1.Deployment) error {
	observedDP := &appsv1.Deployment{}
	return wait.PollImmediateUntil(utils.PollIntervalShort, func() (bool, error) {
		if err := m.k8sClient.Get(ctx, k8s.NamespacedName(dp), observedDP); err != nil {
			if apierrs.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	}, ctx.Done())
}
