package virtualgateway_test

import (
	"context"
	"fmt"
	"sync"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/algorithm"
	appmeshk8s "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/k8s"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/manifest"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/utils"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/integration/mesh"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/integration/virtualgateway"

	"github.com/aws/aws-sdk-go/aws"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("VirtualGateway", func() {

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

	Context("Virtual Gateway scenaries", func() {
		var meshTest mesh.MeshTest
		var vgTest virtualgateway.VirtualGatewayTest

		meshTest = mesh.MeshTest{
			Meshes: make(map[string]*appmesh.Mesh),
		}

		vgTest = virtualgateway.VirtualGatewayTest{
			VirtualGateways: make(map[string]*appmesh.VirtualGateway),
		}

		vgBuilder := &manifest.VGBuilder{}

		AfterEach(func() {
			vgTest.Cleanup(ctx, f)
			meshTest.Cleanup(ctx, f)
		})

		It("Virtual Gateway Create Scenaries", func() {

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
				vgBuilder.Namespace = namespace.Name
				vgTest.Namespace = namespace

				oldNS := namespace.DeepCopy()
				namespace.Labels = algorithm.MergeStringMap(map[string]string{
					"appmesh.k8s.aws/sidecarInjectorWebhook": "enabled",
					"mesh":                                   meshName,
				}, namespace.Labels)

				err = f.K8sClient.Patch(ctx, namespace, client.MergeFrom(oldNS))
				Expect(err).NotTo(HaveOccurred())
			})

			vgName := fmt.Sprintf("vg-%s", utils.RandomDNS1123Label(8))
			listeners := []appmesh.VirtualGatewayListener{vgBuilder.BuildVGListener("http", 8080, "/")}
			nsSelector := map[string]string{"gateway": "ingress-gw"}
			vg := vgBuilder.BuildVirtualGateway(vgName, listeners, nsSelector)

			By("Creating a virtual gateway resource in k8s", func() {
				err := vgTest.Create(ctx, f, vg)
				Expect(err).NotTo(HaveOccurred())
			})

			By("validating the virtual gateway in AWS", func() {
				err := vgTest.CheckInAWS(ctx, f, mesh, vg)
				Expect(err).NotTo(HaveOccurred())

			})

			By("Create a virtual gateway resource with invalid listener protocol -  it should fail", func() {
				vgName = fmt.Sprintf("vg-%s", utils.RandomDNS1123Label(8))
				newListeners := []appmesh.VirtualGatewayListener{vgBuilder.BuildVGListener("https", 443, "/")}
				vg = vgBuilder.BuildVirtualGateway(vgName, newListeners, nsSelector)
				err := vgTest.Create(ctx, f, vg)
				Expect(err).To(HaveOccurred())
			})

			By("Set incorrect labels on namespace", func() {
				oldNS := vgTest.Namespace.DeepCopy()
				vgTest.Namespace.Labels = algorithm.MergeStringMap(map[string]string{
					"appmesh.k8s.aws/sidecarInjectorWebhook": "enabled",
					"mesh":                                   "dontmatch",
				}, vgTest.Namespace.Labels)

				err := f.K8sClient.Patch(ctx, vgTest.Namespace, client.MergeFrom(oldNS))
				Expect(err).NotTo(HaveOccurred())
			})

			vgName = fmt.Sprintf("vg-%s", utils.RandomDNS1123Label(8))
			vg = vgBuilder.BuildVirtualGateway(vgName, listeners, nsSelector)

			By("Creating a virtual gateway resource in k8s when no mesh matches namespace", func() {
				err := vgTest.Create(ctx, f, vg)
				Expect(err).To(HaveOccurred())
			})

		})

		It("Virtual Gateway Update Scenaries", func() {

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
				vgBuilder.Namespace = namespace.Name
				vgTest.Namespace = namespace

				oldNS := namespace.DeepCopy()
				namespace.Labels = algorithm.MergeStringMap(map[string]string{
					"appmesh.k8s.aws/sidecarInjectorWebhook": "enabled",
					"mesh":                                   meshName,
				}, namespace.Labels)

				err = f.K8sClient.Patch(ctx, namespace, client.MergeFrom(oldNS))
				Expect(err).NotTo(HaveOccurred())
			})

			vgName := fmt.Sprintf("vg-%s", utils.RandomDNS1123Label(8))
			listeners := []appmesh.VirtualGatewayListener{vgBuilder.BuildVGListener("http", 8080, "/")}
			nsSelector := map[string]string{"gateway": "ingress-gw"}
			vg := vgBuilder.BuildVirtualGateway(vgName, listeners, nsSelector)

			By("Creating a virtual gateway resource in k8s", func() {
				err := vgTest.Create(ctx, f, vg)
				Expect(err).NotTo(HaveOccurred())
			})

			By("validating the virtual gateway in AWS", func() {
				err := vgTest.CheckInAWS(ctx, f, mesh, vg)
				Expect(err).NotTo(HaveOccurred())

			})

			By("Update logging in virtual gateway and validate", func() {
				oldVG := vgTest.VirtualGateways[vg.Name].DeepCopy()
				newLog := &appmesh.VirtualGatewayLogging{
					AccessLog: &appmesh.VirtualGatewayAccessLog{
						File: &appmesh.VirtualGatewayFileAccessLog{
							Path: "/new/path",
						},
					},
				}

				vgTest.VirtualGateways[vg.Name].Spec.Logging = newLog
				updatedVG, err := vgTest.Update(ctx, f, vgTest.VirtualGateways[vg.Name], oldVG)
				Expect(err).NotTo(HaveOccurred())

				err = vgTest.CheckInAWS(ctx, f, mesh, updatedVG)
				Expect(err).NotTo(HaveOccurred())

			})

			By("Update listeners in virtual gateway and validate", func() {
				oldVG := vgTest.VirtualGateways[vg.Name].DeepCopy()
				listeners = []appmesh.VirtualGatewayListener{vgBuilder.BuildVGListener("http2", 443, "/")}

				vgTest.VirtualGateways[vg.Name].Spec.Listeners = listeners
				updatedVG, err := vgTest.Update(ctx, f, vgTest.VirtualGateways[vg.Name], oldVG)
				Expect(err).NotTo(HaveOccurred())

				err = vgTest.CheckInAWS(ctx, f, mesh, updatedVG)
				Expect(err).NotTo(HaveOccurred())

			})

			By("Update AWSName for virtual gateway and validate it cannot be updated", func() {
				oldVG := vgTest.VirtualGateways[vg.Name].DeepCopy()
				vgTest.VirtualGateways[vg.Name].Spec.AWSName = aws.String("newVirtualGatewayAWSName")

				_, err := vgTest.Update(ctx, f, vgTest.VirtualGateways[vg.Name], oldVG)
				Expect(err).To(HaveOccurred())
			})

		})

		It("Virtual Gateway Delete Scenaries", func() {

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
				vgBuilder.Namespace = namespace.Name
				vgTest.Namespace = namespace

				oldNS := namespace.DeepCopy()
				namespace.Labels = algorithm.MergeStringMap(map[string]string{
					"appmesh.k8s.aws/sidecarInjectorWebhook": "enabled",
					"mesh":                                   meshName,
				}, namespace.Labels)

				err = f.K8sClient.Patch(ctx, namespace, client.MergeFrom(oldNS))
				Expect(err).NotTo(HaveOccurred())
			})

			vgName := fmt.Sprintf("vg-%s", utils.RandomDNS1123Label(8))
			listeners := []appmesh.VirtualGatewayListener{vgBuilder.BuildVGListener("http", 8080, "/")}
			nsSelector := map[string]string{"gateway": "ingress-gw"}
			vg := vgBuilder.BuildVirtualGateway(vgName, listeners, nsSelector)

			By("Creating a virtual gateway resource in k8s", func() {
				err := vgTest.Create(ctx, f, vg)
				Expect(err).NotTo(HaveOccurred())
			})

			By("Validating the virtual gateway in AWS", func() {
				err := vgTest.CheckInAWS(ctx, f, mesh, vg)
				Expect(err).NotTo(HaveOccurred())

			})

			By("Check mesh finalizers", func() {
				var wg sync.WaitGroup
				wg.Add(1)

				go func() {
					meshTest.Cleanup(ctx, f)
					wg.Done()
				}()

				By("Wait for deletion timestamp to appear on mesh before we check virtual gateway", func() {
					res := meshTest.WaitForDeletionTimestamp(ctx, f, mesh)
					Expect(res).To(Equal(true))
				})

				By("Check virtual gateway in AWS after mesh deletion - it should exist", func() {
					err := vgTest.CheckInAWS(ctx, f, mesh, vg)
					Expect(err).NotTo(HaveOccurred())
				})

				By("Check the mesh as the virtual is not deleted - the mesh should exist", func() {
					ms, err := meshTest.Get(ctx, f, mesh)
					Expect(err).NotTo(HaveOccurred())

					hasFin := appmeshk8s.HasFinalizer(ms, appmeshk8s.FinalizerAWSAppMeshResources)
					Expect(hasFin).To(Equal(true))
				})

				By("Delete virtual gateway in k8s", func() {
					vgTest.Cleanup(ctx, f)
				})

				By("Check virtual gateway in AWS after delete in k8s - it should not exist", func() {
					err := vgTest.CheckInAWS(ctx, f, mesh, vg)
					Expect(err).To(HaveOccurred())
				})

				wg.Wait()

				By("Check the mesh as the virtual gateway has been deleted -mesh should not exist", func() {
					_, err := meshTest.Get(ctx, f, mesh)
					Expect(apierrs.IsNotFound(err)).To(Equal(true))
				})

			})

		})

		It("Virtual Gateway Connection Pool Scenarios", func() {

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
				vgBuilder.Namespace = namespace.Name
				vgTest.Namespace = namespace

				oldNS := namespace.DeepCopy()
				namespace.Labels = algorithm.MergeStringMap(map[string]string{
					"appmesh.k8s.aws/sidecarInjectorWebhook": "enabled",
					"mesh":                                   meshName,
				}, namespace.Labels)

				err = f.K8sClient.Patch(ctx, namespace, client.MergeFrom(oldNS))
				Expect(err).NotTo(HaveOccurred())
			})

			vgName := fmt.Sprintf("vg-%s", utils.RandomDNS1123Label(8))
			httpConnectionPool := &appmesh.HTTPConnectionPool{
				MaxConnections:     60,
				MaxPendingRequests: 100,
			}
			vgConnectionPoolListener := vgBuilder.BuildListenerWithConnectionPools("http", 8080, httpConnectionPool, nil, nil)
			listeners := []appmesh.VirtualGatewayListener{vgConnectionPoolListener}
			nsSelector := map[string]string{"gateway": "ingress-gw"}
			vg := vgBuilder.BuildVirtualGateway(vgName, listeners, nsSelector)

			By("Creating a virtual gateway with HTTP connectiol pool", func() {
				err := vgTest.Create(ctx, f, vg)
				Expect(err).NotTo(HaveOccurred())
			})

			By("Validate the virtual gateway in AWS", func() {
				err := vgTest.CheckInAWS(ctx, f, mesh, vg)
				Expect(err).NotTo(HaveOccurred())

			})

			By("Validate update of HTTP connection pool thresholds", func() {
				oldVG := vgTest.VirtualGateways[vg.Name].DeepCopy()
				httpConnectionPool := &appmesh.HTTPConnectionPool{
					MaxConnections:     200,
					MaxPendingRequests: 50,
				}
				vgConnectionPoolListener := vgBuilder.BuildListenerWithConnectionPools("http", 8080, httpConnectionPool, nil, nil)
				listeners := []appmesh.VirtualGatewayListener{vgConnectionPoolListener}

				vgTest.VirtualGateways[vg.Name].Spec.Listeners = listeners
				updatedVG, err := vgTest.Update(ctx, f, vgTest.VirtualGateways[vg.Name], oldVG)
				Expect(err).NotTo(HaveOccurred())

				err = vgTest.CheckInAWS(ctx, f, mesh, updatedVG)
				Expect(err).NotTo(HaveOccurred())

			})

			By("Validate update disable connection pool", func() {
				oldVG := vgTest.VirtualGateways[vg.Name].DeepCopy()
				listeners := []appmesh.VirtualGatewayListener{vgBuilder.BuildVGListener("http", 8080, "/")}

				vgTest.VirtualGateways[vg.Name].Spec.Listeners = listeners
				updatedVG, err := vgTest.Update(ctx, f, vgTest.VirtualGateways[vg.Name], oldVG)
				Expect(err).NotTo(HaveOccurred())

				err = vgTest.CheckInAWS(ctx, f, mesh, updatedVG)
				Expect(err).NotTo(HaveOccurred())

			})

			By("Validate update enable connection pool", func() {
				oldVG := vgTest.VirtualGateways[vg.Name].DeepCopy()
				httpConnectionPool := &appmesh.HTTPConnectionPool{
					MaxConnections:     150,
					MaxPendingRequests: 70,
				}
				vgConnectionPoolListener := vgBuilder.BuildListenerWithConnectionPools("http", 8080, httpConnectionPool, nil, nil)
				listeners := []appmesh.VirtualGatewayListener{vgConnectionPoolListener}

				vgTest.VirtualGateways[vg.Name].Spec.Listeners = listeners
				updatedVG, err := vgTest.Update(ctx, f, vgTest.VirtualGateways[vg.Name], oldVG)
				Expect(err).NotTo(HaveOccurred())

				err = vgTest.CheckInAWS(ctx, f, mesh, updatedVG)
				Expect(err).NotTo(HaveOccurred())

			})

			http2ConnectionPool := &appmesh.HTTP2ConnectionPool{
				MaxRequests: 50,
			}
			grpcConnectionPool := &appmesh.GRPCConnectionPool{
				MaxRequests: 30,
			}

			vgName = fmt.Sprintf("vg-%s", utils.RandomDNS1123Label(8))
			vgConnectionPoolListener = vgBuilder.BuildListenerWithConnectionPools("http2", 8080, nil, http2ConnectionPool, nil)
			listeners = []appmesh.VirtualGatewayListener{vgConnectionPoolListener}
			vg = vgBuilder.BuildVirtualGateway(vgName, listeners, nsSelector)

			By("Creating a virtual gateway with HTTP2 connection pool", func() {
				err := vgTest.Create(ctx, f, vg)
				Expect(err).NotTo(HaveOccurred())
			})

			By("Validate the virtual gateway in AWS", func() {
				err := vgTest.CheckInAWS(ctx, f, mesh, vg)
				Expect(err).NotTo(HaveOccurred())

			})

			vgName = fmt.Sprintf("vg-%s", utils.RandomDNS1123Label(8))
			vgConnectionPoolListener = vgBuilder.BuildListenerWithConnectionPools("grpc", 8080, nil, nil, grpcConnectionPool)
			listeners = []appmesh.VirtualGatewayListener{vgConnectionPoolListener}
			vg = vgBuilder.BuildVirtualGateway(vgName, listeners, nsSelector)

			By("Creating a virtual gateway with GRPC connection pool", func() {
				err := vgTest.Create(ctx, f, vg)
				Expect(err).NotTo(HaveOccurred())
			})

			By("Validate the virtual gateway in AWS", func() {
				err := vgTest.CheckInAWS(ctx, f, mesh, vg)
				Expect(err).NotTo(HaveOccurred())

			})

			vgName = fmt.Sprintf("vg-%s", utils.RandomDNS1123Label(8))
			vgConnectionPoolListener = vgBuilder.BuildListenerWithConnectionPools("grpc", 8080, httpConnectionPool, http2ConnectionPool, grpcConnectionPool)
			listeners = []appmesh.VirtualGatewayListener{vgConnectionPoolListener}
			vg = vgBuilder.BuildVirtualGateway(vgName, listeners, nsSelector)

			By("Creating a virtual gateway with HTTP, HTTP2, GRPC connection pool", func() {
				err := vgTest.Create(ctx, f, vg)
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
