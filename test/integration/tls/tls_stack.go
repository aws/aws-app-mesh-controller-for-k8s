package tls

import (
	"context"
	"fmt"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/algorithm"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/k8s"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/manifest"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"io/ioutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
)

const (
	//If you're not able to access below images, try to build them based on the app code under "timeout_app"
	//directory and push it to any accessible ECR repo and update the below values
	defaultFrontEndImage = "928111597794.dkr.ecr.us-west-2.amazonaws.com/amazon/tls-e2e:feapp-tstimg1"
	defaultBackEndImage  = "928111597794.dkr.ecr.us-west-2.amazonaws.com/amazon/tls-e2e:beapp-tstimg1"

	tlsTest          = "tls-e2e"
	AppContainerPort = 8080

	tlsConnectionError         = "upstream connect error"
	expectedBackendResponse = "backend"
)

type TLSStack struct {
	// service discovery type
	ServiceDiscoveryType manifest.ServiceDiscoveryType

	// ====== runtime variables ======
	mesh      *appmesh.Mesh
	namespace *corev1.Namespace

	FrontEndVN *appmesh.VirtualNode
	FrontEndDP *appsv1.Deployment

	BackEndVN  *appmesh.VirtualNode
	BackEndDP  *appsv1.Deployment
	BackEndSVC *corev1.Service

	BackEndVS *appmesh.VirtualService
	BackEndVR *appmesh.VirtualRouter
}

func (s *TLSStack) DeployTLSStack(ctx context.Context, f *framework.Framework) {
	s.createTLSStackMeshAndNamespace(ctx, f)
	s.ServiceDiscoveryType = manifest.DNSServiceDiscovery
	mb := &manifest.ManifestBuilder{
		Namespace:            s.namespace.Name,
		ServiceDiscoveryType: s.ServiceDiscoveryType,
	}
	s.createSecretsForTLSStack(ctx, f, mb)
	s.createVirtualNodeResourcesForTLSStack(ctx, f, mb)
	s.createServicesForTLSStack(ctx, f)
	s.assignBackendVSToFrontEndVN(ctx, f)
}

func (s *TLSStack) CleanupTLSStack(ctx context.Context, f *framework.Framework) {
	var deletionErrors []error
	if errs := s.revokeVirtualNodeBackendAccessForTLSStack(ctx, f); len(errs) != 0 {
		deletionErrors = append(deletionErrors, errs...)
	}
	if errs := s.deleteResourcesForTLSStackServices(ctx, f); len(errs) != 0 {
		deletionErrors = append(deletionErrors, errs...)
	}
	if errs := s.deleteResourcesForTLSStackNodes(ctx, f); len(errs) != 0 {
		deletionErrors = append(deletionErrors, errs...)
	}
	if errs := s.deleteTLSMeshAndNamespace(ctx, f); len(errs) != 0 {
		deletionErrors = append(deletionErrors, errs...)
	}
	for _, err := range deletionErrors {
		f.Logger.Error("clean up failed", zap.Error(err))
	}
	Expect(len(deletionErrors)).To(BeZero())
}

func (s *TLSStack) DeployPartialTLSStack(ctx context.Context, f *framework.Framework) {
	s.createTLSStackMeshAndNamespace(ctx, f)
	time.Sleep(30*time.Second)
	s.ServiceDiscoveryType = manifest.DNSServiceDiscovery
	mb := &manifest.ManifestBuilder{
		Namespace:            s.namespace.Name,
		ServiceDiscoveryType: s.ServiceDiscoveryType,
	}
	s.createSecretsForPartialTLSStack(ctx, f, mb)
	s.createVirtualNodeResourcesForPartialTLSStack(ctx, f, mb)
	s.createServicesForTLSStack(ctx, f)
	s.assignBackendVSToFrontEndVN(ctx, f)
}

func (s *TLSStack) CleanupPartialTLSStack(ctx context.Context, f *framework.Framework) {
	var deletionErrors []error
	if errs := s.revokeVirtualNodeBackendAccessForTLSStack(ctx, f); len(errs) != 0 {
		deletionErrors = append(deletionErrors, errs...)
	}
	if errs := s.deleteResourcesForTLSStackServices(ctx, f); len(errs) != 0 {
		deletionErrors = append(deletionErrors, errs...)
	}
	if errs := s.deleteResourcesForPartialTLSStackNodes(ctx, f); len(errs) != 0 {
		deletionErrors = append(deletionErrors, errs...)
	}
	if errs := s.deleteTLSMeshAndNamespace(ctx, f); len(errs) != 0 {
		deletionErrors = append(deletionErrors, errs...)
	}
	for _, err := range deletionErrors {
		f.Logger.Error("clean up failed", zap.Error(err))
	}
	Expect(len(deletionErrors)).To(BeZero())
}

