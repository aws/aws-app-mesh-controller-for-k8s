package controller

import (
	"testing"

	appmeshv1beta1 "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/apis/appmesh/v1beta1"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws"
	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/appmesh"
)

// newAWSVirtualService is a helper function to generate an Kubernetes Custom Resource API object.
func newAPIVirtualService(meshName string, virtualRouterName string, routes []appmeshv1beta1.Route) appmeshv1beta1.VirtualService {
	vs := appmeshv1beta1.VirtualService{
		Spec: appmeshv1beta1.VirtualServiceSpec{
			MeshName: meshName,
		},
	}

	if virtualRouterName != "" {
		vs.Spec.VirtualRouter = &appmeshv1beta1.VirtualRouter{
			Name: virtualRouterName,
		}
	}

	if routes != nil && len(routes) > 0 {
		vs.Spec.Routes = routes
	}
	return vs
}

func newAPIHttpRoute(routeName string, prefix string, targets []appmeshv1beta1.WeightedTarget) appmeshv1beta1.Route {
	return appmeshv1beta1.Route{
		Http: appmeshv1beta1.HttpRoute{
			Action: appmeshv1beta1.HttpRouteAction{
				WeightedTargets: targets,
			},
			Match: appmeshv1beta1.HttpRouteMatch{
				Prefix: prefix,
			},
		},
		Name: routeName,
	}
}

// newAWSVirtualService is a helper function to generate an App Mesh API object.
func newAWSVirtualService(virtualRouterName string) aws.VirtualService {
	awsVs := aws.VirtualService{
		Data: appmesh.VirtualServiceData{
			Spec: &appmesh.VirtualServiceSpec{},
		},
	}
	if virtualRouterName != "" {
		awsVs.Data.Spec.Provider = &appmesh.VirtualServiceProvider{
			VirtualRouter: &appmesh.VirtualRouterServiceProvider{
				VirtualRouterName: awssdk.String(virtualRouterName),
			},
		}
	}
	return awsVs
}

// newAWSHttpRoute is a helper function to generate an App Mesh API object.
func newAWSHttpRoute(routeName string, prefix string, targets []appmeshv1beta1.WeightedTarget) aws.Route {
	awsRoute := aws.Route{
		Data: appmesh.RouteData{
			Spec: &appmesh.RouteSpec{
				HttpRoute: &appmesh.HttpRoute{
					Action: &appmesh.HttpRouteAction{},
					Match: &appmesh.HttpRouteMatch{
						Prefix: awssdk.String(prefix),
					},
				},
			},
			RouteName: awssdk.String(routeName),
		},
	}

	if targets != nil && len(targets) > 0 {
		var awstargets []*appmesh.WeightedTarget
		for _, t := range targets {
			awstargets = append(awstargets, &appmesh.WeightedTarget{
				VirtualNode: awssdk.String(t.VirtualNodeName),
				Weight:      awssdk.Int64(t.Weight),
			})
		}
		awsRoute.Data.Spec.HttpRoute.Action.WeightedTargets = awstargets
	}
	return awsRoute
}

func TestVServiceNeedsUpdate(t *testing.T) {

	var (
		// defaults
		defaultMeshName   = "example-mesh"
		defaultRouterName = "example-router"
		defaultRouteName  = "example-route"
		defaultPrefix     = "/"
		defaultTargets    = []appmeshv1beta1.WeightedTarget{}

		// Spec with default values
		defaultServiceSpec = newAPIVirtualService(defaultMeshName, defaultRouterName, []appmeshv1beta1.Route{newAPIHttpRoute(defaultRouteName, defaultPrefix, defaultTargets)})

		// result with the same values as spec1_default
		defaultServiceResult              = newAWSVirtualService(defaultRouterName)
		serviceResult_differentRouterName = newAWSVirtualService("router2")
	)

	var vservicetests = []struct {
		name        string
		spec        appmeshv1beta1.VirtualService
		aws         aws.VirtualService
		needsUpdate bool
	}{
		{"vservices are the same", defaultServiceSpec, defaultServiceResult, false},
		{"result has different router name", defaultServiceSpec, serviceResult_differentRouterName, true},
	}

	for _, tt := range vservicetests {
		t.Run(tt.name, func(t *testing.T) {
			if res := vserviceNeedsUpdate(&tt.spec, &tt.aws); res != tt.needsUpdate {
				t.Errorf("got %v, want %v", res, tt.needsUpdate)
			}
		})
	}
}

