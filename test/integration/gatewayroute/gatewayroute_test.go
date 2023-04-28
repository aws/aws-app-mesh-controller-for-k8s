package gatewayroute_test

import (
	"context"
	"fmt"
	"sync"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/algorithm"
	appmeshk8s "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/k8s"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/k8s"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/manifest"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/utils"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/integration/gatewayroute"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/integration/mesh"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/integration/virtualgateway"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/integration/virtualservice"

	"github.com/aws/aws-sdk-go/aws"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("GatewayRoute", func() {

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

	Context("Gateway Route scenarios", func() {
		var meshTest mesh.MeshTest
		var vsTest virtualservice.VirtualServiceTest
		var vgTest virtualgateway.VirtualGatewayTest
		var grTest gatewayroute.GatewayRouteTest

		meshTest = mesh.MeshTest{
			Meshes: make(map[string]*appmesh.Mesh),
		}

		vgTest = virtualgateway.VirtualGatewayTest{
			VirtualGateways: make(map[string]*appmesh.VirtualGateway),
		}

		vsTest = virtualservice.VirtualServiceTest{
			VirtualServices: make(map[string]*appmesh.VirtualService),
		}

		grTest = gatewayroute.GatewayRouteTest{
			GatewayRoutes: make(map[string]*appmesh.GatewayRoute),
		}

		vgBuilder := &manifest.VGBuilder{}
		vsBuilder := &manifest.VSBuilder{}
		grBuilder := &manifest.GRBuilder{}

		AfterEach(func() {
			grTest.Cleanup(ctx, f)
			vsTest.Cleanup(ctx, f)
			vgTest.Cleanup(ctx, f)
			meshTest.Cleanup(ctx, f)
		})

		It("Gateway Route Create Scenarios", func() {

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

			vgName := fmt.Sprintf("vg-%s", utils.RandomDNS1123Label(8))

			By("Create a namespace and add labels", func() {
				namespace, err := f.NSManager.AllocateNamespace(ctx, "appmeshtest")
				Expect(err).NotTo(HaveOccurred())
				vgBuilder.Namespace = namespace.Name
				vgTest.Namespace = namespace
				vsBuilder.Namespace = namespace.Name
				vsTest.Namespace = namespace
				grBuilder.Namespace = namespace.Name
				grTest.Namespace = namespace

				oldNS := namespace.DeepCopy()
				namespace.Labels = algorithm.MergeStringMap(map[string]string{
					"appmesh.k8s.aws/sidecarInjectorWebhook": "enabled",
					"mesh":                                   meshName,
					"gateway":                                vgName,
				}, namespace.Labels)

				err = f.K8sClient.Patch(ctx, namespace, client.MergeFrom(oldNS))
				Expect(err).NotTo(HaveOccurred())
			})

			listeners := []appmesh.VirtualGatewayListener{vgBuilder.BuildVGListener("http", 8080, "/")}
			nsSelector := map[string]string{"gateway": vgName}
			vg := vgBuilder.BuildVirtualGateway(vgName, listeners, nsSelector)

			By("Creating a virtual gateway resource in k8s", func() {
				err := vgTest.Create(ctx, f, vg)
				Expect(err).NotTo(HaveOccurred())
			})

			By("validating the virtual gateway in AWS", func() {
				err := vgTest.CheckInAWS(ctx, f, mesh, vg)
				Expect(err).NotTo(HaveOccurred())

			})

			vsName := fmt.Sprintf("vs-%s", utils.RandomDNS1123Label(8))
			vs := vsBuilder.BuildVirtualServiceNoBackend(vsName)

			By("Creating a virtual service (with no backend) resource in k8s", func() {
				err := vsTest.Create(ctx, f, vs)
				Expect(err).NotTo(HaveOccurred())
			})

			By("Validating the virtual service (with no backend) in AWS", func() {
				err := vsTest.CheckInAWS(ctx, f, mesh, vs)
				Expect(err).NotTo(HaveOccurred())

			})

			grName := fmt.Sprintf("gr-%s", utils.RandomDNS1123Label(8))
			gr := grBuilder.BuildGatewayRouteWithHTTP(grName, vsName, "testPrefix")

			By("Creating a gateway route resource in k8s", func() {
				err := grTest.Create(ctx, f, gr)
				Expect(err).NotTo(HaveOccurred())
			})

			By("Validating the gateway route in AWS", func() {
				err := grTest.CheckInAWS(ctx, f, mesh, vg, gr)
				Expect(err).NotTo(HaveOccurred())

			})

			grName = fmt.Sprintf("gr-%s", utils.RandomDNS1123Label(8))
			gr = grBuilder.BuildGatewayRouteWithHTTP(grName, vsName, "failPrefix")
			gr.Spec.AWSName = aws.String(fmt.Sprintf("vn-%s", utils.RandomDNS1123Label(256)))

			By("Creating a gateway route resource in k8s with a name exceeding the character limit", func() {
				// Not using grTest.Create as it hangs
				err := f.K8sClient.Create(ctx, gr)
				observedVg := &appmesh.VirtualGateway{}
				for i := 0; i < 5; i++ {
					if err := f.K8sClient.Get(ctx, k8s.NamespacedName(vg), observedVg); err != nil {
						if i >= 5 {
							Expect(err).NotTo(HaveOccurred())
						}
					}
					time.Sleep(100 * time.Millisecond)
				}
				grTest.GatewayRoutes[gr.Name] = gr
				Expect(err).NotTo(HaveOccurred())
			})

			By("Check gateway route in AWS - it should not exist", func() {
				err := grTest.CheckInAWS(ctx, f, mesh, vg, gr)
				Expect(err).To(HaveOccurred())
			})

			By("checking events for the BadRequestException", func() {
				clientset, err := kubernetes.NewForConfig(f.RestCfg)
				Expect(err).NotTo(HaveOccurred())
				events, err := clientset.CoreV1().Events(grTest.Namespace.Name).List(ctx, metav1.ListOptions{
					FieldSelector: fmt.Sprintf("involvedObject.name=%s", gr.Name),
					TypeMeta: metav1.TypeMeta{
						Kind: "Pod",
					},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(events.Items).NotTo(BeEmpty())
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

			grName = fmt.Sprintf("gr-%s", utils.RandomDNS1123Label(8))
			gr = grBuilder.BuildGatewayRouteWithHTTP(grName, vsName, "testPrefix")

			By("Creating a gateway route resource in k8s when no mesh matches namespace", func() {
				err := grTest.Create(ctx, f, gr)
				Expect(err).To(HaveOccurred())
			})

		})

		It("Gateway Route Update Scenarios", func() {

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

			vgName := fmt.Sprintf("vg-%s", utils.RandomDNS1123Label(8))

			By("Create a namespace and add labels", func() {
				namespace, err := f.NSManager.AllocateNamespace(ctx, "appmeshtest")
				Expect(err).NotTo(HaveOccurred())
				vgBuilder.Namespace = namespace.Name
				vgTest.Namespace = namespace
				vsBuilder.Namespace = namespace.Name
				vsTest.Namespace = namespace
				grBuilder.Namespace = namespace.Name
				grTest.Namespace = namespace

				oldNS := namespace.DeepCopy()
				namespace.Labels = algorithm.MergeStringMap(map[string]string{
					"appmesh.k8s.aws/sidecarInjectorWebhook": "enabled",
					"mesh":                                   meshName,
					"gateway":                                vgName,
				}, namespace.Labels)

				err = f.K8sClient.Patch(ctx, namespace, client.MergeFrom(oldNS))
				Expect(err).NotTo(HaveOccurred())
			})

			listeners := []appmesh.VirtualGatewayListener{vgBuilder.BuildVGListener("http", 8080, "/")}
			nsSelector := map[string]string{"gateway": vgName}
			vg := vgBuilder.BuildVirtualGateway(vgName, listeners, nsSelector)

			By("Creating a virtual gateway resource in k8s", func() {
				err := vgTest.Create(ctx, f, vg)
				Expect(err).NotTo(HaveOccurred())
			})

			By("validating the virtual gateway in AWS", func() {
				err := vgTest.CheckInAWS(ctx, f, mesh, vg)
				Expect(err).NotTo(HaveOccurred())

			})

			vsName := fmt.Sprintf("vs-%s", utils.RandomDNS1123Label(8))
			vs := vsBuilder.BuildVirtualServiceNoBackend(vsName)

			By("Creating a virtual service (with no backend) resource in k8s", func() {
				err := vsTest.Create(ctx, f, vs)
				Expect(err).NotTo(HaveOccurred())
			})

			By("Validating the virtual service (with no backend) in AWS", func() {
				err := vsTest.CheckInAWS(ctx, f, mesh, vs)
				Expect(err).NotTo(HaveOccurred())

			})

			grName := fmt.Sprintf("gr-%s", utils.RandomDNS1123Label(8))
			gr := grBuilder.BuildGatewayRouteWithHTTP(grName, vsName, "testPrefix")

			By("Creating a gateway route resource in k8s", func() {
				err := grTest.Create(ctx, f, gr)
				Expect(err).NotTo(HaveOccurred())
			})

			By("Validating the gateway route in AWS", func() {
				err := grTest.CheckInAWS(ctx, f, mesh, vg, gr)
				Expect(err).NotTo(HaveOccurred())

			})

			By("Update HTTP route and validate", func() {
				oldGR := grTest.GatewayRoutes[gr.Name].DeepCopy()
				newHTTPRoute := grBuilder.BuildHTTPRoute("newprefix", vsName, grTest.Namespace.Name)

				grTest.GatewayRoutes[gr.Name].Spec.HTTPRoute = newHTTPRoute
				err := grTest.Update(ctx, f, grTest.GatewayRoutes[gr.Name], oldGR)
				Expect(err).NotTo(HaveOccurred())

				err = grTest.CheckInAWS(ctx, f, mesh, vg, grTest.GatewayRoutes[gr.Name])
				Expect(err).NotTo(HaveOccurred())

			})

			By("Update GRPC route and validate", func() {
				oldGR := grTest.GatewayRoutes[gr.Name].DeepCopy()
				GRPCRoute := grBuilder.BuildGRPCRoute("newservice", vsName, grTest.Namespace.Name)

				grTest.GatewayRoutes[gr.Name].Spec.HTTPRoute = nil
				grTest.GatewayRoutes[gr.Name].Spec.GRPCRoute = GRPCRoute
				err := grTest.Update(ctx, f, grTest.GatewayRoutes[gr.Name], oldGR)
				Expect(err).NotTo(HaveOccurred())

				err = grTest.CheckInAWS(ctx, f, mesh, vg, grTest.GatewayRoutes[gr.Name])
				Expect(err).NotTo(HaveOccurred())

			})

			By("Update HTTP2 route and validate", func() {
				oldGR := grTest.GatewayRoutes[gr.Name].DeepCopy()
				HTTP2Route := grBuilder.BuildHTTPRoute("newprefix2", vsName, grTest.Namespace.Name)

				grTest.GatewayRoutes[gr.Name].Spec.HTTPRoute = nil
				grTest.GatewayRoutes[gr.Name].Spec.GRPCRoute = nil
				grTest.GatewayRoutes[gr.Name].Spec.HTTP2Route = HTTP2Route
				err := grTest.Update(ctx, f, grTest.GatewayRoutes[gr.Name], oldGR)
				Expect(err).NotTo(HaveOccurred())

				err = grTest.CheckInAWS(ctx, f, mesh, vg, grTest.GatewayRoutes[gr.Name])
				Expect(err).NotTo(HaveOccurred())

			})

			By("Update AWSName for gateway route and validate it cannot be updated", func() {
				oldGR := grTest.GatewayRoutes[gr.Name].DeepCopy()
				grTest.GatewayRoutes[gr.Name].Spec.AWSName = aws.String("newGatewayRouteAWSName")

				err := grTest.Update(ctx, f, grTest.GatewayRoutes[gr.Name], oldGR)
				Expect(err).To(HaveOccurred())
			})

		})

		It("Gateway Route Delete Scenarios", func() {

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

			vgName := fmt.Sprintf("vg-%s", utils.RandomDNS1123Label(8))

			By("Create a namespace and add labels", func() {
				namespace, err := f.NSManager.AllocateNamespace(ctx, "appmeshtest")
				Expect(err).NotTo(HaveOccurred())
				vgBuilder.Namespace = namespace.Name
				vgTest.Namespace = namespace
				vsBuilder.Namespace = namespace.Name
				vsTest.Namespace = namespace
				grBuilder.Namespace = namespace.Name
				grTest.Namespace = namespace

				oldNS := namespace.DeepCopy()
				namespace.Labels = algorithm.MergeStringMap(map[string]string{
					"appmesh.k8s.aws/sidecarInjectorWebhook": "enabled",
					"mesh":                                   meshName,
					"gateway":                                vgName,
				}, namespace.Labels)

				err = f.K8sClient.Patch(ctx, namespace, client.MergeFrom(oldNS))
				Expect(err).NotTo(HaveOccurred())
			})

			listeners := []appmesh.VirtualGatewayListener{vgBuilder.BuildVGListener("http", 8080, "/")}
			nsSelector := map[string]string{"gateway": vgName}
			vg := vgBuilder.BuildVirtualGateway(vgName, listeners, nsSelector)

			By("Creating a virtual gateway resource in k8s", func() {
				err := vgTest.Create(ctx, f, vg)
				Expect(err).NotTo(HaveOccurred())
			})

			By("Validating the virtual gateway in AWS", func() {
				err := vgTest.CheckInAWS(ctx, f, mesh, vg)
				Expect(err).NotTo(HaveOccurred())

			})

			vsName := fmt.Sprintf("vs-%s", utils.RandomDNS1123Label(8))
			vs := vsBuilder.BuildVirtualServiceNoBackend(vsName)

			By("Creating a virtual service (with no backend) resource in k8s", func() {
				err := vsTest.Create(ctx, f, vs)
				Expect(err).NotTo(HaveOccurred())
			})

			By("Validating the virtual service (with no backend) in AWS", func() {
				err := vsTest.CheckInAWS(ctx, f, mesh, vs)
				Expect(err).NotTo(HaveOccurred())

			})

			grName := fmt.Sprintf("gr-%s", utils.RandomDNS1123Label(8))
			gr := grBuilder.BuildGatewayRouteWithHTTP(grName, vsName, "testPrefix")

			By("Creating a gateway route resource in k8s", func() {
				err := grTest.Create(ctx, f, gr)
				Expect(err).NotTo(HaveOccurred())
			})

			By("Validating the gateway route in AWS", func() {
				err := grTest.CheckInAWS(ctx, f, mesh, vg, gr)
				Expect(err).NotTo(HaveOccurred())

			})

			By("Check mesh finalizers", func() {
				var wg sync.WaitGroup
				wg.Add(1)

				go func() {
					meshTest.Cleanup(ctx, f)
					wg.Done()
				}()

				By("Wait for deletion timestamp to appear on mesh before we check gateway route", func() {
					res := meshTest.WaitForDeletionTimestamp(ctx, f, mesh)
					Expect(res).To(Equal(true))
				})

				By("Check gateway route in AWS after mesh deletion - it should exist", func() {
					err := grTest.CheckInAWS(ctx, f, mesh, vg, gr)
					Expect(err).NotTo(HaveOccurred())
				})

				By("Check the mesh as the gateway route is not deleted - the mesh should exist", func() {
					ms, err := meshTest.Get(ctx, f, mesh)
					Expect(err).NotTo(HaveOccurred())

					hasFin := appmeshk8s.HasFinalizer(ms, appmeshk8s.FinalizerAWSAppMeshResources)
					Expect(hasFin).To(Equal(true))
				})

				By("Delete gateway route in k8s", func() {
					grTest.Cleanup(ctx, f)
				})

				By("Check gateway route in AWS after delete in k8s - it should not exist", func() {
					err := grTest.CheckInAWS(ctx, f, mesh, vg, gr)
					Expect(err).To(HaveOccurred())
				})

				wg.Wait()

				By("Check the mesh as the gateway route has been deleted -mesh should not exist", func() {
					_, err := meshTest.Get(ctx, f, mesh)
					Expect(apierrs.IsNotFound(err)).To(Equal(true))
				})

			})

		})
	})
})
