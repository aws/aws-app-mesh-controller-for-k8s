package virtualrouter

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
	WaitUntilVirtualRouterActive(ctx context.Context, vr *appmesh.VirtualRouter) (*appmesh.VirtualRouter, error)
	WaitUntilVirtualRouterDeleted(ctx context.Context, vr *appmesh.VirtualRouter) error
}

func NewManager(k8sClient client.Client) Manager {
	return &defaultManager{k8sClient: k8sClient}
}

type defaultManager struct {
	k8sClient client.Client
}

func (m *defaultManager) WaitUntilVirtualRouterActive(ctx context.Context, vr *appmesh.VirtualRouter) (*appmesh.VirtualRouter, error) {
	observedVR := &appmesh.VirtualRouter{}
	return observedVR, wait.PollImmediateUntil(utils.PollIntervalShort, func() (bool, error) {
		if err := m.k8sClient.Get(ctx, k8s.NamespacedName(vr), observedVR); err != nil {
			return false, err
		}

		for _, condition := range observedVR.Status.Conditions {
			if condition.Type == appmesh.VirtualRouterActive && condition.Status == corev1.ConditionTrue {
				return true, nil
			}
		}

		return false, nil
	}, ctx.Done())
}

func (m *defaultManager) WaitUntilVirtualRouterDeleted(ctx context.Context, vr *appmesh.VirtualRouter) error {
	observedVR := &appmesh.VirtualRouter{}
	return wait.PollImmediateUntil(utils.PollIntervalShort, func() (bool, error) {
		if err := m.k8sClient.Get(ctx, k8s.NamespacedName(vr), observedVR); err != nil {
			if apierrs.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	}, ctx.Done())
}
