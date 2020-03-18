package framework

import (
	meshclientset "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/client/clientset/versioned"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/e2e/framework/helm"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/e2e/framework/resource/deployment"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/e2e/framework/resource/mesh"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/e2e/framework/resource/namespace"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/e2e/framework/resource/virtualnode"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/e2e/framework/resource/virtualservice"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/e2e/framework/utils"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
	"github.com/aws/aws-sdk-go/service/servicediscovery/servicediscoveryiface"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Framework struct {
	Options       Options
	RestCfg       *rest.Config
	K8sClient     kubernetes.Interface
	K8sMeshClient meshclientset.Interface

	NSManager   namespace.Manager
	DPManager   deployment.Manager
	MeshManager mesh.Manager
	VNManager   virtualnode.Manager
	VSManager   virtualservice.Manager
	HelmManager helm.Manager

	SDClient servicediscoveryiface.ServiceDiscoveryAPI
	Logger   *zap.Logger
}

func New(options Options) *Framework {
	err := options.Validate()
	Expect(err).NotTo(HaveOccurred())

	restCfg, err := buildRestConfig(options)
	Expect(err).NotTo(HaveOccurred())

	k8sClient, err := kubernetes.NewForConfig(restCfg)
	Expect(err).NotTo(HaveOccurred())

	k8sMeshClient, err := meshclientset.NewForConfig(restCfg)
	Expect(err).NotTo(HaveOccurred())

	sess := session.Must(session.NewSession(aws.NewConfig().WithRegion(options.AWSRegion)))
	sdClient := servicediscovery.New(sess)
	f := &Framework{
		Options:       options,
		RestCfg:       restCfg,
		K8sClient:     k8sClient,
		K8sMeshClient: k8sMeshClient,

		HelmManager: helm.NewManager(options.KubeConfig),
		NSManager:   namespace.NewManager(k8sClient),
		DPManager:   deployment.NewManager(k8sClient),
		MeshManager: mesh.NewManager(k8sMeshClient),
		VNManager:   virtualnode.NewManager(k8sMeshClient),
		VSManager:   virtualservice.NewManager(k8sMeshClient),

		SDClient: sdClient,
		Logger:   utils.NewGinkgoLogger(),
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
