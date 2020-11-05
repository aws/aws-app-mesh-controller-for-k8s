package virtualnode_test

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
	appsv1 "k8s.io/api/apps/v1"
	"sync"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"go.uber.org/zap"

	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/algorithm"
	appmeshk8s "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/k8s"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/manifest"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/utils"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/integration/mesh"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/integration/virtualnode"

	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultAppImage  = "970805265562.dkr.ecr.us-west-2.amazonaws.com/colorteller:latest"
	AppContainerPort = 8080
)

var _ = Describe("VirtualNode", func() {

	var (
		ctx context.Context
		f   *framework.Framework
	)

	BeforeEach(func() {
		ctx = context.Background()
		if f == nil {
			f = framework.New(framework.GlobalOptions)
		}

		if f.Options.ControllerImage != "" {
			By("Reset cluster with default controller", func() {
				f.HelmManager.ResetAppMeshController()
			})
		}
		if f.Options.InjectorImage != "" {
			By("Reset cluster with default injector", func() {
				f.HelmManager.ResetAppMeshInjector()
			})
		}
	})

	Context("Virtual Node scenarios", func() {
		var meshTest mesh.MeshTest
		var vnTest virtualnode.VirtualNodeTest

		meshTest = mesh.MeshTest{
			Meshes: make(map[string]*appmesh.Mesh),
		}

		vnTest = virtualnode.VirtualNodeTest{
			VirtualNodes: make(map[string]*appmesh.VirtualNode),
		}

		vnBuilder := &manifest.VNBuilder{
			ServiceDiscoveryType: manifest.DNSServiceDiscovery,
		}

		AfterEach(func() {
			vnTest.Cleanup(ctx, f)
			meshTest.Cleanup(ctx, f)
		})

		It("should create a virtual node in AWS", func() {

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

			By("creating a mesh resource in k8s", func() {
				err := meshTest.Create(ctx, f, mesh)
				Expect(err).NotTo(HaveOccurred())
			})

			By("validating the resources in AWS", func() {
				err := meshTest.CheckInAWS(ctx, f, mesh)
				Expect(err).NotTo(HaveOccurred())

			})

			By("Create a namespace and add labels", func() {
				namespace, err := f.NSManager.AllocateNamespace(ctx, "appmeshtest")
				Expect(err).NotTo(HaveOccurred())
				vnBuilder.Namespace = namespace.Name
				vnTest.Namespace = namespace

				oldNS := namespace.DeepCopy()
				namespace.Labels = algorithm.MergeStringMap(map[string]string{
					"appmesh.k8s.aws/sidecarInjectorWebhook": "enabled",
					"mesh":                                   meshName,
				}, namespace.Labels)

				err = f.K8sClient.Patch(ctx, namespace, client.MergeFrom(oldNS))
				Expect(err).NotTo(HaveOccurred())
			})

			vnName := fmt.Sprintf("vn-%s", utils.RandomDNS1123Label(8))
			listeners := []appmesh.Listener{vnBuilder.BuildListener("http", 8080)}
			backends := []types.NamespacedName{}
			vn := vnBuilder.BuildVirtualNode(vnName, backends, listeners, &appmesh.BackendDefaults{})

			By("Creating a virtual node resource in k8s", func() {
				err := vnTest.Create(ctx, f, vn)
				Expect(err).NotTo(HaveOccurred())
			})

			By("validating the virtual node in AWS", func() {
				err := vnTest.CheckInAWS(ctx, f, mesh, vn)
				Expect(err).NotTo(HaveOccurred())

			})

			By("Set incorrect labels on namespace", func() {
				oldNS := vnTest.Namespace.DeepCopy()
				vnTest.Namespace.Labels = algorithm.MergeStringMap(map[string]string{
					"appmesh.k8s.aws/sidecarInjectorWebhook": "enabled",
					"mesh":                                   "dontmatch",
				}, vnTest.Namespace.Labels)

				err := f.K8sClient.Patch(ctx, vnTest.Namespace, client.MergeFrom(oldNS))
				Expect(err).NotTo(HaveOccurred())
			})

			vnName = fmt.Sprintf("vn-%s", utils.RandomDNS1123Label(8))
			vn = vnBuilder.BuildVirtualNode(vnName, backends, listeners, &appmesh.BackendDefaults{})

			By("Creating a virtual node resource in k8s when no mesh matches namespace", func() {
				err := vnTest.Create(ctx, f, vn)
				Expect(err).To(HaveOccurred())
			})

		})

		It("should create a virtual node with CloudMap ServiceDiscovery enabled", func() {

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

			By("creating a mesh resource in k8s", func() {
				err := meshTest.Create(ctx, f, mesh)
				Expect(err).NotTo(HaveOccurred())
			})

			By("validating the resources in AWS", func() {
				err := meshTest.CheckInAWS(ctx, f, mesh)
				Expect(err).NotTo(HaveOccurred())

			})

			cmNamespace := fmt.Sprintf("%s-%s", f.Options.ClusterName, utils.RandomDNS1123Label(6))
			By(fmt.Sprintf("create cloudMap namespace %s", cmNamespace), func() {
				resp, err := f.CloudMapClient.CreatePrivateDnsNamespaceWithContext(ctx, &servicediscovery.CreatePrivateDnsNamespaceInput{
					Name: aws.String(cmNamespace),
					Vpc:  aws.String(f.Options.AWSVPCID),
				})
				Expect(err).NotTo(HaveOccurred())
				f.Logger.Info("created cloudMap namespace",
					zap.String("namespace", cmNamespace),
					zap.String("operationID", aws.StringValue(resp.OperationId)),
				)
				vnTest.CloudMapNameSpace = cmNamespace
			})
			//Allow CloudMap Namespace to sync
			time.Sleep(30 * time.Second)

			vnBuilder := &manifest.VNBuilder{
				ServiceDiscoveryType: manifest.CloudMapServiceDiscovery,
				CloudMapNamespace:    cmNamespace,
			}

			mb := &manifest.ManifestBuilder{
				ServiceDiscoveryType: manifest.CloudMapServiceDiscovery,
			}

			By("Create a namespace and add labels", func() {
				namespace, err := f.NSManager.AllocateNamespace(ctx, "appmeshtest")
				Expect(err).NotTo(HaveOccurred())
				vnBuilder.Namespace = namespace.Name
				vnTest.Namespace = namespace
				vnTest.Deployments = make(map[string]*appsv1.Deployment)
				mb.Namespace = namespace.Name

				oldNS := namespace.DeepCopy()
				namespace.Labels = algorithm.MergeStringMap(map[string]string{
					"appmesh.k8s.aws/sidecarInjectorWebhook": "enabled",
					"mesh":                                   meshName,
				}, namespace.Labels)

				err = f.K8sClient.Patch(ctx, namespace, client.MergeFrom(oldNS))
				Expect(err).NotTo(HaveOccurred())
			})

			vnName := fmt.Sprintf("vn-%s", utils.RandomDNS1123Label(8))
			listeners := []appmesh.Listener{vnBuilder.BuildListener("http", 8080)}
			backends := []types.NamespacedName{}
			vn := vnBuilder.BuildVirtualNode(vnName, backends, listeners, &appmesh.BackendDefaults{})

			By("Creating a virtual node resource in k8s", func() {
				err := vnTest.Create(ctx, f, vn)
				Expect(err).NotTo(HaveOccurred())
			})

			By(fmt.Sprintf("create a deployment for VirtualNode"), func() {
				containersInfo := []manifest.ContainerInfo{
					{
						Name:          "app",
						AppImage:      defaultAppImage,
						ContainerPort: AppContainerPort,
					},
				}
				containers := mb.BuildContainerSpec(containersInfo)
				dp := mb.BuildDeployment(vnName, 2, containers, map[string]string{})
				//dp := mb.BuildDeployment(vnName, 2, defaultAppImage, AppContainerPort, []corev1.EnvVar{}, map[string]string{})
				err := f.K8sClient.Create(ctx, dp)
				Expect(err).NotTo(HaveOccurred())
				vnTest.Deployments[vnName] = dp
			})

			//Let Instances sync with CloudMap and Pod Readiness Gate go through
			time.Sleep(30 * time.Second)
			By("validating the virtual node in AWS AppMesh & CloudMap", func() {
				err := vnTest.CheckInAWS(ctx, f, mesh, vn)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		It("should delete a virtual node in AWS", func() {

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

			By("creating a mesh resource in k8s", func() {
				err := meshTest.Create(ctx, f, mesh)
				Expect(err).NotTo(HaveOccurred())
			})

			By("validating the resources in AWS", func() {
				err := meshTest.CheckInAWS(ctx, f, mesh)
				Expect(err).NotTo(HaveOccurred())

			})

			By("Create a namespace and add labels", func() {
				namespace, err := f.NSManager.AllocateNamespace(ctx, "appmeshtest")
				Expect(err).NotTo(HaveOccurred())
				vnBuilder.Namespace = namespace.Name
				vnTest.Namespace = namespace

				oldNS := namespace.DeepCopy()
				namespace.Labels = algorithm.MergeStringMap(map[string]string{
					"appmesh.k8s.aws/sidecarInjectorWebhook": "enabled",
					"mesh":                                   meshName,
				}, namespace.Labels)

				err = f.K8sClient.Patch(ctx, namespace, client.MergeFrom(oldNS))
				Expect(err).NotTo(HaveOccurred())
			})

			vnName := fmt.Sprintf("vn-%s", utils.RandomDNS1123Label(8))
			listeners := []appmesh.Listener{vnBuilder.BuildListener("http", 8080)}
			backends := []types.NamespacedName{}
			vn := vnBuilder.BuildVirtualNode(vnName, backends, listeners, &appmesh.BackendDefaults{})

			By("Creating a virtual node resource in k8s", func() {
				err := vnTest.Create(ctx, f, vn)
				Expect(err).NotTo(HaveOccurred())
			})

			By("Validating the virtual node in AWS", func() {
				err := vnTest.CheckInAWS(ctx, f, mesh, vn)
				Expect(err).NotTo(HaveOccurred())

			})

			By("Check mesh finalizers", func() {
				var wg sync.WaitGroup
				wg.Add(1)

				go func() {
					meshTest.Cleanup(ctx, f)
					wg.Done()
				}()

				By("Wait for deletion timestamp to appear on mesh before we check virtual node", func() {
					res := meshTest.WaitForDeletionTimestamp(ctx, f, mesh)
					Expect(res).To(Equal(true))
				})

				By("Check virtual node in AWS after mesh deletion - it should exist", func() {
					err := vnTest.CheckInAWS(ctx, f, mesh, vn)
					Expect(err).NotTo(HaveOccurred())
				})

				By("Check the mesh as the virtual is not deleted - the mesh should exist", func() {
					ms, err := meshTest.Get(ctx, f, mesh)
					Expect(err).NotTo(HaveOccurred())

					hasFin := appmeshk8s.HasFinalizer(ms, appmeshk8s.FinalizerAWSAppMeshResources)
					Expect(hasFin).To(Equal(true))
				})

				By("Delete virtual node in k8s", func() {
					vnTest.Cleanup(ctx, f)
				})

				By("Check virtual node in AWS after delete in k8s - it should not exist", func() {
					err := vnTest.CheckInAWS(ctx, f, mesh, vn)
					Expect(err).To(HaveOccurred())
				})

				wg.Wait()

				By("Check the mesh as the virtual node has been deleted -mesh should not exist", func() {
					_, err := meshTest.Get(ctx, f, mesh)
					Expect(apierrs.IsNotFound(err)).To(Equal(true))
				})

			})

		})

		It("Virtual node outlier detection scenarios", func() {

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

			By("creating a mesh resource in k8s", func() {
				err := meshTest.Create(ctx, f, mesh)
				Expect(err).NotTo(HaveOccurred())
			})

			By("validating the resources in AWS", func() {
				err := meshTest.CheckInAWS(ctx, f, mesh)
				Expect(err).NotTo(HaveOccurred())

			})

			By("Create a namespace and add labels", func() {
				namespace, err := f.NSManager.AllocateNamespace(ctx, "appmeshtest")
				Expect(err).NotTo(HaveOccurred())
				vnBuilder.Namespace = namespace.Name
				vnTest.Namespace = namespace

				oldNS := namespace.DeepCopy()
				namespace.Labels = algorithm.MergeStringMap(map[string]string{
					"appmesh.k8s.aws/sidecarInjectorWebhook": "enabled",
					"mesh":                                   meshName,
				}, namespace.Labels)

				err = f.K8sClient.Patch(ctx, namespace, client.MergeFrom(oldNS))
				Expect(err).NotTo(HaveOccurred())
			})

			vnName := fmt.Sprintf("vn-%s", utils.RandomDNS1123Label(8))
			maxServerErrors := int64(100)
			maxEjectionPercent := int64(50)
			interval := appmesh.Duration{Unit: appmesh.DurationUnitS, Value: 15}
			baseEjectionDuration := appmesh.Duration{Unit: appmesh.DurationUnitS, Value: 10}
			vnOutlierDetectionListener := vnBuilder.BuildListenerWithOutlierDetection("http", 8080, maxServerErrors,
				interval, baseEjectionDuration, maxEjectionPercent)
			listeners := []appmesh.Listener{vnOutlierDetectionListener}
			backends := []types.NamespacedName{}
			vn := vnBuilder.BuildVirtualNode(vnName, backends, listeners, &appmesh.BackendDefaults{})

			By("Creating a virtual node outlier detection normal", func() {
				err := vnTest.Create(ctx, f, vn)
				Expect(err).NotTo(HaveOccurred())
			})

			By("validate the virtual node in AWS", func() {
				err := vnTest.CheckInAWS(ctx, f, mesh, vn)
				Expect(err).NotTo(HaveOccurred())

			})

			By("Validate update of outlier detection thresholds", func() {
				maxServerErrors = int64(90)
				maxEjectionPercent = int64(90)
				interval = appmesh.Duration{Unit: appmesh.DurationUnitS, Value: 30}
				baseEjectionDuration = appmesh.Duration{Unit: appmesh.DurationUnitS, Value: 20}
				vnOutlierDetectionListener = vnBuilder.BuildListenerWithOutlierDetection("http", 8080, maxServerErrors,
					interval, baseEjectionDuration, maxEjectionPercent)
				listeners = []appmesh.Listener{vnOutlierDetectionListener}

				oldVN := vnTest.VirtualNodes[vn.Name].DeepCopy()

				vnTest.VirtualNodes[vn.Name].Spec.Listeners = listeners

				_, err := vnTest.Update(ctx, f, vnTest.VirtualNodes[vn.Name], oldVN)
				Expect(err).NotTo(HaveOccurred())

				err = vnTest.CheckInAWS(ctx, f, mesh, vnTest.VirtualNodes[vn.Name])
				Expect(err).NotTo(HaveOccurred())

			})

			By("Validate update disable outlier detection", func() {
				listeners := []appmesh.Listener{vnBuilder.BuildListener("http", 8080)}

				oldVN := vnTest.VirtualNodes[vn.Name].DeepCopy()

				vnTest.VirtualNodes[vn.Name].Spec.Listeners = listeners

				_, err := vnTest.Update(ctx, f, vnTest.VirtualNodes[vn.Name], oldVN)
				Expect(err).NotTo(HaveOccurred())

				err = vnTest.CheckInAWS(ctx, f, mesh, vnTest.VirtualNodes[vn.Name])
				Expect(err).NotTo(HaveOccurred())

			})

			By("Validate update enable outlier detection", func() {
				maxServerErrors = int64(90)
				maxEjectionPercent = int64(90)
				interval = appmesh.Duration{Unit: appmesh.DurationUnitS, Value: 30}
				baseEjectionDuration = appmesh.Duration{Unit: appmesh.DurationUnitS, Value: 20}
				vnOutlierDetectionListener = vnBuilder.BuildListenerWithOutlierDetection("http", 8080, maxServerErrors,
					interval, baseEjectionDuration, maxEjectionPercent)
				listeners = []appmesh.Listener{vnOutlierDetectionListener}

				oldVN := vnTest.VirtualNodes[vn.Name].DeepCopy()

				vnTest.VirtualNodes[vn.Name].Spec.Listeners = listeners

				_, err := vnTest.Update(ctx, f, vnTest.VirtualNodes[vn.Name], oldVN)
				Expect(err).NotTo(HaveOccurred())

				err = vnTest.CheckInAWS(ctx, f, mesh, vnTest.VirtualNodes[vn.Name])
				Expect(err).NotTo(HaveOccurred())

			})

			vnName = fmt.Sprintf("vn-%s", utils.RandomDNS1123Label(8))
			vnOutlierDetectionListener = vnBuilder.BuildListenerWithOutlierDetection("http", 8080, -5,
				interval, baseEjectionDuration, maxEjectionPercent)
			listeners = []appmesh.Listener{vnOutlierDetectionListener}
			vn = vnBuilder.BuildVirtualNode(vnName, backends, listeners, &appmesh.BackendDefaults{})

			By("Virtual node outlier detection with maxServerErrors -5", func() {
				err := vnTest.Create(ctx, f, vn)
				Expect(err).To(HaveOccurred())
			})

			vnName = fmt.Sprintf("vn-%s", utils.RandomDNS1123Label(8))
			vnOutlierDetectionListener = vnBuilder.BuildListenerWithOutlierDetection("http", 8080, maxServerErrors,
				interval, baseEjectionDuration, -1)
			listeners = []appmesh.Listener{vnOutlierDetectionListener}
			vn = vnBuilder.BuildVirtualNode(vnName, backends, listeners, &appmesh.BackendDefaults{})

			By("Virtual node outlier detection with maxEjectionPercent -1", func() {
				err := vnTest.Create(ctx, f, vn)
				Expect(err).To(HaveOccurred())
			})

			vnName = fmt.Sprintf("vn-%s", utils.RandomDNS1123Label(8))
			vnOutlierDetectionListener = vnBuilder.BuildListenerWithOutlierDetection("http", 8080, maxServerErrors,
				interval, baseEjectionDuration, 105)
			listeners = []appmesh.Listener{vnOutlierDetectionListener}
			vn = vnBuilder.BuildVirtualNode(vnName, backends, listeners, &appmesh.BackendDefaults{})

			By("Virtual node outlier detection with maxEjectionPercent 105", func() {
				err := vnTest.Create(ctx, f, vn)
				Expect(err).To(HaveOccurred())
			})

		})
		It("Virtual node connection pool scenarios", func() {

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

			By("creating a mesh resource in k8s", func() {
				err := meshTest.Create(ctx, f, mesh)
				Expect(err).NotTo(HaveOccurred())
			})

			By("validating the resources in AWS", func() {
				err := meshTest.CheckInAWS(ctx, f, mesh)
				Expect(err).NotTo(HaveOccurred())

			})

			By("Create a namespace and add labels", func() {
				namespace, err := f.NSManager.AllocateNamespace(ctx, "appmeshtest")
				Expect(err).NotTo(HaveOccurred())
				vnBuilder.Namespace = namespace.Name
				vnTest.Namespace = namespace

				oldNS := namespace.DeepCopy()
				namespace.Labels = algorithm.MergeStringMap(map[string]string{
					"appmesh.k8s.aws/sidecarInjectorWebhook": "enabled",
					"mesh":                                   meshName,
				}, namespace.Labels)

				err = f.K8sClient.Patch(ctx, namespace, client.MergeFrom(oldNS))
				Expect(err).NotTo(HaveOccurred())
			})

			vnName := fmt.Sprintf("vn-%s", utils.RandomDNS1123Label(8))
			httpConnectionPool := &appmesh.HTTPConnectionPool{
				MaxConnections:     60,
				MaxPendingRequests: aws.Int64(100),
			}
			vnConnectionPoolListener := vnBuilder.BuildListenerWithConnectionPools("http", 8080, nil, httpConnectionPool, nil, nil)
			listeners := []appmesh.Listener{vnConnectionPoolListener}
			backends := []types.NamespacedName{}
			vn := vnBuilder.BuildVirtualNode(vnName, backends, listeners, &appmesh.BackendDefaults{})

			By("Creating a virtual node with HTTP connection pool", func() {
				err := vnTest.Create(ctx, f, vn)
				Expect(err).NotTo(HaveOccurred())
			})

			By("validate the virtual node in AWS", func() {
				err := vnTest.CheckInAWS(ctx, f, mesh, vn)
				Expect(err).NotTo(HaveOccurred())

			})

			By("Validate update of HTTP connection pool thresholds", func() {
				httpConnectionPool = &appmesh.HTTPConnectionPool{
					MaxConnections:     200,
					MaxPendingRequests: aws.Int64(300),
				}
				vnConnectionPoolListener = vnBuilder.BuildListenerWithConnectionPools("http", 8080, nil, httpConnectionPool, nil, nil)
				listeners = []appmesh.Listener{vnConnectionPoolListener}

				oldVN := vnTest.VirtualNodes[vn.Name].DeepCopy()

				vnTest.VirtualNodes[vn.Name].Spec.Listeners = listeners

				_, err := vnTest.Update(ctx, f, vnTest.VirtualNodes[vn.Name], oldVN)
				Expect(err).NotTo(HaveOccurred())

				err = vnTest.CheckInAWS(ctx, f, mesh, vnTest.VirtualNodes[vn.Name])
				Expect(err).NotTo(HaveOccurred())

			})

			By("Validate update disable connection pool", func() {
				listeners := []appmesh.Listener{vnBuilder.BuildListener("http", 8080)}

				oldVN := vnTest.VirtualNodes[vn.Name].DeepCopy()

				vnTest.VirtualNodes[vn.Name].Spec.Listeners = listeners

				_, err := vnTest.Update(ctx, f, vnTest.VirtualNodes[vn.Name], oldVN)
				Expect(err).NotTo(HaveOccurred())

				err = vnTest.CheckInAWS(ctx, f, mesh, vnTest.VirtualNodes[vn.Name])
				Expect(err).NotTo(HaveOccurred())

			})

			By("Validate update enable connection pool", func() {
				httpConnectionPool = &appmesh.HTTPConnectionPool{
					MaxConnections:     200,
					MaxPendingRequests: aws.Int64(300),
				}
				vnConnectionPoolListener = vnBuilder.BuildListenerWithConnectionPools("http", 8080, nil, httpConnectionPool, nil, nil)
				listeners = []appmesh.Listener{vnConnectionPoolListener}

				oldVN := vnTest.VirtualNodes[vn.Name].DeepCopy()

				vnTest.VirtualNodes[vn.Name].Spec.Listeners = listeners

				_, err := vnTest.Update(ctx, f, vnTest.VirtualNodes[vn.Name], oldVN)
				Expect(err).NotTo(HaveOccurred())

				err = vnTest.CheckInAWS(ctx, f, mesh, vnTest.VirtualNodes[vn.Name])
				Expect(err).NotTo(HaveOccurred())

			})

			httpConnectionPool = &appmesh.HTTPConnectionPool{
				MaxConnections:     60,
				MaxPendingRequests: aws.Int64(100),
			}
			tcpConnectionPool := &appmesh.TCPConnectionPool{
				MaxConnections: 70,
			}
			http2ConnectionPool := &appmesh.HTTP2ConnectionPool{
				MaxRequests: 50,
			}
			grpcConnectionPool := &appmesh.GRPCConnectionPool{
				MaxRequests: 30,
			}

			vnName = fmt.Sprintf("vn-%s", utils.RandomDNS1123Label(8))
			vnConnectionPoolListener = vnBuilder.BuildListenerWithConnectionPools("grpc", 8080, nil, nil, nil, grpcConnectionPool)
			listeners = []appmesh.Listener{vnConnectionPoolListener}
			vn = vnBuilder.BuildVirtualNode(vnName, backends, listeners, &appmesh.BackendDefaults{})

			By("Creating a virtual node with GRPC connection pool", func() {
				err := vnTest.Create(ctx, f, vn)
				Expect(err).NotTo(HaveOccurred())
			})

			By("validate the virtual node in AWS", func() {
				err := vnTest.CheckInAWS(ctx, f, mesh, vn)
				Expect(err).NotTo(HaveOccurred())

			})

			vnName = fmt.Sprintf("vn-%s", utils.RandomDNS1123Label(8))
			vnConnectionPoolListener = vnBuilder.BuildListenerWithConnectionPools("http2", 8080, nil, nil, http2ConnectionPool, nil)
			listeners = []appmesh.Listener{vnConnectionPoolListener}
			vn = vnBuilder.BuildVirtualNode(vnName, backends, listeners, &appmesh.BackendDefaults{})

			By("Creating a virtual node with HTTP2 connection pool", func() {
				err := vnTest.Create(ctx, f, vn)
				Expect(err).NotTo(HaveOccurred())
			})

			By("validate the virtual node in AWS", func() {
				err := vnTest.CheckInAWS(ctx, f, mesh, vn)
				Expect(err).NotTo(HaveOccurred())

			})

			vnName = fmt.Sprintf("vn-%s", utils.RandomDNS1123Label(8))
			vnConnectionPoolListener = vnBuilder.BuildListenerWithConnectionPools("http", 8080, tcpConnectionPool, httpConnectionPool, http2ConnectionPool, grpcConnectionPool)
			listeners = []appmesh.Listener{vnConnectionPoolListener}
			vn = vnBuilder.BuildVirtualNode(vnName, backends, listeners, &appmesh.BackendDefaults{})

			By("Creating a virtual node with HTTP, TCP, HTTP2 and GRPC connection pool", func() {
				err := vnTest.Create(ctx, f, vn)
				Expect(err).To(HaveOccurred())
			})

			vnName = fmt.Sprintf("vn-%s", utils.RandomDNS1123Label(8))
			httpConnectionPool = &appmesh.HTTPConnectionPool{
				MaxConnections:     -30,
				MaxPendingRequests: aws.Int64(100),
			}
			vnConnectionPoolListener = vnBuilder.BuildListenerWithConnectionPools("http", 8080, nil, httpConnectionPool, nil, nil)
			listeners = []appmesh.Listener{vnConnectionPoolListener}
			backends = []types.NamespacedName{}
			vn = vnBuilder.BuildVirtualNode(vnName, backends, listeners, &appmesh.BackendDefaults{})

			By("Virtual node with HTTP connection pool MaxConnections -30", func() {
				err := vnTest.Create(ctx, f, vn)
				Expect(err).To(HaveOccurred())
			})

			vnName = fmt.Sprintf("vn-%s", utils.RandomDNS1123Label(8))
			grpcConnectionPool = &appmesh.GRPCConnectionPool{
				MaxRequests: -40,
			}
			vnConnectionPoolListener = vnBuilder.BuildListenerWithConnectionPools("http", 8080, nil, nil, nil, grpcConnectionPool)
			listeners = []appmesh.Listener{vnConnectionPoolListener}
			backends = []types.NamespacedName{}
			vn = vnBuilder.BuildVirtualNode(vnName, backends, listeners, &appmesh.BackendDefaults{})

			By("Virtual node with GRPC connection pool MaxRequests -30", func() {
				err := vnTest.Create(ctx, f, vn)
				Expect(err).To(HaveOccurred())
			})

		})
	})
})
