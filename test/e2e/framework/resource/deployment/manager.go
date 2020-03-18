package deployment

import (
	"context"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/e2e/framework/utils"
	appsv1 "k8s.io/api/apps/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

type Manager interface {
	WaitUntilDeploymentReady(ctx context.Context, dp *appsv1.Deployment) (*appsv1.Deployment, error)
	WaitUntilDeploymentDeleted(ctx context.Context, dp *appsv1.Deployment) error
}

func NewManager(cs kubernetes.Interface) Manager {
	return &defaultManager{
		cs: cs,
	}
}

type defaultManager struct {
	cs kubernetes.Interface
}

func (m *defaultManager) WaitUntilDeploymentReady(ctx context.Context, dp *appsv1.Deployment) (*appsv1.Deployment, error) {
	var (
		observedDP *appsv1.Deployment
		err        error
	)
	return observedDP, wait.PollImmediateUntil(utils.PollIntervalShort, func() (bool, error) {
		observedDP, err = m.cs.AppsV1().Deployments(dp.Namespace).Get(dp.Name, metav1.GetOptions{})
		if err != nil {
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
	return wait.PollImmediateUntil(utils.PollIntervalShort, func() (bool, error) {
		if _, err := m.cs.AppsV1().Deployments(dp.Namespace).Get(dp.Name, metav1.GetOptions{}); err != nil {
			if apierrs.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	}, ctx.Done())
}
