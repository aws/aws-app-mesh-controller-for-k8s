package backendgroup

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

type BackendGroupTest struct {
	Namespace     *corev1.Namespace
	BackendGroups map[string]*appmesh.BackendGroup
}

func (m *BackendGroupTest) Create(ctx context.Context, f *framework.Framework, bg *appmesh.BackendGroup) error {
	err := f.K8sClient.Create(ctx, bg)
	if err != nil {
		return err
	}
	_, err = f.BGManager.WaitUntilBackendGroupActive(ctx, bg)
	if err != nil {
		return err
	}
	m.BackendGroups[bg.Name] = bg
	return nil
}

func (m *BackendGroupTest) Update(ctx context.Context, f *framework.Framework, newBG *appmesh.BackendGroup, bg *appmesh.BackendGroup) error {
	err := f.K8sClient.Patch(ctx, newBG, client.MergeFrom(bg))
	if err != nil {
		return err
	}
	_, err = f.BGManager.WaitUntilBackendGroupActive(ctx, newBG)
	if err != nil {
		return err
	}

	return nil
}

func (m *BackendGroupTest) Cleanup(ctx context.Context, f *framework.Framework) {
	var deletionErrors []error

	for _, bg := range m.BackendGroups {
		By(fmt.Sprintf("Delete backend group %s", bg.Name), func() {
			if err := f.K8sClient.Delete(ctx, bg,
				client.PropagationPolicy(metav1.DeletePropagationForeground), client.GracePeriodSeconds(0)); err != nil {

				if apierrs.IsNotFound(err) {
					f.Logger.Info("Backend group already deleted",
						zap.String("backend group", bg.Name))
					return
				}

				f.Logger.Error("Failed to delete backend group",
					zap.String("backend group", bg.Name),
					zap.Error(err))
				deletionErrors = append(deletionErrors, err)
				return
			}

			By(fmt.Sprintf("Wait for backend group to be deleted: %s", bg.Name), func() {
				if err := f.BGManager.WaitUntilBackendGroupDeleted(ctx, bg); err != nil {
					f.Logger.Error("failed to wait backend group deletion",
						zap.String("backend group", bg.Name),
						zap.Error(err))
					deletionErrors = append(deletionErrors, err)
				}
			})
			delete(m.BackendGroups, bg.Name)
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
		f.Logger.Error("BackendGroup clean up failed", zap.Error(err))
	}
	Expect(len(deletionErrors)).To(BeZero())
}
