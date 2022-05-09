package sidecar_test

import (
	"context"
	"fmt"

	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework"
	sidecar "github.com/aws/aws-app-mesh-controller-for-k8s/test/integration/sidecar"
	. "github.com/onsi/ginkgo"
)

var _ = Describe("sidecar feature test", func() {
	var (
		ctx context.Context
		f   *framework.Framework
	)

	BeforeEach(func() {
		ctx = context.Background()
		f = framework.New(framework.GlobalOptions)
	})

	Context("sidecar default test dimensions", func() {
		var stacksPendingCleanUp []*sidecar.SidecarStack

		AfterEach(func() {
			for _, stack := range stacksPendingCleanUp {
				stack.CleanupSidecarStack(ctx, f)
			}
		})

		It(fmt.Sprintf("Should wait for sidecar to initialize"), func() {
			stackDefault := sidecar.SidecarStack{}

			By("deploy sidecar stack", func() {
				stacksPendingCleanUp = append(stacksPendingCleanUp, &stackDefault)
				stackDefault.DeploySidecarStack(ctx, f)
			})

			By("check sidecar behavior", func() {
				stackDefault.CheckSidecarBehavior(ctx, f)
			})
		})
	})
})
