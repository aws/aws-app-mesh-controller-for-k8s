package virtualservice

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

type VirtualServiceTest struct {
	Namespace       *corev1.Namespace
	VirtualServices map[string]*appmesh.VirtualService
}

func (m *VirtualServiceTest) Create(ctx context.Context, f *framework.Framework, vs *appmesh.VirtualService) error {
	err := f.K8sClient.Create(ctx, vs)
	if err != nil {
		return err
	}
	_, err = f.VSManager.WaitUntilVirtualServiceActive(ctx, vs)
	if err != nil {
		return err
	}
	m.VirtualServices[vs.Name] = vs
	return nil
}

func (m *VirtualServiceTest) Update(ctx context.Context, f *framework.Framework, newVS *appmesh.VirtualService, vs *appmesh.VirtualService) error {
	err := f.K8sClient.Patch(ctx, newVS, client.MergeFrom(vs))
	if err != nil {
		return err
	}
	_, err = f.VSManager.WaitUntilVirtualServiceActive(ctx, newVS)
	if err != nil {
		return err
	}

	return nil
}

func (m *VirtualServiceTest) Cleanup(ctx context.Context, f *framework.Framework) {
	var deletionErrors []error

	for _, vs := range m.VirtualServices {
		By(fmt.Sprintf("Delete virtual service %s", vs.Name), func() {
			if err := f.K8sClient.Delete(ctx, vs,
				client.PropagationPolicy(metav1.DeletePropagationForeground), client.GracePeriodSeconds(0)); err != nil {

				if apierrs.IsNotFound(err) {
					f.Logger.Info("Virtual service already deleted",
						zap.String("virtual sercice", vs.Name))
					return
				}

				f.Logger.Error("Failed to delete virtual service",
					zap.String("virtual service", vs.Name),
					zap.Error(err))
				deletionErrors = append(deletionErrors, err)
				return
			}

			By(fmt.Sprintf("Wait for virtual service to be deleted: %s", vs.Name), func() {
				if err := f.VSManager.WaitUntilVirtualServiceDeleted(ctx, vs); err != nil {
					f.Logger.Error("failed to wait virtual service deletion",
						zap.String("virtual service", vs.Name),
						zap.Error(err))
					deletionErrors = append(deletionErrors, err)
				}
			})
			delete(m.VirtualServices, vs.Name)
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
		f.Logger.Error("VirtualService clean up failed", zap.Error(err))
	}
	Expect(len(deletionErrors)).To(BeZero())
}

func (m *VirtualServiceTest) CheckInAWS(ctx context.Context, f *framework.Framework, ms *appmesh.Mesh, vs *appmesh.VirtualService) error {
	var err error
	// Reconcile may take a while so add re-tries
	for i := 0; i < utils.PollRetries; i++ {
		err = f.VSManager.CheckVirtualServiceInAWS(ctx, ms, vs)
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
