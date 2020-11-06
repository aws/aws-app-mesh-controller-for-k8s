package mesh_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/utils"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/integration/mesh"

	"github.com/aws/aws-sdk-go/aws"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Mesh", func() {

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

	Context("Mesh create scenaries", func() {
		var meshTest mesh.MeshTest
		meshTest = mesh.MeshTest{
			Meshes: make(map[string]*appmesh.Mesh),
		}

		AfterEach(func() {
			meshTest.Cleanup(ctx, f)
		})

		It("should create a mesh in AWS", func() {

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

			for _, mesh := range meshTest.Meshes {
				By("validating the resources in AWS", func() {
					err := meshTest.CheckInAWS(ctx, f, mesh)
					Expect(err).NotTo(HaveOccurred())

				})
			}
		})
	})
	Context("Mesh update scenaries", func() {
		var meshTest mesh.MeshTest
		meshTest = mesh.MeshTest{
			Meshes: make(map[string]*appmesh.Mesh),
		}

		AfterEach(func() {
			meshTest.Cleanup(ctx, f)
		})
		It("should update egress filter for a mesh in AWS", func() {

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
					EgressFilter: &appmesh.EgressFilter{
						Type: appmesh.EgressFilterTypeDropAll},
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

			By("updating the egress filter to ALLOW_ALL and validating the change", func() {
				oldMesh := meshTest.Meshes[mesh.Name].DeepCopy()
				meshTest.Meshes[mesh.Name].Spec.EgressFilter = &appmesh.EgressFilter{
					Type: appmesh.EgressFilterTypeAllowAll}

				err := meshTest.Update(ctx, f, meshTest.Meshes[mesh.Name], oldMesh)
				Expect(err).NotTo(HaveOccurred())

				err = meshTest.CheckInAWS(ctx, f, meshTest.Meshes[mesh.Name])
				Expect(err).NotTo(HaveOccurred())
			})

			By("updating the egress filter to DROP_ALL and validating the change", func() {
				oldMesh := meshTest.Meshes[mesh.Name].DeepCopy()
				meshTest.Meshes[mesh.Name].Spec.EgressFilter = &appmesh.EgressFilter{
					Type: appmesh.EgressFilterTypeDropAll}

				err := meshTest.Update(ctx, f, meshTest.Meshes[mesh.Name], oldMesh)
				Expect(err).NotTo(HaveOccurred())

				err = meshTest.CheckInAWS(ctx, f, meshTest.Meshes[mesh.Name])
				Expect(err).NotTo(HaveOccurred())
			})

			By("updating AWSName and validating it cannot be updated", func() {
				oldMesh := meshTest.Meshes[mesh.Name].DeepCopy()
				meshTest.Meshes[mesh.Name].Spec.AWSName = aws.String("testMesh")

				err := meshTest.Update(ctx, f, meshTest.Meshes[mesh.Name], oldMesh)
				Expect(err).To(HaveOccurred())
			})

			By("updating the egress filter with an invalid value", func() {
				oldMesh := meshTest.Meshes[mesh.Name].DeepCopy()
				meshTest.Meshes[mesh.Name].Spec.EgressFilter = &appmesh.EgressFilter{
					Type: "DENY"}

				err := meshTest.Update(ctx, f, meshTest.Meshes[mesh.Name], oldMesh)
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
