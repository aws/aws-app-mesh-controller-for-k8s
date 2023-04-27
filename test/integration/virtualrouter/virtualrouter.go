package virtualrouter

import (
	"context"
	"fmt"

	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type VirtualRouterTest struct {
	Namespace      *corev1.Namespace
	VirtualRouters map[string]*appmesh.VirtualRouter
}

func (m *VirtualRouterTest) Create(ctx context.Context, f *framework.Framework, vr *appmesh.VirtualRouter) error {
	err := f.K8sClient.Create(ctx, vr)
	if err != nil {
		return err
	}
	_, err = f.VRManager.WaitUntilVirtualRouterActive(ctx, vr)
	if err != nil {
		return err
	}
	m.VirtualRouters[vr.Name] = vr
	return nil
}

func (m *VirtualRouterTest) Update(ctx context.Context, f *framework.Framework, newVR *appmesh.VirtualRouter, vr *appmesh.VirtualRouter) error {
	err := f.K8sClient.Patch(ctx, newVR, client.MergeFrom(vr))
	if err != nil {
		return err
	}
	_, err = f.VRManager.WaitUntilVirtualRouterActive(ctx, newVR)
	if err != nil {
		return err
	}

	return nil
}

func (m *VirtualRouterTest) Cleanup(ctx context.Context, f *framework.Framework) {
	var deletionErrors []error

	for _, vr := range m.VirtualRouters {
		By(fmt.Sprintf("Delete virtual router %s", vr.Name), func() {
			if err := f.K8sClient.Delete(ctx, vr,
				client.PropagationPolicy(metav1.DeletePropagationForeground), client.GracePeriodSeconds(0)); err != nil {

				if apierrs.IsNotFound(err) {
					f.Logger.Info("Virtual router already deleted",
						zap.String("virtual router", vr.Name))
					return
				}

				f.Logger.Error("Failed to delete virtual router",
					zap.String("virtual router", vr.Name),
					zap.Error(err))
				deletionErrors = append(deletionErrors, err)
				return
			}

			By(fmt.Sprintf("Wait for virtual router to be deleted: %s", vr.Name), func() {
				if err := f.VRManager.WaitUntilVirtualRouterDeleted(ctx, vr); err != nil {
					f.Logger.Error("failed to wait virtual router deletion",
						zap.String("virtual router", vr.Name),
						zap.Error(err))
					deletionErrors = append(deletionErrors, err)
				}
			})
			delete(m.VirtualRouters, vr.Name)
		})
	}

	if m.Namespace != nil {
		By(fmt.Sprintf("Delete namespace: %s", m.Namespace.Name), func() {
			if err := f.K8sClient.Delete(ctx, m.Namespace,
				client.PropagationPolicy(metav1.DeletePropagationForeground), client.GracePeriodSeconds(0)); err != nil {

				if !apierrs.IsNotFound(err) {
					f.Logger.Error("failed to delete namespace",
						zap.String("namespace", m.Namespace.Name),
						zap.Error(err))
					deletionErrors = append(deletionErrors, err)
					return
				}
			}

			By(fmt.Sprintf("Wait for the namespace to be deleted: %s", m.Namespace.Namespace), func() {
				if err := f.NSManager.WaitUntilNamespaceDeleted(ctx, m.Namespace); err != nil {
					f.Logger.Error("failed to wait namespace deletion",
						zap.String("namespace", m.Namespace.Name),
						zap.Error(err))
					deletionErrors = append(deletionErrors, err)
				}
			})
		})
	}

	for _, err := range deletionErrors {
		f.Logger.Error("VirtualRouter clean up failed", zap.Error(err))
	}
	Expect(len(deletionErrors)).To(BeZero())
}

func (m *VirtualRouterTest) CheckInAWS(ctx context.Context, f *framework.Framework, ms *appmesh.Mesh, vr *appmesh.VirtualRouter) error {
	return f.VRManager.CheckVirtualRouterInAWS(ctx, ms, vr)
}
