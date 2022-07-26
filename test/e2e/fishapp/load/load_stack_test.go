package load

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/manifest"
	. "github.com/onsi/ginkgo"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"time"
)

var basePath string
var configPath string
var loadDriverPath string

func init() {
	flag.StringVar(&basePath, "base-path", "/Users/eavishal/appmesh/AppMeshLoadTester/", "Load Driver base path")
	configPath = filepath.Join(basePath, "config.json")
	loadDriverPath = filepath.Join(basePath, "scripts", "load_driver.py")
}

type Config struct {
	IsTLSEnabled                bool                `json:"isTLSEnabled"`
	IsmTLSEnabled               bool                `json:"ismTLSEnabled"`
	RoutesCountPerVirtualRouter int                 `json:"RoutesCountPerVirtualRouter"`
	BackendsCountPerVirtualNode int                 `json:"BackendsCountPerVirtualNode"`
	ReplicasPerVirtualNode      int32               `json:"ReplicasPerVirtualNode"`
	ConnectivityCheckPerURL     int                 `json:"ConnectivityCheckPerURL"`
	Backends_map                map[string][]string `json:"backends_map"`
	Load_tests                  []map[string]string `json:"load_tests"`
	Metrics                     map[string]string   `json:"metrics"`
}

func readDriverConfig(configPath string) Config {
	var config Config
	file, fileErr := ioutil.ReadFile(configPath)
	if fileErr != nil {
		panic(fileErr)
	}
	if jsonErr := json.Unmarshal([]byte(file), &config); jsonErr != nil {
		panic(jsonErr)
	}

	return config
}

func createResourcesStack(config Config) (DynamicStack, []*DynamicStack) {
	var stackPrototype DynamicStack
	var stacksPendingCleanUp []*DynamicStack

	stackPrototype = DynamicStack{
		IsTLSEnabled: config.IsTLSEnabled,
		//Please set "enable-sds" to true in controller, prior to enabling this.
		//*TODO* Rename it to include SDS in it's name once we convert file based TLS test -> file based mTLS test.
		IsmTLSEnabled:               config.IsmTLSEnabled,
		VirtualServicesCount:        2,
		VirtualNodesCount:           2,
		RoutesCountPerVirtualRouter: config.RoutesCountPerVirtualRouter,
		TargetsCountPerRoute:        1,
		BackendsCountPerVirtualNode: config.BackendsCountPerVirtualNode,
		ReplicasPerVirtualNode:      config.ReplicasPerVirtualNode,
		ConnectivityCheckPerURL:     config.ConnectivityCheckPerURL,
	}
	stacksPendingCleanUp = nil

	return stackPrototype, stacksPendingCleanUp
}

func upgradeControllerInjector(stackDefault DynamicStack, stackPrototype DynamicStack, stacksPendingCleanUp []*DynamicStack, ctx context.Context, f *framework.Framework) []*DynamicStack {
	if f.Options.ControllerImage != "" || f.Options.InjectorImage != "" {
		if f.Options.ControllerImage != "" {
			fmt.Println(fmt.Sprintf("upgrade cluster into new controller"))
			f.HelmManager.UpgradeAppMeshController(f.Options.ControllerImage)
		}
		if f.Options.InjectorImage != "" {
			fmt.Println(fmt.Sprintf("upgrade cluster into new injector"))
			f.HelmManager.UpgradeAppMeshInjector(f.Options.InjectorImage)
		}

		fmt.Println(fmt.Sprintf("sleep 1 minute to give controller/injector a break"))
		time.Sleep(1 * time.Minute)

		fmt.Println(fmt.Sprintf("check stack behavior on cluster with upgraded controller/injector"))
		stackDefault.Check(ctx, f)

		stackNew := stackPrototype
		fmt.Println(fmt.Sprintf("deploy new stack into cluster with upgraded controller/injector"))
		stacksPendingCleanUp = append(stacksPendingCleanUp, &stackNew)
		stackNew.Deploy(ctx, f, basePath)

		fmt.Println(fmt.Sprintf("sleep 1 minute to give controller/injector a break"))
		time.Sleep(1 * time.Minute)

		fmt.Println(fmt.Sprintf("check new stack behavior on cluster with upgraded controller/injector"))
		stackNew.Check(ctx, f)
	}
	return stacksPendingCleanUp
}

func spinUpResources(stackPrototype DynamicStack, stacksPendingCleanUp []*DynamicStack, sdType manifest.ServiceDiscoveryType, checkConnectivity bool, ctx context.Context, f *framework.Framework) []*DynamicStack {
	//for _, sdType := range []manifest.ServiceDiscoveryType{manifest.DNSServiceDiscovery, manifest.CloudMapServiceDiscovery} {
	fmt.Println(fmt.Sprintf("Service discovery type -: %v", sdType))
	stackPrototype.ServiceDiscoveryType = sdType
	stackDefault := stackPrototype

	fmt.Println("deploy stack into cluster with default controller/injector")
	stacksPendingCleanUp = append(stacksPendingCleanUp, &stackDefault)
	stackDefault.Deploy(ctx, f, basePath)

	fmt.Println("sleep 1 minute to give controller/injector a break")
	time.Sleep(1 * time.Minute)

	if checkConnectivity {
		fmt.Println(fmt.Sprintf("Checking stack behavior/connectivity on cluster with default controller/injector"))
		stackDefault.Check(ctx, f)
	}

	fmt.Println("Entering upgradeControllerInjector")
	stacksPendingCleanUp = upgradeControllerInjector(stackDefault, stackPrototype, stacksPendingCleanUp, ctx, f)

	return stacksPendingCleanUp
}

func spinDownResources(stacksPendingCleanUp []*DynamicStack, ctx context.Context, f *framework.Framework) {
	fmt.Println("Length of stacksPendingCleanUp = ", len(stacksPendingCleanUp))
	for _, stack := range stacksPendingCleanUp {
		stack.Cleanup(ctx, f)
	}
}

var _ = Describe("Running Load Test Driver", func() {
	var ctx context.Context
	var f *framework.Framework

	Context("normal test dimensions", func() {
		var stackPrototype DynamicStack
		var stacksPendingCleanUp []*DynamicStack

		fmt.Println("Reading config from -: ", configPath)
		config := readDriverConfig(configPath)

		BeforeEach(func() {
			ctx = context.Background()
			f = framework.New(framework.GlobalOptions)
			stackPrototype, stacksPendingCleanUp = createResourcesStack(config)
		})

		sdType := manifest.DNSServiceDiscovery

		It(fmt.Sprintf("should behaves correctly with service discovery type %v", sdType), func() {

			checkConnectivity := false
			stacksPendingCleanUp = spinUpResources(stackPrototype, stacksPendingCleanUp, sdType, checkConnectivity, ctx, f)

			// Run Python load driver script
			fmt.Printf("Running Fortio Load Driver from %s\n", loadDriverPath)
			if err := exec.Command("python3", loadDriverPath, configPath, basePath).Run(); err != nil {
				fmt.Printf("Load Driver Failed with error -: %v\n", err)
			}

			fmt.Println("Fortio Load Driver success")
			f.Logger.Info("Spinning down resources")
			spinDownResources(stacksPendingCleanUp, ctx, f)
		})

	})
})
