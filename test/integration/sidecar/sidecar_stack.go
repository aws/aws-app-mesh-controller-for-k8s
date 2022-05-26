package sidecar

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"

	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultFrontendImage = "public.ecr.aws/b7m0w2t6/color-fe-app:2.0.3"
	defaultBackendImage  = "public.ecr.aws/b7m0w2t6/color-be-app:2.0.2"
)

type SidecarStack struct {
	appContainerPort int
	color            string
	testName         string

	mesh       *appmesh.Mesh
	namespace  *corev1.Namespace
	frontendVN *appmesh.VirtualNode
	frontendDP *appsv1.Deployment
	backendVN  *appmesh.VirtualNode
	backendVS  *appmesh.VirtualService
	backendDP  *appsv1.Deployment
	backendSVC *corev1.Service
}

func newSidecarStack(name string, appContainerPort int, color string) *SidecarStack {
	return &SidecarStack{
		appContainerPort: appContainerPort,
		color:            color,
		testName:         name,
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
				EgressFilter: &appmesh.EgressFilter{
					Type: appmesh.EgressFilterTypeAllowAll,
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

	By("create Role", func() {
		role := &rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "default",
				Namespace: s.testName,
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"pods"},
					Verbs:     []string{"list", "update"},
				},
			},
		}

		_, err := f.Clientset.RbacV1().Roles(s.testName).Create(ctx, role, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
	})

	By("create RoleBinding", func() {
		roleB := &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "default",
				Namespace: s.testName,
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      "default",
					Namespace: s.testName,
				},
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "Role",
				Name:     "default",
			},
		}

		_, err := f.Clientset.RbacV1().RoleBindings(s.testName).Create(ctx, roleB, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
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
				Backends: []appmesh.Backend{
					{
						appmesh.VirtualServiceBackend{
							VirtualServiceRef: &appmesh.VirtualServiceReference{
								Name: "color",
							},
						},
					},
				},
				Listeners: []appmesh.Listener{
					{
						PortMapping: appmesh.PortMapping{
							Port:     appmesh.PortNumber(int64(8080)),
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
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "app",
								Image: defaultFrontendImage,
								Ports: []corev1.ContainerPort{
									{
										ContainerPort: int32(s.appContainerPort),
									},
								},
								Env: []corev1.EnvVar{
									{
										Name:  "HOST",
										Value: fmt.Sprintf("color.%s.svc.cluster.local:%d", s.testName, s.appContainerPort),
									},
									{
										Name:  "NAMESPACE",
										Value: s.testName,
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

		_, err = f.DPManager.WaitUntilDeploymentReady(ctx, dp)
		Expect(err).NotTo(HaveOccurred())

		s.frontendDP = dp
	})
}

func (s *SidecarStack) createBackendResources(ctx context.Context, f *framework.Framework) {
	By("create backend VirtualNode", func() {
		vn := &appmesh.VirtualNode{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "blue",
				Namespace: s.testName,
			},
			Spec: appmesh.VirtualNodeSpec{
				Listeners: []appmesh.Listener{
					{
						PortMapping: appmesh.PortMapping{
							Port:     appmesh.PortNumber(int64(8080)),
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
						"app":     "color",
						"version": "blue",
					},
				},
				ServiceDiscovery: &appmesh.ServiceDiscovery{
					DNS: &appmesh.DNSServiceDiscovery{
						Hostname: fmt.Sprintf("color-blue.%s.svc.cluster.local", s.testName),
					},
				},
			},
		}

		err := f.K8sClient.Create(ctx, vn)
		Expect(err).NotTo(HaveOccurred())

		_, err = f.VNManager.WaitUntilVirtualNodeActive(ctx, vn)
		Expect(err).NotTo(HaveOccurred())

		s.backendVN = vn
	})

	By("create backend VirtualService", func() {
		vs := &appmesh.VirtualService{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "color",
				Namespace: s.testName,
			},
			Spec: appmesh.VirtualServiceSpec{
				AWSName: aws.String(fmt.Sprintf("color.%s.svc.cluster.local", s.testName)),
				Provider: &appmesh.VirtualServiceProvider{
					VirtualNode: &appmesh.VirtualNodeServiceProvider{
						VirtualNodeRef: &appmesh.VirtualNodeReference{
							Name: "blue",
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
		dp := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "blue",
				Namespace: s.testName,
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{MatchLabels: map[string]string{
					"app":     "color",
					"version": "blue",
				}},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app":     "color",
							"version": "blue",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "app",
								Image: defaultBackendImage,
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

		_, err = f.DPManager.WaitUntilDeploymentReady(ctx, dp)
		Expect(err).NotTo(HaveOccurred())

		s.backendDP = dp
	})

	By("create color Service", func() {
		svc := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "color",
				Namespace: s.testName,
			},
			Spec: corev1.ServiceSpec{
				Ports: []corev1.ServicePort{
					{
						Port: int32(s.appContainerPort),
					},
				},
			},
		}

		err := f.K8sClient.Create(ctx, svc)
		Expect(err).NotTo(HaveOccurred())

		s.backendSVC = svc
	})

	By("create color-blue Service", func() {
		svc := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "color-blue",
				Namespace: s.testName,
			},
			Spec: corev1.ServiceSpec{
				Ports: []corev1.ServicePort{
					{
						Port: int32(s.appContainerPort),
					},
				},
				Selector: map[string]string{
					"app":     "color",
					"version": "blue",
				},
			},
		}

		err := f.K8sClient.Create(ctx, svc)
		Expect(err).NotTo(HaveOccurred())
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

func newStringPtr(s string) *string {
	return &s
}