//Check Timeout behavior with and with timeout configured
func (s *TLSStack) CheckTLSBehavior(ctx context.Context, f *framework.Framework, tlsEnabled bool) {
	By(fmt.Sprintf("verify frontend to backend connectivity with TLS enabled"), func() {
		err := s.checkExpectedRouteBehavior(ctx, f, s.FrontEndDP, "tlsroute", tlsEnabled)
		Expect(err).NotTo(HaveOccurred())
	})
}

func (s *TLSStack) createTLSStackMeshAndNamespace(ctx context.Context, f *framework.Framework) {
	By("create a mesh", func() {
		meshName := tlsTest
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

	By(fmt.Sprintf("wait for mesh %s become active", s.mesh.Name), func() {
		mesh, err := f.MeshManager.WaitUntilMeshActive(ctx, s.mesh)
		Expect(err).NotTo(HaveOccurred())
		s.mesh = mesh
	})

	By("allocate test namespace", func() {
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: tlsTest,
			},
		}
		err := f.K8sClient.Create(ctx, namespace)
		Expect(err).NotTo(HaveOccurred())
		s.namespace = namespace
	})

	By("label namespace with appMesh inject", func() {
		oldNS := s.namespace.DeepCopy()
		s.namespace.Labels = algorithm.MergeStringMap(map[string]string{
			"appmesh.k8s.aws/sidecarInjectorWebhook": "enabled",
			"mesh":                                   s.mesh.Name,
		}, s.namespace.Labels)
		err := f.K8sClient.Patch(ctx, s.namespace, client.MergeFrom(oldNS))
		Expect(err).NotTo(HaveOccurred())
	})
}

func (s *TLSStack) deleteTLSMeshAndNamespace(ctx context.Context, f *framework.Framework) []error {
	var deletionErrors []error
	if s.namespace != nil {
		By(fmt.Sprintf("delete namespace: %s", s.namespace.Name), func() {
			if err := f.K8sClient.Delete(ctx, s.namespace,
				client.PropagationPolicy(metav1.DeletePropagationForeground), client.GracePeriodSeconds(0)); err != nil {
				f.Logger.Error("failed to delete namespace",
					zap.String("namespace", s.namespace.Name),
					zap.Error(err))
				deletionErrors = append(deletionErrors, err)
				return
			}

			By(fmt.Sprintf("wait namespace to be deleted: %s", s.namespace.Namespace), func() {
				if err := f.NSManager.WaitUntilNamespaceDeleted(ctx, s.namespace); err != nil {
					f.Logger.Error("failed to wait namespace deletion",
						zap.String("namespace", s.namespace.Name),
						zap.Error(err))
					deletionErrors = append(deletionErrors, err)
				}
			})
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

			By(fmt.Sprintf("wait mesh to be deleted: %s", s.mesh.Name), func() {
				if err := f.MeshManager.WaitUntilMeshDeleted(ctx, s.mesh); err != nil {
					f.Logger.Error("failed to wait mesh deletion",
						zap.String("mesh", s.mesh.Name),
						zap.Error(err))
					deletionErrors = append(deletionErrors, err)
				}
			})
		})
	}

	return deletionErrors
}

func (s *TLSStack) createSecretsForTLSStack(ctx context.Context, f *framework.Framework, mb *manifest.ManifestBuilder) {
	By("create secrets to be used by frontend for validation", func() {
		frontendTLSFiles := []string{"ca_1_cert.pem"}
		secret := mb.BuildK8SSecretsFromPemFile("tls_app/tls/", frontendTLSFiles,
			"ca1-cert-tls", f)
		err := f.K8sClient.Create(ctx, secret)
		Expect(err).NotTo(HaveOccurred())
	})

	By("create secrets to be used by backend", func() {
		backendTLSFiles := []string{"backend-tls_cert_chain.pem", "backend-tls_key.pem"}
		secret := mb.BuildK8SSecretsFromPemFile("tls_app/tls/", backendTLSFiles,
			"backend-tls-tls", f)
		err := f.K8sClient.Create(ctx, secret)
		Expect(err).NotTo(HaveOccurred())
	})
}

