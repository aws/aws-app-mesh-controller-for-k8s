package framework

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws/services"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws/throttle"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/helm"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/resource/deployment"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/resource/gatewayroute"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/resource/mesh"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/resource/namespace"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/resource/virtualgateway"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/resource/virtualnode"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/resource/virtualrouter"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/resource/virtualservice"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/utils"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Framework struct {
	Options   Options
	RestCfg   *rest.Config
	K8sClient client.Client

	NSManager   namespace.Manager
	DPManager   deployment.Manager
	MeshManager mesh.Manager
	VNManager   virtualnode.Manager
	VSManager   virtualservice.Manager
	VRManager   virtualrouter.Manager
	VGManager   virtualgateway.Manager
	GRManager   gatewayroute.Manager
	HelmManager helm.Manager

	CloudMapClient services.CloudMap
	Logger         *zap.Logger
	StopChan       <-chan struct{}
}

func New(options Options) *Framework {
	err := options.Validate()
	Expect(err).NotTo(HaveOccurred())

	restCfg, err := buildRestConfig(options)
	Expect(err).NotTo(HaveOccurred())

	k8sSchema := runtime.NewScheme()
	clientgoscheme.AddToScheme(k8sSchema)
	appmesh.AddToScheme(k8sSchema)

	signalCtx := ctrl.SetupSignalHandler()
	cache, err := cache.New(restCfg, cache.Options{Scheme: k8sSchema})
	Expect(err).NotTo(HaveOccurred())
	go func() {
		cache.Start(signalCtx)
	}()
	cache.WaitForCacheSync(signalCtx)
	realClient, err := client.New(restCfg, client.Options{Scheme: k8sSchema})
	Expect(err).NotTo(HaveOccurred())
	k8sClient, err := client.NewDelegatingClient(client.NewDelegatingClientInput{
		CacheReader: cache,
		Client:      realClient,
	})
	Expect(err).NotTo(HaveOccurred())

	cloud, err := aws.NewCloud(aws.CloudConfig{
		Region:         options.AWSRegion,
		ThrottleConfig: throttle.NewDefaultServiceOperationsThrottleConfig(),
	}, nil)
	Expect(err).NotTo(HaveOccurred())

	f := &Framework{
		Options:   options,
		RestCfg:   restCfg,
		K8sClient: k8sClient,

		HelmManager:    helm.NewManager(options.KubeConfig),
		NSManager:      namespace.NewManager(k8sClient),
		DPManager:      deployment.NewManager(k8sClient),
		MeshManager:    mesh.NewManager(k8sClient, cloud.AppMesh()),
		VNManager:      virtualnode.NewManager(k8sClient, cloud.AppMesh(), cloud.CloudMap()),
		VSManager:      virtualservice.NewManager(k8sClient, cloud.AppMesh()),
		VRManager:      virtualrouter.NewManager(k8sClient, cloud.AppMesh()),
		VGManager:      virtualgateway.NewManager(k8sClient, cloud.AppMesh()),
		GRManager:      gatewayroute.NewManager(k8sClient, cloud.AppMesh()),
		CloudMapClient: cloud.CloudMap(),
		Logger:         utils.NewGinkgoLogger(),
		StopChan:       signalCtx.Done(),
	}
	return f
}

func buildRestConfig(options Options) (*rest.Config, error) {
	restCfg, err := clientcmd.BuildConfigFromFlags("", options.KubeConfig)
	if err != nil {
		return nil, err
	}
	restCfg.QPS = 20
	restCfg.Burst = 50
	return restCfg, nil
}
