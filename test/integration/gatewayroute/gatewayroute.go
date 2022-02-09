package gatewayroute

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

type GatewayRouteTest struct {
	Namespace     *corev1.Namespace
	GatewayRoutes map[string]*appmesh.GatewayRoute
}

func (m *GatewayRouteTest) Create(ctx context.Context, f *framework.Framework, gr *appmesh.GatewayRoute) error {
	err := f.K8sClient.Create(ctx, gr)
	if err != nil {
		return err
	}
	_, err = f.GRManager.WaitUntilGatewayRouteActive(ctx, gr)
	if err != nil {
		return err
	}
	m.GatewayRoutes[gr.Name] = gr
	return nil
}

func (m *GatewayRouteTest) Update(ctx context.Context, f *framework.Framework, newGR *appmesh.GatewayRoute, gr *appmesh.GatewayRoute) error {
	err := f.K8sClient.Patch(ctx, newGR, client.MergeFrom(gr))
	if err != nil {
		return err
	}
	_, err = f.GRManager.WaitUntilGatewayRouteActive(ctx, newGR)
	if err != nil {
		return err
	}

	return nil
}

func (m *GatewayRouteTest) Cleanup(ctx context.Context, f *framework.Framework) {
	var deletionErrors []error

	for _, gr := range m.GatewayRoutes {
		By(fmt.Sprintf("Delete gateway route %s", gr.Name), func() {
			if err := f.K8sClient.Delete(ctx, gr,
				client.PropagationPolicy(metav1.DeletePropagationForeground), client.GracePeriodSeconds(0)); err != nil {

				if apierrs.IsNotFound(err) {
					f.Logger.Info("Gateway route already deleted",
						zap.String("gateway route", gr.Name))
					return
				}

				f.Logger.Error("Failed to delete gateway route",
					zap.String("gateway route", gr.Name),
					zap.Error(err))
				deletionErrors = append(deletionErrors, err)
				return
			}

			By(fmt.Sprintf("Wait for gateway route to be deleted: %s", gr.Name), func() {
				if err := f.GRManager.WaitUntilGatewayRouteDeleted(ctx, gr); err != nil {
					f.Logger.Error("failed to wait gateway route deletion",
						zap.String("gateway route", gr.Name),
						zap.Error(err))
					deletionErrors = append(deletionErrors, err)
				}
			})
			delete(m.GatewayRoutes, gr.Name)
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
		f.Logger.Error("GatewayRoute clean up failed", zap.Error(err))
	}
	Expect(len(deletionErrors)).To(BeZero())
}

func (m *GatewayRouteTest) CheckInAWS(ctx context.Context, f *framework.Framework, ms *appmesh.Mesh, vg *appmesh.VirtualGateway, gr *appmesh.GatewayRoute) error {
	var err error
	// Reconcile may take a while so add re-tries
	for i := 0; i < utils.PollRetries; i++ {
		err = f.GRManager.CheckGatewayRouteInAWS(ctx, ms, vg, gr)
		if err != nil {
			if i >= utils.PollRetries {
				return err
			}
			time.Sleep(utils.AWSPollIntervalShort)
			continue
		}
		return err
	}
	return err
}
