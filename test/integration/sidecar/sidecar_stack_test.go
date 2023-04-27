package sidecar

import (
	"context"

	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("sidecar features", func() {
	var ctx context.Context
	var f *framework.Framework

	BeforeEach(func() {
		ctx = context.Background()
		f = framework.New(framework.GlobalOptions)
	})

	Context("wait for sidecar to initialize", func() {
		var stack *SidecarStack
		var err error

		BeforeEach(func() {
			stack, err = newSidecarStack("sidecar-test", framework.GlobalOptions.KubeConfig)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			stack.cleanup(ctx, f)
		})

		It("should have the color annotation", func() {
			stack.createMeshAndNamespace(ctx, f)
			stack.createBackendResources(ctx, f)
			stack.createFrontendResources(ctx, f)

			pods, err := stack.k8client.CoreV1().Pods(stack.testName).List(ctx, metav1.ListOptions{})
			Expect(err).NotTo(HaveOccurred())

			for _, pod := range pods.Items {
				ann := pod.ObjectMeta.Annotations
				color := ann["color"]

				Expect(color).To(Equal(stack.color))
			}
		})
	})
})
