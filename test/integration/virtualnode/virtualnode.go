package virtualnode

import (
	"context"
	"fmt"

	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type VirtualNodeTest struct {
	Namespace    *corev1.Namespace
	VirtualNodes map[string]*appmesh.VirtualNode
}

func (m *VirtualNodeTest) Create(ctx context.Context, f *framework.Framework, vn *appmesh.VirtualNode) error {
	err := f.K8sClient.Create(ctx, vn)
	if err != nil {
		return err
	}
	_, err = f.VNManager.WaitUntilVirtualNodeActive(ctx, vn)
	if err != nil {
		return err
	}
	m.VirtualNodes[vn.Name] = vn
	return nil
}

func (m *VirtualNodeTest) Update(ctx context.Context, f *framework.Framework, newVN *appmesh.VirtualNode, vn *appmesh.VirtualNode) (*appmesh.VirtualNode, error) {
	err := f.K8sClient.Patch(ctx, newVN, client.MergeFrom(vn))
	if err != nil {
		return nil, err
	}
	updatedVN, err := f.VNManager.WaitUntilVirtualNodeActive(ctx, newVN)
	if err != nil {
		return nil, err
	}

	m.VirtualNodes[vn.Name] = updatedVN
	return updatedVN, nil
}

func (m *VirtualNodeTest) Cleanup(ctx context.Context, f *framework.Framework) {
	var deletionErrors []error

	for _, vn := range m.VirtualNodes {
		By(fmt.Sprintf("Delete virtual node %s", vn.Name), func() {
			if err := f.K8sClient.Delete(ctx, vn,
				client.PropagationPolicy(metav1.DeletePropagationForeground), client.GracePeriodSeconds(0)); err != nil {
				f.Logger.Error("Failed to delete virtual node",
					zap.String("virtual node", vn.Name),
					zap.Error(err))
				deletionErrors = append(deletionErrors, err)
				return
			}

			By(fmt.Sprintf("Wait for virtual node to be deleted: %s", vn.Name), func() {
				if err := f.VNManager.WaitUntilVirtualNodeDeleted(ctx, vn); err != nil {
					f.Logger.Error("failed to wait virtual node deletion",
						zap.String("virtual node", vn.Name),
						zap.Error(err))
					deletionErrors = append(deletionErrors, err)
				}
			})
		})
	}

	if m.Namespace != nil {
		By(fmt.Sprintf("Delete namespace: %s", m.Namespace.Name), func() {
			if err := f.K8sClient.Delete(ctx, m.Namespace,
				client.PropagationPolicy(metav1.DeletePropagationForeground), client.GracePeriodSeconds(0)); err != nil {
				f.Logger.Error("failed to delete namespace",
					zap.String("namespace", m.Namespace.Name),
					zap.Error(err))
				deletionErrors = append(deletionErrors, err)
				return
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
		f.Logger.Error("VirtualNode clean up failed", zap.Error(err))
	}
	Expect(len(deletionErrors)).To(BeZero())
}

func (m *VirtualNodeTest) CheckInAWS(ctx context.Context, f *framework.Framework, ms *appmesh.Mesh, vn *appmesh.VirtualNode) error {
	return f.VNManager.CheckVirtualNodeInAWS(ctx, ms, vn)
}
