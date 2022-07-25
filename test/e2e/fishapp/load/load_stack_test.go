package load

import (
	"context"
	"fmt"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/e2e/fishapp"
	"time"

	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/manifest"
	. "github.com/onsi/ginkgo"
	"go.uber.org/zap"
)

func initParams() (fishapp.DynamicStack, []*fishapp.DynamicStack) {
	var stackPrototype fishapp.DynamicStack
	var stacksPendingCleanUp []*fishapp.DynamicStack

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

	return stackPrototype, stacksPendingCleanUp
}

func upgradeControllerInjector(stackDefault fishapp.DynamicStack, stackPrototype fishapp.DynamicStack, stacksPendingCleanUp []*fishapp.DynamicStack, ctx context.Context, f *framework.Framework) {
	if f.Options.ControllerImage != "" || f.Options.InjectorImage != "" {
		if f.Options.ControllerImage != "" {
			fmt.Sprintf("upgrade cluster into new controller")
			f.HelmManager.UpgradeAppMeshController(f.Options.ControllerImage)
		}
		if f.Options.InjectorImage != "" {
			fmt.Sprintf("upgrade cluster into new injector")
			f.HelmManager.UpgradeAppMeshInjector(f.Options.InjectorImage)
		}

		fmt.Sprintf("sleep 1 minute to give controller/injector a break")
		time.Sleep(1 * time.Minute)

		fmt.Sprintf("check stack behavior on cluster with upgraded controller/injector")
		stackDefault.Check(ctx, f)

		stackNew := stackPrototype
		fmt.Sprintf("deploy new stack into cluster with upgraded controller/injector")
		stacksPendingCleanUp = append(stacksPendingCleanUp, &stackNew)
		stackNew.Deploy(ctx, f)

		fmt.Sprintf("sleep 1 minute to give controller/injector a break")
		time.Sleep(1 * time.Minute)

		fmt.Sprintf("check new stack behavior on cluster with upgraded controller/injector")
		stackNew.Check(ctx, f)
	}
}

func spinUpResources(stackPrototype fishapp.DynamicStack, stacksPendingCleanUp []*fishapp.DynamicStack, sdType manifest.ServiceDiscoveryType, checkConnectivity bool, ctx context.Context, f *framework.Framework) {
	//for _, sdType := range []manifest.ServiceDiscoveryType{manifest.DNSServiceDiscovery, manifest.CloudMapServiceDiscovery} {
	fmt.Sprintf("Service discovery type -: %v", sdType)
	stackPrototype.ServiceDiscoveryType = sdType
	stackDefault := stackPrototype

	fmt.Sprintf("deploy stack into cluster with default controller/injector")
	stacksPendingCleanUp = append(stacksPendingCleanUp, &stackDefault)
	stackDefault.Deploy(ctx, f)

	fmt.Sprintf("sleep 1 minute to give controller/injector a break")
	time.Sleep(1 * time.Minute)

	//fmt.Sprintf("stackDefault -:")
	if checkConnectivity {
		fmt.Sprintf("Checking stack behavior/connectivity on cluster with default controller/injector")
		stackDefault.Check(ctx, f)
	}

	fmt.Sprintf("Entering upgradeControllerInjector")
	upgradeControllerInjector(stackDefault, stackPrototype, stacksPendingCleanUp, ctx, f)
}

func spinDownResources(stacksPendingCleanUp []*fishapp.DynamicStack, ctx context.Context, f *framework.Framework) {
	for _, stack := range stacksPendingCleanUp {
		stack.Cleanup(ctx, f)
	}
}

var _ = Describe("Running Load Test Driver", func() {
	var ctx context.Context
	var f *framework.Framework
	Context("normal test dimensions", func() {
		var stackPrototype fishapp.DynamicStack
		var stacksPendingCleanUp []*fishapp.DynamicStack

		BeforeEach(func() {
			ctx = context.Background()
			f = framework.New(framework.GlobalOptions)
			stackPrototype, stacksPendingCleanUp = initParams()

			//stackPrototype = fishapp.DynamicStack{
			//	IsTLSEnabled: true,
			//	//Please set "enable-sds" to true in controller, prior to enabling this.
			//	//*TODO* Rename it to include SDS in it's name once we convert file based TLS test -> file based mTLS test.
			//	IsmTLSEnabled:               false,
			//	VirtualServicesCount:        5,
			//	VirtualNodesCount:           5,
			//	RoutesCountPerVirtualRouter: 2,
			//	TargetsCountPerRoute:        4,
			//	BackendsCountPerVirtualNode: 2,
			//	ReplicasPerVirtualNode:      3,
			//	ConnectivityCheckPerURL:     400,
			//}
			//stacksPendingCleanUp = nil
		})

		f.Logger.Info("stackPrototype Values -: ", zap.Int("VirtualServicesCount", stackPrototype.VirtualServicesCount), zap.Int("VirtualNodesCount", stackPrototype.VirtualNodesCount))
		sdType := manifest.DNSServiceDiscovery

		It(fmt.Sprintf("should behaves correctly with service discovery type %v", sdType), func() {

			checkConnectivity := false
			spinUpResources(stackPrototype, stacksPendingCleanUp, sdType, checkConnectivity, ctx, f)
			fmt.Sprintf("should behaves correctly with service discovery type %v", stackPrototype.ServiceDiscoveryType)
			//sleep_duration := 6 * time.Hour
			//fmt.Printf("Sleeping %d mins", sleep_duration)
			//time.Sleep(sleep_duration)

			// Run Python load driver script
			//err := exec.Command("python3", "/Users/eavishal/appmesh/AppMeshLoadTester/scripts/load_driver.py").Run()
			//fmt.Sprintf("Ran Load Driver with Error -: %v", err)
			f.Logger.Info("Spinning down resources")
			spinDownResources(stacksPendingCleanUp, ctx, f)
		})

	})
})