func (s *TLSStack) createSecretsForPartialTLSStack(ctx context.Context, f *framework.Framework, mb *manifest.ManifestBuilder) {
	By("create secrets to be used by frontend for validation", func() {
		frontendTLSFiles := []string{"ca_1_cert.pem"}
		secret := mb.BuildK8SSecretsFromPemFile("tls_app/tls/", frontendTLSFiles,
			"ca1-cert-tls", f)
		f.Logger.Error("Secret: ", zap.String("Name: ", secret.Name), zap.String("Namespace: ", secret.Namespace))
		err := f.K8sClient.Create(ctx, secret)
		Expect(err).NotTo(HaveOccurred())
	})
}

func (s *TLSStack) createVirtualNodeResourcesForTLSStack(ctx context.Context, f *framework.Framework, mb *manifest.ManifestBuilder) {
	By("create virtualNode resources", func() {
		vnBuilder := &manifest.VNBuilder{
			ServiceDiscoveryType: manifest.DNSServiceDiscovery,
			Namespace:            tlsTest,
		}

		By(fmt.Sprintf("create frontend virtualNode with TLS enabled"), func() {
			listeners := []appmesh.Listener{vnBuilder.BuildListener("http", 8080)}
			backends := []types.NamespacedName{}
			tlsEnforce := true
			tlsBackendDefaults := &appmesh.BackendDefaults{
				ClientPolicy: &appmesh.ClientPolicy{
					TLS: &appmesh.ClientPolicyTLS{
						Enforce:    &tlsEnforce,
						Ports:      nil,
						Validation: appmesh.TLSValidationContext{
							Trust: appmesh.TLSValidationContextTrust{
								ACM:  nil,
								File: &appmesh.TLSValidationContextFileTrust{CertificateChain: "/certs/ca_1_cert.pem"},
							},
						},
					},
				},
			}
			vn := vnBuilder.BuildVirtualNode("frontend-tls", backends, listeners, tlsBackendDefaults)
			err := f.K8sClient.Create(ctx, vn)
			Expect(err).NotTo(HaveOccurred())
			s.FrontEndVN = vn
		})

		By(fmt.Sprintf("create frontend-tls deployment"), func() {
			//annotations := map[string]string{}
			annotations := map[string]string {
				"appmesh.k8s.aws/secretMounts": "ca1-cert-tls:/certs/",
			}
			dp := mb.BuildDeployment("frontend-tls", 1, defaultFrontEndImage, AppContainerPort, []corev1.EnvVar{}, annotations)
			err := f.K8sClient.Create(ctx, dp)
			Expect(err).NotTo(HaveOccurred())
			s.FrontEndDP = dp
		})

		By(fmt.Sprintf("create backend virtualNode with tls enabled"), func() {
			backendListenerTLS := &appmesh.ListenerTLS{
				Certificate: appmesh.ListenerTLSCertificate{
					File: &appmesh.ListenerTLSFileCertificate{
						CertificateChain: "/certs/backend-tls_cert_chain.pem",
						PrivateKey:       "/certs/backend-tls_key.pem",
					},
				},
				Mode:        "STRICT",
			}
			listeners := []appmesh.Listener{vnBuilder.BuildListenerWithTLS("http", 8080, backendListenerTLS)}
			backends := []types.NamespacedName{}

			vn := vnBuilder.BuildVirtualNode("backend-tls", backends, listeners, &appmesh.BackendDefaults{})
			err := f.K8sClient.Create(ctx, vn)
			Expect(err).NotTo(HaveOccurred())
			s.BackEndVN = vn
		})

		By(fmt.Sprintf("create backend deployment"), func() {
			env := []corev1.EnvVar{
				{
					Name:  "SERVER_PORT",
					Value: fmt.Sprintf("%d", AppContainerPort),
				},
				{
					Name:  "WHO_AM_I",
					Value: "backend",
				},
			}
			//annotations := map[string]string{}
			annotations := map[string]string {
				"appmesh.k8s.aws/secretMounts": "backend-tls-tls:/certs/",
			}
			dp := mb.BuildDeployment("backend-tls", 1, defaultBackEndImage, AppContainerPort, env, annotations)
			err := f.K8sClient.Create(ctx, dp)
			Expect(err).NotTo(HaveOccurred())
			s.BackEndDP = dp
		})

		By(fmt.Sprintf("create service for backend-tls virtualnode"), func() {
			svc := mb.BuildServiceWithSelector("backend-tls", AppContainerPort, AppContainerPort)
			err := f.K8sClient.Create(ctx, svc)
			Expect(err).NotTo(HaveOccurred())
			s.BackEndSVC = svc
		})

		By("wait until all VirtualNodes become active", func() {
			_, err := f.VNManager.WaitUntilVirtualNodeActive(ctx, s.FrontEndVN)
			if err != nil {
				f.Logger.Error("failed to wait all VirtualNode become deleted", zap.Error(err))
				return
			}
			_, err = f.VNManager.WaitUntilVirtualNodeActive(ctx, s.BackEndVN)
			if err != nil {
				f.Logger.Error("failed to wait all VirtualNode become deleted", zap.Error(err))
				return
			}
		})

		By("wait all deployments become ready", func() {
			_, err := f.DPManager.WaitUntilDeploymentReady(ctx, s.FrontEndDP)
			if err != nil {
				f.Logger.Error("failed while waiting for Frontend VirtualNode to become active", zap.Error(err))
				return
			}
			_, err = f.DPManager.WaitUntilDeploymentReady(ctx, s.BackEndDP)
			if err != nil {
				f.Logger.Error("failed while waiting for Backend VirtualNode to become active", zap.Error(err))
				return
			}
		})

		By("check all VirtualNode in aws", func() {
			err := f.VNManager.CheckVirtualNodeInAWS(ctx, s.mesh, s.FrontEndVN)
			if err != nil {
				f.Logger.Error("failed while validating Frontend VirtualNode aws resource", zap.Error(err))
				return
			}
			err = f.VNManager.CheckVirtualNodeInAWS(ctx, s.mesh, s.BackEndVN)
			if err != nil {
				f.Logger.Error("failed while validating Backend VirtualNode aws resource", zap.Error(err))
				return
			}
		})
	})
}

