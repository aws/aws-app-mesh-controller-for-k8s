package backendgroup

import (
	"context"

	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws/services"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/k8s"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/utils"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Manager interface {
	WaitUntilBackendGroupActive(ctx context.Context, bg *appmesh.BackendGroup) (*appmesh.BackendGroup, error)
	WaitUntilBackendGroupDeleted(ctx context.Context, bg *appmesh.BackendGroup) error
}

func NewManager(k8sClient client.Client, appMeshSDK services.AppMesh) Manager {
	return &defaultManager{
		k8sClient:  k8sClient,
		appMeshSDK: appMeshSDK,
	}
}

type defaultManager struct {
	k8sClient  client.Client
	appMeshSDK services.AppMesh
}

func (m *defaultManager) WaitUntilBackendGroupActive(ctx context.Context, bg *appmesh.BackendGroup) (*appmesh.BackendGroup, error) {
	observedBG := &appmesh.BackendGroup{}
	retryCount := 0
	return observedBG, wait.PollImmediateUntil(utils.PollIntervalShort, func() (bool, error) {

		err := m.k8sClient.Get(ctx, k8s.NamespacedName(bg), observedBG)
		if err != nil {
			if retryCount >= utils.PollRetries {
				return false, err
			}
			retryCount++
			return false, nil
		}

		/*
			for _, condition := range observedBG.Status.Conditions {
				if condition.Type == appmesh.BackendGroupActive && condition.Status == corev1.ConditionTrue {
					return true, nil
				}
			}
		*/

		//return false, nil
		return true, nil
	}, ctx.Done())
}

func (m *defaultManager) WaitUntilBackendGroupDeleted(ctx context.Context, bg *appmesh.BackendGroup) error {
	observedBG := &appmesh.BackendGroup{}
	return wait.PollImmediateUntil(utils.PollIntervalShort, func() (bool, error) {
		if err := m.k8sClient.Get(ctx, k8s.NamespacedName(bg), observedBG); err != nil {
			if apierrs.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	}, ctx.Done())
}
