package fishapp

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"
	"time"

	appmeshv1beta1 "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/apis/appmesh/v1beta1"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/e2e/fishapp/shared"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/e2e/framework"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/e2e/framework/collection"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/e2e/framework/k8s"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/e2e/framework/utils"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"gonum.org/v1/gonum/stat"
	"gonum.org/v1/gonum/stat/distuv"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
)

const (
	connectivityCheckRate                  = time.Second / 100
	connectivityCheckProxyPort             = 8899
	connectivityCheckUniformDistributionSL = 0.001 // Significance level that traffic to targets are uniform distributed.
)

// A dynamic generated stack designed to test app mesh integration :D
// Suppose given configuration below:
//		5 VirtualServicesCount
//		10 VirtualNodesCount
//		2 RoutesCountPerVirtualRouter
//		2 TargetsCountPerRoute
//		4 BackendsCountPerVirtualNode
// We will generate virtual service configuration & virtual node configuration follows:
// =======virtual services =========
//	vs1 -> /path1 -> vn1(50)
//				  -> vn2(50)
//		-> /path2 -> vn3(50)
//				  -> vn4(50)
//	vs2 -> /path1 -> vn5(50)
//				  -> vn6(50)
//		-> /path2 -> vn7(50)
//				  -> vn8(50)
//	vs3 -> /path1 -> vn9(50)
//				  -> vn10(50)
//		-> /path2 -> vn1(50)
//				  -> vn2(50)
//	vs4 -> /path1 -> vn3(50)
//				  -> vn4(50)
//		-> /path2 -> vn5(50)
//				  -> vn6(50)
//	vs5 -> /path1 -> vn7(50)
//				  -> vn8(50)
//		-> /path2 -> vn9(50)
//				  -> vn10(50)
// =======virtual nodes =========
//  vn1 -> vs1,vs2,vs3,vs4
//  vn2 -> vs5,vs1,vs2,vs3
//  vn3 -> vs4,vs5,vs1,vs2
//  ...
//
// then we validate each virtual node can access each virtual service at every path, and calculates the target distribution
type DynamicStack struct {
	// service discovery type
	ServiceDiscoveryType shared.ServiceDiscoveryType

	// number of virtual service
	VirtualServicesCount int

	// number of virtual nodes count
	VirtualNodesCount int

	// number of routes per virtual router
	RoutesCountPerVirtualRouter int

	// number of targets per route
	TargetsCountPerRoute int

	// number of backends per virtual node
	BackendsCountPerVirtualNode int

	// number of replicas per virtual node
	ReplicasPerVirtualNode int32

	// how many time to check connectivity per URL
	ConnectivityCheckPerURL int

	// ====== runtime variables ======
	mesh              *appmeshv1beta1.Mesh
	namespace         *corev1.Namespace
	cloudMapNamespace string

	createdNodeVNs  []*appmeshv1beta1.VirtualNode
	createdNodeDPs  []*appsv1.Deployment
	createdNodeSVCs []*corev1.Service

	createdServiceVSs  []*appmeshv1beta1.VirtualService
	createdServiceSVCs []*corev1.Service
}

// expects the stack can be deployed to namespace successfully
func (s *DynamicStack) Deploy(ctx context.Context, f *framework.Framework) {
	s.createMeshAndNamespace(ctx, f)
	if s.ServiceDiscoveryType == shared.CloudMapServiceDiscovery {
		s.createCloudMapNamespace(ctx, f)
	}
	mb := &shared.ManifestBuilder{
		MeshName:             s.mesh.Name,
		Namespace:            s.namespace.Name,
		ServiceDiscoveryType: s.ServiceDiscoveryType,
		CloudMapNamespace:    s.cloudMapNamespace,
	}
	s.createResourcesForNodes(ctx, f, mb)
	s.createResourcesForServices(ctx, f, mb)
	s.grantVirtualNodesBackendAccess(ctx, f)
}

// expects the stack can be cleaned up from namespace successfully
func (s *DynamicStack) Cleanup(ctx context.Context, f *framework.Framework) {
	var deletionErrors []error
	if errs := s.revokeVirtualNodeBackendAccess(ctx, f); len(errs) != 0 {
		deletionErrors = append(deletionErrors, errs...)
	}
	if errs := s.deleteResourcesForServices(ctx, f); len(errs) != 0 {
		deletionErrors = append(deletionErrors, errs...)
	}
	if errs := s.deleteResourcesForNodes(ctx, f); len(errs) != 0 {
		deletionErrors = append(deletionErrors, errs...)
	}
	if s.ServiceDiscoveryType == shared.CloudMapServiceDiscovery {
		if errs := s.deleteCloudMapNamespace(ctx, f); len(errs) != 0 {
			deletionErrors = append(deletionErrors, errs...)
		}
	}
	if errs := s.deleteMeshAndNamespace(ctx, f); len(errs) != 0 {
		deletionErrors = append(deletionErrors, errs...)
	}
	for _, err := range deletionErrors {
		f.Logger.Error("clean up failed", zap.Error(err))
	}
	Expect(len(deletionErrors)).To(BeZero())
}

