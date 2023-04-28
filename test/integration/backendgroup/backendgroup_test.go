package backendgroup_test

import (
	"context"
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/algorithm"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework"
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

			bg := bgBuilder.BuildBackendGroup(bgName, expectedBackends)

			By("Creating a backend group resource in k8s", func() {
				err := bgTest.Create(ctx, f, bg)
				Expect(err).NotTo(HaveOccurred())
			})

			By("Creating a virtual node resource in k8s", func() {
				err := vnTest.Create(ctx, f, vn)
				Expect(err).NotTo(HaveOccurred())
			})

			By("validating the virtual node backends", func() {
				err := vnTest.ValidateBackends(ctx, f, mesh, vn, expectedBackends)
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

			By("validating the virtual node backends", func() {
				err := vnTest.ValidateBackends(ctx, f, mesh, vn, expectedBackends)
				Expect(err).NotTo(HaveOccurred())
			})

		})

		It("Wildcard backend group scenarios", func() {
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

			vnName := fmt.Sprintf("vn-%s", utils.RandomDNS1123Label(8))
			listeners := []appmesh.Listener{vnBuilder.BuildListener("http", 8080)}
			backends := []types.NamespacedName{}
			expectedBackends := []types.NamespacedName{
				{
					Namespace: vs.Namespace,
					Name:      vs.Name,
				},
			}
			vn := vnBuilder.BuildVirtualNode(vnName, backends, listeners, &appmesh.BackendDefaults{})
			vn.Spec.BackendGroups = []appmesh.BackendGroupReference{
				{
					Namespace: aws.String(vn.Namespace),
					Name:      "*",
				},
			}

			By("Creating a virtualnode with a wildcard backend group", func() {
				err := vnTest.Create(ctx, f, vn)
				Expect(err).NotTo(HaveOccurred())
			})

			By("validating the virtual node backends", func() {
				err := vnTest.ValidateBackends(ctx, f, mesh, vn, expectedBackends)
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

			By("Creating a second virtual service", func() {
				err := vsTest.Create(ctx, f, vs2)
				Expect(err).NotTo(HaveOccurred())
			})

			By("Validating the second virtual service in AWS", func() {
				err := vsTest.CheckInAWS(ctx, f, mesh, vs2)
				Expect(err).NotTo(HaveOccurred())
			})

			vnName := fmt.Sprintf("vn-%s", utils.RandomDNS1123Label(8))
			listeners := []appmesh.Listener{vnBuilder.BuildListener("http", 8080)}
			backends := []types.NamespacedName{}

			// start with one backend in the group, then update the group to include a second one
			expectedBackends := []types.NamespacedName{
				{
					Namespace: vs.Namespace,
					Name:      vs.Name,
				},
			}
			vn := vnBuilder.BuildVirtualNode(vnName, backends, listeners, &appmesh.BackendDefaults{})
			bgName := fmt.Sprintf("bg-%s", utils.RandomDNS1123Label(8))
			vn.Spec.BackendGroups = []appmesh.BackendGroupReference{
				{
					Namespace: aws.String(vs.Namespace),
					Name:      bgName,
				},
			}

			bg := bgBuilder.BuildBackendGroup(bgName, expectedBackends)

			By("Creating a backend group resource in k8s", func() {
				err := bgTest.Create(ctx, f, bg)
				Expect(err).NotTo(HaveOccurred())
			})

			By("Creating a virtual node resource in k8s", func() {
				err := vnTest.Create(ctx, f, vn)
				Expect(err).NotTo(HaveOccurred())
			})

			By("validating the virtual node backends", func() {
				err := vnTest.ValidateBackends(ctx, f, mesh, vn, expectedBackends)
				Expect(err).NotTo(HaveOccurred())
			})

			expectedBackends = append(expectedBackends, types.NamespacedName{
				Namespace: vs2.Namespace,
				Name:      vs2.Name,
			})

			oldBG := bgTest.BackendGroups[bg.Name].DeepCopy()
			bgTest.BackendGroups[bg.Name].Spec.VirtualServices = append(
				bgTest.BackendGroups[bg.Name].Spec.VirtualServices,
				appmesh.VirtualServiceReference{
					Namespace: &vs2.Namespace,
					Name:      vs2.Name,
				})

			By("Updating the backend group", func() {
				err := bgTest.Update(ctx, f, bgTest.BackendGroups[bg.Name], oldBG)
				Expect(err).NotTo(HaveOccurred())
			})

			By("validating the virtual node backends", func() {
				err := vnTest.ValidateBackends(ctx, f, mesh, vn, expectedBackends)
				Expect(err).NotTo(HaveOccurred())
			})

		})
	})
})
