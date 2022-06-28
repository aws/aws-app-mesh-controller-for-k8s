package sidecar_v1_22

import (
	"context"
	"time"

	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
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
			stack, err = newSidecarStack_v1_22("sidecar-test", framework.GlobalOptions.KubeConfig, 8090)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			stack.cleanup(ctx, f)
		})

		It("expect pod status to be Running", func() {
			stack.createMeshAndNamespace(ctx, f)
			stack.createFrontendResources(ctx, f)

			err := wait.Poll(1*time.Second, 30*time.Second, func() (done bool, err error) {
				pods, err := stack.k8client.CoreV1().Pods(stack.testName).List(ctx, metav1.ListOptions{})
				if err != nil {
					return false, err
				}

				for _, pod := range pods.Items {
					if name, ok := pod.ObjectMeta.Labels["app"]; ok && name == "front" && pod.Status.Phase == corev1.PodRunning {
						return true, nil
					}
				}

				return false, nil
			})

			Expect(err).NotTo(HaveOccurred())
		})
	})
})
