package load

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/manifest"
	"github.com/aws/aws-sdk-go/aws"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

var basePath string
var configPath string
var loadDriverPath string

func init() {
	flag.StringVar(&basePath, "base-path", "", "Load Driver base path")
}

type Config struct {
	IsTLSEnabled            bool                `json:"isTLSEnabled"`
	IsmTLSEnabled           bool                `json:"ismTLSEnabled"`
	ReplicasPerVirtualNode  int32               `json:"ReplicasPerVirtualNode"`
	ConnectivityCheckPerURL int                 `json:"ConnectivityCheckPerURL"`
	Backends_map            map[string][]string `json:"backends_map"`
	Load_tests              []map[string]string `json:"load_tests"`
	Metrics                 map[string]string   `json:"metrics"`
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

func getNodeNamesFromConfig(backendsMap map[string][]string) []string {
	var nodeMap = make(map[string]bool)
	for node, backends := range backendsMap {
		if _, ok := nodeMap[node]; !ok {
			nodeMap[node] = true
		}
		for _, backendNode := range backends {
			if _, ok := nodeMap[backendNode]; !ok {
				nodeMap[backendNode] = true
			}
		}
	}
	nodeNames := make([]string, len(nodeMap))
	i := 0
	for k := range nodeMap {
		nodeNames[i] = k
		i++
	}
	return nodeNames
}

func createResourcesStack(config Config) (DynamicStackNew, []*DynamicStackNew) {
	var stackPrototype DynamicStackNew
	var stacksPendingCleanUp []*DynamicStackNew

	nodeNames := getNodeNamesFromConfig(config.Backends_map)

	stackPrototype = DynamicStackNew{
		IsTLSEnabled: config.IsTLSEnabled,
		//Please set "enable-sds" to true in controller, prior to enabling this.
		//*TODO* Rename it to include SDS in it's name once we convert file based TLS test -> file based mTLS test.
		IsmTLSEnabled:           config.IsmTLSEnabled,
		InputNodeNamesList:      nodeNames,
		InputNodeBackendsMap:    config.Backends_map,
		ReplicasPerVirtualNode:  config.ReplicasPerVirtualNode,
		ConnectivityCheckPerURL: config.ConnectivityCheckPerURL,
	}
	stacksPendingCleanUp = nil

	return stackPrototype, stacksPendingCleanUp
}

func upgradeControllerInjector(stackDefault DynamicStackNew, stackPrototype DynamicStackNew, stacksPendingCleanUp []*DynamicStackNew, ctx context.Context, f *framework.Framework) []*DynamicStackNew {
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

		//fmt.Println(fmt.Sprintf("check stack behavior on cluster with upgraded controller/injector"))
		//stackDefault.Check(ctx, f)

		stackNew := stackPrototype
		fmt.Println(fmt.Sprintf("deploy new stack into cluster with upgraded controller/injector"))
		stacksPendingCleanUp = append(stacksPendingCleanUp, &stackNew)
		stackNew.Deploy(ctx, f, basePath)

		fmt.Println(fmt.Sprintf("sleep 1 minute to give controller/injector a break"))
		time.Sleep(1 * time.Minute)

		//fmt.Println(fmt.Sprintf("check new stack behavior on cluster with upgraded controller/injector"))
		//stackNew.Check(ctx, f)
	}
	return stacksPendingCleanUp
}

func spinUpResources(stackPrototype DynamicStackNew, stacksPendingCleanUp []*DynamicStackNew, sdType manifest.ServiceDiscoveryType, checkConnectivity bool, ctx context.Context, f *framework.Framework) []*DynamicStackNew {
	fmt.Println(fmt.Sprintf("Service discovery type -: %v", sdType))
	stackPrototype.ServiceDiscoveryType = sdType
	stackDefault := stackPrototype

	fmt.Println("deploy stack into cluster with default controller/injector")
	stacksPendingCleanUp = append(stacksPendingCleanUp, &stackDefault)
	stackDefault.Deploy(ctx, f, basePath)

	fmt.Println("sleep 1 minute to give controller/injector a break")
	time.Sleep(1 * time.Minute)

	//if checkConnectivity {
	//	fmt.Println(fmt.Sprintf("Checking stack behavior/connectivity on cluster with default controller/injector"))
	//	stackDefault.Check(ctx, f)
	//}

	stacksPendingCleanUp = upgradeControllerInjector(stackDefault, stackPrototype, stacksPendingCleanUp, ctx, f)

	return stacksPendingCleanUp
}

func spinDownResources(stacksPendingCleanUp []*DynamicStackNew, ctx context.Context, f *framework.Framework) {
	fmt.Println("Spinning down resources")
	fmt.Println("Length of stacksPendingCleanUp = ", len(stacksPendingCleanUp))
	for _, stack := range stacksPendingCleanUp {
		stack.Cleanup(ctx, f)
	}
}

