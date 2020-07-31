package manifest

import (
	"fmt"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-sdk-go/aws"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type RouteToWeightedVirtualNodes struct {
	Path            string
	WeightedTargets []WeightedVirtualNode
}

type WeightedVirtualNode struct {
	VirtualNode types.NamespacedName
	Weight      int64
}

type VRBuilder struct {
	Namespace string
	Listeners []appmesh.VirtualRouterListener
}

func (b *VRBuilder) BuildVirtualRouter(instanceName string, routes []appmesh.Route) *appmesh.VirtualRouter {
	vrName := b.buildServiceName(instanceName)

	vr := &appmesh.VirtualRouter{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: b.Namespace,
			Name:      vrName,
		},
		Spec: appmesh.VirtualRouterSpec{
			Listeners: b.Listeners,
			Routes:    routes,
		},
	}
	return vr
}

func (b *VRBuilder) BuildRoutes(routeCfgs []RouteToWeightedVirtualNodes) []appmesh.Route {
	var routes []appmesh.Route
	for index, routeCfg := range routeCfgs {
		var targets []appmesh.WeightedTarget
		for _, weightedTarget := range routeCfg.WeightedTargets {
			targets = append(targets, appmesh.WeightedTarget{
				VirtualNodeRef: &appmesh.VirtualNodeReference{
					Namespace: aws.String(weightedTarget.VirtualNode.Namespace),
					Name:      weightedTarget.VirtualNode.Name,
				},
				Weight: weightedTarget.Weight,
			})
		}
		routes = append(routes, appmesh.Route{
			Name: fmt.Sprintf("route-%d", index),
			HTTPRoute: &appmesh.HTTPRoute{
				Match: appmesh.HTTPRouteMatch{
					Prefix: routeCfg.Path,
				},
				Action: appmesh.HTTPRouteAction{
					WeightedTargets: targets,
				},
			},
		})
	}
	return routes

}

func (b *VRBuilder) BuildVirtualRouterListener(protocol appmesh.PortProtocol, port appmesh.PortNumber) appmesh.VirtualRouterListener {
	return appmesh.VirtualRouterListener{
		PortMapping: appmesh.PortMapping{
			Port:     port,
			Protocol: protocol,
		},
	}
}

func (b *VRBuilder) buildName(instanceName string) string {
	return instanceName
}

func (b *VRBuilder) buildServiceName(instanceName string) string {
	return instanceName
}
