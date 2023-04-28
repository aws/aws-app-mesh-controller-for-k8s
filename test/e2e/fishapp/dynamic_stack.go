package fishapp

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/references"
	"k8s.io/apimachinery/pkg/util/sets"

	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/algorithm"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/k8s"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/manifest"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/utils"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
	. "github.com/onsi/ginkgo/v2"
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
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	connectivityCheckRate                  = time.Second / 100
	connectivityCheckProxyPort             = 8899
	connectivityCheckUniformDistributionSL = 0.001 // Significance level that traffic to targets are uniform distributed.
	AppContainerPort                       = 9080
	HttpProxyContainerPort                 = 8899
	defaultAppImage                        = "public.ecr.aws/e6v3k1j4/colorteller:v1"
	defaultHTTPProxyImage                  = "abhinavsingh/proxy.py:latest"
	caCertScript                           = "certs/ca_certs.sh"
	nodeCertScript                         = "certs/node_certs.sh"
	genericNodeCertCfgFile                 = "certs/node_cert.cfg"
	certsBasePath                          = "certs/"
	certsCfgFileSuffix                     = "_cert.cfg"
	certChainSuffix                        = "_cert_chain.pem"
	certKeySuffix                          = "_key.pem"
	caCertFile                             = "ca_cert.pem"
	envoyCACertPath                        = "/certs/ca_cert.pem"
	certCleanupScript                      = "certs/cleanup.sh"
	sdsDeployScript                        = "certs/sds_provider.sh"
	registerAgentIdentity                  = "certs/register_agent_entry.sh"
	registerWorkloadIdentity               = "certs/register_workload_entry.sh"
)

var (
	mTLSe2eValidationContext = "spiffe://mtls-e2e.aws"
)

// A dynamic generated stack designed to test app mesh integration :D
// Suppose given configuration below:
//
//	5 VirtualServicesCount
//	10 VirtualNodesCount
//	2 RoutesCountPerVirtualRouter
//	2 TargetsCountPerRoute
//	4 BackendsCountPerVirtualNode
//
// We will generate virtual service configuration & virtual node configuration follows:
// =======virtual services =========
//
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
//
// =======virtual nodes =========
//
//	vn1 -> vs1,vs2,vs3,vs4
//	vn2 -> vs5,vs1,vs2,vs3
//	vn3 -> vs4,vs5,vs1,vs2
//	...
//
// then we validate each virtual node can access each virtual service at every path, and calculates the target distribution
type DynamicStack struct {
	// service discovery type
	ServiceDiscoveryType manifest.ServiceDiscoveryType

	// tls
	IsTLSEnabled bool

	//mtls
	IsmTLSEnabled bool

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
	mesh              *appmesh.Mesh
	namespace         *corev1.Namespace
	cloudMapNamespace string

	createdNodeVNs  []*appmesh.VirtualNode
	createdNodeDPs  []*appsv1.Deployment
	createdNodeSVCs []*corev1.Service

	createdServiceVSs  []*appmesh.VirtualService
	createdServiceVRs  []*appmesh.VirtualRouter
	createdServiceSVCs []*corev1.Service

	BackendVNsByVR map[string][]string
	VNReferenceMap map[string][]*string
}

// expects the stack can be deployed to namespace successfully
func (s *DynamicStack) Deploy(ctx context.Context, f *framework.Framework) {
	s.createMeshAndNamespace(ctx, f)
	if s.ServiceDiscoveryType == manifest.CloudMapServiceDiscovery {
		s.createCloudMapNamespace(ctx, f)
		time.Sleep(1 * time.Minute)
	}
	mb := &manifest.ManifestBuilder{
		Namespace:            s.namespace.Name,
		ServiceDiscoveryType: s.ServiceDiscoveryType,
		CloudMapNamespace:    s.cloudMapNamespace,
	}
	if s.IsTLSEnabled {
		err := s.createCertificateAuthority(ctx, f)
		if err != nil {
			return
		}
		s.createResourcesForNodesWithTLS(ctx, f, mb)
	} else if s.IsmTLSEnabled {
		err := s.deploySDSProvider(ctx, f)
		if err != nil {
			f.Logger.Error("error creating sds provider")
			return
		}
		err = s.registerAgentSDSEntry()
		if err != nil {
			return
		}
		s.createResourcesForNodesWithmTLS(ctx, f, mb)
	} else {
		s.createResourcesForNodes(ctx, f, mb)
	}
	s.createResourcesForServices(ctx, f, mb)
	s.grantVirtualNodesBackendAccess(ctx, f)
	if s.IsmTLSEnabled {
		s.updateListenerSANsForNodes(ctx, f)
	}
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
	if s.ServiceDiscoveryType == manifest.CloudMapServiceDiscovery {
		if errs := s.deleteCloudMapNamespace(ctx, f); len(errs) != 0 {
			deletionErrors = append(deletionErrors, errs...)
		}
	}
	if errs := s.deleteMeshAndNamespace(ctx, f); len(errs) != 0 {
		deletionErrors = append(deletionErrors, errs...)
	}
	if s.IsTLSEnabled {
		if err := s.deleteCerts(); err != nil {
			f.Logger.Error("Certs clean up failed", zap.Error(err))
		}
	} else if s.IsmTLSEnabled {
		if err := s.deleteSDSProvider(); err != nil {
			f.Logger.Error("Certs clean up failed", zap.Error(err))
		}
	}

	for _, err := range deletionErrors {
		f.Logger.Error("clean up failed", zap.Error(err))
	}
	Expect(len(deletionErrors)).To(BeZero())
}

