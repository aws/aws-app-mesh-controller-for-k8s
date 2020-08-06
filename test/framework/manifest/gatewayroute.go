package manifest

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-sdk-go/aws"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type GRBuilder struct {
	Namespace string
}

func (b *GRBuilder) BuildGatewayRouteWithHTTP(instanceName string, vsName string, prefix string) *appmesh.GatewayRoute {
	grName := b.buildServiceName(instanceName)

	gr := &appmesh.GatewayRoute{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: b.Namespace,
			Name:      grName,
		},
		Spec: appmesh.GatewayRouteSpec{
			HTTPRoute: b.BuildHTTPRoute(prefix, vsName, b.Namespace),
		},
	}
	return gr
}

func (b *GRBuilder) BuildGRPCRoute(grpcServiceName string, vsName string, nsName string) *appmesh.GRPCGatewayRoute {
	return &appmesh.GRPCGatewayRoute{
		Match: appmesh.GRPCGatewayRouteMatch{
			ServiceName: aws.String(grpcServiceName),
		},
		Action: appmesh.GRPCGatewayRouteAction{
			Target: appmesh.GatewayRouteTarget{
				VirtualService: appmesh.GatewayRouteVirtualService{
					VirtualServiceRef: &appmesh.VirtualServiceReference{
						Namespace: aws.String(nsName),
						Name:      vsName,
					},
				},
			},
		},
	}

}

func (b *GRBuilder) BuildHTTPRoute(prefix string, vsName string, nsName string) *appmesh.HTTPGatewayRoute {
	return &appmesh.HTTPGatewayRoute{
		Match: appmesh.HTTPGatewayRouteMatch{
			Prefix: aws.String(prefix),
		},
		Action: appmesh.HTTPGatewayRouteAction{
			Target: appmesh.GatewayRouteTarget{
				VirtualService: appmesh.GatewayRouteVirtualService{
					VirtualServiceRef: &appmesh.VirtualServiceReference{
						Namespace: aws.String(nsName),
						Name:      vsName,
					},
				},
			},
		},
	}
}

func (b *GRBuilder) buildName(instanceName string) string {
	return instanceName
}

func (b *GRBuilder) buildServiceName(instanceName string) string {
	return instanceName
}
