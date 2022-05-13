package sidecar

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/k8s"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/manifest"
	"github.com/aws/aws-sdk-go/aws"

	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/algorithm"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultFrontendImage = "public.ecr.aws/b7m0w2t6/color-fe-app:1.0.0"
	defaultBackendImage  = "public.ecr.aws/b7m0w2t6/color-be-app:1.0.0"

	sidecarTest      = "sidecar-e2e"
	AppContainerPort = 8080
)

type SidecarStack struct {
	mesh      *appmesh.Mesh
	namespace *corev1.Namespace

	FrontendVN *appmesh.VirtualNode
	FrontendDP *appsv1.Deployment

	BackendVN  *appmesh.VirtualNode
	BackendVS  *appmesh.VirtualService
	BackendDP  *appsv1.Deployment
	BackendSVC *corev1.Service
}

func (s *SidecarStack) DeploySidecarStack(ctx context.Context, f *framework.Framework) {
	s.createSidecarStackMeshAndNamespace(ctx, f)
	mb := &manifest.ManifestBuilder{
		Namespace:            s.namespace.Name,
		ServiceDiscoveryType: manifest.DNSServiceDiscovery,
	}
	s.createSidecarStackFrontendResources(ctx, f, mb)
	s.createSidecarStackBackendResources(ctx, f, mb)
	s.assignBackendSVC(ctx, f)
}

func (s *SidecarStack) CleanupSidecarStack(ctx context.Context, f *framework.Framework) {
	var deletionErrors []error

	if errs := s.revokeBackendVS(ctx, f); len(errs) != 0 {
		deletionErrors = append(deletionErrors, errs...)
	}
	if errs := s.deleteBackendResourcesForSidecarStack(ctx, f); len(errs) != 0 {
		deletionErrors = append(deletionErrors, errs...)
	}
	if errs := s.deleteSidecarStackFrontendResources(ctx, f); len(errs) != 0 {
		deletionErrors = append(deletionErrors, errs...)
	}
	if errs := s.deleteSidecarMeshAndNamespace(ctx, f); len(errs) != 0 {
		deletionErrors = append(deletionErrors, errs...)
	}
	for _, err := range deletionErrors {
		f.Logger.Error("clean up failed", zap.Error(err))
	}

	Expect(len(deletionErrors)).To(BeZero())
}

func (s *SidecarStack) CheckSidecarBehavior(ctx context.Context, f *framework.Framework) {
	By("verify sidecar behavior", func() {
		sel := labels.Set(s.FrontendDP.Spec.Selector.MatchLabels)
		pods := &corev1.PodList{}

		err := f.K8sClient.List(ctx, pods, client.InNamespace(s.FrontendDP.Namespace), client.MatchingLabelsSelector{Selector: sel.AsSelector()})
		if err != nil {
			f.Logger.Error(fmt.Sprintf("failed to get pods for Deployment: %v", k8s.NamespacedName(s.FrontendDP).String()), zap.Error(err))
		}

		for _, pod := range pods.Items {
			errCh, readyCh := make(chan error), make(chan struct{})

			portForwarder, err := k8s.NewPortForwarder(ctx, f.RestCfg, &pod, []string{fmt.Sprintf("%d:%d", 9901, 9901), fmt.Sprintf("%d:%d", AppContainerPort, AppContainerPort)}, readyCh)
			if err != nil {
				f.Logger.Error("failed to initialize port forwarder", zap.Error(err))
			}

			go func() {
				errCh <- portForwarder.ForwardPorts()
			}()

			<-readyCh

			res, err := http.Get("http://localhost:9901/server_info")
			if err != nil {
				f.Logger.Error("GET request failed", zap.Error(err))
			}

			defer res.Body.Close()
			body, _ := io.ReadAll(res.Body)

			fmt.Printf("%v\n", string(body))

			res, err = http.Get(fmt.Sprintf("http://localhost:%d/color", AppContainerPort))
			if err != nil {
				f.Logger.Error("GET request failed", zap.Error(err))
			}

			Expect(res.Status).To(Equal(200))
			// timing out after one minute should succeed
			// time.Sleep(60 * time.Second)
			// res, err = http.Get(podURL)
		}
	})
}

