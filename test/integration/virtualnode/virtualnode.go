package virtualnode

import (
	"context"
	"fmt"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/manifest"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"

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

type VirtualNodeTest struct {
	Namespace         *corev1.Namespace
	VirtualNodes      map[string]*appmesh.VirtualNode
	Deployments       map[string]*appsv1.Deployment
	Pods              map[string]*corev1.Pod
	CloudMapNameSpace string
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

func (m *VirtualNodeTest) Update(ctx context.Context, f *framework.Framework, newVN *appmesh.VirtualNode, vn *appmesh.VirtualNode) error {
	err := f.K8sClient.Patch(ctx, newVN, client.MergeFrom(vn))
	if err != nil {
		return err
	}
	_, err = f.VNManager.WaitUntilVirtualNodeActive(ctx, newVN)
	if err != nil {
		return err
	}

	return nil
}

func (m *VirtualNodeTest) CreatePodGroup(ctx context.Context, f *framework.Framework, group manifest.PodGroupInfo) {
	By(fmt.Sprintf("create pod group %s", group.GroupLabel), func() {
		for _, pod := range group.Pods {
			err := f.K8sClient.Create(ctx, pod)
			Expect(err).NotTo(HaveOccurred())
			m.Pods[pod.Name] = pod
		}
	})
}

func (m *VirtualNodeTest) Cleanup(ctx context.Context, f *framework.Framework) {
	var deletionErrors []error

	for _, pod := range m.Pods {
		if pod == nil {
			continue
		}
		By(fmt.Sprintf("delete Pod %s", pod.Name), func() {
			if err := f.K8sClient.Delete(ctx, pod,
				client.PropagationPolicy(metav1.DeletePropagationForeground), client.GracePeriodSeconds(0)); err != nil {
				if apierrs.IsNotFound(err) {
					f.Logger.Info("Pod already deleted",
						zap.String("Pod", pod.Name))
					return
				}
				f.Logger.Error("failed to delete pod",
					zap.String("deployment", pod.Name),
					zap.Error(err))
				deletionErrors = append(deletionErrors, err)
				return
			}
		})
	}

	for _, dp := range m.Deployments {
		if dp == nil {
			continue
		}
		By(fmt.Sprintf("delete Deployment %s", dp.Name), func() {
			if err := f.K8sClient.Delete(ctx, dp,
				client.PropagationPolicy(metav1.DeletePropagationForeground), client.GracePeriodSeconds(0)); err != nil {
				if apierrs.IsNotFound(err) {
					f.Logger.Info("Deployment already deleted",
						zap.String("deployment", dp.Name))
					return
				}
				f.Logger.Error("failed to delete deployment",
					zap.String("deployment", dp.Name),
					zap.Error(err))
				deletionErrors = append(deletionErrors, err)
				return
			}

			By(fmt.Sprintf("Wait for deployment to be deleted: %s", dp.Name), func() {
				if err := f.DPManager.WaitUntilDeploymentDeleted(ctx, dp); err != nil {
					f.Logger.Error("failed while waiting for deployment deletion",
						zap.String("virtual node", dp.Name),
						zap.Error(err))
					deletionErrors = append(deletionErrors, err)
				}
			})
			delete(m.Deployments, dp.Name)
		})
	}

	for _, vn := range m.VirtualNodes {
		By(fmt.Sprintf("Delete virtual node %s", vn.Name), func() {
			if err := f.K8sClient.Delete(ctx, vn,
				client.PropagationPolicy(metav1.DeletePropagationForeground), client.GracePeriodSeconds(0)); err != nil {

				if apierrs.IsNotFound(err) {
					f.Logger.Info("Virtual node already deleted",
						zap.String("virtual node", vn.Name))
					return
				}
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
			delete(m.VirtualNodes, vn.Name)
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

	if m.CloudMapNameSpace != "" {
		//Delete CloudMap Namespace
		By(fmt.Sprintf("delete cloudMap namespace %s", m.CloudMapNameSpace), func() {
			var cmNamespaceID string
			f.CloudMapClient.ListNamespacesPagesWithContext(ctx, &servicediscovery.ListNamespacesInput{}, func(output *servicediscovery.ListNamespacesOutput, b bool) bool {
				for _, ns := range output.Namespaces {
					if aws.StringValue(ns.Name) == m.CloudMapNameSpace {
						cmNamespaceID = aws.StringValue(ns.Id)
						return true
					}
				}
				return false
			})
			if _, err := f.CloudMapClient.DeleteNamespaceWithContext(ctx, &servicediscovery.DeleteNamespaceInput{
				Id: aws.String(cmNamespaceID),
			}); err != nil {
				f.Logger.Error("failed to delete cloudMap namespace",
					zap.String("namespaceID", cmNamespaceID),
					zap.Error(err),
				)
			}
			m.CloudMapNameSpace = ""
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

func (m *VirtualNodeTest) CheckInAWSWithExpectedPods(ctx context.Context, f *framework.Framework, ms *appmesh.Mesh, vn *appmesh.VirtualNode, expectedRegisteredPods []*corev1.Pod) error {
	return f.VNManager.CheckVirtualNodeInCloudMapWithExpectedRegisteredPods(ctx, ms, vn, expectedRegisteredPods)
}

func (m *VirtualNodeTest) ValidateBackends(ctx context.Context, f *framework.Framework, ms *appmesh.Mesh, vn *appmesh.VirtualNode, expectedBackends []types.NamespacedName) error {
	return f.VNManager.ValidateVirtualNodeBackends(ctx, ms, vn, expectedBackends)
}
