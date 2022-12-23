package timeout

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/manifest"

	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/algorithm"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/k8s"
	"github.com/aws/aws-sdk-go/aws"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	//If you're not able to access below images, try to build them based on the app code under "timeout_app"
	//directory and push it to any accessible ECR repo and update the below values
	defaultFrontEndImage = "public.ecr.aws/e6v3k1j4/appmesh-test-feapp:v1"
	defaultBackEndImage  = "public.ecr.aws/e6v3k1j4/appmesh-test-beapp:v1"

	timeoutTest      = "timeout-e2e"
	AppContainerPort = 8080

	timeoutMessage          = "upstream request timeout"
	expectedBackendResponse = "backend"
)

// Timeout stack is setup as below
//	FrontEnd ->
//        - Exposes two endpoints "/defaultroute" and "/timeoutroute" and reaches out to backend for both the endpoints
//        - Refers to "backend" VirtualService which in turn refers to "backend" VirtualRouter
//
//  Backend ->
//        - Exposes "/defaultroute" and "/timeoutroute" with a configured 45 seconds delay
//        - "backend" VN is configured with a Listener timeout of 60 seconds
//        - "backend" VR has two routes "/default" and "/timeout"
//        - "/defaultroute" path uses default timeout of 15s and "/timeoutroute" path is configured with 60s timeout
//
//  We then validate the timeout feature.
//       - Call to "/default" from frontend pod should timeout with "upstream request timeout"
//       - Call to "/timeout" from frontend pod should get a valid response which will be "backend"(config.WHO_AM_I)

type TimeoutStack struct {
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

	TimeoutValue int
}

func (s *TimeoutStack) DeployTimeoutStack(ctx context.Context, f *framework.Framework) {
	s.createTimeoutStackMeshAndNamespace(ctx, f)

	s.ServiceDiscoveryType = manifest.DNSServiceDiscovery
	mb := &manifest.ManifestBuilder{
		Namespace:            s.namespace.Name,
		ServiceDiscoveryType: s.ServiceDiscoveryType,
		DisableIPv6:          true, // for github action compatibility
	}
	s.createVirtualNodeResourcesForTimeoutStack(ctx, f, mb)
	s.createServicesForTimeoutStack(ctx, f)
	s.assignBackendVSToFrontEndVN(ctx, f)
}

func (s *TimeoutStack) CleanupTimeoutStack(ctx context.Context, f *framework.Framework) {
	var deletionErrors []error
	if errs := s.revokeVirtualNodeBackendAccessForTimeoutStack(ctx, f); len(errs) != 0 {
		deletionErrors = append(deletionErrors, errs...)
	}
	if errs := s.deleteResourcesForTimeoutStackServices(ctx, f); len(errs) != 0 {
		deletionErrors = append(deletionErrors, errs...)
	}
	if errs := s.deleteResourcesForTimeoutStackNodes(ctx, f); len(errs) != 0 {
		deletionErrors = append(deletionErrors, errs...)
	}
	if errs := s.deleteTimeoutMeshAndNamespace(ctx, f); len(errs) != 0 {
		deletionErrors = append(deletionErrors, errs...)
	}
	for _, err := range deletionErrors {
		f.Logger.Error("clean up failed", zap.Error(err))
	}
	Expect(len(deletionErrors)).To(BeZero())
}

// Check Timeout behavior with and with timeout configured
func (s *TimeoutStack) CheckTimeoutBehavior(ctx context.Context, f *framework.Framework) {
	By(fmt.Sprintf("verify route timesout if it takes more than default 15s w/o listener timeout configured"), func() {
		err := s.checkExpectedRouteBehavior(ctx, f, s.FrontEndDP, "defaultroute", false)
		Expect(err).NotTo(HaveOccurred())
	})

	By(fmt.Sprintf("verify the timeout behaviour with configured timeout value"), func() {
		err := s.checkExpectedRouteBehavior(ctx, f, s.FrontEndDP, "timeoutroute", true)
		Expect(err).NotTo(HaveOccurred())
	})
}

