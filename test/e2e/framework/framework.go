package framework

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/e2e/framework/helm"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/e2e/framework/resource/deployment"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/e2e/framework/resource/mesh"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/e2e/framework/resource/namespace"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/e2e/framework/resource/virtualnode"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/e2e/framework/resource/virtualrouter"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/e2e/framework/resource/virtualservice"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/e2e/framework/utils"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
	"github.com/aws/aws-sdk-go/service/servicediscovery/servicediscoveryiface"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
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
	HelmManager helm.Manager

	SDClient servicediscoveryiface.ServiceDiscoveryAPI
	Logger   *zap.Logger
}

func New(options Options) *Framework {
	err := options.Validate()
	Expect(err).NotTo(HaveOccurred())

	restCfg, err := buildRestConfig(options)
	Expect(err).NotTo(HaveOccurred())

	k8sSchema := runtime.NewScheme()
	clientgoscheme.AddToScheme(k8sSchema)
	appmesh.AddToScheme(k8sSchema)
	k8sClient, err := client.New(restCfg, client.Options{Scheme: k8sSchema})
	Expect(err).NotTo(HaveOccurred())

	sess := session.Must(session.NewSession(aws.NewConfig().WithRegion(options.AWSRegion)))
	sdClient := servicediscovery.New(sess)
	f := &Framework{
		Options:   options,
		RestCfg:   restCfg,
		K8sClient: k8sClient,

		HelmManager: helm.NewManager(options.KubeConfig),
		NSManager:   namespace.NewManager(k8sClient),
		DPManager:   deployment.NewManager(k8sClient),
		MeshManager: mesh.NewManager(k8sClient),
		VNManager:   virtualnode.NewManager(k8sClient),
		VSManager:   virtualservice.NewManager(k8sClient),
		VRManager:   virtualrouter.NewManager(k8sClient),
		SDClient:    sdClient,
		Logger:      utils.NewGinkgoLogger(),
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