// Check connectivity and routing works correctly
func (s *DynamicStack) Check(ctx context.Context, f *framework.Framework) {
	vsByName := make(map[string]*appmeshv1beta1.VirtualService)
	for i := 0; i != s.VirtualServicesCount; i++ {
		vs := s.createdServiceVSs[i]
		vsByName[vs.Name] = vs
	}

	var checkErrors []error
	for i := 0; i != s.VirtualNodesCount; i++ {
		dp := s.createdNodeDPs[i]
		vn := s.createdNodeVNs[i]
		var vsList []*appmeshv1beta1.VirtualService
		for _, backend := range vn.Spec.Backends {
			vsList = append(vsList, vsByName[backend.VirtualService.VirtualServiceName])
		}
		if errs := s.checkDeploymentToVirtualServiceConnectivity(ctx, f, dp, vsList); len(errs) != 0 {
			checkErrors = append(checkErrors, errs...)
		}
	}
	for _, err := range checkErrors {
		f.Logger.Error("connectivity check failed", zap.Error(err))
	}
	Expect(len(checkErrors)).To(BeZero())
}

func (s *DynamicStack) createMeshAndNamespace(ctx context.Context, f *framework.Framework) {
	By("create a mesh", func() {
		meshName := fmt.Sprintf("%s-%s", f.Options.ClusterName, utils.RandomDNS1123Label(6))
		mesh, err := f.K8sMeshClient.AppmeshV1beta1().Meshes().Create(&appmeshv1beta1.Mesh{
			ObjectMeta: metav1.ObjectMeta{
				Name: meshName,
			},
			Spec: appmeshv1beta1.MeshSpec{},
		})
		Expect(err).NotTo(HaveOccurred())
		s.mesh = mesh
	})

	By(fmt.Sprintf("wait for mesh %s become active", s.mesh.Name), func() {
		mesh, err := f.MeshManager.WaitUntilMeshActive(ctx, s.mesh)
		Expect(err).NotTo(HaveOccurred())
		s.mesh = mesh
	})

	By("allocates a namespace", func() {
		namespace, err := f.NSManager.AllocateNamespace(ctx, "appmesh")
		Expect(err).NotTo(HaveOccurred())
		s.namespace = namespace
	})

	By("label namespace with appMesh inject", func() {
		namespace := s.namespace.DeepCopy()
		namespace.Labels = collection.MergeStringMap(s.namespace.Labels, map[string]string{
			"appmesh.k8s.aws/sidecarInjectorWebhook": "enabled",
		})
		patch, err := k8s.CreateStrategicTwoWayMergePatch(s.namespace, namespace, corev1.Namespace{})
		Expect(err).NotTo(HaveOccurred())
		namespace, err = f.K8sClient.CoreV1().Namespaces().Patch(namespace.Name, types.StrategicMergePatchType, patch)
		Expect(err).NotTo(HaveOccurred())
		s.namespace = namespace
	})
}

