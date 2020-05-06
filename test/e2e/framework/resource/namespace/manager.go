package namespace

import (
	"context"
	"fmt"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/e2e/framework/k8s"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/e2e/framework/utils"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Manager interface {
	AllocateNamespace(ctx context.Context, baseName string) (*corev1.Namespace, error)
	WaitUntilNamespaceDeleted(ctx context.Context, ns *corev1.Namespace) error
}

func NewManager(k8sClient client.Client) Manager {
	return &defaultManager{
		k8sClient: k8sClient,
	}
}

type defaultManager struct {
	k8sClient client.Client
}

func (m *defaultManager) AllocateNamespace(ctx context.Context, baseName string) (*corev1.Namespace, error) {
	name, err := m.findAvailableNamespaceName(ctx, baseName)
	if err != nil {
		return nil, err
	}

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	err = m.k8sClient.Create(ctx, ns)
	return ns, err
}

func (m *defaultManager) WaitUntilNamespaceDeleted(ctx context.Context, ns *corev1.Namespace) error {
	gotNS := &corev1.Namespace{}
	return wait.PollImmediateUntil(utils.PollIntervalShort, func() (bool, error) {
		if err := m.k8sClient.Get(ctx, k8s.NamespacedName(ns), gotNS); err != nil {
			if apierrs.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	}, ctx.Done())
}

// findAvailableNamespaceName random namespace name starting with baseName.
func (m *defaultManager) findAvailableNamespaceName(ctx context.Context, baseName string) (string, error) {
	var name string
	gotNS := &corev1.Namespace{}
	err := wait.PollImmediateUntil(utils.PollIntervalShort, func() (bool, error) {
		name = fmt.Sprintf("%v-%v", baseName, utils.RandomDNS1123Label(6))
		if err := m.k8sClient.Get(ctx, types.NamespacedName{Name: name}, gotNS); err != nil {
			if apierrs.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	}, ctx.Done())
	return name, err
}
