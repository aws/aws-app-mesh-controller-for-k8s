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

func (b *VGBuilder) BuildVGListener(protocol appmesh.VirtualGatewayPortProtocol, port appmesh.PortNumber) appmesh.VirtualGatewayListener {
	return appmesh.VirtualGatewayListener{
		PortMapping: appmesh.VirtualGatewayPortMapping{
			Port:     port,
			Protocol: protocol,
		},
		HealthCheck: &appmesh.VirtualGatewayHealthCheckPolicy{
			HealthyThreshold:   3,
			IntervalMillis:     6000,
			Path:               aws.String("/"),
			Port:               &port,
			Protocol:           protocol,
			TimeoutMillis:      3000,
			UnhealthyThreshold: 2,
		},
	}
}

func (b *VGBuilder) buildName(instanceName string) string {
	return instanceName
}

func (b *VGBuilder) buildServiceName(instanceName string) string {
	return instanceName
}
