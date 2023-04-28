package fishapp_test

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-app-mesh-controller-for-k8s/test/e2e/fishapp"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/manifest"
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("test dynamically generated symmetrical mesh", func() {
	var (
		ctx context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()

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

	Context("normal test dimensions", func() {
		var stackPrototype fishapp.DynamicStack
		var stacksPendingCleanUp []*fishapp.DynamicStack

		BeforeEach(func() {
			stackPrototype = fishapp.DynamicStack{
				IsTLSEnabled: true,
				//Please set "enable-sds" to true in controller, prior to enabling this.
				//*TODO* Rename it to include SDS in it's name once we convert file based TLS test -> file based mTLS test.
				IsmTLSEnabled:               false,
				VirtualServicesCount:        5,
				VirtualNodesCount:           5,
				RoutesCountPerVirtualRouter: 2,
				TargetsCountPerRoute:        4,
				BackendsCountPerVirtualNode: 2,
				ReplicasPerVirtualNode:      3,
				ConnectivityCheckPerURL:     400,
			}
			stacksPendingCleanUp = nil
		})

		AfterEach(func() {
			for _, stack := range stacksPendingCleanUp {
				stack.Cleanup(ctx, f)
			}
		})

		for _, sdType := range []manifest.ServiceDiscoveryType{manifest.DNSServiceDiscovery, manifest.CloudMapServiceDiscovery} {
			func(sdType manifest.ServiceDiscoveryType) {
				It(fmt.Sprintf("should behaves correctly with service discovery type %v", sdType), func() {
					stackPrototype.ServiceDiscoveryType = sdType
					stackDefault := stackPrototype

					By("deploy stack into cluster with default controller/injector", func() {
						stacksPendingCleanUp = append(stacksPendingCleanUp, &stackDefault)
						stackDefault.Deploy(ctx, f)
					})

					By("sleep 1 minute to give controller/injector a break", func() {
						time.Sleep(1 * time.Minute)
					})

					By("check stack behavior on cluster with default controller/injector", func() {
						stackDefault.Check(ctx, f)
					})

					if f.Options.ControllerImage != "" || f.Options.InjectorImage != "" {
						if f.Options.ControllerImage != "" {
							By("upgrade cluster into new controller", func() {
								f.HelmManager.UpgradeAppMeshController(f.Options.ControllerImage)
							})
						}
						if f.Options.InjectorImage != "" {
							By("upgrade cluster into new injector", func() {
								f.HelmManager.UpgradeAppMeshInjector(f.Options.InjectorImage)
							})
						}

						By("sleep 1 minute to give controller/injector a break", func() {
							time.Sleep(1 * time.Minute)
						})

						By("check stack behavior on cluster with upgraded controller/injector", func() {
							stackDefault.Check(ctx, f)
						})

						stackNew := stackPrototype
						By("deploy new stack into cluster with upgraded controller/injector", func() {
							stacksPendingCleanUp = append(stacksPendingCleanUp, &stackNew)
							stackNew.Deploy(ctx, f)
						})

						By("sleep 1 minute to give controller/injector a break", func() {
							time.Sleep(1 * time.Minute)
						})

						By("check new stack behavior on cluster with upgraded controller/injector", func() {
							stackNew.Check(ctx, f)
						})
					}
				})
			}(sdType)
		}
	})
})
