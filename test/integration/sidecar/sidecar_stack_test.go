package sidecar

import (
	"context"

	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("sidecar features", func() {
	ctx, f := context.Background(), framework.New(framework.GlobalOptions)
	var stack *SidecarStack

	BeforeEach(func() {
		stack = newSidecarStack("sidecar-test")
	})

	AfterEach(func() {
		stack.cleanup(ctx, f)
	})

	Context("wait for sidecar to initialize", func() {
		It("should have the color annotation", func() {
			stack.createMeshAndNamespace(ctx, f)
			stack.createBackendResources(ctx, f)
			stack.createFrontendResources(ctx, f)

			// on startup frontend will annotate pod with color
			pods := &corev1.PodList{}

			if err := f.K8sClient.List(
				ctx,
				pods,
				client.InNamespace(stack.frontendDP.Namespace),
				client.MatchingLabelsSelector{
					Selector: labels.Set(stack.frontendDP.Spec.Selector.MatchLabels).AsSelector(),
				},
			); err != nil {
				Expect(err).NotTo(HaveOccurred())
			}

			for _, pod := range pods.Items {
				ann := pod.ObjectMeta.Annotations
				_, ok := ann["color"]

				Expect(ok).To(Equal(true))
			}
		})
	})
})
