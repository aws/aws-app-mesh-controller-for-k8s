package mesh_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/k8s"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/utils"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/integration/mesh"

	"github.com/aws/aws-sdk-go/aws"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"time"
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

	Context("Mesh create scenarios", func() {
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

		It("should show errors if the mesh cannot be created in AWS", func() {
			meshName := fmt.Sprintf("mesh-%s", utils.RandomDNS1123Label(8))
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
					AWSName: aws.String(fmt.Sprintf("mesh-%s", utils.RandomDNS1123Label(256))),
				},
			}

			By("Creating a mesh resource in k8s with a name exceeding the character limit", func() {
				// Not using meshTest.Create as it hangs
				err := f.K8sClient.Create(ctx, mesh)
				// sometimes there's a delay in the resource showing up
				observedMesh := &appmesh.Mesh{}
				for i := 0; i < 5; i++ {
					if err := f.K8sClient.Get(ctx, k8s.NamespacedName(mesh), observedMesh); err != nil {
						if i >= 5 {
							Expect(err).NotTo(HaveOccurred())
						}
					}
					time.Sleep(100 * time.Millisecond)
				}
				meshTest.Meshes[mesh.Name] = mesh
				Expect(err).NotTo(HaveOccurred())
			})

			By("Check mesh in AWS - it should not exist", func() {
				err := meshTest.CheckInAWS(ctx, f, mesh)
				Expect(err).To(HaveOccurred())
			})

			By("checking events for the BadRequestException", func() {
				clientset, err := kubernetes.NewForConfig(f.RestCfg)
				Expect(err).NotTo(HaveOccurred())
				events, err := clientset.CoreV1().Events(mesh.Namespace).List(ctx, metav1.ListOptions{
					FieldSelector: fmt.Sprintf("involvedObject.name=%s", mesh.Name),
					TypeMeta: metav1.TypeMeta{
						Kind: "Pod",
					},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(events.Items).NotTo(BeEmpty())
			})
		})

		It("should create a mesh with MeshDiscoverySpec", func() {

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
					ServiceDiscovery: &appmesh.MeshServiceDiscovery{
						IpPreference: aws.String(appmesh.IpPreferenceIPv4),
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

	Context("Mesh update scenarios", func() {
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
		It("should update Ip Preference for a mesh in AWS", func() {

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
					ServiceDiscovery: &appmesh.MeshServiceDiscovery{
						IpPreference: aws.String(appmesh.IpPreferenceIPv4),
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

			By("updating the ipPreference and validating the change", func() {
				oldMesh := meshTest.Meshes[mesh.Name].DeepCopy()
				meshTest.Meshes[mesh.Name].Spec.ServiceDiscovery = &appmesh.MeshServiceDiscovery{
					IpPreference: aws.String(appmesh.IpPreferenceIPv6),
				}
				err := meshTest.Update(ctx, f, meshTest.Meshes[mesh.Name], oldMesh)

				if f.Options.IpFamily == utils.IPv6 {
					Expect(meshTest.Meshes[mesh.Name].Spec.ServiceDiscovery.IpPreference).To(Equal(aws.String(appmesh.IpPreferenceIPv6)))
				} else {
					Expect(meshTest.Meshes[mesh.Name].Spec.ServiceDiscovery.IpPreference).To(Equal(aws.String(appmesh.IpPreferenceIPv4)))
				}

				Expect(err).NotTo(HaveOccurred())

				err = meshTest.CheckInAWS(ctx, f, meshTest.Meshes[mesh.Name])
				Expect(err).NotTo(HaveOccurred())
			})

			By("updating to not have Service Discovery for mesh and validating the change", func() {
				oldMesh := meshTest.Meshes[mesh.Name].DeepCopy()
				meshTest.Meshes[mesh.Name].Spec.ServiceDiscovery = nil
				err := meshTest.Update(ctx, f, meshTest.Meshes[mesh.Name], oldMesh)

				if f.Options.IpFamily == utils.IPv6 {
					Expect(meshTest.Meshes[mesh.Name].Spec.ServiceDiscovery.IpPreference).To(Equal(aws.String(appmesh.IpPreferenceIPv6)))
				} else {
					Expect(meshTest.Meshes[mesh.Name].Spec.ServiceDiscovery).To(BeNil())
				}

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

		})
	})
})