func (s *TLSStack) deleteResourcesForTLSStackNodes(ctx context.Context, f *framework.Framework) []error {
	var deletionErrors []error
	By("delete all resources for nodes", func() {
		By(fmt.Sprintf("delete Backend Service"), func() {
			if err := f.K8sClient.Delete(ctx, s.BackEndSVC); err != nil {
				f.Logger.Error("failed to delete Service",
					zap.String("namespace", s.BackEndSVC.Namespace),
					zap.String("name", s.BackEndSVC.Name),
					zap.Error(err),
				)
				deletionErrors = append(deletionErrors, err)
			}
		})

		By(fmt.Sprintf("delete Frontend Deployment"), func() {
			if err := f.K8sClient.Delete(ctx, s.FrontEndDP,
				client.PropagationPolicy(metav1.DeletePropagationForeground), client.GracePeriodSeconds(0)); err != nil {
				f.Logger.Error("failed to delete Deployment",
					zap.String("namespace", s.FrontEndDP.Namespace),
					zap.String("name", s.FrontEndDP.Name),
					zap.Error(err),
				)
				deletionErrors = append(deletionErrors, err)
			}
		})

		By(fmt.Sprintf("delete Backend Deployment"), func() {
			if err := f.K8sClient.Delete(ctx, s.BackEndDP,
				client.PropagationPolicy(metav1.DeletePropagationForeground), client.GracePeriodSeconds(0)); err != nil {
				f.Logger.Error("failed to delete Deployment",
					zap.String("namespace", s.BackEndDP.Namespace),
					zap.String("name", s.BackEndDP.Name),
					zap.Error(err),
				)
				deletionErrors = append(deletionErrors, err)
			}
		})

		By(fmt.Sprintf("delete Frontend VirtualNode for node"), func() {
			if err := f.K8sClient.Delete(ctx, s.FrontEndVN); err != nil {
				f.Logger.Error("failed to delete VirtualNode",
					zap.String("namespace", s.FrontEndVN.Namespace),
					zap.String("name", s.FrontEndVN.Name),
					zap.Error(err),
				)
				deletionErrors = append(deletionErrors, err)
			}
		})

		By(fmt.Sprintf("delete Backend VirtualNode"), func() {
			if err := f.K8sClient.Delete(ctx, s.BackEndVN); err != nil {
				f.Logger.Error("failed to delete VirtualNode",
					zap.String("namespace", s.BackEndVN.Namespace),
					zap.String("name", s.BackEndVN.Name),
					zap.Error(err),
				)
				deletionErrors = append(deletionErrors, err)
			}
		})

		By("wait all deployments become deleted", func() {
			if err := f.DPManager.WaitUntilDeploymentDeleted(ctx, s.FrontEndDP); err != nil {
				f.Logger.Error("failed while waiting for Frontend deployment to be deleted", zap.Error(err))
				return
			}

			if err := f.DPManager.WaitUntilDeploymentDeleted(ctx, s.BackEndDP); err != nil {
				f.Logger.Error("failed while waiting for Backend deployment to be deleted", zap.Error(err))
				return
			}
		})

		By("wait all VirtualNodes become deleted", func() {
			if err := f.VNManager.WaitUntilVirtualNodeDeleted(ctx, s.FrontEndVN); err != nil {
				f.Logger.Error("failed while waiting for Frontend VN to be deleted", zap.Error(err))
				return
			}
			if err := f.VNManager.WaitUntilVirtualNodeDeleted(ctx, s.BackEndVN); err != nil {
				f.Logger.Error("failed while waiting for Backend VN to be deleted", zap.Error(err))
				return
			}
		})
	})
	return deletionErrors
}

