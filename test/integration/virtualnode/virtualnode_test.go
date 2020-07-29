package virtualnode_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/algorithm"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/manifest"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/utils"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/integration/mesh"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/integration/virtualnode"

	//"github.com/aws/aws-sdk-go/aws"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

	Context("Virtual Node create scenaries", func() {
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
			vn := vnBuilder.BuildVirtualNode(vnName, backends, listeners)

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
			vn = vnBuilder.BuildVirtualNode(vnName, backends, listeners)

			By("Creating a virtual node resource in k8s when no mesh matches namespace", func() {
				err := vnTest.Create(ctx, f, vn)
				Expect(err).To(HaveOccurred())
			})

		})
	})
})
