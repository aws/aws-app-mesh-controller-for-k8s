package virtualnode_test

import (
	"context"
	"fmt"
	appmeshk8s "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/k8s"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sync"
	"time"

	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/algorithm"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/k8s"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/manifest"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/utils"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/integration/mesh"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/integration/virtualnode"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultAppImage  = "public.ecr.aws/e6v3k1j4/colorteller:v1"
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

			vnName = fmt.Sprintf("vn-%s", utils.RandomDNS1123Label(8))
			vn = vnBuilder.BuildVirtualNode(vnName, backends, listeners, &appmesh.BackendDefaults{})
			vn.Spec.AWSName = aws.String(fmt.Sprintf("vn-%s", utils.RandomDNS1123Label(256)))

			By("Creating a virtual node resource in k8s with a name exceeding the character limit", func() {
				// Not using vnTest.Create as it hangs
				err := f.K8sClient.Create(ctx, vn)
				observedVn := &appmesh.VirtualNode{}
				for i := 0; i < 5; i++ {
					if err := f.K8sClient.Get(ctx, k8s.NamespacedName(vn), observedVn); err != nil {
						if i >= 5 {
							Expect(err).NotTo(HaveOccurred())
						}
					}
					time.Sleep(100 * time.Millisecond)
				}
				vnTest.VirtualNodes[vn.Name] = vn
				Expect(err).NotTo(HaveOccurred())
			})

			By("Check virtual node in AWS - it should not exist", func() {
				err := vnTest.CheckInAWS(ctx, f, mesh, vn)
				Expect(err).To(HaveOccurred())
			})

			By("checking events for the BadRequestException", func() {
				clientset, err := kubernetes.NewForConfig(f.RestCfg)
				Expect(err).NotTo(HaveOccurred())
				events, err := clientset.CoreV1().Events(vnTest.Namespace.Name).List(ctx, metav1.ListOptions{
					FieldSelector: fmt.Sprintf("involvedObject.name=%s", vn.Name),
					TypeMeta: metav1.TypeMeta{
						Kind: "Pod",
					},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(events.Items).NotTo(BeEmpty())
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

				err := vnTest.Update(ctx, f, vnTest.VirtualNodes[vn.Name], oldVN)
				Expect(err).NotTo(HaveOccurred())

				err = vnTest.CheckInAWS(ctx, f, mesh, vnTest.VirtualNodes[vn.Name])
				Expect(err).NotTo(HaveOccurred())

			})

			By("Validate update disable outlier detection", func() {
				listeners := []appmesh.Listener{vnBuilder.BuildListener("http", 8080)}

				oldVN := vnTest.VirtualNodes[vn.Name].DeepCopy()

				vnTest.VirtualNodes[vn.Name].Spec.Listeners = listeners

				err := vnTest.Update(ctx, f, vnTest.VirtualNodes[vn.Name], oldVN)
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

				err := vnTest.Update(ctx, f, vnTest.VirtualNodes[vn.Name], oldVN)
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

				err := vnTest.Update(ctx, f, vnTest.VirtualNodes[vn.Name], oldVN)
				Expect(err).NotTo(HaveOccurred())

				err = vnTest.CheckInAWS(ctx, f, mesh, vnTest.VirtualNodes[vn.Name])
				Expect(err).NotTo(HaveOccurred())

			})

			By("Validate update disable connection pool", func() {
				listeners := []appmesh.Listener{vnBuilder.BuildListener("http", 8080)}

				oldVN := vnTest.VirtualNodes[vn.Name].DeepCopy()

				vnTest.VirtualNodes[vn.Name].Spec.Listeners = listeners

				err := vnTest.Update(ctx, f, vnTest.VirtualNodes[vn.Name], oldVN)
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

				err := vnTest.Update(ctx, f, vnTest.VirtualNodes[vn.Name], oldVN)
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
		It("Virtual node mTLS scenarios", func() {
			mTLSValidationContext := "spiffe://integration-test.aws"
			mTLSValidationContext_upd := "spiffe://integration-test-updated.aws"
			tlsEnforce := true
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
			nodeSVID := mTLSValidationContext + "/" + vnName
			nodeBackendDefaultsSDS := &appmesh.BackendDefaults{
				ClientPolicy: &appmesh.ClientPolicy{
					TLS: &appmesh.ClientPolicyTLS{
						Enforce: &tlsEnforce,
						Ports:   nil,
						Validation: appmesh.TLSValidationContext{
							Trust: appmesh.TLSValidationContextTrust{
								SDS: &appmesh.TLSValidationContextSDSTrust{
									SecretName: &mTLSValidationContext,
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

			nodeListenerTLSSDS := &appmesh.ListenerTLS{
				Certificate: appmesh.ListenerTLSCertificate{
					SDS: &appmesh.ListenerTLSSDSCertificate{
						SecretName: &nodeSVID,
					},
				},
				Validation: &appmesh.ListenerTLSValidationContext{
					Trust: appmesh.ListenerTLSValidationContextTrust{
						SDS: &appmesh.TLSValidationContextSDSTrust{
							SecretName: &mTLSValidationContext,
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
			listeners := []appmesh.Listener{vnBuilder.BuildListenerWithTLS("http", AppContainerPort, nodeListenerTLSSDS)}
			vn := vnBuilder.BuildVirtualNode(vnName, nil, listeners, nodeBackendDefaultsSDS)
			By("Creating a virtual node with SDS based mTLS enabled", func() {
				err := vnTest.Create(ctx, f, vn)
				Expect(err).NotTo(HaveOccurred())
			})

			By("validate the virtual node in AWS", func() {
				err := vnTest.CheckInAWS(ctx, f, mesh, vn)
				Expect(err).NotTo(HaveOccurred())

			})

			By("Validate update of mTLS - SDS validation context", func() {
				nodeListenerTLS_SDS := &appmesh.ListenerTLS{
					Certificate: appmesh.ListenerTLSCertificate{
						SDS: &appmesh.ListenerTLSSDSCertificate{
							SecretName: &nodeSVID,
						},
					},
					Validation: &appmesh.ListenerTLSValidationContext{
						Trust: appmesh.ListenerTLSValidationContextTrust{
							SDS: &appmesh.TLSValidationContextSDSTrust{
								SecretName: &mTLSValidationContext_upd,
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
				listeners := []appmesh.Listener{vnBuilder.BuildListenerWithTLS("http", AppContainerPort, nodeListenerTLS_SDS)}
				oldVN := vnTest.VirtualNodes[vn.Name].DeepCopy()

				vnTest.VirtualNodes[vn.Name].Spec.Listeners = listeners

				err := vnTest.Update(ctx, f, vnTest.VirtualNodes[vn.Name], oldVN)
				Expect(err).NotTo(HaveOccurred())

				err = vnTest.CheckInAWS(ctx, f, mesh, vnTest.VirtualNodes[vn.Name])
				Expect(err).NotTo(HaveOccurred())
			})

			fileCACertPath := "/certs/caCert.pem"
			fileCACertPathNew := "/certs/caCertNew.pem"
			appCertPath := "/certs/appCert.pem"
			privateKeyPath := "/certs/appKey.pem"
			serviceDNS := "app.namespace.svc.cluster.local"

			nodeBackendDefaultsFile := &appmesh.BackendDefaults{
				ClientPolicy: &appmesh.ClientPolicy{
					TLS: &appmesh.ClientPolicyTLS{
						Enforce: &tlsEnforce,
						Ports:   nil,
						Validation: appmesh.TLSValidationContext{
							Trust: appmesh.TLSValidationContextTrust{
								File: &appmesh.TLSValidationContextFileTrust{
									CertificateChain: fileCACertPath,
								},
							},
							SubjectAlternativeNames: &appmesh.SubjectAlternativeNames{
								Match: &appmesh.SubjectAlternativeNameMatchers{
									Exact: []*string{
										&serviceDNS,
									},
								},
							},
						},
						Certificate: &appmesh.ClientTLSCertificate{
							File: &appmesh.ListenerTLSFileCertificate{
								CertificateChain: appCertPath,
								PrivateKey:       privateKeyPath,
							},
						},
					},
				},
			}

			nodeListenerTLSFile := &appmesh.ListenerTLS{
				Certificate: appmesh.ListenerTLSCertificate{
					File: &appmesh.ListenerTLSFileCertificate{
						CertificateChain: appCertPath,
						PrivateKey:       privateKeyPath,
					},
				},
				Validation: &appmesh.ListenerTLSValidationContext{
					Trust: appmesh.ListenerTLSValidationContextTrust{
						File: &appmesh.TLSValidationContextFileTrust{
							CertificateChain: fileCACertPath,
						},
					},
					SubjectAlternativeNames: &appmesh.SubjectAlternativeNames{
						Match: &appmesh.SubjectAlternativeNameMatchers{
							Exact: []*string{
								&serviceDNS,
							},
						},
					},
				},
				Mode: "STRICT",
			}
			listeners = []appmesh.Listener{vnBuilder.BuildListenerWithTLS("http", AppContainerPort, nodeListenerTLSFile)}
			vnName = fmt.Sprintf("vn-%s", utils.RandomDNS1123Label(8))
			vn = vnBuilder.BuildVirtualNode(vnName, nil, listeners, nodeBackendDefaultsFile)
			By("Creating a virtual node with File based mTLS enabled", func() {
				err := vnTest.Create(ctx, f, vn)
				Expect(err).NotTo(HaveOccurred())
			})

			By("validate the virtual node in AWS", func() {
				err := vnTest.CheckInAWS(ctx, f, mesh, vn)
				Expect(err).NotTo(HaveOccurred())

			})

			By("Validate update of mTLS - File validation context", func() {
				nodeListenerTLSFile := &appmesh.ListenerTLS{
					Certificate: appmesh.ListenerTLSCertificate{
						File: &appmesh.ListenerTLSFileCertificate{
							CertificateChain: appCertPath,
							PrivateKey:       privateKeyPath,
						},
					},
					Validation: &appmesh.ListenerTLSValidationContext{
						Trust: appmesh.ListenerTLSValidationContextTrust{
							File: &appmesh.TLSValidationContextFileTrust{
								CertificateChain: fileCACertPathNew,
							},
						},
						SubjectAlternativeNames: &appmesh.SubjectAlternativeNames{
							Match: &appmesh.SubjectAlternativeNameMatchers{
								Exact: []*string{
									&serviceDNS,
								},
							},
						},
					},
					Mode: "STRICT",
				}
				listeners := []appmesh.Listener{vnBuilder.BuildListenerWithTLS("http", AppContainerPort, nodeListenerTLSFile)}
				oldVN := vnTest.VirtualNodes[vn.Name].DeepCopy()

				vnTest.VirtualNodes[vn.Name].Spec.Listeners = listeners

				err := vnTest.Update(ctx, f, vnTest.VirtualNodes[vn.Name], oldVN)
				Expect(err).NotTo(HaveOccurred())

				err = vnTest.CheckInAWS(ctx, f, mesh, vnTest.VirtualNodes[vn.Name])
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Context("Virtual Node CloudMap scenarios", func() {
		var (
			meshSpec  *appmesh.Mesh
			meshTest  mesh.MeshTest
			vnTest    virtualnode.VirtualNodeTest
			vnBuilder *manifest.VNBuilder
			mb        *manifest.ManifestBuilder
		)

		BeforeEach(func() {
			meshTest = mesh.MeshTest{
				Meshes: make(map[string]*appmesh.Mesh),
			}

			vnTest = virtualnode.VirtualNodeTest{
				VirtualNodes: make(map[string]*appmesh.VirtualNode),
				Pods:         make(map[string]*corev1.Pod),
				Deployments:  make(map[string]*appsv1.Deployment),
			}

			meshName := fmt.Sprintf("%s-%s", f.Options.ClusterName, utils.RandomDNS1123Label(6))
			meshSpec = &appmesh.Mesh{
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

			By("creating a mesh resource in k8s: "+meshName, func() {
				err := meshTest.Create(ctx, f, meshSpec)
				Expect(err).NotTo(HaveOccurred())
			})

			By("validating the resources in AWS", func() {
				err := meshTest.CheckInAWS(ctx, f, meshSpec)
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

			vnBuilder = &manifest.VNBuilder{
				ServiceDiscoveryType: manifest.CloudMapServiceDiscovery,
				CloudMapNamespace:    cmNamespace,
			}

			mb = &manifest.ManifestBuilder{
				ServiceDiscoveryType: manifest.CloudMapServiceDiscovery,
				DisableIPv6:          true, // for github action compatibility
			}

			By("Create a namespace and add labels", func() {
				namespace, err := f.NSManager.AllocateNamespace(ctx, "appmeshtest")
				Expect(err).NotTo(HaveOccurred())
				vnBuilder.Namespace = namespace.Name
				vnTest.Namespace = namespace
				mb.Namespace = namespace.Name

				oldNS := namespace.DeepCopy()
				namespace.Labels = algorithm.MergeStringMap(map[string]string{
					"appmesh.k8s.aws/sidecarInjectorWebhook": "enabled",
					"mesh":                                   meshName,
				}, namespace.Labels)

				err = f.K8sClient.Patch(ctx, namespace, client.MergeFrom(oldNS))
				Expect(err).NotTo(HaveOccurred())
			})
		})

		AfterEach(func() {
			vnTest.Cleanup(ctx, f)
			meshTest.Cleanup(ctx, f)
		})

		Context("Virtual Node CloudMap Scenarios", func() {
			It("should create a virtual node with CloudMap ServiceDiscovery enabled", func() {
				containersInfo := []manifest.ContainerInfo{
					{
						Name:          "app",
						AppImage:      defaultAppImage,
						ContainerPort: AppContainerPort,
					},
				}
				containers := mb.BuildContainerSpec(containersInfo)
				podGroup := mb.BuildPodGroup(containers, "pods", 2)

				vnName := fmt.Sprintf("vn-%s", utils.RandomDNS1123Label(8))
				listeners := []appmesh.Listener{vnBuilder.BuildListener("http", 8080)}
				vn := vnBuilder.BuildVirtualNodeWithPodSelector(vnName, nil, listeners, &appmesh.BackendDefaults{}, podGroup.MatchLabels)

				By("Creating a virtual node resource in k8s", func() {
					err := vnTest.Create(ctx, f, vn)
					Expect(err).NotTo(HaveOccurred())
				})

				vnTest.CreatePodGroup(ctx, f, podGroup)

				//Let Instances sync with CloudMap and Pod Readiness Gate go through
				time.Sleep(60 * time.Second)
				By("validating the virtual node in AWS AppMesh & CloudMap", func() {
					err := vnTest.CheckInAWSWithExpectedPods(ctx, f, meshSpec, vn, podGroup.Pods)
					Expect(err).NotTo(HaveOccurred())
				})

				By("delete one of the pods for the virtual node", func() {
					err := f.K8sClient.Delete(ctx, podGroup.Pods[0])
					Expect(err).NotTo(HaveOccurred())
				})

				//Let instances sync with CloudMap
				time.Sleep(60 * time.Second)
				By(fmt.Sprint("validate the virtual node instance registrations were removed"), func() {
					err := vnTest.CheckInAWSWithExpectedPods(ctx, f, meshSpec, vn, podGroup.Pods[1:2])
					Expect(err).NotTo(HaveOccurred())
				})
			})

			It("should update instance registrations when pod selector changes on virtual node", func() {
				containersInfo := []manifest.ContainerInfo{
					{
						Name:          "app",
						AppImage:      defaultAppImage,
						ContainerPort: AppContainerPort,
					},
				}
				containers := mb.BuildContainerSpec(containersInfo)
				podGroup := mb.BuildPodGroup(containers, "pods", 2)

				vnName := fmt.Sprintf("vn-%s", utils.RandomDNS1123Label(8))
				listeners := []appmesh.Listener{vnBuilder.BuildListener("http", 8080)}
				vn := vnBuilder.BuildVirtualNodeWithPodSelector(vnName, nil, listeners, &appmesh.BackendDefaults{}, podGroup.MatchLabels)

				By("Creating a virtual node resource in k8s", func() {
					err := vnTest.Create(ctx, f, vn)
					Expect(err).NotTo(HaveOccurred())
				})

				vnTest.CreatePodGroup(ctx, f, podGroup)

				//Let Instances sync with CloudMap and Pod Readiness Gate go through
				time.Sleep(60 * time.Second)
				By("validating the virtual node in AWS AppMesh & CloudMap", func() {
					err := vnTest.CheckInAWSWithExpectedPods(ctx, f, meshSpec, vn, podGroup.Pods)
					Expect(err).NotTo(HaveOccurred())
				})

				By("updating virtual node pod selector", func() {
					oldVn := vn.DeepCopy()
					vn.Spec.PodSelector.MatchLabels = map[string]string{
						"wont": "match",
					}
					err := vnTest.Update(ctx, f, vn, oldVn)
					Expect(err).NotTo(HaveOccurred())
				})

				//Let instances sync with CloudMap
				time.Sleep(60 * time.Second)
				By(fmt.Sprint("validate the virtual node instance registrations were removed"), func() {
					err := vnTest.CheckInAWSWithExpectedPods(ctx, f, meshSpec, vn, nil)
					Expect(err).NotTo(HaveOccurred())
				})
			})

			It("should remove all instance registrations when pod selector is removed from virtual node", func() {
				containersInfo := []manifest.ContainerInfo{
					{
						Name:          "app",
						AppImage:      defaultAppImage,
						ContainerPort: AppContainerPort,
					},
				}
				containers := mb.BuildContainerSpec(containersInfo)
				podGroup := mb.BuildPodGroup(containers, "pods", 2)

				vnName := fmt.Sprintf("vn-%s", utils.RandomDNS1123Label(8))

				listeners := []appmesh.Listener{vnBuilder.BuildListener("http", 8080)}
				vn := vnBuilder.BuildVirtualNodeWithPodSelector(vnName, nil, listeners, &appmesh.BackendDefaults{}, podGroup.MatchLabels)

				By("Creating a virtual node resource in k8s", func() {
					err := vnTest.Create(ctx, f, vn)
					Expect(err).NotTo(HaveOccurred())
				})

				vnTest.CreatePodGroup(ctx, f, podGroup)

				//Let Instances sync with CloudMap and Pod Readiness Gate go through
				time.Sleep(60 * time.Second)
				By("validating the virtual node in AWS AppMesh & CloudMap", func() {
					err := vnTest.CheckInAWSWithExpectedPods(ctx, f, meshSpec, vn, podGroup.Pods)
					Expect(err).NotTo(HaveOccurred())
				})

				By("updating virtual node pod selector", func() {
					oldVn := vn.DeepCopy()
					vn.Spec.PodSelector = nil
					err := vnTest.Update(ctx, f, vn, oldVn)
					Expect(err).NotTo(HaveOccurred())
				})

				//Let instances sync with CloudMap
				time.Sleep(60 * time.Second)
				By(fmt.Sprint("validate the virtual node instance registrations were removed"), func() {
					err := vnTest.CheckInAWSWithExpectedPods(ctx, f, meshSpec, vn, nil)
					Expect(err).NotTo(HaveOccurred())
				})
			})

			It("virtual node registrations should be updated when associated pod labels change", func() {
				containersInfo := []manifest.ContainerInfo{
					{
						Name:          "app",
						AppImage:      defaultAppImage,
						ContainerPort: AppContainerPort,
					},
				}
				containers := mb.BuildContainerSpec(containersInfo)
				podGroup := mb.BuildPodGroup(containers, "pods", 2)

				vnName := fmt.Sprintf("vn-%s", utils.RandomDNS1123Label(8))

				listeners := []appmesh.Listener{vnBuilder.BuildListener("http", 8080)}
				vn := vnBuilder.BuildVirtualNodeWithPodSelector(vnName, nil, listeners, &appmesh.BackendDefaults{}, podGroup.MatchLabels)

				By("Creating a virtual node resource in k8s", func() {
					err := vnTest.Create(ctx, f, vn)
					Expect(err).NotTo(HaveOccurred())
				})

				vnTest.CreatePodGroup(ctx, f, podGroup)

				//Let Instances sync with CloudMap and Pod Readiness Gate go through
				time.Sleep(60 * time.Second)
				By("validating the virtual node in AWS AppMesh & CloudMap", func() {
					err := vnTest.CheckInAWSWithExpectedPods(ctx, f, meshSpec, vn, podGroup.Pods)
					Expect(err).NotTo(HaveOccurred())
				})

				By("updating pod labels", func() {
					pod := podGroup.Pods[1]
					oldPod := pod.DeepCopy()
					pod.ObjectMeta.Labels["app"] = "nope"
					err := f.K8sClient.Patch(ctx, pod, client.MergeFrom(oldPod))
					Expect(err).NotTo(HaveOccurred())
				})

				//Let instances sync with CloudMap
				time.Sleep(60 * time.Second)
				By(fmt.Sprint("validate the virtual node instance registrations were removed"), func() {
					err := vnTest.CheckInAWSWithExpectedPods(ctx, f, meshSpec, vn, podGroup.Pods[0:1])
					Expect(err).NotTo(HaveOccurred())
				})
			})
		})
	})
})
