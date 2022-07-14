package backendgroup_test

import (
	"context"
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/wait"

	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/algorithm"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/k8s"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/manifest"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/utils"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/integration/backendgroup"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/integration/mesh"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/integration/virtualnode"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/integration/virtualrouter"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/integration/virtualservice"

	"github.com/aws/aws-sdk-go/aws"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("BackendGroup", func() {

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

	Context("Backend Group scenarios", func() {
		var meshTest mesh.MeshTest
		var vnTest virtualnode.VirtualNodeTest
		var vrTest virtualrouter.VirtualRouterTest
		var vsTest virtualservice.VirtualServiceTest
		var bgTest backendgroup.BackendGroupTest

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

		bgTest = backendgroup.BackendGroupTest{
			BackendGroups: make(map[string]*appmesh.BackendGroup),
		}

		vnBuilder := &manifest.VNBuilder{
			ServiceDiscoveryType: manifest.DNSServiceDiscovery,
		}

		vrBuilder := &manifest.VRBuilder{}
		vsBuilder := &manifest.VSBuilder{}

		bgBuilder := &manifest.BGBuilder{}

		AfterEach(func() {
			vsTest.Cleanup(ctx, f)
			vrTest.Cleanup(ctx, f)
			vnTest.Cleanup(ctx, f)
			meshTest.Cleanup(ctx, f)
			bgTest.Cleanup(ctx, f)
		})

		It("Create backend group scenarios", func() {

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
				bgBuilder.Namespace = namespace.Name
				vnTest.Namespace = namespace
				vrTest.Namespace = namespace
				vsTest.Namespace = namespace
				bgTest.Namespace = namespace

				oldNS := namespace.DeepCopy()
				namespace.Labels = algorithm.MergeStringMap(map[string]string{
					"appmesh.k8s.aws/sidecarInjectorWebhook": "enabled",
					"mesh":                                   meshName,
				}, namespace.Labels)

				err = f.K8sClient.Patch(ctx, namespace, client.MergeFrom(oldNS))
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

			vs2Name := fmt.Sprintf("vs-%s", utils.RandomDNS1123Label(8))
			vs2 := vsBuilder.BuildVirtualServiceNoBackend(vs2Name)

			By("Create a second namespace and add labels", func() {
				namespace, err := f.NSManager.AllocateNamespace(ctx, "appmeshtest-2")
				Expect(err).NotTo(HaveOccurred())

				// A VirtualService will be created in the new namespace
				vsTest.Namespace = namespace

				oldNS := namespace.DeepCopy()
				namespace.Labels = algorithm.MergeStringMap(map[string]string{
					"appmesh.k8s.aws/sidecarInjectorWebhook": "enabled",
					"mesh":                                   meshName,
				}, namespace.Labels)

				err = f.K8sClient.Patch(ctx, namespace, client.MergeFrom(oldNS))
				Expect(err).NotTo(HaveOccurred())
			})

			By("Creating a virtual service (with no backend) in the second namespace", func() {
				err := vsTest.Create(ctx, f, vs2)
				Expect(err).NotTo(HaveOccurred())
			})

			By("Validating the second virtual service in AWS", func() {
				err := vsTest.CheckInAWS(ctx, f, mesh, vs2)
				Expect(err).NotTo(HaveOccurred())
			})

			// Reset vsTest to original namespace
			// vsTest.Namespace = bgTest.Namespace

			vnName := fmt.Sprintf("vn-%s", utils.RandomDNS1123Label(8))
			listeners := []appmesh.Listener{vnBuilder.BuildListener("http", 8080)}
			backends := []types.NamespacedName{}
			expectedBackends := []types.NamespacedName{
				{
					Namespace: vs.Namespace,
					Name:      vs.Name,
				},
				{
					Namespace: vs2.Namespace,
					Name:      vs2.Name,
				},
			}
			vn := vnBuilder.BuildVirtualNode(vnName, backends, listeners, &appmesh.BackendDefaults{})
			bgName := fmt.Sprintf("bg-%s", utils.RandomDNS1123Label(8))
			vn.Spec.BackendGroups = []appmesh.BackendGroupReference{
				{
					Namespace: aws.String(vn.Namespace),
					Name:      bgName,
				},
			}
			expectedVN := vnBuilder.BuildVirtualNode(vnName, expectedBackends, listeners, &appmesh.BackendDefaults{})

			By("Creating a virtual node resource in k8s", func() {
				err := vnTest.Create(ctx, f, vn)
				Expect(err).NotTo(HaveOccurred())
			})

			By("validating the virtual node in AWS", func() {
				err := vnTest.CheckInAWS(ctx, f, mesh, vn)
				Expect(err).NotTo(HaveOccurred())

			})

			bg := bgBuilder.BuildBackendGroup(bgName, expectedBackends)

			By("Creating a backend group resource in k8s", func() {
				err := bgTest.Create(ctx, f, bg)
				Expect(err).NotTo(HaveOccurred())
			})

			By("validating the virtual node backends", func() {
				expectedVN.Spec.AWSName = vn.Spec.AWSName
				retryCount := 0
				err := wait.PollImmediateUntil(utils.PollIntervalShort, func() (bool, error) {
					// Expected VN contains the backends in the group
					err := vnTest.CheckInAWS(ctx, f, mesh, expectedVN)
					if err != nil {
						if retryCount >= utils.PollRetries {
							return false, err
						}
						retryCount++
						return false, nil
					}
					return true, nil
				}, ctx.Done())
				Expect(err).NotTo(HaveOccurred())
			})

			// The new VirtualService will be created after the BackendGroup.
			vsName = fmt.Sprintf("vs-%s", utils.RandomDNS1123Label(8))
			vs = vsBuilder.BuildVirtualServiceNoBackend(vsName)

			expectedBackends = []types.NamespacedName{
				{
					Namespace: vs.Namespace,
					Name:      vs.Name,
				},
			}

			vnName = fmt.Sprintf("vn-%s", utils.RandomDNS1123Label(8))
			vn = vnBuilder.BuildVirtualNode(vnName, backends, listeners, &appmesh.BackendDefaults{})

			bgName = fmt.Sprintf("bg-%s", utils.RandomDNS1123Label(8))
			bg = bgBuilder.BuildBackendGroup(bgName, expectedBackends)

			By("Creating a backend group resource in k8s", func() {
				err := bgTest.Create(ctx, f, bg)
				Expect(err).NotTo(HaveOccurred())
			})

			vn.Spec.BackendGroups = []appmesh.BackendGroupReference{
				{
					Namespace: aws.String(vn.Namespace),
					Name:      bgName,
				},
			}

			expectedVN = vnBuilder.BuildVirtualNode(vnName, expectedBackends, listeners, &appmesh.BackendDefaults{})

			By("Creating a virtual node resource in k8s", func() {
				err := f.K8sClient.Create(ctx, vn)
				vnTest.VirtualNodes[vn.Name] = vn
				Expect(err).NotTo(HaveOccurred())
			})

			By("Creating a virtual service (with no backend) resource in k8s", func() {
				err := vsTest.Create(ctx, f, vs)
				Expect(err).NotTo(HaveOccurred())
			})

			By("Validating the virtual service (with no backend) in AWS", func() {
				err := vsTest.CheckInAWS(ctx, f, mesh, vs)
				Expect(err).NotTo(HaveOccurred())
			})

			// TODO move into helper function, probably in virtualnode manager
			By("validating the virtual node backends", func() {
				expectedVN.Spec.AWSName = vn.Spec.AWSName
				retryCount := 0
				err := wait.PollImmediateUntil(utils.PollIntervalShort, func() (bool, error) {
					// Expected VN contains the backends in the group
					err := vnTest.CheckInAWS(ctx, f, mesh, expectedVN)
					if err != nil {
						if retryCount >= utils.PollRetries {
							return false, err
						}
						retryCount++
						return false, nil
					}
					return true, nil
				}, ctx.Done())
				Expect(err).NotTo(HaveOccurred())
			})

		})

		It("Update backend group scenarios", func() {

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
				bgTest.Namespace = namespace

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
		/*
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
		*/
	})
})