func (s *SidecarStack) createSidecarStackMeshAndNamespace(ctx context.Context, f *framework.Framework) {
	By("create a Mesh", func() {
		meshName := sidecarTest
		mesh := &appmesh.Mesh{
			ObjectMeta: metav1.ObjectMeta{
				Name: meshName,
			},
			Spec: appmesh.MeshSpec{
				NamespaceSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"mesh": meshName,
					},
				},
			},
		}
		err := f.K8sClient.Create(ctx, mesh)
		Expect(err).NotTo(HaveOccurred())
		s.mesh = mesh
	})

	By(fmt.Sprintf("wait for Mesh %s become active", s.mesh.Name), func() {
		mesh, err := f.MeshManager.WaitUntilMeshActive(ctx, s.mesh)
		Expect(err).NotTo(HaveOccurred())
		s.mesh = mesh
	})

	By("allocate test Namespace", func() {
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: sidecarTest,
			},
		}
		err := f.K8sClient.Create(ctx, namespace)
		Expect(err).NotTo(HaveOccurred())
		s.namespace = namespace
	})

	By("label Namespace with AppMesh inject", func() {
		oldNS := s.namespace.DeepCopy()
		s.namespace.Labels = algorithm.MergeStringMap(map[string]string{
			"appmesh.k8s.aws/sidecarInjectorWebhook": "enabled",
			"mesh":                                   s.mesh.Name,
		}, s.namespace.Labels)
		err := f.K8sClient.Patch(ctx, s.namespace, client.MergeFrom(oldNS))
		Expect(err).NotTo(HaveOccurred())
	})
}

func (s *SidecarStack) createSidecarStackFrontendResources(ctx context.Context, f *framework.Framework, mb *manifest.ManifestBuilder) {
	By("create frontend resources", func() {
		vnBuilder := &manifest.VNBuilder{
			ServiceDiscoveryType: manifest.DNSServiceDiscovery,
			Namespace:            sidecarTest,
		}

		By("create frontend VirtualNode", func() {
			vn := vnBuilder.BuildVirtualNode(
				"frontend",
				[]types.NamespacedName{},
				[]appmesh.Listener{vnBuilder.BuildListener("http", 8080)},
				&appmesh.BackendDefaults{},
			)

			err := f.K8sClient.Create(ctx, vn)
			Expect(err).NotTo(HaveOccurred())
			s.FrontendVN = vn

			_, err = f.VNManager.WaitUntilVirtualNodeActive(ctx, s.FrontendVN)
			if err != nil {
				f.Logger.Error("failed while waiting for VirtualNode", zap.Error(err))
				return
			}
		})

		By("create frontend Deployment", func() {
			dp := mb.BuildDeployment(
				"frontend",
				1,
				mb.BuildContainerSpec([]manifest.ContainerInfo{
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
								Name:  "COLOR_HOST",
								Value: fmt.Sprintf("backend.%s.svc.cluster.local:%d", sidecarTest, AppContainerPort),
							},
						},
					},
				}),
				map[string]string{},
			)

			err := f.K8sClient.Create(ctx, dp)
			Expect(err).NotTo(HaveOccurred())
			s.FrontendDP = dp

			_, err = f.DPManager.WaitUntilDeploymentReady(ctx, s.FrontendDP)
			if err != nil {
				f.Logger.Error("failed while waiting for Frontend VirtualNode to become active", zap.Error(err))
				return
			}
		})
	})
}