func (s *TimeoutStack) createTimeoutStackMeshAndNamespace(ctx context.Context, f *framework.Framework) {
	By("create a mesh", func() {
		meshName := timeoutTest
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
				Name: timeoutTest,
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

func (s *TimeoutStack) deleteTimeoutMeshAndNamespace(ctx context.Context, f *framework.Framework) []error {
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

func (s *TimeoutStack) createVirtualNodeResourcesForTimeoutStack(ctx context.Context, f *framework.Framework, mb *manifest.ManifestBuilder) {
	By("create virtualNode resources", func() {
		vnBuilder := &manifest.VNBuilder{
			ServiceDiscoveryType: manifest.DNSServiceDiscovery,
			Namespace:            timeoutTest,
		}

		annotations := map[string]string{}
		By(fmt.Sprintf("create frontend virtualNode with default timeout"), func() {
			listeners := []appmesh.Listener{vnBuilder.BuildListener("http", 8080)}
			backends := []types.NamespacedName{}
			vn := vnBuilder.BuildVirtualNode("frontend", backends, listeners, &appmesh.BackendDefaults{})
			err := f.K8sClient.Create(ctx, vn)
			Expect(err).NotTo(HaveOccurred())
			s.FrontEndVN = vn
		})

		By(fmt.Sprintf("create frontend deployment"), func() {
			containersInfo := []manifest.ContainerInfo{
				{
					Name:          "app",
					AppImage:      defaultFrontEndImage,
					ContainerPort: AppContainerPort,
					Env: []corev1.EnvVar{
						{
							Name:  "PORT",
							Value: fmt.Sprintf("%d", AppContainerPort),
						},
						{
							Name:  "BACKEND_TIMEOUT_HOST",
							Value: "backend.timeout-e2e.svc.cluster.local",
						},
					},
				},
			}
			containers := mb.BuildContainerSpec(containersInfo)
			dp := mb.BuildDeployment("frontend", 1, containers, annotations)
			err := f.K8sClient.Create(ctx, dp)
			Expect(err).NotTo(HaveOccurred())
			s.FrontEndDP = dp
		})

		By(fmt.Sprintf("create backend virtualNode with non default timeout"), func() {
			listeners := []appmesh.Listener{vnBuilder.BuildListenerWithTimeout("http", 8080, 60, appmesh.DurationUnitS)}
			backends := []types.NamespacedName{}
			vn := vnBuilder.BuildVirtualNode("backend", backends, listeners, &appmesh.BackendDefaults{})
			err := f.K8sClient.Create(ctx, vn)
			Expect(err).NotTo(HaveOccurred())
			s.BackEndVN = vn
		})

		By(fmt.Sprintf("create backend deployment"), func() {
			containersInfo := []manifest.ContainerInfo{
				{
					Name:          "app",
					AppImage:      defaultBackEndImage,
					ContainerPort: AppContainerPort,
					Env: []corev1.EnvVar{
						{
							Name:  "SERVER_PORT",
							Value: fmt.Sprintf("%d", AppContainerPort),
						},
						{
							Name:  "WHO_AM_I",
							Value: "backend",
						},
						{
							Name:  "TIMEOUT_VALUE",
							Value: fmt.Sprintf("%d", s.TimeoutValue),
						},
					},
				},
			}
			containers := mb.BuildContainerSpec(containersInfo)
			dp := mb.BuildDeployment("backend", 1, containers, annotations)
			err := f.K8sClient.Create(ctx, dp)
			Expect(err).NotTo(HaveOccurred())
			s.BackEndDP = dp
		})

		By(fmt.Sprintf("create service for backend virtualnode"), func() {
			svc := mb.BuildServiceWithSelector("backend", AppContainerPort, AppContainerPort)
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

func (s *TimeoutStack) deleteResourcesForTimeoutStackNodes(ctx context.Context, f *framework.Framework) []error {
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

func (s *TimeoutStack) createServicesForTimeoutStack(ctx context.Context, f *framework.Framework) {
	By("create all resources for services", func() {
		testNameSpace := timeoutTest
		By(fmt.Sprintf("Create VirtualRouter for backend service"), func() {
			var routes []appmesh.Route
			var weightedTargets []appmesh.WeightedTarget
			vrBuilder := &manifest.VRBuilder{
				Namespace: timeoutTest,
			}

			weightedTargets = append(weightedTargets, appmesh.WeightedTarget{
				VirtualNodeRef: &appmesh.VirtualNodeReference{
					Namespace: &testNameSpace,
					Name:      "backend",
				},
				Weight: 1,
			})

			//route with a configured timeout value of 60s
			routes = append(routes, appmesh.Route{
				Name: "Timeout",
				HTTPRoute: &appmesh.HTTPRoute{
					Match: appmesh.HTTPRouteMatch{
						Prefix: aws.String("/timeoutroute"),
					},
					Action: appmesh.HTTPRouteAction{
						WeightedTargets: weightedTargets,
					},
					Timeout: &appmesh.HTTPTimeout{
						PerRequest: &appmesh.Duration{
							Unit:  "s",
							Value: 60,
						},
					},
				},
			})

			//route using default timeout value
			routes = append(routes, appmesh.Route{
				Name: "No-Timeout",
				HTTPRoute: &appmesh.HTTPRoute{
					Match: appmesh.HTTPRouteMatch{
						Prefix: aws.String("/defaultroute"),
					},
					Action: appmesh.HTTPRouteAction{
						WeightedTargets: weightedTargets,
					},
				},
			})

			vrBuilder.Listeners = []appmesh.VirtualRouterListener{vrBuilder.BuildVirtualRouterListener("http", 8080)}
			vr := vrBuilder.BuildVirtualRouter("backend", routes)
			err := f.K8sClient.Create(ctx, vr)
			Expect(err).NotTo(HaveOccurred())
			s.BackEndVR = vr
		})

		By(fmt.Sprintf("create VirtualService for backend"), func() {
			vsDNS := fmt.Sprintf("%s.%s.svc.cluster.local", "backend", timeoutTest)
			vs := &appmesh.VirtualService{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNameSpace,
					Name:      "backend",
				},
				Spec: appmesh.VirtualServiceSpec{
					AWSName: aws.String(vsDNS),
					Provider: &appmesh.VirtualServiceProvider{
						VirtualRouter: &appmesh.VirtualRouterServiceProvider{
							VirtualRouterRef: &appmesh.VirtualRouterReference{
								Namespace: &testNameSpace,
								Name:      "backend",
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

func (s *TimeoutStack) deleteResourcesForTimeoutStackServices(ctx context.Context, f *framework.Framework) []error {
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

func (s *TimeoutStack) assignBackendVSToFrontEndVN(ctx context.Context, f *framework.Framework) {
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

func (s *TimeoutStack) revokeVirtualNodeBackendAccessForTimeoutStack(ctx context.Context, f *framework.Framework) []error {
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

func (s *TimeoutStack) checkExpectedRouteBehavior(ctx context.Context, f *framework.Framework,
	dp *appsv1.Deployment, path string, timeoutConfigured bool) error {
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
		if response, err := s.verifyFrontEndPodToRouteConnectivity(ctx, f, pod, path); err != nil {
			f.Logger.Error("error while reaching out to default endpoint of backend")
			return err
		} else {
			if (!timeoutConfigured && response != timeoutMessage) || (timeoutConfigured && response != expectedBackendResponse) {
				return fmt.Errorf("failed to verify route timeout behavior")
			}
		}
	}
	return nil
}

func (s *TimeoutStack) verifyFrontEndPodToRouteConnectivity(ctx context.Context, f *framework.Framework,
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
