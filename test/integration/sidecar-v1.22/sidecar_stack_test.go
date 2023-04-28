package sidecar_v1_22

import (
	"context"
	"time"

	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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
			stack, err = newSidecarStack_v1_22("sidecar-test-v1-22-2-0", framework.GlobalOptions.KubeConfig, 8090)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			stack.cleanup(ctx, f)
		})

		It("expect pod status to be Running", func() {
			stack.createMeshAndNamespace(ctx, f)
			stack.createFrontendResources(ctx, f)

			err := wait.Poll(5*time.Second, 300*time.Second, func() (done bool, err error) {
				pods, err := stack.k8client.CoreV1().Pods(stack.testName).List(ctx, metav1.ListOptions{})
				if err != nil {
					return false, err
				}

				for _, pod := range pods.Items {
					allReady := true

					for _, status := range pod.Status.ContainerStatuses {
						if !status.Ready {
							allReady = false
							break
						}
					}

					if allReady {
						return true, nil
					}
				}

				return false, nil
			})

			Expect(err).NotTo(HaveOccurred())
		})
	})
})