func (s *DynamicStack) deleteMeshAndNamespace(ctx context.Context, f *framework.Framework) []error {
	var deletionErrors []error
	if s.namespace != nil {
		By(fmt.Sprintf("delete namespace: %s", s.namespace.Name), func() {
			foregroundDeletion := metav1.DeletePropagationForeground
			if err := f.K8sClient.CoreV1().Namespaces().Delete(s.namespace.Name, &metav1.DeleteOptions{
				GracePeriodSeconds: aws.Int64(0),
				PropagationPolicy:  &foregroundDeletion,
			}); err != nil {
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
			foregroundDeletion := metav1.DeletePropagationForeground
			if err := f.K8sMeshClient.AppmeshV1beta1().Meshes().Delete(s.mesh.Name, &metav1.DeleteOptions{
				GracePeriodSeconds: aws.Int64(0),
				PropagationPolicy:  &foregroundDeletion,
			}); err != nil {
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

func (s *DynamicStack) createCloudMapNamespace(ctx context.Context, f *framework.Framework) {
	cmNamespace := fmt.Sprintf("%s-%s", f.Options.ClusterName, utils.RandomDNS1123Label(6))
	By(fmt.Sprintf("create cloudMap namespace %s", cmNamespace), func() {
		resp, err := f.SDClient.CreatePrivateDnsNamespaceWithContext(ctx, &servicediscovery.CreatePrivateDnsNamespaceInput{
			Name: aws.String(cmNamespace),
			Vpc:  aws.String(f.Options.AWSVPCID),
		})
		Expect(err).NotTo(HaveOccurred())
		s.cloudMapNamespace = cmNamespace
		f.Logger.Info("created cloudMap namespace",
			zap.String("namespace", cmNamespace),
			zap.String("operationID", aws.StringValue(resp.OperationId)),
		)
	})
}

func (s *DynamicStack) deleteCloudMapNamespace(ctx context.Context, f *framework.Framework) []error {
	var deletionErrors []error
	if s.cloudMapNamespace != "" {
		By(fmt.Sprintf("delete cloudMap namespace %s", s.cloudMapNamespace), func() {
			var cmNamespaceID string
			f.SDClient.ListNamespacesPagesWithContext(ctx, &servicediscovery.ListNamespacesInput{}, func(output *servicediscovery.ListNamespacesOutput, b bool) bool {
				for _, ns := range output.Namespaces {
					if aws.StringValue(ns.Name) == s.cloudMapNamespace {
						cmNamespaceID = aws.StringValue(ns.Id)
						return true
					}
				}
				return false
			})
			if cmNamespaceID == "" {
				err := errors.Errorf("cannot find cloudMap namespace with name %s", s.cloudMapNamespace)
				f.Logger.Error("failed to delete cloudMap namespace",
					zap.String("namespace", s.cloudMapNamespace),
					zap.Error(err))
				deletionErrors = append(deletionErrors, err)
				return
			}

			// hummm, let's fix the controller bug in test cases first xD:
			// https://github.com/aws/aws-app-mesh-controller-for-k8s/issues/107
			// https://github.com/aws/aws-app-mesh-controller-for-k8s/issues/131
			By(fmt.Sprintf("[bug workaround] clean up resources in cloudMap namespace %s", s.cloudMapNamespace), func() {
				// give controller a break to deregister instance xD
				time.Sleep(1 * time.Minute)
				var cmServiceIDs []string
				f.SDClient.ListServicesPagesWithContext(ctx, &servicediscovery.ListServicesInput{
					Filters: []*servicediscovery.ServiceFilter{
						{
							Condition: aws.String(servicediscovery.FilterConditionEq),
							Name:      aws.String("NAMESPACE_ID"),
							Values:    aws.StringSlice([]string{cmNamespaceID}),
						},
					},
				}, func(output *servicediscovery.ListServicesOutput, b bool) bool {
					for _, svc := range output.Services {
						cmServiceIDs = append(cmServiceIDs, aws.StringValue(svc.Id))
					}
					return false
				})
				for _, cmServiceID := range cmServiceIDs {
					var cmInstanceIDs []string
					f.SDClient.ListInstancesPagesWithContext(ctx, &servicediscovery.ListInstancesInput{
						ServiceId: aws.String(cmServiceID),
					}, func(output *servicediscovery.ListInstancesOutput, b bool) bool {
						for _, ins := range output.Instances {
							cmInstanceIDs = append(cmInstanceIDs, aws.StringValue(ins.Id))
						}
						return false
					})

					for _, cmInstanceID := range cmInstanceIDs {
						if _, err := f.SDClient.DeregisterInstanceWithContext(ctx, &servicediscovery.DeregisterInstanceInput{
							ServiceId:  aws.String(cmServiceID),
							InstanceId: aws.String(cmInstanceID),
						}); err != nil {
							f.Logger.Error("failed to deregister cloudMap instance",
								zap.String("namespaceID", cmNamespaceID),
								zap.String("serviceID", cmServiceID),
								zap.String("instanceID", cmInstanceID),
								zap.Error(err),
							)
							deletionErrors = append(deletionErrors, err)
						}
					}
					time.Sleep(30 * time.Second)

					if _, err := f.SDClient.DeleteServiceWithContext(ctx, &servicediscovery.DeleteServiceInput{
						Id: aws.String(cmServiceID),
					}); err != nil {
						f.Logger.Error("failed to delete cloudMap service",
							zap.String("namespaceID", cmNamespaceID),
							zap.String("serviceID", cmServiceID),
							zap.Error(err),
						)
						deletionErrors = append(deletionErrors, err)
					}
				}
			})

			time.Sleep(30 * time.Second)
			if _, err := f.SDClient.DeleteNamespaceWithContext(ctx, &servicediscovery.DeleteNamespaceInput{
				Id: aws.String(cmNamespaceID),
			}); err != nil {
				f.Logger.Error("failed to delete cloudMap namespace",
					zap.String("namespaceID", cmNamespaceID),
					zap.Error(err),
				)
				deletionErrors = append(deletionErrors, err)
			}
		})
	}
	return deletionErrors
}

func (s *DynamicStack) createResourcesForNodes(ctx context.Context, f *framework.Framework, mb *shared.ManifestBuilder) {
	By("create all resources for nodes", func() {
		s.createdNodeVNs = make([]*appmeshv1beta1.VirtualNode, s.VirtualNodesCount)
		s.createdNodeDPs = make([]*appsv1.Deployment, s.VirtualNodesCount)
		s.createdNodeSVCs = make([]*corev1.Service, s.VirtualNodesCount)

		var err error
		for i := 0; i != s.VirtualNodesCount; i++ {
			instanceName := fmt.Sprintf("node-%d", i)
			By(fmt.Sprintf("create VirtualNode for node #%d", i), func() {
				vn := mb.BuildNodeVirtualNode(instanceName, nil)
				vn, err = f.K8sMeshClient.AppmeshV1beta1().VirtualNodes(s.namespace.Name).Create(vn)
				Expect(err).NotTo(HaveOccurred())
				s.createdNodeVNs[i] = vn
			})

			By(fmt.Sprintf("create Deployment for node #%d", i), func() {
				dp := mb.BuildNodeDeployment(instanceName, s.ReplicasPerVirtualNode)
				dp, err = f.K8sClient.AppsV1().Deployments(s.namespace.Name).Create(dp)
				Expect(err).NotTo(HaveOccurred())
				s.createdNodeDPs[i] = dp
			})

			By(fmt.Sprintf("create Service for node #%d", i), func() {
				svc := mb.BuildNodeService(instanceName)
				svc, err := f.K8sClient.CoreV1().Services(s.namespace.Name).Create(svc)
				Expect(err).NotTo(HaveOccurred())
				s.createdNodeSVCs[i] = svc
			})
		}

		By("wait all VirtualNodes become active", func() {
			var waitErrors []error
			waitErrorsMutex := &sync.Mutex{}
			var wg sync.WaitGroup
			for i := 0; i != s.VirtualNodesCount; i++ {
				wg.Add(1)
				go func(nodeIndex int) {
					defer wg.Done()
					vn := s.createdNodeVNs[nodeIndex]
					vn, err := f.VNManager.WaitUntilVirtualNodeActive(ctx, vn)
					if err != nil {
						waitErrorsMutex.Lock()
						waitErrors = append(waitErrors, errors.Wrapf(err, "VirtualNode: %v", k8s.NamespacedName(vn).String()))
						waitErrorsMutex.Unlock()
						return
					}
					s.createdNodeVNs[nodeIndex] = vn
				}(i)
			}
			wg.Wait()
			for _, waitErr := range waitErrors {
				f.Logger.Error("failed to wait all VirtualNodes become active", zap.Error(waitErr))
			}
			Expect(len(waitErrors)).To(BeZero())
		})

		By("wait all deployments become ready", func() {
			var waitErrors []error
			waitErrorsMutex := &sync.Mutex{}
			var wg sync.WaitGroup
			for i := 0; i != s.VirtualNodesCount; i++ {
				wg.Add(1)
				go func(nodeIndex int) {
					defer wg.Done()
					dp := s.createdNodeDPs[nodeIndex]
					dp, err := f.DPManager.WaitUntilDeploymentReady(ctx, dp)
					if err != nil {
						waitErrorsMutex.Lock()
						waitErrors = append(waitErrors, errors.Wrapf(err, "Deployment: %v", k8s.NamespacedName(dp).String()))
						waitErrorsMutex.Unlock()
						return
					}
					s.createdNodeDPs[nodeIndex] = dp
				}(i)
			}
			wg.Wait()
			for _, waitErr := range waitErrors {
				f.Logger.Error("failed to wait all Deployments become active", zap.Error(waitErr))
			}
			Expect(len(waitErrors)).To(BeZero())
		})
	})
}

func (s *DynamicStack) deleteResourcesForNodes(ctx context.Context, f *framework.Framework) []error {
	var deletionErrors []error
	By("delete all resources for nodes", func() {
		for i, svc := range s.createdNodeSVCs {
			if svc == nil {
				continue
			}
			By(fmt.Sprintf("delete Service for node #%d", i), func() {
				if err := f.K8sClient.CoreV1().Services(svc.Namespace).Delete(svc.Name, &metav1.DeleteOptions{}); err != nil {
					f.Logger.Error("failed to delete Service",
						zap.String("namespace", svc.Namespace),
						zap.String("name", svc.Name),
						zap.Error(err),
					)
					deletionErrors = append(deletionErrors, err)
				}
			})
		}
		for i, dp := range s.createdNodeDPs {
			if dp == nil {
				continue
			}
			By(fmt.Sprintf("delete Deployment for node #%d", i), func() {
				foregroundDeletion := metav1.DeletePropagationForeground
				if err := f.K8sClient.AppsV1().Deployments(dp.Namespace).Delete(dp.Name, &metav1.DeleteOptions{
					GracePeriodSeconds: aws.Int64(0),
					PropagationPolicy:  &foregroundDeletion,
				}); err != nil {
					f.Logger.Error("failed to delete Deployment",
						zap.String("namespace", dp.Namespace),
						zap.String("name", dp.Name),
						zap.Error(err),
					)
					deletionErrors = append(deletionErrors, err)
				}
			})
		}
		for i, vn := range s.createdNodeVNs {
			if vn == nil {
				continue
			}
			By(fmt.Sprintf("delete VirtualNode for node #%d", i), func() {
				if err := f.K8sMeshClient.AppmeshV1beta1().VirtualNodes(vn.Namespace).Delete(vn.Name, &metav1.DeleteOptions{}); err != nil {
					f.Logger.Error("failed to delete VirtualNode",
						zap.String("namespace", vn.Namespace),
						zap.String("name", vn.Name),
						zap.Error(err),
					)
					deletionErrors = append(deletionErrors, err)
				}
			})
		}

		By("wait all deployments become deleted", func() {
			var waitErrors []error
			waitErrorsMutex := &sync.Mutex{}
			var wg sync.WaitGroup
			for i, dp := range s.createdNodeDPs {
				if dp == nil {
					continue
				}
				wg.Add(1)
				go func(nodeIndex int) {
					defer wg.Done()
					dp := s.createdNodeDPs[nodeIndex]
					if err := f.DPManager.WaitUntilDeploymentDeleted(ctx, dp); err != nil {
						waitErrorsMutex.Lock()
						waitErrors = append(waitErrors, errors.Wrapf(err, "Deployment: %v", k8s.NamespacedName(dp).String()))
						waitErrorsMutex.Unlock()
						return
					}
				}(i)
			}
			wg.Wait()
			for _, waitErr := range waitErrors {
				f.Logger.Error("failed to wait all Deployments become deleted", zap.Error(waitErr))
			}
			Expect(len(waitErrors)).To(BeZero())
		})

		By("wait all VirtualNodes become deleted", func() {
			var waitErrors []error
			waitErrorsMutex := &sync.Mutex{}
			var wg sync.WaitGroup
			for i, vn := range s.createdNodeVNs {
				if vn == nil {
					continue
				}
				wg.Add(1)
				go func(nodeIndex int) {
					defer wg.Done()
					vn := s.createdNodeVNs[nodeIndex]
					if err := f.VNManager.WaitUntilVirtualNodeDeleted(ctx, vn); err != nil {
						waitErrorsMutex.Lock()
						waitErrors = append(waitErrors, errors.Wrapf(err, "VirtualNode: %v", k8s.NamespacedName(vn).String()))
						waitErrorsMutex.Unlock()
						return
					}
				}(i)
			}
			wg.Wait()
			for _, waitErr := range waitErrors {
				f.Logger.Error("failed to wait all VirtualNode become deleted", zap.Error(waitErr))
			}
			Expect(len(waitErrors)).To(BeZero())
		})
	})
	return deletionErrors
}

func (s *DynamicStack) createResourcesForServices(ctx context.Context, f *framework.Framework, mb *shared.ManifestBuilder) {
	By("create all resources for services", func() {
		s.createdServiceVSs = make([]*appmeshv1beta1.VirtualService, s.VirtualServicesCount)
		s.createdServiceSVCs = make([]*corev1.Service, s.VirtualServicesCount)

		var err error
		nextVirtualNodeIndex := 0
		for i := 0; i != s.VirtualServicesCount; i++ {
			instanceName := fmt.Sprintf("service-%d", i)
			By(fmt.Sprintf("create VirtualService for service #%d", i), func() {
				var routeCfgs []shared.RouteToWeightedVirtualNodes
				for routeIndex := 0; routeIndex != s.RoutesCountPerVirtualRouter; routeIndex++ {
					var weightedTargets []shared.WeightedVirtualNode
					for targetIndex := 0; targetIndex != s.TargetsCountPerRoute; targetIndex++ {
						weightedTargets = append(weightedTargets, shared.WeightedVirtualNode{
							VirtualNodeName: s.createdNodeVNs[nextVirtualNodeIndex].Name,
							Weight:          1,
						})
						nextVirtualNodeIndex = (nextVirtualNodeIndex + 1) % s.VirtualNodesCount
					}
					routeCfgs = append(routeCfgs, shared.RouteToWeightedVirtualNodes{
						Path:            fmt.Sprintf("/path-%d", routeIndex),
						WeightedTargets: weightedTargets,
					})
				}
				vs := mb.BuildServiceVirtualService(instanceName, routeCfgs)
				vs, err := f.K8sMeshClient.AppmeshV1beta1().VirtualServices(s.namespace.Name).Create(vs)
				Expect(err).NotTo(HaveOccurred())
				s.createdServiceVSs[i] = vs
			})

			By(fmt.Sprintf("create Service for service #%d", i), func() {
				svc := mb.BuildServiceService(instanceName)
				svc, err = f.K8sClient.CoreV1().Services(s.namespace.Name).Create(svc)
				Expect(err).NotTo(HaveOccurred())
				s.createdServiceSVCs[i] = svc
			})
		}

		By("wait all VirtualService become active", func() {
			var waitErrors []error
			waitErrorsMutex := &sync.Mutex{}
			var wg sync.WaitGroup
			for i := 0; i != s.VirtualServicesCount; i++ {
				wg.Add(1)
				go func(serviceIndex int) {
					defer wg.Done()
					vs, err := f.VSManager.WaitUntilVirtualServiceActive(ctx, s.createdServiceVSs[serviceIndex])
					if err != nil {
						waitErrorsMutex.Lock()
						waitErrors = append(waitErrors, errors.Wrapf(err, "VirtualService: %v", k8s.NamespacedName(vs).String()))
						waitErrorsMutex.Unlock()
						return
					}
					s.createdServiceVSs[serviceIndex] = vs
				}(i)
			}
			wg.Wait()
			for _, waitErr := range waitErrors {
				f.Logger.Error("failed to wait all VirtualService become active", zap.Error(waitErr))
			}
			Expect(len(waitErrors)).To(BeZero())
		})
	})
}

func (s *DynamicStack) deleteResourcesForServices(ctx context.Context, f *framework.Framework) []error {
	var deletionErrors []error
	By("delete all resources for services", func() {
		for i, svc := range s.createdServiceSVCs {
			if svc == nil {
				continue
			}

			By(fmt.Sprintf("delete Service for service #%d", i), func() {
				if err := f.K8sClient.CoreV1().Services(svc.Namespace).Delete(svc.Name, &metav1.DeleteOptions{}); err != nil {
					f.Logger.Error("failed to delete Service",
						zap.String("namespace", svc.Namespace),
						zap.String("name", svc.Name),
						zap.Error(err),
					)
					deletionErrors = append(deletionErrors, err)
				}
			})
		}
		for i, vs := range s.createdServiceVSs {
			if vs == nil {
				continue
			}
			By(fmt.Sprintf("delete VirtualService for service #%d", i), func() {
				if err := f.K8sMeshClient.AppmeshV1beta1().VirtualServices(vs.Namespace).Delete(vs.Name, &metav1.DeleteOptions{}); err != nil {
					f.Logger.Error("failed to delete VirtualService",
						zap.String("namespace", vs.Namespace),
						zap.String("name", vs.Name),
						zap.Error(err),
					)
					deletionErrors = append(deletionErrors, err)
				}
			})
		}

		By("wait all VirtualService become deleted", func() {
			var waitErrors []error
			waitErrorsMutex := &sync.Mutex{}
			var wg sync.WaitGroup
			for i, vs := range s.createdServiceVSs {
				if vs == nil {
					continue
				}
				wg.Add(1)
				go func(serviceIndex int) {
					defer wg.Done()
					vs := s.createdServiceVSs[serviceIndex]
					if err := f.VSManager.WaitUntilVirtualServiceDeleted(ctx, vs); err != nil {
						waitErrorsMutex.Lock()
						waitErrors = append(waitErrors, errors.Wrapf(err, "VirtualService: %v", k8s.NamespacedName(vs).String()))
						waitErrorsMutex.Unlock()
						return
					}
				}(i)
			}
			wg.Wait()
			for _, waitErr := range waitErrors {
				f.Logger.Error("failed to wait all VirtualService become deleted", zap.Error(waitErr))
			}
			Expect(len(waitErrors)).To(BeZero())
		})
	})
	return deletionErrors
}

func (s *DynamicStack) grantVirtualNodesBackendAccess(ctx context.Context, f *framework.Framework) {
	By("granting VirtualNodes backend access", func() {
		nextVirtualServiceIndex := 0
		for i, vn := range s.createdNodeVNs {
			if vn == nil {
				continue
			}
			By(fmt.Sprintf("granting VirtualNode backend access for node #%d", i), func() {
				var vnBackends []appmeshv1beta1.Backend
				for backendIndex := 0; backendIndex != s.BackendsCountPerVirtualNode; backendIndex++ {
					vnBackends = append(vnBackends, appmeshv1beta1.Backend{
						VirtualService: appmeshv1beta1.VirtualServiceBackend{
							VirtualServiceName: s.createdServiceVSs[nextVirtualServiceIndex].Name,
						},
					})
					nextVirtualServiceIndex = (nextVirtualServiceIndex + 1) % s.VirtualServicesCount
				}

				vnNew := vn.DeepCopy()
				vnNew.Spec.Backends = vnBackends
				patch, err := k8s.CreateJSONMergePatch(vn, vnNew, appmeshv1beta1.VirtualNode{})
				Expect(err).NotTo(HaveOccurred())
				vnNew, err = f.K8sMeshClient.AppmeshV1beta1().VirtualNodes(vnNew.Namespace).Patch(vnNew.Name, types.MergePatchType, patch)
				Expect(err).NotTo(HaveOccurred())
				s.createdNodeVNs[i] = vnNew
			})
		}
	})
}

func (s *DynamicStack) revokeVirtualNodeBackendAccess(ctx context.Context, f *framework.Framework) []error {
	var deletionErrors []error
	By("revoking VirtualNodes backend access", func() {
		for i, vn := range s.createdNodeVNs {
			if vn == nil || len(vn.Spec.Backends) == 0 {
				continue
			}
			By(fmt.Sprintf("revoking VirtualNode backend access for node #%d", i), func() {
				vnNew := vn.DeepCopy()
				vnNew.Spec.Backends = nil
				patch, err := k8s.CreateJSONMergePatch(vn, vnNew, appmeshv1beta1.VirtualNode{})
				if err != nil {
					f.Logger.Error("failed to generate merge patch",
						zap.String("namespace", vn.Namespace),
						zap.String("name", vn.Name),
						zap.Error(err),
					)
					deletionErrors = append(deletionErrors, err)
					return
				}
				vnNew, err = f.K8sMeshClient.AppmeshV1beta1().VirtualNodes(vnNew.Namespace).Patch(vnNew.Name, types.MergePatchType, patch)
				if err != nil {
					f.Logger.Error("failed to revoke VirtualNode backend access",
						zap.String("namespace", vn.Namespace),
						zap.String("name", vn.Name),
						zap.Error(err),
					)
					deletionErrors = append(deletionErrors, err)
				}
			})
		}
	})
	return deletionErrors
}

func (s *DynamicStack) checkDeploymentToVirtualServiceConnectivity(ctx context.Context, f *framework.Framework,
	dp *appsv1.Deployment, vsList []*appmeshv1beta1.VirtualService) []error {
	sel := labels.Set(dp.Spec.Selector.MatchLabels)
	opts := metav1.ListOptions{LabelSelector: sel.AsSelector().String()}
	pods, err := f.K8sClient.CoreV1().Pods(dp.Namespace).List(opts)
	if err != nil {
		return []error{errors.Wrapf(err, "failed to get pods for Deployment: %v", k8s.NamespacedName(dp).String())}
	}
	if len(pods.Items) == 0 {
		return []error{errors.Wrapf(err, "Deployment have zero pods: %v", k8s.NamespacedName(dp).String())}
	}

	var checkErrors []error
	for i := range pods.Items {
		pod := pods.Items[i].DeepCopy()
		By(fmt.Sprintf("check pod %s/%s connectivity to services", pod.Namespace, pod.Name), func() {
			if errs := s.checkPodToVirtualServiceConnectivity(ctx, f, pod, vsList); len(errs) != 0 {
				checkErrors = append(checkErrors, errs...)
			}
		})
	}
	return checkErrors
}

func (s *DynamicStack) checkPodToVirtualServiceConnectivity(ctx context.Context, f *framework.Framework,
	pod *corev1.Pod, vsList []*appmeshv1beta1.VirtualService) []error {
	connectivityCheckEntries, err := s.obtainPodToVirtualServiceConnectivityEntries(ctx, f, pod, vsList)
	if err != nil {
		return []error{err}
	}

	retErrCounterByRUL := make(map[string]map[string]int)
	retStatusNotOKCounterByURL := make(map[string]map[int]int)
	retBodyCounterByURL := make(map[string]map[string]int)
	for _, checkEntry := range connectivityCheckEntries {
		if checkEntry.retErr != nil {
			if _, ok := retErrCounterByRUL[checkEntry.dstURL]; !ok {
				retErrCounterByRUL[checkEntry.dstURL] = make(map[string]int)
			}
			retErrCounterByRUL[checkEntry.dstURL][checkEntry.retErr.Error()] += 1
			continue
		}
		if checkEntry.retHTTPStatusCode != http.StatusOK {
			if _, ok := retStatusNotOKCounterByURL[checkEntry.dstURL]; !ok {
				retStatusNotOKCounterByURL[checkEntry.dstURL] = make(map[int]int)
			}
			retStatusNotOKCounterByURL[checkEntry.dstURL][checkEntry.retHTTPStatusCode] += 1
			continue
		}
		if _, ok := retBodyCounterByURL[checkEntry.dstURL]; !ok {
			retBodyCounterByURL[checkEntry.dstURL] = make(map[string]int)
		}
		retBodyCounterByURL[checkEntry.dstURL][checkEntry.retHTTPBody] += 1
	}

	var checkErrors []error
	for url, retErrCounter := range retErrCounterByRUL {
		for retErr, count := range retErrCounter {
			if s.ServiceDiscoveryType == shared.CloudMapServiceDiscovery {
				f.Logger.Warn("expect traffic from pod to URL succeed",
					zap.String("pod", k8s.NamespacedName(pod).String()),
					zap.String("url", url),
					zap.String("error", retErr),
					zap.Int("count", count),
				)
				continue
			}
			checkErrors = append(checkErrors, errors.Errorf("expect traffic from pod %v to URL %v succeed, got err: %v, count: %v",
				k8s.NamespacedName(pod).String(), url, retErr, count,
			))
		}
	}
	for url, retStatusNotOKCounter := range retStatusNotOKCounterByURL {
		for retStatusNotOK, count := range retStatusNotOKCounter {
			if s.ServiceDiscoveryType == shared.CloudMapServiceDiscovery {
				f.Logger.Warn("expect traffic from pod to URL succeed",
					zap.String("pod", k8s.NamespacedName(pod).String()),
					zap.String("url", url),
					zap.Int("status_code", retStatusNotOK),
					zap.Int("count", count),
				)
				continue
			}
			checkErrors = append(checkErrors, errors.Errorf("expect traffic from pod %v to URL %v succeed, got status_code: %v, count: %v",
				k8s.NamespacedName(pod).String(), url, retStatusNotOK, count,
			))
		}
	}

	uniformDist := distuv.ChiSquared{K: float64(s.TargetsCountPerRoute - 1)}
	var expectedHTTPRetCounts []float64
	for i := 0; i != s.TargetsCountPerRoute; i++ {
		expectedHTTPRetCounts = append(expectedHTTPRetCounts, float64(s.ConnectivityCheckPerURL)/float64(s.TargetsCountPerRoute))
	}
	for url, retBodyCounter := range retBodyCounterByURL {
		var actualHTTPRetCounts []float64
		actualHTTPRetCountLogFields := []zap.Field{zap.Namespace("distribution")}
		for retBody, count := range retBodyCounter {
			actualHTTPRetCounts = append(actualHTTPRetCounts, float64(count))
			actualHTTPRetCountLogFields = append(actualHTTPRetCountLogFields, zap.Int(retBody, count))
		}
		f.Logger.With(actualHTTPRetCountLogFields...).Info("traffic from pod to URL",
			zap.String("pod", k8s.NamespacedName(pod).String()),
			zap.String("url", url),
		)

		chiSqStatics := stat.ChiSquare(actualHTTPRetCounts, expectedHTTPRetCounts)
		pv := 1 - uniformDist.CDF(chiSqStatics)
		if pv < connectivityCheckUniformDistributionSL {
			f.Logger.Warn("expect traffic from pod to URL to be even distributed",
				zap.String("pod", k8s.NamespacedName(pod).String()),
				zap.String("url", url),
				zap.Float64("significance level", connectivityCheckUniformDistributionSL),
				zap.Float64("pValue", pv),
			)
		}
	}

	return checkErrors
}

// one entry of connectivity check result.
type connectivityCheckEntry struct {
	dstVirtualService types.NamespacedName
	dstURL            string

	retHTTPStatusCode int
	retHTTPBody       string
	retErr            error
}

func (s *DynamicStack) obtainPodToVirtualServiceConnectivityEntries(ctx context.Context, f *framework.Framework,
	pod *corev1.Pod, vsList []*appmeshv1beta1.VirtualService) ([]connectivityCheckEntry, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var checkEntries []connectivityCheckEntry
	for _, vs := range vsList {
		for _, route := range vs.Spec.Routes {
			path := route.Http.Match.Prefix
			checkEntries = append(checkEntries, connectivityCheckEntry{
				dstVirtualService: k8s.NamespacedName(vs),
				dstURL:            fmt.Sprintf("http://%s:%d%s", vs.Name, shared.AppContainerPort, path),
			})
		}
	}

	pfErrChan := make(chan error)
	pfReadyChan := make(chan struct{})
	portForwarder, err := k8s.NewPortForwarder(ctx, f.RestCfg, pod, []string{fmt.Sprintf("%d:%d", connectivityCheckProxyPort, shared.HttpProxyContainerPort)}, pfReadyChan)
	if err != nil {
		return nil, err
	}
	go func() {
		pfErrChan <- portForwarder.ForwardPorts()
	}()

	proxyURL, err := url.Parse(fmt.Sprintf("http://localhost:%d", connectivityCheckProxyPort))
	if err != nil {
		return nil, err
	}
	proxyClient := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}}

	checkEntriesRetChan := make(chan connectivityCheckEntry)
	go func() {
		var wg sync.WaitGroup
		throttle := time.Tick(connectivityCheckRate)
		for _, entry := range checkEntries {
			for i := 0; i != s.ConnectivityCheckPerURL; i++ {
				<-throttle
				wg.Add(1)
				go func(entry connectivityCheckEntry) {
					defer wg.Done()
					<-pfReadyChan
					resp, err := proxyClient.Get(entry.dstURL)
					if err != nil {
						entry.retErr = err
						checkEntriesRetChan <- entry
						return
					}
					entry.retHTTPStatusCode = resp.StatusCode
					body, err := ioutil.ReadAll(resp.Body)
					if err != nil {
						entry.retErr = err
						checkEntriesRetChan <- entry
						return
					}
					entry.retHTTPBody = string(body)
					checkEntriesRetChan <- entry
				}(entry)
			}
		}
		wg.Wait()
		close(checkEntriesRetChan)
	}()

	var checkEntriesRet []connectivityCheckEntry
	for ret := range checkEntriesRetChan {
		checkEntriesRet = append(checkEntriesRet, ret)
	}
	cancel()
	return checkEntriesRet, <-pfErrChan
}