func TestRouteNeedUpdate(t *testing.T) {

	var (
		// defaults
		defaultRouteName = "example-route"
		defaultPrefix    = "/"
		defaultNodeName  = "example-node"
		defaultTargets   = []appmeshv1beta1.WeightedTarget{
			{Weight: int64(1), VirtualNodeName: defaultNodeName},
		}
		defaultNamespace = "dummyNamespace"

		awsExtraTarget = []appmeshv1beta1.WeightedTarget{
			{Weight: int64(1), VirtualNodeName: defaultNodeName + "-" + defaultNamespace},
			{Weight: int64(1), VirtualNodeName: "extra-node"},
		}

		awsdefaultTargets = []appmeshv1beta1.WeightedTarget{
			{Weight: int64(1), VirtualNodeName: defaultNodeName + "-" + defaultNamespace},
		}

		// Spec with default values
		defaultRouteSpec = newAPIHttpRoute(defaultRouteName, defaultPrefix, defaultTargets)

		// result with the same values as defaultRouteSpec
		defaultRouteResult = newAWSHttpRoute(defaultRouteName, defaultPrefix, awsdefaultTargets)

		extraTargetResult     = newAWSHttpRoute(defaultRouteName, defaultPrefix, awsExtraTarget)
		extraTargetSpec       = newAPIHttpRoute(defaultRouteName, defaultPrefix, awsExtraTarget)
		noTargetsResult       = newAWSHttpRoute(defaultRouteName, defaultPrefix, []appmeshv1beta1.WeightedTarget{})
		noTargetSpec          = newAPIHttpRoute(defaultRouteName, defaultPrefix, []appmeshv1beta1.WeightedTarget{})
		differentPrefixResult = newAWSHttpRoute(defaultRouteName, "/foo", defaultTargets)

		crdTargets   = []appmeshv1beta1.WeightedTarget{
			{Weight: int64(1), VirtualNodeName: "foo.bar"},
			{Weight: int64(2), VirtualNodeName: "foo.bar.zoo"},
			{Weight: int64(3), VirtualNodeName: "foo"},
		}
		crdRouteSpec = newAPIHttpRoute(defaultRouteName, defaultPrefix, crdTargets)

		awsTargets   = []appmeshv1beta1.WeightedTarget{
			{Weight: int64(1), VirtualNodeName: "foo-bar"},
			{Weight: int64(2), VirtualNodeName: "foo-bar-zoo"},
			{Weight: int64(3), VirtualNodeName: "foo-dummyNamespace"},
		}
		awsRouteResult = newAWSHttpRoute(defaultRouteName, defaultPrefix, awsTargets)

	)

	var routetests = []struct {
		name        string
		spec        appmeshv1beta1.Route
		routes      aws.Route
		needsUpdate bool
	}{
		{"routes are the same", defaultRouteSpec, defaultRouteResult, false},
		{"extra weighted target in result", defaultRouteSpec, extraTargetResult, true},
		{"extra weighted target in spec", extraTargetSpec, defaultRouteResult, true},
		{"no targets in result", extraTargetSpec, noTargetsResult, true},
		{"no targets in spec", noTargetSpec, defaultRouteResult, true},
		{"different prefix", defaultRouteSpec, differentPrefixResult, true},
		{"routes with different weights are the same", crdRouteSpec, awsRouteResult, false},
	}

	for _, tt := range routetests {
		t.Run(tt.name, func(t *testing.T) {
			if res := routeNeedsUpdate(tt.spec, tt.routes); res != tt.needsUpdate {
				t.Errorf("got %v, want %v", res, tt.needsUpdate)
			}
		})
	}
}