func (s *TLSStack) createVirtualNodeResourcesForPartialTLSStack(ctx context.Context, f *framework.Framework, mb *manifest.ManifestBuilder) {
	By("create virtualNode resources", func() {
		vnBuilder := &manifest.VNBuilder{
			ServiceDiscoveryType: manifest.DNSServiceDiscovery,
			Namespace:            tlsTest,
		}

		By(fmt.Sprintf("create frontend virtualNode with TLS enabled"), func() {
			listeners := []appmesh.Listener{vnBuilder.BuildListener("http", 8080)}
			backends := []types.NamespacedName{}
			tlsEnforce := true
			tlsBackendDefaults := &appmesh.BackendDefaults{
				ClientPolicy: &appmesh.ClientPolicy{
					TLS: &appmesh.ClientPolicyTLS{
						Enforce:    &tlsEnforce,
						Ports:      nil,
						Validation: appmesh.TLSValidationContext{
							Trust: appmesh.TLSValidationContextTrust{
								ACM:  nil,
								File: &appmesh.TLSValidationContextFileTrust{CertificateChain: "/certs/ca_1_cert.pem"},
							},
						},
					},
				},
			}
			vn := vnBuilder.BuildVirtualNode("frontend-tls", backends, listeners, tlsBackendDefaults)
			err := f.K8sClient.Create(ctx, vn)
			Expect(err).NotTo(HaveOccurred())
			s.FrontEndVN = vn
		})

		By(fmt.Sprintf("create frontend-tls deployment"), func() {
			//annotations := map[string]string{}
			annotations := map[string]string {
				"appmesh.k8s.aws/secretMounts": "ca1-cert-tls:/certs/",
			}
			dp := mb.BuildDeployment("frontend-tls", 1, defaultFrontEndImage, AppContainerPort, []corev1.EnvVar{}, annotations)
			err := f.K8sClient.Create(ctx, dp)
			Expect(err).NotTo(HaveOccurred())
			s.FrontEndDP = dp
		})

		By(fmt.Sprintf("create backend virtualNode without tls"), func() {
			listeners := []appmesh.Listener{vnBuilder.BuildListener("http", 8080)}
			backends := []types.NamespacedName{}

			vn := vnBuilder.BuildVirtualNode("backend-tls", backends, listeners, &appmesh.BackendDefaults{})
			err := f.K8sClient.Create(ctx, vn)
			Expect(err).NotTo(HaveOccurred())
			s.BackEndVN = vn
		})

		By(fmt.Sprintf("create backend deployment"), func() {
			env := []corev1.EnvVar{
				{
					Name:  "SERVER_PORT",
					Value: fmt.Sprintf("%d", AppContainerPort),
				},
				{
					Name:  "WHO_AM_I",
					Value: "backend",
				},
			}

			dp := mb.BuildDeployment("backend-tls", 1, defaultBackEndImage, AppContainerPort, env, map[string]string{})
			err := f.K8sClient.Create(ctx, dp)
			Expect(err).NotTo(HaveOccurred())
			s.BackEndDP = dp
		})

		By(fmt.Sprintf("create service for backend-tls virtualnode"), func() {
			svc := mb.BuildServiceWithSelector("backend-tls", AppContainerPort, AppContainerPort)
			err := f.K8sClient.Create(ctx, svc)
			Expect(err).NotTo(HaveOccurred())
			s.BackEndSVC = svc
		})

		By("wait until all VirtualNodes become active", func() {
			_, err := f.VNManager.WaitUntilVirtualNodeActive(ctx, s.FrontEndVN)
			if err != nil {
				f.Logger.Error("failed to wait all VirtualNode become deleted", zap.Error(err))
				return
			}
			_, err = f.VNManager.WaitUntilVirtualNodeActive(ctx, s.BackEndVN)
			if err != nil {
				f.Logger.Error("failed to wait all VirtualNode become deleted", zap.Error(err))
				return
			}
		})

		By("wait all deployments become ready", func() {
			_, err := f.DPManager.WaitUntilDeploymentReady(ctx, s.FrontEndDP)
			if err != nil {
				f.Logger.Error("failed while waiting for Frontend VirtualNode to become active", zap.Error(err))
				return
			}
			_, err = f.DPManager.WaitUntilDeploymentReady(ctx, s.BackEndDP)
			if err != nil {
				f.Logger.Error("failed while waiting for Backend VirtualNode to become active", zap.Error(err))
				return
			}
		})

		By("check all VirtualNode in aws", func() {
			err := f.VNManager.CheckVirtualNodeInAWS(ctx, s.mesh, s.FrontEndVN)
			if err != nil {
				f.Logger.Error("failed while validating Frontend VirtualNode aws resource", zap.Error(err))
				return
			}
			err = f.VNManager.CheckVirtualNodeInAWS(ctx, s.mesh, s.BackEndVN)
			if err != nil {
				f.Logger.Error("failed while validating Backend VirtualNode aws resource", zap.Error(err))
				return
			}
		})
	})
}