func (s *SidecarStack) createSidecarStackBackendResources(ctx context.Context, f *framework.Framework, mb *manifest.ManifestBuilder) {
	By("create backend resources", func() {
		vnBuilder := &manifest.VNBuilder{
			ServiceDiscoveryType: manifest.DNSServiceDiscovery,
			Namespace:            sidecarTest,
		}

		By("create backend VirtualNode", func() {
			vn := vnBuilder.BuildVirtualNode(
				"backend",
				[]types.NamespacedName{},
				[]appmesh.Listener{vnBuilder.BuildListener("http", 8080)},
				&appmesh.BackendDefaults{},
			)

			err := f.K8sClient.Create(ctx, vn)
			Expect(err).NotTo(HaveOccurred())
			s.BackendVN = vn

			_, err = f.VNManager.WaitUntilVirtualNodeActive(ctx, s.BackendVN)
			if err != nil {
				f.Logger.Error("failed while waiting for VirtualNode", zap.Error(err))
				return
			}
		})

		By("create backend VirtualService", func() {
			ns := sidecarTest
			vs := &appmesh.VirtualService{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: ns,
					Name:      "backend",
				},
				Spec: appmesh.VirtualServiceSpec{
					AWSName: aws.String(fmt.Sprintf("%s.%s.svc.cluster.local", "backend", ns)),
					Provider: &appmesh.VirtualServiceProvider{
						VirtualNode: &appmesh.VirtualNodeServiceProvider{
							VirtualNodeRef: &appmesh.VirtualNodeReference{
								Namespace: &ns,
								Name:      "backend",
							},
						},
					},
				},
			}

			err := f.K8sClient.Create(ctx, vs)
			Expect(err).NotTo(HaveOccurred())
			s.BackendVS = vs

			_, err = f.VSManager.WaitUntilVirtualServiceActive(ctx, s.BackendVS)
			if err != nil {
				f.Logger.Error("failed to check backend VirtualService", zap.Error(err))
				return
			}
		})

		By("create backend Deployment", func() {
			dp := mb.BuildDeployment(
				"backend",
				1,
				mb.BuildContainerSpec([]manifest.ContainerInfo{
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
						},
					},
				}),
				map[string]string{},
			)

			err := f.K8sClient.Create(ctx, dp)
			Expect(err).NotTo(HaveOccurred())
			s.BackendDP = dp

			_, err = f.DPManager.WaitUntilDeploymentReady(ctx, s.BackendDP)
			if err != nil {
				f.Logger.Error("failed while waiting for backend VirtualNode to become active", zap.Error(err))
				return
			}
		})

		By("create service for backend VirtualNode", func() {
			svc := mb.BuildServiceWithSelector("backend", AppContainerPort, AppContainerPort)

			err := f.K8sClient.Create(ctx, svc)
			Expect(err).NotTo(HaveOccurred())
			s.BackendSVC = svc
		})
	})
}

func (s *SidecarStack) assignBackendSVC(ctx context.Context, f *framework.Framework) {
	By(fmt.Sprintf("assigning backend"), func() {
		vnNew := s.FrontendVN.DeepCopy()
		vnNew.Spec.Backends = []appmesh.Backend{
			appmesh.Backend{
				VirtualService: appmesh.VirtualServiceBackend{
					VirtualServiceRef: &appmesh.VirtualServiceReference{
						Namespace: aws.String(s.BackendSVC.Namespace),
						Name:      s.BackendSVC.Name,
					},
				},
			},
		}

		err := f.K8sClient.Patch(ctx, vnNew, client.MergeFrom(s.FrontendVN))
		Expect(err).NotTo(HaveOccurred())
		s.FrontendVN = vnNew
	})
}

func (s *SidecarStack) deleteSidecarMeshAndNamespace(ctx context.Context, f *framework.Framework) []error {
	var deletionErrors []error

	if s.namespace != nil {
		By(fmt.Sprintf("delete namespace %s", s.namespace.Name), func() {
			if err := f.K8sClient.Delete(ctx, s.namespace,
				client.PropagationPolicy(metav1.DeletePropagationForeground), client.GracePeriodSeconds(0)); err != nil {
				f.Logger.Error("failed to delete namespace",
					zap.String("namespace", s.namespace.Name),
					zap.Error(err))
				deletionErrors = append(deletionErrors, err)
				return
			}

			if err := f.NSManager.WaitUntilNamespaceDeleted(ctx, s.namespace); err != nil {
				f.Logger.Error("failed to wait namespace deletion",
					zap.String("namespace", s.namespace.Name),
					zap.Error(err))
				deletionErrors = append(deletionErrors, err)
			}
		})
	}

	if s.mesh != nil {
		By(fmt.Sprintf("delete mesh %s", s.mesh.Name), func() {
			if err := f.K8sClient.Delete(ctx, s.mesh,
				client.PropagationPolicy(metav1.DeletePropagationForeground), client.GracePeriodSeconds(0)); err != nil {
				f.Logger.Error("failed to delete mesh",
					zap.String("mesh", s.mesh.Name),
					zap.Error(err))
				deletionErrors = append(deletionErrors, err)
				return
			}

			if err := f.MeshManager.WaitUntilMeshDeleted(ctx, s.mesh); err != nil {
				f.Logger.Error("failed to wait mesh deletion",
					zap.String("mesh", s.mesh.Name),
					zap.Error(err))
				deletionErrors = append(deletionErrors, err)
			}
		})
	}

	return deletionErrors
}

