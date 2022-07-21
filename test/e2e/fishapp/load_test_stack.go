package fishapp

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/manifest"
)

func initParams() (DynamicStack, []*DynamicStack) {
	var stackPrototype DynamicStack
	var stacksPendingCleanUp []*DynamicStack

	stackPrototype = DynamicStack{
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

func upgradeControllerInjector(stackDefault DynamicStack, stackPrototype DynamicStack, stacksPendingCleanUp []*DynamicStack, ctx context.Context, f *framework.Framework) {
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

func spinUpResources(stackPrototype DynamicStack, stacksPendingCleanUp []*DynamicStack, sdType manifest.ServiceDiscoveryType, checkConnectivity bool, ctx context.Context, f *framework.Framework) {
	//for _, sdType := range []manifest.ServiceDiscoveryType{manifest.DNSServiceDiscovery, manifest.CloudMapServiceDiscovery} {
	fmt.Sprintf("Service discovery type -: %v", sdType)
	stackPrototype.ServiceDiscoveryType = sdType
	stackDefault := stackPrototype

	fmt.Sprintf("deploy stack into cluster with default controller/injector")
	stacksPendingCleanUp = append(stacksPendingCleanUp, &stackDefault)
	stackDefault.Deploy(ctx, f)

	fmt.Sprintf("sleep 1 minute to give controller/injector a break")
	time.Sleep(1 * time.Minute)

	if checkConnectivity {
		fmt.Sprintf("Checking stack behavior/connectivity on cluster with default controller/injector")
		stackDefault.Check(ctx, f)
	}

	fmt.Sprintf("Entering upgradeControllerInjector")
	upgradeControllerInjector(stackDefault, stackPrototype, stacksPendingCleanUp, ctx, f)
}

func spinDownResources(stacksPendingCleanUp []*DynamicStack, ctx context.Context, f *framework.Framework) {
	for _, stack := range stacksPendingCleanUp {
		stack.Cleanup(ctx, f)
	}
}

func main() {
	var ctx context.Context
	f := framework.New(framework.GlobalOptions)
	var stackPrototype DynamicStack
	var stacksPendingCleanUp []*DynamicStack
	stackPrototype, stacksPendingCleanUp = initParams()

	sdType := manifest.DNSServiceDiscovery
	checkConnectivity := true
	spinUpResources(stackPrototype, stacksPendingCleanUp, sdType, checkConnectivity, ctx, f)

	//sleep_duration := 6 * time.Hour
	//fmt.Printf("Sleeping %d mins", sleep_duration)
	//time.Sleep(sleep_duration)

	// Run Python load driver script
	//err := exec.Command("python3", "/Users/eavishal/appmesh/AppMeshLoadTester/scripts/load_driver.py").Run()
	//fmt.Sprintf("Ran Load Driver with Error -: %v", err)
	fmt.Sprintf("Spinning down resources")
	spinDownResources(stacksPendingCleanUp, ctx, f)
}
