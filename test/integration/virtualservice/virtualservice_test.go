package virtualservice_test

import (
	"context"
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sync"
	"time"

	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/algorithm"
	appmeshk8s "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/k8s"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/k8s"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/manifest"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/utils"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/integration/mesh"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/integration/virtualnode"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/integration/virtualrouter"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/integration/virtualservice"

	"github.com/aws/aws-sdk-go/aws"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("VirtualService", func() {

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

	Context("Virtual Router scenarios", func() {
		var meshTest mesh.MeshTest
		var vnTest virtualnode.VirtualNodeTest
		var vrTest virtualrouter.VirtualRouterTest
		var vsTest virtualservice.VirtualServiceTest

		meshTest = mesh.MeshTest{
			Meshes: make(map[string]*appmesh.Mesh),
		}

		vnTest = virtualnode.VirtualNodeTest{
			VirtualNodes: make(map[string]*appmesh.VirtualNode),
		}

		vrTest = virtualrouter.VirtualRouterTest{
			VirtualRouters: make(map[string]*appmesh.VirtualRouter),
		}

		vsTest = virtualservice.VirtualServiceTest{
			VirtualServices: make(map[string]*appmesh.VirtualService),
		}

		vnBuilder := &manifest.VNBuilder{
			ServiceDiscoveryType: manifest.DNSServiceDiscovery,
		}

		vrBuilder := &manifest.VRBuilder{}
		vsBuilder := &manifest.VSBuilder{}

		AfterEach(func() {
			vsTest.Cleanup(ctx, f)
			vrTest.Cleanup(ctx, f)
			vnTest.Cleanup(ctx, f)
			meshTest.Cleanup(ctx, f)
		})

		It("Create virtual service scenarios", func() {

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
				vrBuilder.Namespace = namespace.Name
				vsBuilder.Namespace = namespace.Name
				vnTest.Namespace = namespace
				vrTest.Namespace = namespace
				vsTest.Namespace = namespace

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

			weightedTargets := []manifest.WeightedVirtualNode{{
				VirtualNode: k8s.NamespacedName(vn),
				Weight:      1,
			}}

			routeCfgs := []manifest.RouteToWeightedVirtualNodes{{
				Path:            "/route-1",
				WeightedTargets: weightedTargets,
			},
			}

			routes := vrBuilder.BuildRoutes(routeCfgs)

			vrName := fmt.Sprintf("vr-%s", utils.RandomDNS1123Label(8))
			vrBuilder.Listeners = []appmesh.VirtualRouterListener{vrBuilder.BuildVirtualRouterListener("http", 8080)}

			vr := vrBuilder.BuildVirtualRouter(vrName, routes)

			By("Creating a virtual router resource in k8s", func() {
				err := vrTest.Create(ctx, f, vr)
				Expect(err).NotTo(HaveOccurred())
			})

			By("Validating the virtual router in AWS", func() {
				err := vrTest.CheckInAWS(ctx, f, mesh, vr)
				Expect(err).NotTo(HaveOccurred())

			})

			vsName := fmt.Sprintf("vs-%s", utils.RandomDNS1123Label(8))
			vs := vsBuilder.BuildVirtualServiceWithRouterBackend(vsName, vr.Name)

			By("Creating a virtual service (with virtual router backend) resource in k8s", func() {
				err := vsTest.Create(ctx, f, vs)
				Expect(err).NotTo(HaveOccurred())
			})

			By("Validating the virtual service (with virtual router backend) in AWS", func() {
				err := vsTest.CheckInAWS(ctx, f, mesh, vs)
				Expect(err).NotTo(HaveOccurred())

			})

			vsName = fmt.Sprintf("vs-%s", utils.RandomDNS1123Label(8))
			vs = vsBuilder.BuildVirtualServiceWithNodeBackend(vsName, vn.Name)

			By("Creating a virtual service (with virtual node backend) resource in k8s", func() {
				err := vsTest.Create(ctx, f, vs)
				Expect(err).NotTo(HaveOccurred())
			})

			By("Validating the virtual service (with virtual node backend) in AWS", func() {
				err := vsTest.CheckInAWS(ctx, f, mesh, vs)
				Expect(err).NotTo(HaveOccurred())

			})

			vsName = fmt.Sprintf("vs-%s", utils.RandomDNS1123Label(8))
			vs = vsBuilder.BuildVirtualServiceNoBackend(vsName)

			By("Creating a virtual service (with no backend) resource in k8s", func() {
				err := vsTest.Create(ctx, f, vs)
				Expect(err).NotTo(HaveOccurred())
			})

			By("Validating the virtual service (with no backend) in AWS", func() {
				err := vsTest.CheckInAWS(ctx, f, mesh, vs)
				Expect(err).NotTo(HaveOccurred())

			})

			vsName = fmt.Sprintf("vs-%s", utils.RandomDNS1123Label(8))
			vs = vsBuilder.BuildVirtualServiceNoBackend(vsName)
			vs.Spec.AWSName = aws.String(fmt.Sprintf("vs-%s", utils.RandomDNS1123Label(256)))

			By("Creating a virtual service resource in k8s with a name exceeding the character limit", func() {
				// Not using vsTest.Create as it hangs
				err := f.K8sClient.Create(ctx, vs)
				vsTest.VirtualServices[vs.Name] = vs
				observedVs := &appmesh.VirtualService{}
				for i := 0; i < 5; i++ {
					if err := f.K8sClient.Get(ctx, k8s.NamespacedName(vs), observedVs); err != nil {
						if i >= 5 {
							Expect(err).NotTo(HaveOccurred())
						}
					}
					time.Sleep(100 * time.Millisecond)
				}
				Expect(err).NotTo(HaveOccurred())
			})

			By("Check virtual service in AWS - it should not exist", func() {
				err := vsTest.CheckInAWS(ctx, f, mesh, vs)
				Expect(err).To(HaveOccurred())
			})

			By("checking events for the BadRequestException", func() {
				clientset, err := kubernetes.NewForConfig(f.RestCfg)
				Expect(err).NotTo(HaveOccurred())
				events, err := clientset.CoreV1().Events(vsTest.Namespace.Name).List(ctx, metav1.ListOptions{
					FieldSelector: fmt.Sprintf("involvedObject.name=%s", vs.Name),
					TypeMeta: metav1.TypeMeta{
						Kind: "Pod",
					},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(events.Items).NotTo(BeEmpty())
			})

			By("Set incorrect labels on namespace", func() {
				oldNS := vrTest.Namespace.DeepCopy()
				vrTest.Namespace.Labels = algorithm.MergeStringMap(map[string]string{
					"appmesh.k8s.aws/sidecarInjectorWebhook": "enabled",
					"mesh":                                   "dontmatch",
				}, vrTest.Namespace.Labels)

				err := f.K8sClient.Patch(ctx, vrTest.Namespace, client.MergeFrom(oldNS))
				Expect(err).NotTo(HaveOccurred())
			})

			vsName = fmt.Sprintf("vs-%s", utils.RandomDNS1123Label(8))
			vs = vsBuilder.BuildVirtualServiceNoBackend(vsName)

			By("Creating a virtual service resource in k8s when no mesh matches namespace", func() {
				err := vsTest.Create(ctx, f, vs)
				Expect(err).To(HaveOccurred())

			})

		})

		It("Update virtual service scenarios", func() {

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
				vrBuilder.Namespace = namespace.Name
				vsBuilder.Namespace = namespace.Name
				vnTest.Namespace = namespace
				vrTest.Namespace = namespace
				vsTest.Namespace = namespace

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

			weightedTargets := []manifest.WeightedVirtualNode{{
				VirtualNode: k8s.NamespacedName(vn),
				Weight:      1,
			}}

			routeCfgs := []manifest.RouteToWeightedVirtualNodes{{
				Path:            "/route-1",
				WeightedTargets: weightedTargets,
			},
			}

			routes := vrBuilder.BuildRoutes(routeCfgs)

			vrName := fmt.Sprintf("vr-%s", utils.RandomDNS1123Label(8))
			vrBuilder.Listeners = []appmesh.VirtualRouterListener{vrBuilder.BuildVirtualRouterListener("http", 8080)}

			vr := vrBuilder.BuildVirtualRouter(vrName, routes)

			By("Creating a virtual router resource in k8s", func() {
				err := vrTest.Create(ctx, f, vr)
				Expect(err).NotTo(HaveOccurred())
			})

			By("Validating the virtual router in AWS", func() {
				err := vrTest.CheckInAWS(ctx, f, mesh, vr)
				Expect(err).NotTo(HaveOccurred())

			})

			vsName := fmt.Sprintf("vs-%s", utils.RandomDNS1123Label(8))
			vs := vsBuilder.BuildVirtualServiceWithRouterBackend(vsName, vr.Name)

			By("Creating a virtual service (with virtual router backend) resource in k8s", func() {
				err := vsTest.Create(ctx, f, vs)
				Expect(err).NotTo(HaveOccurred())
			})

			By("Validating the virtual service (with virtual router backend) in AWS", func() {
				err := vsTest.CheckInAWS(ctx, f, mesh, vs)
				Expect(err).NotTo(HaveOccurred())

			})

			By("Update AWSName for virtual service and validate it cannot be updated", func() {
				oldVS := vsTest.VirtualServices[vs.Name].DeepCopy()
				vsTest.VirtualServices[vs.Name].Spec.AWSName = aws.String("newVirtualServiceAWSName")

				err := vsTest.Update(ctx, f, vsTest.VirtualServices[vs.Name], oldVS)
				Expect(err).To(HaveOccurred())
			})

			By("Update backend from virtualrouter to virtualnode for virtual router and validate", func() {
				oldVS := vsTest.VirtualServices[vs.Name].DeepCopy()
				vsTest.VirtualServices[vs.Name].Spec.Provider = &appmesh.VirtualServiceProvider{
					VirtualNode: &appmesh.VirtualNodeServiceProvider{
						VirtualNodeRef: &appmesh.VirtualNodeReference{
							Namespace: aws.String(vsTest.Namespace.Name),
							Name:      vn.Name,
						},
					},
					VirtualRouter: nil,
				}

				err := vsTest.Update(ctx, f, vsTest.VirtualServices[vs.Name], oldVS)
				Expect(err).NotTo(HaveOccurred())
			})

			By("Validating the virtual service (with updated virtual node backend) in AWS", func() {
				err := vsTest.CheckInAWS(ctx, f, mesh, vsTest.VirtualServices[vs.Name])
				Expect(err).NotTo(HaveOccurred())

			})

			By("Update backend from virtualnode to no backend for virtual router and validate", func() {
				oldVS := vsTest.VirtualServices[vs.Name].DeepCopy()
				vsTest.VirtualServices[vs.Name].Spec.Provider = nil

				err := vsTest.Update(ctx, f, vsTest.VirtualServices[vs.Name], oldVS)
				Expect(err).NotTo(HaveOccurred())
			})

			By("Validating the virtual service (with updated no backend) in AWS", func() {
				err := vsTest.CheckInAWS(ctx, f, mesh, vsTest.VirtualServices[vs.Name])
				Expect(err).NotTo(HaveOccurred())

			})

		})

		It("Delete virtual service scenarios", func() {

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
				vrBuilder.Namespace = namespace.Name
				vsBuilder.Namespace = namespace.Name
				vnTest.Namespace = namespace
				vrTest.Namespace = namespace
				vsTest.Namespace = namespace

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

			weightedTargets := []manifest.WeightedVirtualNode{{
				VirtualNode: k8s.NamespacedName(vn),
				Weight:      1,
			}}

			routeCfgs := []manifest.RouteToWeightedVirtualNodes{{
				Path:            "/route-1",
				WeightedTargets: weightedTargets,
			},
			}

			routes := vrBuilder.BuildRoutes(routeCfgs)

			vrName := fmt.Sprintf("vr-%s", utils.RandomDNS1123Label(8))
			vrBuilder.Listeners = []appmesh.VirtualRouterListener{vrBuilder.BuildVirtualRouterListener("http", 8080)}

			vr := vrBuilder.BuildVirtualRouter(vrName, routes)

			By("Creating a virtual router resource in k8s", func() {
				err := vrTest.Create(ctx, f, vr)
				Expect(err).NotTo(HaveOccurred())
			})

			By("Validating the virtual router in AWS", func() {
				err := vrTest.CheckInAWS(ctx, f, mesh, vr)
				Expect(err).NotTo(HaveOccurred())

			})

			vsName := fmt.Sprintf("vs-%s", utils.RandomDNS1123Label(8))
			vs := vsBuilder.BuildVirtualServiceWithRouterBackend(vsName, vr.Name)

			By("Creating a virtual service resource in k8s", func() {
				err := vsTest.Create(ctx, f, vs)
				Expect(err).NotTo(HaveOccurred())
			})

			By("Validating the virtual service in AWS", func() {
				err := vsTest.CheckInAWS(ctx, f, mesh, vs)
				Expect(err).NotTo(HaveOccurred())

			})

			By("Check mesh finalizers", func() {
				var wg sync.WaitGroup
				wg.Add(1)

				go func() {
					meshTest.Cleanup(ctx, f)
					wg.Done()
				}()

				By("Wait for deletion timestamp to appear on mesh before we check virtual service", func() {
					res := meshTest.WaitForDeletionTimestamp(ctx, f, mesh)
					Expect(res).To(Equal(true))
				})

				By("Check virtual service in AWS after mesh deletion - it should exist", func() {
					err := vsTest.CheckInAWS(ctx, f, mesh, vs)
					Expect(err).NotTo(HaveOccurred())
				})

				By("Check the mesh as the virtual service is not deleted - the mesh should exist", func() {
					ms, err := meshTest.Get(ctx, f, mesh)
					Expect(err).NotTo(HaveOccurred())

					hasFin := appmeshk8s.HasFinalizer(ms, appmeshk8s.FinalizerAWSAppMeshResources)
					Expect(hasFin).To(Equal(true))
				})

				By("Delete virtual service in k8s", func() {
					vsTest.Cleanup(ctx, f)
				})

				By("Check virtual service in AWS after delete in k8s - it should not exist", func() {
					err := vsTest.CheckInAWS(ctx, f, mesh, vs)
					Expect(err).To(HaveOccurred())
				})

				wg.Wait()

				By("Check the mesh as the virtual service has been deleted - mesh should not exist", func() {
					_, err := meshTest.Get(ctx, f, mesh)
					Expect(apierrs.IsNotFound(err)).To(Equal(true))
				})

			})

		})
	})
})
