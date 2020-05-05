package virtualnode

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
	WaitUntilVirtualNodeActive(ctx context.Context, vn *appmeshv1beta1.VirtualNode) (*appmeshv1beta1.VirtualNode, error)
	WaitUntilVirtualNodeDeleted(ctx context.Context, vn *appmeshv1beta1.VirtualNode) error
}

func NewManager(cs meshclientset.Interface) Manager {
	return &defaultManager{cs: cs}
}

type defaultManager struct {
	cs meshclientset.Interface
}

func (m *defaultManager) WaitUntilVirtualNodeActive(ctx context.Context, vn *appmeshv1beta1.VirtualNode) (*appmeshv1beta1.VirtualNode, error) {
	var (
		observedVN *appmeshv1beta1.VirtualNode
		err        error
	)
	return observedVN, wait.PollImmediateUntil(utils.PollIntervalShort, func() (bool, error) {
		observedVN, err = m.cs.AppmeshV1beta1().VirtualNodes(vn.Namespace).Get(vn.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		for _, condition := range observedVN.Status.Conditions {
			if condition.Type == appmeshv1beta1.VirtualNodeActive && condition.Status == corev1.ConditionTrue {
				return true, nil
			}
		}

		return false, nil
	}, ctx.Done())
}

func (m *defaultManager) WaitUntilVirtualNodeDeleted(ctx context.Context, vn *appmeshv1beta1.VirtualNode) error {
	return wait.PollImmediateUntil(utils.PollIntervalShort, func() (bool, error) {
		if _, err := m.cs.AppmeshV1beta1().VirtualNodes(vn.Namespace).Get(vn.Name, metav1.GetOptions{}); err != nil {
			if apierrs.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	}, ctx.Done())
}
