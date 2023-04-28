package virtualgateway

import (
	"context"
	"fmt"
	"time"

	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type VirtualGatewayTest struct {
	Namespace       *corev1.Namespace
	VirtualGateways map[string]*appmesh.VirtualGateway
}

func (m *VirtualGatewayTest) Create(ctx context.Context, f *framework.Framework, vg *appmesh.VirtualGateway) error {
	err := f.K8sClient.Create(ctx, vg)
	if err != nil {
		return err
	}
	_, err = f.VGManager.WaitUntilVirtualGatewayActive(ctx, vg)
	if err != nil {
		return err
	}
	m.VirtualGateways[vg.Name] = vg
	return nil
}

func (m *VirtualGatewayTest) Update(ctx context.Context, f *framework.Framework, newVG *appmesh.VirtualGateway, vg *appmesh.VirtualGateway) error {
	err := f.K8sClient.Patch(ctx, newVG, client.MergeFrom(vg))
	if err != nil {
		return err
	}
	_, err = f.VGManager.WaitUntilVirtualGatewayActive(ctx, newVG)
	if err != nil {
		return err
	}

	return nil
}

func (m *VirtualGatewayTest) Cleanup(ctx context.Context, f *framework.Framework) {
	var deletionErrors []error

	for _, vg := range m.VirtualGateways {
		By(fmt.Sprintf("Delete virtual gateway %s", vg.Name), func() {
			if err := f.K8sClient.Delete(ctx, vg,
				client.PropagationPolicy(metav1.DeletePropagationForeground), client.GracePeriodSeconds(0)); err != nil {

				if apierrs.IsNotFound(err) {
					f.Logger.Info("Virtual gateway already deleted",
						zap.String("virtual sercice", vg.Name))
					return
				}

				f.Logger.Error("Failed to delete virtual gateway",
					zap.String("virtual gateway", vg.Name),
					zap.Error(err))
				deletionErrors = append(deletionErrors, err)
				return
			}

			By(fmt.Sprintf("Wait for virtual gateway to be deleted: %s", vg.Name), func() {
				if err := f.VGManager.WaitUntilVirtualGatewayDeleted(ctx, vg); err != nil {
					f.Logger.Error("failed to wait virtual gateway deletion",
						zap.String("virtual gateway", vg.Name),
						zap.Error(err))
					deletionErrors = append(deletionErrors, err)
				}
			})
			delete(m.VirtualGateways, vg.Name)
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
		f.Logger.Error("VirtualGateway clean up failed", zap.Error(err))
	}
	Expect(len(deletionErrors)).To(BeZero())
}

func (m *VirtualGatewayTest) CheckInAWS(ctx context.Context, f *framework.Framework, ms *appmesh.Mesh, vg *appmesh.VirtualGateway) error {
	var err error
	// Reconcile may take a while so add re-tries
	for i := 0; i < utils.PollRetries; i++ {
		err = f.VGManager.CheckVirtualGatewayInAWS(ctx, ms, vg)
		if err != nil {
			if i >= utils.PollRetries {
				return err
			}
			time.Sleep(utils.AWSPollIntervalMedium)
			continue
		}
		return err
	}
	return err
}
