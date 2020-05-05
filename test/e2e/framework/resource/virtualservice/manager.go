package virtualservice

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
	WaitUntilVirtualServiceActive(ctx context.Context, vs *appmeshv1beta1.VirtualService) (*appmeshv1beta1.VirtualService, error)
	WaitUntilVirtualServiceDeleted(ctx context.Context, vs *appmeshv1beta1.VirtualService) error
}

func NewManager(cs meshclientset.Interface) Manager {
	return &defaultManager{cs: cs}
}

type defaultManager struct {
	cs meshclientset.Interface
}

func (m *defaultManager) WaitUntilVirtualServiceActive(ctx context.Context, vs *appmeshv1beta1.VirtualService) (*appmeshv1beta1.VirtualService, error) {
	var (
		observedVS *appmeshv1beta1.VirtualService
		err        error
	)
	return observedVS, wait.PollImmediateUntil(utils.PollIntervalShort, func() (bool, error) {
		observedVS, err = m.cs.AppmeshV1beta1().VirtualServices(vs.Namespace).Get(vs.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		conditions := make(map[appmeshv1beta1.VirtualServiceConditionType]corev1.ConditionStatus)
		for _, condition := range observedVS.Status.Conditions {
			conditions[condition.Type] = condition.Status
		}
		readyConditionTypes := []appmeshv1beta1.VirtualServiceConditionType{
			appmeshv1beta1.VirtualServiceActive,
			appmeshv1beta1.VirtualRouterActive,
			appmeshv1beta1.RoutesActive,
		}
		for _, conditionType := range readyConditionTypes {
			status, ok := conditions[conditionType]
			if !ok || status != corev1.ConditionTrue {
				return false, nil
			}
		}
		return true, nil
	}, ctx.Done())
}

func (m *defaultManager) WaitUntilVirtualServiceDeleted(ctx context.Context, vs *appmeshv1beta1.VirtualService) error {
	return wait.PollImmediateUntil(utils.PollIntervalShort, func() (bool, error) {
		if _, err := m.cs.AppmeshV1beta1().VirtualServices(vs.Namespace).Get(vs.Name, metav1.GetOptions{}); err != nil {
			if apierrs.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	}, ctx.Done())
}