func (s *SidecarStack) deleteSidecarStackFrontendResources(ctx context.Context, f *framework.Framework) []error {
	var deletionErrors []error

	By("delete frontend resources", func() {
		By("delete frontend Deployment", func() {
			if err := f.K8sClient.Delete(ctx, s.FrontendDP,
				client.PropagationPolicy(metav1.DeletePropagationForeground), client.GracePeriodSeconds(0)); err != nil {
				f.Logger.Error("failed to delete Deployment",
					zap.String("namespace", s.FrontendDP.Namespace),
					zap.String("name", s.FrontendDP.Name),
					zap.Error(err),
				)
				deletionErrors = append(deletionErrors, err)
			}

			if err := f.DPManager.WaitUntilDeploymentDeleted(ctx, s.FrontendDP); err != nil {
				f.Logger.Error("failed to delete frontend deployment", zap.Error(err))
				return
			}
		})

		By("delete frontend VirtualNode", func() {
			if err := f.K8sClient.Delete(ctx, s.FrontendVN); err != nil {
				f.Logger.Error("failed to delete VirtualNode",
					zap.String("namespace", s.FrontendVN.Namespace),
					zap.String("name", s.FrontendVN.Name),
					zap.Error(err),
				)
				deletionErrors = append(deletionErrors, err)
			}

			if err := f.VNManager.WaitUntilVirtualNodeDeleted(ctx, s.FrontendVN); err != nil {
				f.Logger.Error("failed to delete frontend VirtualNode", zap.Error(err))
				return
			}
		})
	})

	return deletionErrors
}

func (s *SidecarStack) deleteBackendResourcesForSidecarStack(ctx context.Context, f *framework.Framework) []error {
	var deletionErrors []error

	By("delete backend resources", func() {
		By("delete backend Deployment", func() {
			if err := f.K8sClient.Delete(ctx, s.BackendDP,
				client.PropagationPolicy(metav1.DeletePropagationForeground), client.GracePeriodSeconds(0)); err != nil {
				f.Logger.Error("failed to delete Deployment",
					zap.String("namespace", s.BackendDP.Namespace),
					zap.String("name", s.BackendDP.Name),
					zap.Error(err),
				)
				deletionErrors = append(deletionErrors, err)
			}

			if err := f.DPManager.WaitUntilDeploymentDeleted(ctx, s.BackendDP); err != nil {
				f.Logger.Error("failed to delete backend deployment", zap.Error(err))
				return
			}
		})

		By("delete backend VirtualService", func() {
			if err := f.K8sClient.Delete(ctx, s.BackendVS); err != nil {
				f.Logger.Error("failed to delete VirtualService",
					zap.String("namespace", s.BackendVS.Namespace),
					zap.String("name", s.BackendVS.Name),
					zap.Error(err),
				)
				deletionErrors = append(deletionErrors, err)
			}

			if err := f.VSManager.WaitUntilVirtualServiceDeleted(ctx, s.BackendVS); err != nil {
				f.Logger.Error("failed to delete backend VirtualService", zap.Error(err))
				return
			}
		})

		By("delete backend VirtualNode", func() {
			if err := f.K8sClient.Delete(ctx, s.BackendVN); err != nil {
				f.Logger.Error("failed to delete VirtualNode",
					zap.String("namespace", s.BackendVN.Namespace),
					zap.String("name", s.BackendVN.Name),
					zap.Error(err),
				)
				deletionErrors = append(deletionErrors, err)
			}

			if err := f.VNManager.WaitUntilVirtualNodeDeleted(ctx, s.BackendVN); err != nil {
				f.Logger.Error("failed to delete backend VirtualNode", zap.Error(err))
				return
			}
		})

		By("delete backend Service", func() {
			if err := f.K8sClient.Delete(ctx, s.BackendSVC); err != nil {
				f.Logger.Error("failed to delete Service",
					zap.String("namespace", s.BackendSVC.Namespace),
					zap.String("name", s.BackendSVC.Name),
					zap.Error(err),
				)
				deletionErrors = append(deletionErrors, err)
			}
		})
	})

	return deletionErrors
}

func (s *SidecarStack) revokeBackendVS(ctx context.Context, f *framework.Framework) []error {
	var deletionErrors []error

	By("revoking backend", func() {
		vnNew := s.FrontendVN.DeepCopy()
		vnNew.Spec.Backends = nil

		err := f.K8sClient.Patch(ctx, vnNew, client.MergeFrom(s.FrontendVN))
		if err != nil {
			f.Logger.Error("failed to revoke VirtualNode backend access",
				zap.String("namespace", s.FrontendVN.Namespace),
				zap.String("name", s.FrontendVN.Name),
				zap.Error(err),
			)

			deletionErrors = append(deletionErrors, err)
		}
	})

	return deletionErrors
}
