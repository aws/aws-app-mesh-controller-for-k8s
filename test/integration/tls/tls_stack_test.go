package tls_test

import (
	"context"
	"fmt"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/manifest"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/integration/tls"
	. "github.com/onsi/ginkgo"
	"time"
)

var _ = Describe("tls feature test", func() {
	var (
		ctx context.Context
		f   *framework.Framework
	)

	ctx = context.Background()

	Context("tls default test dimensions", func() {
		var stackPrototype tls.TLSStack
		var stacksPendingCleanUp []*tls.TLSStack

		for _, sdType := range []manifest.ServiceDiscoveryType{manifest.DNSServiceDiscovery} {
			func(sdType manifest.ServiceDiscoveryType) {
				It(fmt.Sprintf("Should behave correctly with end-to-end TLS configuration"), func() {
					f = framework.New(framework.GlobalOptions)
					stackPrototype.ServiceDiscoveryType = sdType
					stackDefault := stackPrototype

					By("deploy tls stack into cluster", func() {
						stackDefault.DeployTLSStack(ctx, f)
					})

					By("sleep 30 seconds for Envoys to be configured", func() {
						time.Sleep(30 * time.Second)
					})

					By("check tls behavior", func() {
						stackDefault.CheckTLSBehavior(ctx, f, true)
					})

					By("clean up TLS test stack", func() {
						stackDefault.CleanupTLSStack(ctx, f)
					})
				})

				It(fmt.Sprintf("Should behave correctly without end-to-end TLS configuration"), func() {
					stackPrototype.ServiceDiscoveryType = sdType
					stackDefault := stackPrototype

					By("deploy tls stack into cluster", func() {
						stacksPendingCleanUp = append(stacksPendingCleanUp, &stackDefault)
						stackDefault.DeployPartialTLSStack(ctx, f)
					})

					By("sleep 30 seconds for Envoys to be configured", func() {
						time.Sleep(30 * time.Second)
					})

					By("check tls behavior", func() {
						stackDefault.CheckTLSBehavior(ctx, f, false)
					})

					By("clean up TLS test stack", func() {
						stackDefault.CleanupTLSStack(ctx, f)
					})
				})

				It(fmt.Sprintf("Should behave correctly when cert validation fails"), func() {
					stackPrototype.ServiceDiscoveryType = sdType
					stackDefault := stackPrototype

					By("deploy tls stack into cluster", func() {
						stacksPendingCleanUp = append(stacksPendingCleanUp, &stackDefault)
						stackDefault.DeployTLSValidationStack(ctx, f)
					})

					By("sleep 30 seconds for Envoys to be configured", func() {
						time.Sleep(30 * time.Second)
					})

					By("check tls behavior", func() {
						stackDefault.CheckTLSBehavior(ctx, f, false)
					})

					By("clean up TLS test stack", func() {
						stackDefault.CleanupTLSStack(ctx, f)
					})
				})
			}(sdType)
		}
	})
})