func (s *TLSStack) deleteResourcesForPartialTLSStackNodes(ctx context.Context, f *framework.Framework) []error {
	var deletionErrors []error
	By("delete all resources for nodes", func() {
		By(fmt.Sprintf("delete Backend Service"), func() {
			if err := f.K8sClient.Delete(ctx, s.BackEndSVC); err != nil {
				f.Logger.Error("failed to delete Service",
					zap.String("namespace", s.BackEndSVC.Namespace),
					zap.String("name", s.BackEndSVC.Name),
					zap.Error(err),
				)
				deletionErrors = append(deletionErrors, err)
			}
		})

		By(fmt.Sprintf("delete Frontend Deployment"), func() {
			if err := f.K8sClient.Delete(ctx, s.FrontEndDP,
				client.PropagationPolicy(metav1.DeletePropagationForeground), client.GracePeriodSeconds(0)); err != nil {
				f.Logger.Error("failed to delete Deployment",
					zap.String("namespace", s.FrontEndDP.Namespace),
					zap.String("name", s.FrontEndDP.Name),
					zap.Error(err),
				)
				deletionErrors = append(deletionErrors, err)
			}
		})

		By(fmt.Sprintf("delete Backend Deployment"), func() {
			if err := f.K8sClient.Delete(ctx, s.BackEndDP,
				client.PropagationPolicy(metav1.DeletePropagationForeground), client.GracePeriodSeconds(0)); err != nil {
				f.Logger.Error("failed to delete Deployment",
					zap.String("namespace", s.BackEndDP.Namespace),
					zap.String("name", s.BackEndDP.Name),
					zap.Error(err),
				)
				deletionErrors = append(deletionErrors, err)
			}
		})

		By(fmt.Sprintf("delete Frontend VirtualNode for node"), func() {
			if err := f.K8sClient.Delete(ctx, s.FrontEndVN); err != nil {
				f.Logger.Error("failed to delete VirtualNode",
					zap.String("namespace", s.FrontEndVN.Namespace),
					zap.String("name", s.FrontEndVN.Name),
					zap.Error(err),
				)
				deletionErrors = append(deletionErrors, err)
			}
		})

		By(fmt.Sprintf("delete Backend VirtualNode"), func() {
			if err := f.K8sClient.Delete(ctx, s.BackEndVN); err != nil {
				f.Logger.Error("failed to delete VirtualNode",
					zap.String("namespace", s.BackEndVN.Namespace),
					zap.String("name", s.BackEndVN.Name),
					zap.Error(err),
				)
				deletionErrors = append(deletionErrors, err)
			}
		})

		By("wait all deployments become deleted", func() {
			if err := f.DPManager.WaitUntilDeploymentDeleted(ctx, s.FrontEndDP); err != nil {
				f.Logger.Error("failed while waiting for Frontend deployment to be deleted", zap.Error(err))
				return
			}

			if err := f.DPManager.WaitUntilDeploymentDeleted(ctx, s.BackEndDP); err != nil {
				f.Logger.Error("failed while waiting for Backend deployment to be deleted", zap.Error(err))
				return
			}
		})

		By("wait all VirtualNodes become deleted", func() {
			if err := f.VNManager.WaitUntilVirtualNodeDeleted(ctx, s.FrontEndVN); err != nil {
				f.Logger.Error("failed while waiting for Frontend VN to be deleted", zap.Error(err))
				return
			}
			if err := f.VNManager.WaitUntilVirtualNodeDeleted(ctx, s.BackEndVN); err != nil {
				f.Logger.Error("failed while waiting for Backend VN to be deleted", zap.Error(err))
				return
			}
		})
	})
	return deletionErrors
}