func deployFortio(ctx context.Context, f *framework.Framework, stacksPendingCleanUp []*DynamicStackNew) *exec.Cmd {
	By("Applying Fortio components using kubectl", func() {
		fortioCmd := exec.Command("kubectl", "apply", "-f", filepath.Join(basePath, "fortio.yaml"))
		fmt.Printf("Running kubectl apply -: %s\n", fortioCmd.String())
		var out bytes.Buffer
		var stderr bytes.Buffer
		fortioCmd.Stdout = &out
		fortioCmd.Stderr = &stderr
		fortioErr := fortioCmd.Run()
		if fortioErr != nil {
			fmt.Printf("Fortio kubectl failed with error -: %v -: %s\n", fortioErr, stderr.String())
		} else {
			fmt.Println("Successfully applied Fortio components using kubectl -: \n", out.String())
		}
		Expect(fortioErr).NotTo(HaveOccurred())
	})

	fortio_vn, _ := stacksPendingCleanUp[0].waitUntilFortioComponentsActive(ctx, f, stacksPendingCleanUp[0].namespace.Name, "fortio")
	fmt.Println("All Fortio Components Active")

	var portForwardCmd *exec.Cmd
	By("Port-forwarding Fortio service to local", func() {
		portForwardCmd = exec.Command("kubectl", fmt.Sprintf("--namespace"), stacksPendingCleanUp[0].namespace.Name, "port-forward", "service/fortio", "9091:8080")
		var out bytes.Buffer
		var stderr bytes.Buffer
		portForwardCmd.Stdout = &out
		portForwardCmd.Stderr = &stderr

		fmt.Printf("Running port-forwarding command -: %s\n", portForwardCmd.String())
		portForwardErr := portForwardCmd.Start()
		if portForwardErr != nil {
			fmt.Printf("Fortio port-forwarding failed with error -: %v -: %s\n", portForwardErr, stderr.String())
		} else {
			fmt.Println(out.String())
		}
		Expect(portForwardErr).NotTo(HaveOccurred())
	})

	By("Adding all VirtualServices as backends of Fortio", func() {
		var vnBackends []appmesh.Backend
		for _, vs := range stacksPendingCleanUp[0].createdServiceVSs {
			fmt.Printf("Granting backend access for Fortio to service -: %s\n", vs.Name)
			vnBackends = append(vnBackends, appmesh.Backend{
				VirtualService: appmesh.VirtualServiceBackend{
					VirtualServiceRef: &appmesh.VirtualServiceReference{
						Namespace: aws.String(vs.Namespace),
						Name:      vs.Name,
					},
				},
			})
		}
		vnNew := fortio_vn.DeepCopy()
		fmt.Printf("Number of backends for Fortio = %d\n", len(vnBackends))
		vnNew.Spec.Backends = vnBackends
		//if s.IsmTLSEnabled {
		//	vnNew.Spec.BackendDefaults.ClientPolicy.TLS.Validation.SubjectAlternativeNames.Match.Exact = backendSANs
		//}

		err := f.K8sClient.Patch(ctx, vnNew, client.MergeFrom(fortio_vn))
		Expect(err).NotTo(HaveOccurred())
		stacksPendingCleanUp[0].createdNodeVNs["fortio"] = vnNew
	})

	return portForwardCmd
}

var _ = Describe("Running Load Test Driver", func() {
	var ctx context.Context
	var f *framework.Framework

	Context("normal test dimensions", func() {
		var stackPrototype DynamicStackNew
		var stacksPendingCleanUp []*DynamicStackNew

		BeforeEach(func() {
			ctx = context.Background()
			f = framework.New(framework.GlobalOptions)

			// Ginkgo flag parsing happens in BeforeEach or It blocks only
			configPath = filepath.Join(basePath, "config.json")
			loadDriverPath = filepath.Join(basePath, "scripts", "load_driver.py")
			fmt.Println("Reading config from -: ", configPath)
			config := readDriverConfig(configPath)
			stackPrototype, stacksPendingCleanUp = createResourcesStack(config)
		})

		sdType := manifest.DNSServiceDiscovery

		It(fmt.Sprintf("should behaves correctly with service discovery type %v", sdType), func() {

			checkConnectivity := false
			stacksPendingCleanUp = spinUpResources(stackPrototype, stacksPendingCleanUp, sdType, checkConnectivity, ctx, f)
			defer spinDownResources(stacksPendingCleanUp, ctx, f)

			portForwardCmd := deployFortio(ctx, f, stacksPendingCleanUp)
			defer func() {
				fmt.Println("killing Fortio port-forward process")
				if portForwardKillErr := portForwardCmd.Process.Kill(); portForwardKillErr != nil {
					fmt.Println("failed to kill process. Error -: ", portForwardKillErr, "Continuing...")
				}
			}()

			time.Sleep(2 * time.Minute)

			// Run Fortio load driver script
			By(fmt.Sprintf("Running Fortio Load Driver from %s\n", loadDriverPath), func() {
				loadDriverCmd := exec.Command("python3", loadDriverPath, configPath, basePath)
				fmt.Printf("Load Driver Command -: %s\n", loadDriverCmd)
				var stderr bytes.Buffer
				var stdout bytes.Buffer
				loadDriverCmd.Stdout = &stdout
				loadDriverCmd.Stderr = &stderr
				err := loadDriverCmd.Run()
				if err != nil {
					fmt.Printf("Load Driver Failed with error -: \n%v -: %s\n", err, stderr.String())
				} else {
					fmt.Println("Fortio Load Driver success -: \n", stdout.String())
				}
				Expect(err).NotTo(HaveOccurred())
			})
		})

	})
})
