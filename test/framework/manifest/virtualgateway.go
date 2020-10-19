package manifest

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-sdk-go/aws"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type VGBuilder struct {
	Namespace string
}

func (b *VGBuilder) BuildVirtualGateway(instanceName string, listeners []appmesh.VirtualGatewayListener, nsSelector map[string]string) *appmesh.VirtualGateway {
	vgName := b.buildServiceName(instanceName)

	vg := &appmesh.VirtualGateway{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: b.Namespace,
			Name:      vgName,
		},
		Spec: appmesh.VirtualGatewaySpec{
			NamespaceSelector: &metav1.LabelSelector{
				MatchLabels: nsSelector,
			},
			Listeners: listeners,
			Logging: &appmesh.VirtualGatewayLogging{
				AccessLog: &appmesh.VirtualGatewayAccessLog{
					File: &appmesh.VirtualGatewayFileAccessLog{
						Path: "/",
					},
				},
			},
			MeshRef: nil,
		},
	}
	return vg
}

func (b *VGBuilder) BuildVGListener(protocol appmesh.VirtualGatewayPortProtocol, port appmesh.PortNumber, healthCheckPath string) appmesh.VirtualGatewayListener {
	return appmesh.VirtualGatewayListener{
		PortMapping: appmesh.VirtualGatewayPortMapping{
			Port:     port,
			Protocol: protocol,
		},
		HealthCheck: b.BuildHealthCheckPolicy(healthCheckPath, protocol, port),
	}
}

func (b *VGBuilder) BuildListenerWithConnectionPools(protocol appmesh.VirtualGatewayPortProtocol, port appmesh.PortNumber, http *appmesh.HTTPConnectionPool,
	http2 *appmesh.HTTP2ConnectionPool, grpc *appmesh.GRPCConnectionPool) appmesh.VirtualGatewayListener {

	vgConnectionPool := &appmesh.VirtualGatewayConnectionPool{}

	if http != nil {
		vgConnectionPool.HTTP = http
	}
	if http2 != nil {
		vgConnectionPool.HTTP2 = http2
	}
	if grpc != nil {
		vgConnectionPool.GRPC = grpc
	}

	return appmesh.VirtualGatewayListener{
		PortMapping: appmesh.VirtualGatewayPortMapping{
			Port:     port,
			Protocol: protocol,
		},
		ConnectionPool: vgConnectionPool,
	}
}

func (b *VGBuilder) BuildHealthCheckPolicy(path string, protocol appmesh.VirtualGatewayPortProtocol, port appmesh.PortNumber) *appmesh.VirtualGatewayHealthCheckPolicy {
	return &appmesh.VirtualGatewayHealthCheckPolicy{
		HealthyThreshold:   3,
		IntervalMillis:     6000,
		Path:               aws.String(path),
		Port:               &port,
		Protocol:           protocol,
		TimeoutMillis:      3000,
		UnhealthyThreshold: 2,
	}
}

func (b *VGBuilder) buildName(instanceName string) string {
	return instanceName
}

func (b *VGBuilder) buildServiceName(instanceName string) string {
	return instanceName
}