func (s *TLSStack) createServicesForTLSStack(ctx context.Context, f *framework.Framework) {
	By("create all resources for services", func() {
		testNameSpace := tlsTest
		By(fmt.Sprintf("Create VirtualRouter for backend service"), func() {
			var routes []appmesh.Route
			var weightedTargets []appmesh.WeightedTarget
			vrBuilder := &manifest.VRBuilder{
				Namespace: tlsTest,
			}

			weightedTargets = append(weightedTargets, appmesh.WeightedTarget{
				VirtualNodeRef: &appmesh.VirtualNodeReference{
					Namespace: &testNameSpace,
					Name:      "backend-tls",
				},
				Weight: 1,
			})

			//TLS configured route
			routes = append(routes, appmesh.Route{
				Name: "TLSRoute",
				HTTPRoute: &appmesh.HTTPRoute{
					Match: appmesh.HTTPRouteMatch{
						Prefix: "/tlsroute",
					},
					Action: appmesh.HTTPRouteAction{
						WeightedTargets: weightedTargets,
					},
				},
			})

			vrBuilder.Listeners = []appmesh.VirtualRouterListener{vrBuilder.BuildVirtualRouterListener("http", 8080)}
			vr := vrBuilder.BuildVirtualRouter("backend-tls", routes)
			err := f.K8sClient.Create(ctx, vr)
			Expect(err).NotTo(HaveOccurred())
			s.BackEndVR = vr
		})

		By(fmt.Sprintf("create VirtualService for backend"), func() {
			vsDNS := fmt.Sprintf("%s.%s.svc.cluster.local", "backend-tls", tlsTest)
			vs := &appmesh.VirtualService{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNameSpace,
					Name:      "backend-tls",
				},
				Spec: appmesh.VirtualServiceSpec{
					AWSName: aws.String(vsDNS),
					Provider: &appmesh.VirtualServiceProvider{
						VirtualRouter: &appmesh.VirtualRouterServiceProvider{
							VirtualRouterRef: &appmesh.VirtualRouterReference{
								Namespace: &testNameSpace,
								Name:      "backend-tls",
							},
						},
					},
				},
			}
			err := f.K8sClient.Create(ctx, vs)
			Expect(err).NotTo(HaveOccurred())
			s.BackEndVS = vs
		})

		By("check backend VirtualRouter in AWS", func() {
			err := f.VRManager.CheckVirtualRouterInAWS(ctx, s.mesh, s.BackEndVR)
			if err != nil {
				f.Logger.Error("failed to check backend VirtualRouter in AWS", zap.Error(err))
				return
			}
		})

		By("check backend VirtualService in AWS", func() {
			err := f.VSManager.CheckVirtualServiceInAWS(ctx, s.mesh, s.BackEndVS)
			if err != nil {
				f.Logger.Error("failed to check backend VirtualService in AWS", zap.Error(err))
				return
			}
		})
	})
}

func (s *TLSStack) deleteResourcesForTLSStackServices(ctx context.Context, f *framework.Framework) []error {
	var deletionErrors []error
	By("delete all resources for services", func() {
		By(fmt.Sprintf("Delete backend VirtualService"), func() {
			if err := f.K8sClient.Delete(ctx, s.BackEndVS); err != nil {
				f.Logger.Error("failed to delete VirtualService",
					zap.String("namespace", s.BackEndVS.Namespace),
					zap.String("name", s.BackEndVS.Name),
					zap.Error(err),
				)
				deletionErrors = append(deletionErrors, err)
			}
		})

		By(fmt.Sprintf("Delete backend VirtualRouter"), func() {
			if err := f.K8sClient.Delete(ctx, s.BackEndVR); err != nil {
				f.Logger.Error("failed to delete VirtualRouter",
					zap.String("namespace", s.BackEndVR.Namespace),
					zap.String("name", s.BackEndVR.Name),
					zap.Error(err),
				)
				deletionErrors = append(deletionErrors, err)
			}
		})

		By("Wait for backend VirtualService to be deleted", func() {
			if err := f.VSManager.WaitUntilVirtualServiceDeleted(ctx, s.BackEndVS); err != nil {
				f.Logger.Error("failed to delete backend VirtualService", zap.Error(err))
				return
			}
		})
		By("Wait for backend VirtualRouter to be deleted", func() {
			if err := f.VRManager.WaitUntilVirtualRouterDeleted(ctx, s.BackEndVR); err != nil {
				f.Logger.Error("failed to delete backend VirtualRouter", zap.Error(err))
				return
			}
		})
	})
	return deletionErrors
}

