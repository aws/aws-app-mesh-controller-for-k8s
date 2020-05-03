package services

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
	"github.com/aws/aws-sdk-go/service/servicediscovery/servicediscoveryiface"
)

type CloudMap interface {
	servicediscoveryiface.ServiceDiscoveryAPI
}

const (
	CreateServiceTimeout       = 10
	DeregisterInstanceTimeout  = 10
	GetServiceTimeout          = 10
	ListInstancesPagesTimeout  = 10
	ListNamespacesPagesTimeout = 10
	ListServicesPagesTimeout   = 10
	RegisterInstanceTimeout    = 10

	HealthStatusFailureThreshold = 2

	//AttrAwsInstanceIPV4 is a special attribute expected by CloudMap.
	//See https://github.com/aws/aws-sdk-go/blob/fd304fe4cb2ea1027e7fc7e21062beb768915fcc/service/servicediscovery/api.go#L5161
	AttrAwsInstanceIPV4         = "AWS_INSTANCE_IPV4"
	AttrAwsInstancePort         = "AWS_INSTANCE_PORT"
	AttrAwsInstanceHealthStatus = "AWS_INIT_HEALTH_STATUS"

	//AttrK8sPod is a custom attribute injected by app-mesh controller
	AttrK8sPod = "k8s.io/pod"
	//AttrK8sNamespace is a custom attribute injected by app-mesh controller
	AttrK8sNamespace = "k8s.io/namespace"
)

// NewCloudMap constructs new CloudMap implementation.
func NewCloudMap(session *session.Session) CloudMap {
	return &defaultCloudMap{
		ServiceDiscoveryAPI: servicediscovery.New(session),
	}
}

type defaultCloudMap struct {
	servicediscoveryiface.ServiceDiscoveryAPI
}