// Check connectivity and routing works correctly
func (s *DynamicStack) Check(ctx context.Context, f *framework.Framework) {
	// TODO: we can just record the mapping when allocate vn->vs, instead of re-compute it here
	vsIndexByKey := make(map[types.NamespacedName]int, len(s.createdServiceVSs))
	for i := 0; i != s.VirtualServicesCount; i++ {
		vsKey := k8s.NamespacedName(s.createdServiceVSs[i])
		vsIndexByKey[vsKey] = i
	}

	var checkErrors []error
	for i := 0; i != s.VirtualNodesCount; i++ {
		dp := s.createdNodeDPs[i]
		vn := s.createdNodeVNs[i]

		vsIndexes := sets.NewInt()
		for _, backend := range vn.Spec.Backends {
			vsKey := references.ObjectKeyForVirtualServiceReference(vn, *backend.VirtualService.VirtualServiceRef)
			vsIndex := vsIndexByKey[vsKey]
			vsIndexes.Insert(vsIndex)
		}

		if errs := s.checkDeploymentToVirtualServiceConnectivity(ctx, f, dp, vsIndexes); len(errs) != 0 {
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

	By("allocates a namespace", func() {
		if s.IsTLSEnabled {
			namespace := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "tls-e2e",
				},
			}
			err := f.K8sClient.Create(ctx, namespace)
			Expect(err).NotTo(HaveOccurred())
			s.namespace = namespace
		} else if s.IsmTLSEnabled {
			namespace := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "mtls-e2e",
				},
			}
			err := f.K8sClient.Create(ctx, namespace)
			Expect(err).NotTo(HaveOccurred())
			s.namespace = namespace
		} else {
			namespace, err := f.NSManager.AllocateNamespace(ctx, "appmesh")
			Expect(err).NotTo(HaveOccurred())
			s.namespace = namespace
		}
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

func (s *DynamicStack) deleteMeshAndNamespace(ctx context.Context, f *framework.Framework) []error {
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

func (s *DynamicStack) createCloudMapNamespace(ctx context.Context, f *framework.Framework) {
	cmNamespace := fmt.Sprintf("%s-%s", f.Options.ClusterName, utils.RandomDNS1123Label(6))
	if s.IsTLSEnabled {
		cmNamespace = "tls-e2e.svc.cluster.local"
	}
	By(fmt.Sprintf("create cloudMap namespace %s", cmNamespace), func() {
		resp, err := f.CloudMapClient.CreatePrivateDnsNamespaceWithContext(ctx, &servicediscovery.CreatePrivateDnsNamespaceInput{
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
			f.CloudMapClient.ListNamespacesPagesWithContext(ctx, &servicediscovery.ListNamespacesInput{}, func(output *servicediscovery.ListNamespacesOutput, b bool) bool {
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
				f.CloudMapClient.ListServicesPagesWithContext(ctx, &servicediscovery.ListServicesInput{
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
					f.CloudMapClient.ListInstancesPagesWithContext(ctx, &servicediscovery.ListInstancesInput{
						ServiceId: aws.String(cmServiceID),
					}, func(output *servicediscovery.ListInstancesOutput, b bool) bool {
						for _, ins := range output.Instances {
							cmInstanceIDs = append(cmInstanceIDs, aws.StringValue(ins.Id))
						}
						return false
					})

					for _, cmInstanceID := range cmInstanceIDs {
						if _, err := f.CloudMapClient.DeregisterInstanceWithContext(ctx, &servicediscovery.DeregisterInstanceInput{
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

					if _, err := f.CloudMapClient.DeleteServiceWithContext(ctx, &servicediscovery.DeleteServiceInput{
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
			if _, err := f.CloudMapClient.DeleteNamespaceWithContext(ctx, &servicediscovery.DeleteNamespaceInput{
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

func (s *DynamicStack) createCertificateAuthority(ctx context.Context, f *framework.Framework) error {
	_, err := exec.Command("/bin/sh", caCertScript).Output()
	if err != nil {
		return fmt.Errorf("error: %s", err)
	}
	return nil
}

func (s *DynamicStack) createTLSCertsForNodes(nodeName string) error {
	nodeCertCfgFile := certsBasePath + nodeName + certsCfgFileSuffix
	replaceExpr := "s/node/" + nodeName + "/g"
	cmd := exec.Command("sed", "-e", replaceExpr, genericNodeCertCfgFile)
	certFile, err := os.OpenFile(nodeCertCfgFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("error opening file: %s", err)
	}
	defer certFile.Close()
	cmd.Stdout = certFile
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("error: %s", err)
	}

	_, err = exec.Command("/bin/sh", nodeCertScript, nodeName).Output()
	if err != nil {
		return fmt.Errorf("error: %s", err)
	}
	return nil
}

func (s *DynamicStack) deleteCerts() error {
	fmt.Printf("Delete Certs")
	_, err := exec.Command("/bin/sh", certCleanupScript).Output()
	if err != nil {
		return fmt.Errorf("error %s", err)
	}
	return nil
}

func (s *DynamicStack) createSecretsForNodeResource(ctx context.Context, f *framework.Framework, mb *manifest.ManifestBuilder,
	nodeName string, nodeSecretName string) error {
	nodeCertChain := nodeName + certChainSuffix
	nodeKey := nodeName + certKeySuffix
	frontendTLSFiles := []string{caCertFile, nodeCertChain, nodeKey}
	secret := mb.BuildK8SSecretsFromPemFile(certsBasePath, frontendTLSFiles, nodeSecretName, f)
	err := f.K8sClient.Create(ctx, secret)
	if err != nil {
		return err
	}
	return nil
}

func (s *DynamicStack) deploySDSProvider(ctx context.Context, f *framework.Framework) error {
	_, err := exec.Command("/bin/sh", sdsDeployScript, "deploy").Output()
	if err != nil {
		return fmt.Errorf("error: %s", err)
	}
	// TODO - Convert this to a watch
	time.Sleep(30 * time.Second)
	return nil
}

func (s *DynamicStack) deleteSDSProvider() error {
	_, err := exec.Command("/bin/sh", sdsDeployScript, "delete").Output()
	if err != nil {
		return fmt.Errorf("error: %s", err)
	}
	return nil
}

func (s *DynamicStack) registerAgentSDSEntry() error {
	_, err := exec.Command("/bin/sh", registerAgentIdentity).Output()
	if err != nil {
		return fmt.Errorf("error: %s", err)
	}
	return nil
}

func (s *DynamicStack) registerVirtualNodeSDSEntry(nodeName string) error {
	_, err := exec.Command("/bin/sh", registerWorkloadIdentity, nodeName).Output()
	if err != nil {
		return fmt.Errorf("error: %s", err)
	}
	return nil
}

func (s *DynamicStack) createResourcesForNodes(ctx context.Context, f *framework.Framework, mb *manifest.ManifestBuilder) {
	By("create all resources for nodes", func() {
		s.createdNodeVNs = make([]*appmesh.VirtualNode, s.VirtualNodesCount)
		s.createdNodeDPs = make([]*appsv1.Deployment, s.VirtualNodesCount)
		s.createdNodeSVCs = make([]*corev1.Service, s.VirtualNodesCount)

		vnBuilder := &manifest.VNBuilder{
			ServiceDiscoveryType: s.ServiceDiscoveryType,
			Namespace:            s.namespace.Name,
			CloudMapNamespace:    s.cloudMapNamespace,
		}

		for i := 0; i != s.VirtualNodesCount; i++ {
			instanceName := fmt.Sprintf("node-%d", i)

			By(fmt.Sprintf("create VirtualNode for node #%d", i), func() {
				listeners := []appmesh.Listener{vnBuilder.BuildListener("http", 9080)}
				backends := []types.NamespacedName{}
				vn := vnBuilder.BuildVirtualNode(instanceName, backends, listeners, &appmesh.BackendDefaults{})
				err := f.K8sClient.Create(ctx, vn)
				Expect(err).NotTo(HaveOccurred())
				s.createdNodeVNs[i] = vn
			})

			By(fmt.Sprintf("create Deployment for node #%d", i), func() {
				containersInfo := []manifest.ContainerInfo{
					{
						Name:          "app",
						AppImage:      defaultAppImage,
						ContainerPort: AppContainerPort,
						Env: []corev1.EnvVar{
							{
								Name:  "SERVER_PORT",
								Value: fmt.Sprintf("%d", AppContainerPort),
							},
							{
								Name:  "COLOR",
								Value: instanceName,
							},
						},
					},
					{
						Name:          "http-proxy",
						AppImage:      defaultHTTPProxyImage,
						ContainerPort: HttpProxyContainerPort,
						Args: []string{
							"--hostname=0.0.0.0",
							fmt.Sprintf("--port=%d", HttpProxyContainerPort),
						},
					},
				}
				containers := mb.BuildContainerSpec(containersInfo)
				dp := mb.BuildDeployment(instanceName, s.ReplicasPerVirtualNode, containers, map[string]string{})
				err := f.K8sClient.Create(ctx, dp)
				Expect(err).NotTo(HaveOccurred())
				s.createdNodeDPs[i] = dp
			})

			By(fmt.Sprintf("create Service for node #%d", i), func() {
				svc := mb.BuildServiceWithSelector(instanceName, AppContainerPort, AppContainerPort)
				err := f.K8sClient.Create(ctx, svc)
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

		By("check all VirtualNode in aws", func() {
			var checkErrors []error
			checkErrorsMutex := &sync.Mutex{}
			var wg sync.WaitGroup
			for i := 0; i != s.VirtualNodesCount; i++ {
				wg.Add(1)
				go func(nodeIndex int) {
					defer wg.Done()
					vn := s.createdNodeVNs[nodeIndex]
					err := f.VNManager.CheckVirtualNodeInAWS(ctx, s.mesh, vn)
					if err != nil {
						checkErrorsMutex.Lock()
						checkErrors = append(checkErrors, errors.Wrapf(err, "VirtualNode: %v", k8s.NamespacedName(vn).String()))
						checkErrorsMutex.Unlock()
						return
					}
				}(i)
			}
			wg.Wait()
			for _, checkErr := range checkErrors {
				f.Logger.Error("failed to check all VirtualNodes in aws", zap.Error(checkErr))
			}
			Expect(len(checkErrors)).To(BeZero())
		})
	})
}

func (s *DynamicStack) createResourcesForNodesWithTLS(ctx context.Context, f *framework.Framework, mb *manifest.ManifestBuilder) {
	By("create all resources for nodes", func() {
		s.createdNodeVNs = make([]*appmesh.VirtualNode, s.VirtualNodesCount)
		s.createdNodeDPs = make([]*appsv1.Deployment, s.VirtualNodesCount)
		s.createdNodeSVCs = make([]*corev1.Service, s.VirtualNodesCount)

		vnBuilder := &manifest.VNBuilder{
			ServiceDiscoveryType: s.ServiceDiscoveryType,
			Namespace:            s.namespace.Name,
			CloudMapNamespace:    s.cloudMapNamespace,
		}

		for i := 0; i != s.VirtualNodesCount; i++ {
			instanceName := fmt.Sprintf("node-%d", i)
			By(fmt.Sprintf("create certs for node #%d", i), func() {
				err := s.createTLSCertsForNodes(instanceName)
				Expect(err).NotTo(HaveOccurred())
			})
			By(fmt.Sprintf("create certs for node #%d", i), func() {
				nodeSecretName := fmt.Sprintf("node-%d-tls", i)
				err := s.createSecretsForNodeResource(ctx, f, mb, instanceName, nodeSecretName)
				Expect(err).NotTo(HaveOccurred())
			})

			By(fmt.Sprintf("create VirtualNode for node #%d", i), func() {
				tlsEnforce := true
				nodeBackendDefaults := &appmesh.BackendDefaults{
					ClientPolicy: &appmesh.ClientPolicy{
						TLS: &appmesh.ClientPolicyTLS{
							Enforce: &tlsEnforce,
							Ports:   nil,
							Validation: appmesh.TLSValidationContext{
								Trust: appmesh.TLSValidationContextTrust{
									ACM:  nil,
									File: &appmesh.TLSValidationContextFileTrust{CertificateChain: envoyCACertPath},
								},
							},
						},
					},
				}

				nodeCertificateChain := "/certs/" + instanceName + certChainSuffix
				nodePrivateKey := "/certs/" + instanceName + certKeySuffix

				nodeListenerTLS := &appmesh.ListenerTLS{
					Certificate: appmesh.ListenerTLSCertificate{
						File: &appmesh.ListenerTLSFileCertificate{
							CertificateChain: nodeCertificateChain,
							PrivateKey:       nodePrivateKey,
						},
					},
					Mode: "STRICT",
				}
				listeners := []appmesh.Listener{vnBuilder.BuildListenerWithTLS("http", AppContainerPort, nodeListenerTLS)}
				vn := vnBuilder.BuildVirtualNode(instanceName, nil, listeners, nodeBackendDefaults)
				err := f.K8sClient.Create(ctx, vn)
				Expect(err).NotTo(HaveOccurred())
				s.createdNodeVNs[i] = vn
			})

			By(fmt.Sprintf("create Deployment for node #%d", i), func() {
				nodeSecretName := fmt.Sprintf("node-%d-tls", i)
				certsPath := nodeSecretName + ":/certs/"
				annotations := map[string]string{
					"appmesh.k8s.aws/secretMounts": certsPath,
				}
				containersInfo := []manifest.ContainerInfo{
					{
						Name:          "app",
						AppImage:      defaultAppImage,
						ContainerPort: AppContainerPort,
						Env: []corev1.EnvVar{
							{
								Name:  "SERVER_PORT",
								Value: fmt.Sprintf("%d", AppContainerPort),
							},
							{
								Name:  "COLOR",
								Value: instanceName,
							},
						},
					},
					{
						Name:          "http-proxy",
						AppImage:      defaultHTTPProxyImage,
						ContainerPort: HttpProxyContainerPort,
						Args: []string{
							"--hostname=0.0.0.0",
							fmt.Sprintf("--port=%d", HttpProxyContainerPort),
						},
					},
				}
				containers := mb.BuildContainerSpec(containersInfo)
				dp := mb.BuildDeployment(instanceName, s.ReplicasPerVirtualNode, containers, annotations)
				err := f.K8sClient.Create(ctx, dp)
				Expect(err).NotTo(HaveOccurred())
				s.createdNodeDPs[i] = dp
			})

			By(fmt.Sprintf("create Service for node #%d", i), func() {
				svc := mb.BuildServiceWithSelector(instanceName, AppContainerPort, AppContainerPort)
				err := f.K8sClient.Create(ctx, svc)
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

		By("check all VirtualNode in aws", func() {
			var checkErrors []error
			checkErrorsMutex := &sync.Mutex{}
			var wg sync.WaitGroup
			for i := 0; i != s.VirtualNodesCount; i++ {
				wg.Add(1)
				go func(nodeIndex int) {
					defer wg.Done()
					vn := s.createdNodeVNs[nodeIndex]
					err := f.VNManager.CheckVirtualNodeInAWS(ctx, s.mesh, vn)
					if err != nil {
						checkErrorsMutex.Lock()
						checkErrors = append(checkErrors, errors.Wrapf(err, "VirtualNode: %v", k8s.NamespacedName(vn).String()))
						checkErrorsMutex.Unlock()
						return
					}
				}(i)
			}
			wg.Wait()
			for _, checkErr := range checkErrors {
				f.Logger.Error("failed to check all VirtualNodes in aws", zap.Error(checkErr))
			}
			Expect(len(checkErrors)).To(BeZero())
		})
	})
}

func (s *DynamicStack) createResourcesForNodesWithmTLS(ctx context.Context, f *framework.Framework, mb *manifest.ManifestBuilder) {
	By("create all resources for nodes", func() {
		s.createdNodeVNs = make([]*appmesh.VirtualNode, s.VirtualNodesCount)
		s.createdNodeDPs = make([]*appsv1.Deployment, s.VirtualNodesCount)
		s.createdNodeSVCs = make([]*corev1.Service, s.VirtualNodesCount)

		vnBuilder := &manifest.VNBuilder{
			ServiceDiscoveryType: s.ServiceDiscoveryType,
			Namespace:            s.namespace.Name,
			CloudMapNamespace:    s.cloudMapNamespace,
		}

		for i := 0; i != s.VirtualNodesCount; i++ {
			instanceName := fmt.Sprintf("node-%d", i)
			By(fmt.Sprintf("register workload entry with SDS Provider for node #%d", i), func() {
				err := s.registerVirtualNodeSDSEntry(instanceName)
				Expect(err).NotTo(HaveOccurred())
			})

			By(fmt.Sprintf("create VirtualNode for node #%d", i), func() {
				tlsEnforce := true
				nodeSVID := mTLSe2eValidationContext + "/" + instanceName
				nodeBackendDefaults := &appmesh.BackendDefaults{
					ClientPolicy: &appmesh.ClientPolicy{
						TLS: &appmesh.ClientPolicyTLS{
							Enforce: &tlsEnforce,
							Ports:   nil,
							Validation: appmesh.TLSValidationContext{
								Trust: appmesh.TLSValidationContextTrust{
									SDS: &appmesh.TLSValidationContextSDSTrust{
										SecretName: &mTLSe2eValidationContext,
									},
								},
								SubjectAlternativeNames: &appmesh.SubjectAlternativeNames{
									Match: &appmesh.SubjectAlternativeNameMatchers{
										Exact: []*string{
											&nodeSVID,
										},
									},
								},
							},
							Certificate: &appmesh.ClientTLSCertificate{
								SDS: &appmesh.ListenerTLSSDSCertificate{
									SecretName: &nodeSVID,
								},
							},
						},
					},
				}

				nodeListenerTLS := &appmesh.ListenerTLS{
					Certificate: appmesh.ListenerTLSCertificate{
						SDS: &appmesh.ListenerTLSSDSCertificate{
							SecretName: &nodeSVID,
						},
					},
					Validation: &appmesh.ListenerTLSValidationContext{
						Trust: appmesh.ListenerTLSValidationContextTrust{
							SDS: &appmesh.TLSValidationContextSDSTrust{
								SecretName: &mTLSe2eValidationContext,
							},
						},
						SubjectAlternativeNames: &appmesh.SubjectAlternativeNames{
							Match: &appmesh.SubjectAlternativeNameMatchers{
								Exact: []*string{
									&nodeSVID,
								},
							},
						},
					},
					Mode: "STRICT",
				}
				listeners := []appmesh.Listener{vnBuilder.BuildListenerWithTLS("http", AppContainerPort, nodeListenerTLS)}
				vn := vnBuilder.BuildVirtualNode(instanceName, nil, listeners, nodeBackendDefaults)
				err := f.K8sClient.Create(ctx, vn)
				Expect(err).NotTo(HaveOccurred())
				s.createdNodeVNs[i] = vn
			})

			By(fmt.Sprintf("create Deployment for node #%d", i), func() {
				containersInfo := []manifest.ContainerInfo{
					{
						Name:          "app",
						AppImage:      defaultAppImage,
						ContainerPort: AppContainerPort,
						Env: []corev1.EnvVar{
							{
								Name:  "SERVER_PORT",
								Value: fmt.Sprintf("%d", AppContainerPort),
							},
							{
								Name:  "COLOR",
								Value: instanceName,
							},
						},
					},
					{
						Name:          "http-proxy",
						AppImage:      defaultHTTPProxyImage,
						ContainerPort: HttpProxyContainerPort,
						Args: []string{
							"--hostname=0.0.0.0",
							fmt.Sprintf("--port=%d", HttpProxyContainerPort),
						},
					},
				}
				containers := mb.BuildContainerSpec(containersInfo)
				dp := mb.BuildDeployment(instanceName, s.ReplicasPerVirtualNode, containers, nil)
				err := f.K8sClient.Create(ctx, dp)
				Expect(err).NotTo(HaveOccurred())
				s.createdNodeDPs[i] = dp
			})

			By(fmt.Sprintf("create Service for node #%d", i), func() {
				svc := mb.BuildServiceWithSelector(instanceName, AppContainerPort, AppContainerPort)
				err := f.K8sClient.Create(ctx, svc)
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

		By("check all VirtualNode in aws", func() {
			var checkErrors []error
			checkErrorsMutex := &sync.Mutex{}
			var wg sync.WaitGroup
			for i := 0; i != s.VirtualNodesCount; i++ {
				wg.Add(1)
				go func(nodeIndex int) {
					defer wg.Done()
					vn := s.createdNodeVNs[nodeIndex]
					err := f.VNManager.CheckVirtualNodeInAWS(ctx, s.mesh, vn)
					if err != nil {
						checkErrorsMutex.Lock()
						checkErrors = append(checkErrors, errors.Wrapf(err, "VirtualNode: %v", k8s.NamespacedName(vn).String()))
						checkErrorsMutex.Unlock()
						return
					}
				}(i)
			}
			wg.Wait()
			for _, checkErr := range checkErrors {
				f.Logger.Error("failed to check all VirtualNodes in aws", zap.Error(checkErr))
			}
			Expect(len(checkErrors)).To(BeZero())
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
				if err := f.K8sClient.Delete(ctx, svc); err != nil {
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
				if err := f.K8sClient.Delete(ctx, dp,
					client.PropagationPolicy(metav1.DeletePropagationForeground), client.GracePeriodSeconds(0)); err != nil {
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
				if err := f.K8sClient.Delete(ctx, vn); err != nil {
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

func (s *DynamicStack) createResourcesForServices(ctx context.Context, f *framework.Framework, mb *manifest.ManifestBuilder) {
	By("create all resources for services", func() {
		s.createdServiceVSs = make([]*appmesh.VirtualService, s.VirtualServicesCount)
		s.createdServiceVRs = make([]*appmesh.VirtualRouter, s.VirtualServicesCount)
		s.createdServiceSVCs = make([]*corev1.Service, s.VirtualServicesCount)
		s.BackendVNsByVR = make(map[string][]string)

		vrBuilder := &manifest.VRBuilder{
			Namespace: s.namespace.Name,
		}
		vsBuilder := &manifest.VSBuilder{
			Namespace: s.namespace.Name,
		}

		nextVirtualNodeIndex := 0
		for i := 0; i != s.VirtualServicesCount; i++ {
			instanceName := fmt.Sprintf("service-%d", i)
			var VRBackends []string
			By(fmt.Sprintf("create VirtualRouter for service #%d", i), func() {
				var routeCfgs []manifest.RouteToWeightedVirtualNodes
				for routeIndex := 0; routeIndex != s.RoutesCountPerVirtualRouter; routeIndex++ {
					var weightedTargets []manifest.WeightedVirtualNode
					for targetIndex := 0; targetIndex != s.TargetsCountPerRoute; targetIndex++ {
						weightedTargets = append(weightedTargets, manifest.WeightedVirtualNode{
							VirtualNode: k8s.NamespacedName(s.createdNodeVNs[nextVirtualNodeIndex]),
							Weight:      1,
						})
						VRBackends = append(VRBackends, s.createdNodeVNs[nextVirtualNodeIndex].Name)
						nextVirtualNodeIndex = (nextVirtualNodeIndex + 1) % s.VirtualNodesCount
					}
					routeCfgs = append(routeCfgs, manifest.RouteToWeightedVirtualNodes{
						Path:            fmt.Sprintf("/path-%d", routeIndex),
						WeightedTargets: weightedTargets,
					})
				}

				routes := vrBuilder.BuildRoutes(routeCfgs)
				vrBuilder.Listeners = []appmesh.VirtualRouterListener{vrBuilder.BuildVirtualRouterListener("http", AppContainerPort)}

				vr := vrBuilder.BuildVirtualRouter(instanceName, routes)
				err := f.K8sClient.Create(ctx, vr)
				Expect(err).NotTo(HaveOccurred())
				s.createdServiceVRs[i] = vr
				s.BackendVNsByVR[instanceName] = VRBackends
			})

			By(fmt.Sprintf("create VirtualService for service #%d", i), func() {
				vs := vsBuilder.BuildVirtualServiceWithRouterBackend(instanceName, instanceName)
				err := f.K8sClient.Create(ctx, vs)
				Expect(err).NotTo(HaveOccurred())
				s.createdServiceVSs[i] = vs
			})

			By(fmt.Sprintf("create Service for service #%d", i), func() {
				svc := mb.BuildServiceWithSelector(instanceName, AppContainerPort, AppContainerPort)
				err := f.K8sClient.Create(ctx, svc)
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

		By("check all VirtualRouters in AWS", func() {
			var checkErrors []error
			checkErrorsMutex := &sync.Mutex{}
			var wg sync.WaitGroup
			for i := 0; i != s.VirtualServicesCount; i++ {
				wg.Add(1)
				go func(serviceIndex int) {
					defer wg.Done()
					vr := s.createdServiceVRs[serviceIndex]
					err := f.VRManager.CheckVirtualRouterInAWS(ctx, s.mesh, vr)
					if err != nil {
						checkErrorsMutex.Lock()
						checkErrors = append(checkErrors, errors.Wrapf(err, "VirtualRouter: %v", k8s.NamespacedName(vr).String()))
						checkErrorsMutex.Unlock()
						return
					}
				}(i)
			}
			wg.Wait()
			for _, checkErr := range checkErrors {
				f.Logger.Error("failed to check all VirtualRouters in AWS", zap.Error(checkErr))
			}
			Expect(len(checkErrors)).To(BeZero())
		})

		By("check all VirtualService in AWS", func() {
			var checkErrors []error
			checkErrorsMutex := &sync.Mutex{}
			var wg sync.WaitGroup
			for i := 0; i != s.VirtualServicesCount; i++ {
				wg.Add(1)
				go func(serviceIndex int) {
					defer wg.Done()
					vs := s.createdServiceVSs[serviceIndex]
					err := f.VSManager.CheckVirtualServiceInAWS(ctx, s.mesh, vs)
					if err != nil {
						checkErrorsMutex.Lock()
						checkErrors = append(checkErrors, errors.Wrapf(err, "VirtualService: %v", k8s.NamespacedName(vs).String()))
						checkErrorsMutex.Unlock()
						return
					}
				}(i)
			}
			wg.Wait()
			for _, checkErr := range checkErrors {
				f.Logger.Error("failed to check all VirtualService in AWS", zap.Error(checkErr))
			}
			Expect(len(checkErrors)).To(BeZero())
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
				if err := f.K8sClient.Delete(ctx, svc); err != nil {
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
				if err := f.K8sClient.Delete(ctx, vs); err != nil {
					f.Logger.Error("failed to delete VirtualService",
						zap.String("namespace", vs.Namespace),
						zap.String("name", vs.Name),
						zap.Error(err),
					)
					deletionErrors = append(deletionErrors, err)
				}
			})
		}
		for i, vr := range s.createdServiceVRs {
			if vr == nil {
				continue
			}
			By(fmt.Sprintf("delete VirtualRouter for service #%d", i), func() {
				if err := f.K8sClient.Delete(ctx, vr); err != nil {
					f.Logger.Error("failed to delete VirtualRouter",
						zap.String("namespace", vr.Namespace),
						zap.String("name", vr.Name),
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
		By("wait all VirtualRouter become deleted", func() {
			var waitErrors []error
			waitErrorsMutex := &sync.Mutex{}
			var wg sync.WaitGroup
			for i, vr := range s.createdServiceVRs {
				if vr == nil {
					continue
				}
				wg.Add(1)
				go func(serviceIndex int) {
					defer wg.Done()
					vr := s.createdServiceVRs[serviceIndex]
					if err := f.VRManager.WaitUntilVirtualRouterDeleted(ctx, vr); err != nil {
						waitErrorsMutex.Lock()
						waitErrors = append(waitErrors, errors.Wrapf(err, "VirtualRouter: %v", k8s.NamespacedName(vr).String()))
						waitErrorsMutex.Unlock()
						return
					}
				}(i)
			}
			wg.Wait()
			for _, waitErr := range waitErrors {
				f.Logger.Error("failed to wait all VirtualRouter become deleted", zap.Error(waitErr))
			}
			Expect(len(waitErrors)).To(BeZero())
		})
	})
	return deletionErrors
}

func (s *DynamicStack) grantVirtualNodesBackendAccess(ctx context.Context, f *framework.Framework) {
	By("granting VirtualNodes backend access", func() {
		s.VNReferenceMap = make(map[string][]*string)
		nextVirtualServiceIndex := 0
		for i, vn := range s.createdNodeVNs {
			if vn == nil {
				continue
			}
			By(fmt.Sprintf("granting VirtualNode backend access for node #%d", i), func() {
				var vnBackends []appmesh.Backend
				var backendSANs []*string
				backendSANsMap := make(map[string]bool)
				for backendIndex := 0; backendIndex != s.BackendsCountPerVirtualNode; backendIndex++ {
					vs := s.createdServiceVSs[nextVirtualServiceIndex]
					vnBackends = append(vnBackends, appmesh.Backend{
						VirtualService: appmesh.VirtualServiceBackend{
							VirtualServiceRef: &appmesh.VirtualServiceReference{
								Namespace: aws.String(vs.Namespace),
								Name:      vs.Name,
							},
						},
					})
					if s.IsmTLSEnabled {
						for _, backendVN := range s.BackendVNsByVR[vs.Name] {
							if _, ok := backendSANsMap[backendVN]; ok {
								continue
							}
							backendSANsMap[backendVN] = true
							listenerSVID := mTLSe2eValidationContext + "/" + vn.Name
							backendSVID := mTLSe2eValidationContext + "/" + backendVN
							s.VNReferenceMap[backendVN] = append(s.VNReferenceMap[backendVN], &listenerSVID)
							backendSANs = append(backendSANs, &backendSVID)
						}
					}
					nextVirtualServiceIndex = (nextVirtualServiceIndex + 1) % s.VirtualServicesCount
				}

				vnNew := vn.DeepCopy()
				vnNew.Spec.Backends = vnBackends
				if s.IsmTLSEnabled {
					vnNew.Spec.BackendDefaults.ClientPolicy.TLS.Validation.SubjectAlternativeNames.Match.Exact = backendSANs
				}

				err := f.K8sClient.Patch(ctx, vnNew, client.MergeFrom(vn))
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

				err := f.K8sClient.Patch(ctx, vnNew, client.MergeFrom(vn))
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

func (s *DynamicStack) updateListenerSANsForNodes(ctx context.Context, f *framework.Framework) {
	By("updating VirtualNodes listener SAN values", func() {
		for i, vn := range s.createdNodeVNs {
			if vn == nil {
				continue
			}
			By(fmt.Sprintf("updating VirtualNodes listener SAN values for node #%d", i), func() {
				vnNew := vn.DeepCopy()
				listenerSANs := s.VNReferenceMap[vn.Name]
				vnNew.Spec.Listeners[0].TLS.Validation.SubjectAlternativeNames.Match.Exact = listenerSANs
				err := f.K8sClient.Patch(ctx, vnNew, client.MergeFrom(vn))
				Expect(err).NotTo(HaveOccurred())
			})
		}
	})
}

func (s *DynamicStack) checkDeploymentToVirtualServiceConnectivity(ctx context.Context, f *framework.Framework,
	dp *appsv1.Deployment, vsIndexes sets.Int) []error {
	sel := labels.Set(dp.Spec.Selector.MatchLabels)
	podList := &corev1.PodList{}
	err := f.K8sClient.List(ctx, podList, client.InNamespace(dp.Namespace), client.MatchingLabelsSelector{Selector: sel.AsSelector()})
	if err != nil {
		return []error{errors.Wrapf(err, "failed to get pods for Deployment: %v", k8s.NamespacedName(dp).String())}
	}
	if len(podList.Items) == 0 {
		return []error{errors.Wrapf(err, "Deployment have zero pods: %v", k8s.NamespacedName(dp).String())}
	}

	var checkErrors []error
	for i := range podList.Items {
		pod := podList.Items[i].DeepCopy()
		By(fmt.Sprintf("check pod %s/%s connectivity to services", pod.Namespace, pod.Name), func() {
			if errs := s.checkPodToVirtualServiceConnectivity(ctx, f, pod, vsIndexes); len(errs) != 0 {
				checkErrors = append(checkErrors, errs...)
			}
		})
	}
	return checkErrors
}

func (s *DynamicStack) checkPodToVirtualServiceConnectivity(ctx context.Context, f *framework.Framework,
	pod *corev1.Pod, vsIndexes sets.Int) []error {
	connectivityCheckEntries, err := s.obtainPodToVirtualServiceConnectivityEntries(ctx, f, pod, vsIndexes)
	if err != nil {
		return []error{err}
	}

	retErrCounterByRUL := make(map[string]map[string]int)
	retStatusNotOKCounterByURL := make(map[string]map[int]int)
	retBodyCounterByURL := make(map[string]map[string]int)
	for _, checkEntry := range connectivityCheckEntries {
		if _, ok := retBodyCounterByURL[checkEntry.dstURL]; !ok {
			retBodyCounterByURL[checkEntry.dstURL] = make(map[string]int)
		}
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

		retBodyCounterByURL[checkEntry.dstURL][checkEntry.retHTTPBody] += 1
	}

	var checkErrors []error
	for url, retErrCounter := range retErrCounterByRUL {
		for retErr, count := range retErrCounter {
			f.Logger.Warn("expect traffic from pod to URL succeed",
				zap.String("pod", k8s.NamespacedName(pod).String()),
				zap.String("url", url),
				zap.String("error", retErr),
				zap.Int("count", count),
			)
			checkErrors = append(checkErrors, errors.Errorf("expect traffic from pod %v to URL %v succeed, got err: %v, count: %v",
				k8s.NamespacedName(pod).String(), url, retErr, count,
			))
		}
	}
	for url, retStatusNotOKCounter := range retStatusNotOKCounterByURL {
		for retStatusNotOK, count := range retStatusNotOKCounter {
			f.Logger.Warn("expect traffic from pod to URL succeed",
				zap.String("pod", k8s.NamespacedName(pod).String()),
				zap.String("url", url),
				zap.Int("status_code", retStatusNotOK),
				zap.Int("count", count),
			)
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
		httpRetCountsDiff := len(expectedHTTPRetCounts) - len(actualHTTPRetCounts)
		for i := 0; i < httpRetCountsDiff; i++ {
			actualHTTPRetCounts = append(actualHTTPRetCounts, 0)
		}

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
	pod *corev1.Pod, vsIndexes sets.Int) ([]connectivityCheckEntry, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var checkEntries []connectivityCheckEntry
	for vsIndex := range vsIndexes {
		vs := s.createdServiceVSs[vsIndex]
		vr := s.createdServiceVRs[vsIndex]
		for _, route := range vr.Spec.Routes {
			path := aws.StringValue(route.HTTPRoute.Match.Prefix)
			checkEntries = append(checkEntries, connectivityCheckEntry{
				dstVirtualService: k8s.NamespacedName(vs),
				dstURL:            fmt.Sprintf("http://%s:%d%s", aws.StringValue(vs.Spec.AWSName), AppContainerPort, path),
			})
		}
	}

	pfErrChan := make(chan error)
	pfReadyChan := make(chan struct{})
	portForwarder, err := k8s.NewPortForwarder(ctx, f.RestCfg, pod, []string{fmt.Sprintf("%d:%d", connectivityCheckProxyPort, HttpProxyContainerPort)}, pfReadyChan)
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