func (s *TLSStack) assignBackendVSToFrontEndVN(ctx context.Context, f *framework.Framework) {
	By("granting VirtualNodes backend access", func() {
		By(fmt.Sprintf("assigning backend VS to frontend VN"), func() {
			var vnBackends []appmesh.Backend
			vs := s.BackEndVS
			vnBackends = append(vnBackends, appmesh.Backend{
				VirtualService: appmesh.VirtualServiceBackend{
					VirtualServiceRef: &appmesh.VirtualServiceReference{
						Namespace: aws.String(vs.Namespace),
						Name:      vs.Name,
					},
				},
			})

			vnNew := s.FrontEndVN.DeepCopy()
			vnNew.Spec.Backends = vnBackends
			err := f.K8sClient.Patch(ctx, vnNew, client.MergeFrom(s.FrontEndVN))
			Expect(err).NotTo(HaveOccurred())
			s.FrontEndVN = vnNew
		})
	})
}

func (s *TLSStack) revokeVirtualNodeBackendAccessForTLSStack(ctx context.Context, f *framework.Framework) []error {
	var deletionErrors []error
	By("revoking VirtualNodes backend access", func() {
		By(fmt.Sprintf("revoking Frontend VirtualNode access to backend"), func() {
			vnNew := s.FrontEndVN.DeepCopy()
			vnNew.Spec.Backends = nil

			err := f.K8sClient.Patch(ctx, vnNew, client.MergeFrom(s.FrontEndVN))
			if err != nil {
				f.Logger.Error("failed to revoke VirtualNode backend access",
					zap.String("namespace", s.FrontEndVN.Namespace),
					zap.String("name", s.FrontEndVN.Name),
					zap.Error(err),
				)
				deletionErrors = append(deletionErrors, err)
			}
		})
	})
	return deletionErrors
}

func (s *TLSStack) checkExpectedRouteBehavior(ctx context.Context, f *framework.Framework,
	dp *appsv1.Deployment, path string, tlsEnabled bool) error {
	sel := labels.Set(dp.Spec.Selector.MatchLabels)
	podList := &corev1.PodList{}
	err := f.K8sClient.List(ctx, podList, client.InNamespace(dp.Namespace), client.MatchingLabelsSelector{Selector: sel.AsSelector()})
	if err != nil {
		return errors.Wrapf(err, "failed to get pods for Deployment: %v", k8s.NamespacedName(dp).String())
	}
	if len(podList.Items) == 0 {
		return errors.Wrapf(err, "Deployment have zero pods: %v", k8s.NamespacedName(dp).String())
	}

	for i := range podList.Items {
		pod := podList.Items[i].DeepCopy()
		if response, err := s.verifyFrontEndPodToBackendPodConnectivity(ctx, f, pod, path); err != nil {
			f.Logger.Error("error while reaching out to default endpoint of backend", zap.Error(err))
			return err
		} else {
			if (!tlsEnabled && !strings.HasPrefix(response, tlsConnectionError)) || (tlsEnabled && response != expectedBackendResponse) {
				return fmt.Errorf("failed to verify TLS behavior")
			}
		}
	}

	return nil
}

func (s *TLSStack) verifyFrontEndPodToBackendPodConnectivity(ctx context.Context, f *framework.Framework,
	pod *corev1.Pod, path string) (response string, err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	pfErrChan := make(chan error)
	pfReadyChan := make(chan struct{})
	portForwarder, err := k8s.NewPortForwarder(ctx, f.RestCfg, pod, []string{fmt.Sprintf("%d:%d", AppContainerPort, AppContainerPort)}, pfReadyChan)
	if err != nil {
		return response, err
	}
	go func() {
		pfErrChan <- portForwarder.ForwardPorts()
	}()

	podURL := fmt.Sprintf("http://localhost:%d/%s", AppContainerPort, path)
	<-pfReadyChan
	resp, err := http.Get(podURL)
	if err != nil {
		f.Logger.Error("failed to check backend VirtualService in AWS", zap.Error(err))
		return response, err
	}
	respPayload, err := ioutil.ReadAll(resp.Body)
	response = string(respPayload)
	if err != nil {
		return response, err
	}
	f.Logger.Warn("Received Response: ", zap.String("Response: ", response), zap.Int("StatusCode: ", resp.StatusCode))
	cancel()
	return response, <-pfErrChan
}
