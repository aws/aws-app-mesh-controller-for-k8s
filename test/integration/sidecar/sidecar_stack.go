package sidecar

import (
	"context"
	"fmt"

	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/manifest"
	"github.com/aws/aws-sdk-go/aws"

	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultFrontendImage = "public.ecr.aws/b7m0w2t6/color-fe-app:2.0.0"
	defaultBackendImage  = "public.ecr.aws/b7m0w2t6/color-be-app:2.0.0"
	AppContainerPort     = 8080
)

type SidecarStack struct {
	testName string
	mb       *manifest.ManifestBuilder
	vnB      *manifest.VNBuilder

	mesh      *appmesh.Mesh
	namespace *corev1.Namespace

	frontendVN *appmesh.VirtualNode
	frontendDP *appsv1.Deployment

	backendVN  *appmesh.VirtualNode
	backendVS  *appmesh.VirtualService
	backendDP  *appsv1.Deployment
	backendSVC *corev1.Service
}

func newSidecarStack(name string) *SidecarStack {
	mb := &manifest.ManifestBuilder{
		Namespace:            name,
		ServiceDiscoveryType: manifest.DNSServiceDiscovery,
	}
	vnB := &manifest.VNBuilder{
		Namespace:            name,
		ServiceDiscoveryType: manifest.DNSServiceDiscovery,
	}

	return &SidecarStack{
		testName: name,
		mb:       mb,
		vnB:      vnB,
	}
}

func (s *SidecarStack) createMeshAndNamespace(ctx context.Context, f *framework.Framework) {
	By("create Mesh", func() {
		mesh := &appmesh.Mesh{
			ObjectMeta: metav1.ObjectMeta{Name: s.testName},
			Spec: appmesh.MeshSpec{
				NamespaceSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"mesh": s.testName,
					},
				},
			},
		}

		err := f.K8sClient.Create(ctx, mesh)
		Expect(err).NotTo(HaveOccurred())

		_, err = f.MeshManager.WaitUntilMeshActive(ctx, mesh)
		Expect(err).NotTo(HaveOccurred())

		s.mesh = mesh
	})

	By("create Namespace", func() {
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: s.testName,
				Labels: map[string]string{
					"appmesh.k8s.aws/sidecarInjectorWebhook": "enabled",
					"mesh":                                   s.mesh.Name,
				},
			},
		}

		err := f.K8sClient.Create(ctx, namespace)
		Expect(err).NotTo(HaveOccurred())

		s.namespace = namespace
	})
}

func (s *SidecarStack) createFrontendResources(ctx context.Context, f *framework.Framework) {
	By("create frontend VirtualNode", func() {
		vn := s.vnB.BuildVirtualNode(
			"frontend",
			[]types.NamespacedName{
				{
					Namespace: s.backendSVC.Namespace,
					Name:      s.backendSVC.Name,
				},
			},
			[]appmesh.Listener{s.vnB.BuildListener("http", 8080)},
			&appmesh.BackendDefaults{},
		)

		err := f.K8sClient.Create(ctx, vn)
		Expect(err).NotTo(HaveOccurred())

		_, err = f.VNManager.WaitUntilVirtualNodeActive(ctx, vn)
		Expect(err).NotTo(HaveOccurred())

		s.frontendVN = vn
	})

	By("create frontend Deployment", func() {
		dp := s.mb.BuildDeployment(
			"frontend",
			1,
			s.mb.BuildContainerSpec([]manifest.ContainerInfo{
				{
					Name:          "app",
					AppImage:      defaultFrontendImage,
					ContainerPort: AppContainerPort,
					Env: []corev1.EnvVar{
						{
							Name:  "PORT",
							Value: fmt.Sprintf("%d", AppContainerPort),
						},
						{
							Name:  "HOST",
							Value: fmt.Sprintf("backend.%s.svc.cluster.local:%d", s.testName, AppContainerPort),
						},
						{
							Name:  "NAMESPACE",
							Value: s.testName,
						},
					},
				},
			}),
			map[string]string{},
		)

		err := f.K8sClient.Create(ctx, dp)
		Expect(err).NotTo(HaveOccurred())

		_, err = f.DPManager.WaitUntilDeploymentReady(ctx, dp)
		Expect(err).NotTo(HaveOccurred())

		s.frontendDP = dp
	})
}

func (s *SidecarStack) createBackendResources(ctx context.Context, f *framework.Framework) {
	By("create backend VirtualNode", func() {
		vn := s.vnB.BuildVirtualNode(
			"backend",
			[]types.NamespacedName{},
			[]appmesh.Listener{s.vnB.BuildListener("http", 8080)},
			&appmesh.BackendDefaults{},
		)

		err := f.K8sClient.Create(ctx, vn)
		Expect(err).NotTo(HaveOccurred())

		_, err = f.VNManager.WaitUntilVirtualNodeActive(ctx, vn)
		Expect(err).NotTo(HaveOccurred())

		s.backendVN = vn
	})

	By("create backend VirtualService", func() {
		vs := &appmesh.VirtualService{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: s.testName,
				Name:      "backend",
			},
			Spec: appmesh.VirtualServiceSpec{
				AWSName: aws.String(fmt.Sprintf("backend.%s.svc.cluster.local", s.testName)),
				Provider: &appmesh.VirtualServiceProvider{
					VirtualNode: &appmesh.VirtualNodeServiceProvider{
						VirtualNodeRef: &appmesh.VirtualNodeReference{
							Namespace: &s.testName,
							Name:      "backend",
						},
					},
				},
			},
		}

		err := f.K8sClient.Create(ctx, vs)
		Expect(err).NotTo(HaveOccurred())

		_, err = f.VSManager.WaitUntilVirtualServiceActive(ctx, vs)
		Expect(err).NotTo(HaveOccurred())

		s.backendVS = vs
	})

	By("create backend Deployment", func() {
		dp := s.mb.BuildDeployment(
			"backend",
			1,
			s.mb.BuildContainerSpec([]manifest.ContainerInfo{
				{
					Name:          "app",
					AppImage:      defaultBackendImage,
					ContainerPort: AppContainerPort,
					Env: []corev1.EnvVar{
						{
							Name:  "PORT",
							Value: fmt.Sprintf("%d", AppContainerPort),
						},
						{
							Name:  "COLOR",
							Value: "red",
						},
						{
							Name:  "Namespace",
							Value: s.testName,
						},
					},
				},
			}),
			map[string]string{},
		)

		err := f.K8sClient.Create(ctx, dp)
		Expect(err).NotTo(HaveOccurred())

		_, err = f.DPManager.WaitUntilDeploymentReady(ctx, dp)
		Expect(err).NotTo(HaveOccurred())

		s.backendDP = dp
	})

	By("create backend Service", func() {
		svc := s.mb.BuildServiceWithSelector("backend", AppContainerPort, AppContainerPort)

		err := f.K8sClient.Create(ctx, svc)
		Expect(err).NotTo(HaveOccurred())
		s.backendSVC = svc
	})

}

func (s *SidecarStack) cleanup(ctx context.Context, f *framework.Framework) {
	if err := f.K8sClient.Delete(
		ctx,
		s.namespace,
		client.PropagationPolicy(metav1.DeletePropagationForeground), client.GracePeriodSeconds(0),
	); err != nil {
		f.Logger.Error("failed to delete namespace")
	}

	if err := f.K8sClient.Delete(
		ctx,
		s.mesh,
		client.PropagationPolicy(metav1.DeletePropagationForeground), client.GracePeriodSeconds(0),
	); err != nil {
		f.Logger.Error("failed to delete mesh")
	}
}
