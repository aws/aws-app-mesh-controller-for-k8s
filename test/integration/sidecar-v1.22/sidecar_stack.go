package sidecar_v1_22

import (
	"context"
	"fmt"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/inject"

	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	defaultImage = "public.ecr.aws/b7m0w2t6/color-be-app:2.0.2"
)

type SidecarStack struct {
	appContainerPort int
	color            string
	k8client         *kubernetes.Clientset
	testName         string

	mesh       *appmesh.Mesh
	namespace  *corev1.Namespace
	frontendVN *appmesh.VirtualNode
	frontendDP *appsv1.Deployment
}

func newSidecarStack_v1_22(name, kubecfg string, port int) (*SidecarStack, error) {
	cfg, err := clientcmd.BuildConfigFromFlags("", kubecfg)
	if err != nil {
		return nil, err
	}

	k8client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	return &SidecarStack{
		appContainerPort: port,
		color:            "blue",
		testName:         name,
		k8client:         k8client,
	}, nil
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
		vn := &appmesh.VirtualNode{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "front",
				Namespace: s.testName,
			},
			Spec: appmesh.VirtualNodeSpec{
				Listeners: []appmesh.Listener{
					{
						PortMapping: appmesh.PortMapping{
							Port:     appmesh.PortNumber(int64(8090)),
							Protocol: appmesh.PortProtocolHTTP,
						},
						HealthCheck: &appmesh.HealthCheckPolicy{
							IntervalMillis:     int64(5000),
							HealthyThreshold:   int64(2),
							Protocol:           appmesh.PortProtocolHTTP,
							Path:               newStringPtr("/ping"),
							TimeoutMillis:      int64(2000),
							UnhealthyThreshold: int64(2),
						},
					},
				},
				PodSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": "front",
					},
				},
				ServiceDiscovery: &appmesh.ServiceDiscovery{
					DNS: &appmesh.DNSServiceDiscovery{
						Hostname: fmt.Sprintf("front.%s.svc.cluster.local", s.testName),
					},
				},
			},
		}

		err := f.K8sClient.Create(ctx, vn)
		Expect(err).NotTo(HaveOccurred())

		_, err = f.VNManager.WaitUntilVirtualNodeActive(ctx, vn)
		Expect(err).NotTo(HaveOccurred())

		s.frontendVN = vn
	})

	By("create frontend Deployment", func() {
		dp := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "front",
				Namespace: s.testName,
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{MatchLabels: map[string]string{
					"app": "front",
				}},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": "front",
						},
						Annotations: map[string]string{
							inject.AppMeshIPV6Annotation: "disabled", // for github action compatibility
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "app",
								Image: defaultImage,
								Ports: []corev1.ContainerPort{
									{
										ContainerPort: int32(s.appContainerPort),
									},
								},
								Env: []corev1.EnvVar{
									{
										Name:  "COLOR",
										Value: s.color,
									},
									{
										Name:  "PORT",
										Value: fmt.Sprintf("%d", s.appContainerPort),
									},
								},
							},
						},
					},
				},
			},
		}

		err := f.K8sClient.Create(ctx, dp)
		Expect(err).NotTo(HaveOccurred())

		s.frontendDP = dp
	})
}

func (s *SidecarStack) cleanup(ctx context.Context, f *framework.Framework) {
	if err := f.K8sClient.Delete(ctx, s.frontendVN); err != nil {
		f.Logger.Error("failed to delete frontend virtual node")
	}

	if err := f.K8sClient.Delete(ctx, s.frontendDP); err != nil {
		f.Logger.Error("failed to delete frontend deployment")
	}

	if err := f.K8sClient.Delete(ctx, s.namespace); err != nil {
		f.Logger.Error("failed to delete namespace")
	}

	if err := f.K8sClient.Delete(ctx, s.mesh); err != nil {
		f.Logger.Error("failed to delete mesh")
	}
}

func newStringPtr(s string) *string {
	return &s
}
