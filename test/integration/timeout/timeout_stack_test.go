package timeout_test

import (
	"context"
	"fmt"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/manifest"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/integration/timeout"
	. "github.com/onsi/ginkgo/v2"
	"time"
)

var _ = Describe("timeout feature test", func() {
	var (
		ctx context.Context
		f   *framework.Framework
	)

	BeforeEach(func() {
		ctx = context.Background()
		f = framework.New(framework.GlobalOptions)
	})

	Context("timeout default test dimensions", func() {
		var stackPrototype timeout.TimeoutStack
		var stacksPendingCleanUp []*timeout.TimeoutStack

		BeforeEach(func() {
			stackPrototype = timeout.TimeoutStack{
				TimeoutValue: 45,
			}
			stacksPendingCleanUp = nil
		})

		AfterEach(func() {
			for _, stack := range stacksPendingCleanUp {
				stack.CleanupTimeoutStack(ctx, f)
			}
		})

		for _, sdType := range []manifest.ServiceDiscoveryType{manifest.DNSServiceDiscovery} {
			func(sdType manifest.ServiceDiscoveryType) {
				It(fmt.Sprintf("Should behave correctly with and without timeout configured"), func() {
					stackPrototype.ServiceDiscoveryType = sdType
					stackDefault := stackPrototype

					By("deploy timeout stack into cluster", func() {
						stacksPendingCleanUp = append(stacksPendingCleanUp, &stackDefault)
						stackDefault.DeployTimeoutStack(ctx, f)
					})

					By("sleep 30 seconds for Envoys to be configured", func() {
						time.Sleep(30 * time.Second)
					})

					By("check timeout behavior", func() {
						stackDefault.CheckTimeoutBehavior(ctx, f)
					})
				})
			}(sdType)
		}
	})
})
