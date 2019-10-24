package controller

import (
	"fmt"
	"reflect"
	"testing"

	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/appmesh"
	"github.com/stretchr/testify/mock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	appmeshv1beta1 "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/apis/appmesh/v1beta1"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws"
	ctrlawsmocks "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws/mocks"
	appmeshv1beta1mocks "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/client/clientset/versioned/mocks"
	appmeshv1beta1typedmocks "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/client/clientset/versioned/typed/appmesh/v1beta1/mocks"
)

// newAWSVirtualService is a helper function to generate an Kubernetes Custom Resource API object.
func newAPIVirtualService(meshName string, virtualRouter *appmeshv1beta1.VirtualRouter, routes []appmeshv1beta1.Route) appmeshv1beta1.VirtualService {
	vs := appmeshv1beta1.VirtualService{
		Spec: appmeshv1beta1.VirtualServiceSpec{
			MeshName:      meshName,
			VirtualRouter: virtualRouter,
			Routes:        routes,
		},
	}

	return vs
}

func newAPIHttpRoute(routeName string, prefix string, targets []appmeshv1beta1.WeightedTarget) appmeshv1beta1.Route {
	return appmeshv1beta1.Route{
		Http: &appmeshv1beta1.HttpRoute{
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

func newAPIHttp2Route(routeName string, prefix string, targets []appmeshv1beta1.WeightedTarget) appmeshv1beta1.Route {
	return appmeshv1beta1.Route{
		Http2: &appmeshv1beta1.HttpRoute{
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

func newAPIGrpcRoute(routeName string, serviceName string, methodName string, targets []appmeshv1beta1.WeightedTarget) appmeshv1beta1.Route {
	return appmeshv1beta1.Route{
		Grpc: &appmeshv1beta1.GrpcRoute{
			Action: appmeshv1beta1.GrpcRouteAction{
				WeightedTargets: targets,
			},
			Match: appmeshv1beta1.GrpcRouteMatch{
				ServiceName: &serviceName,
				MethodName:  &methodName,
			},
		},
		Name: routeName,
	}
}

func newAPITcpRoute(routeName string, targets []appmeshv1beta1.WeightedTarget) appmeshv1beta1.Route {
	return appmeshv1beta1.Route{
		Tcp: &appmeshv1beta1.TcpRoute{
			Action: appmeshv1beta1.TcpRouteAction{
				WeightedTargets: targets,
			},
		},
		Name: routeName,
	}
}

// newAWSVirtualService is a helper function to generate an App Mesh API object.
func newAWSVirtualService(virtualRouter *appmesh.VirtualRouterData) aws.VirtualService {
	awsVs := aws.VirtualService{
		Data: appmesh.VirtualServiceData{
			Spec: &appmesh.VirtualServiceSpec{},
		},
	}
	if virtualRouter != nil {
		awsVs.Data.Spec.Provider = &appmesh.VirtualServiceProvider{
			VirtualRouter: &appmesh.VirtualRouterServiceProvider{
				VirtualRouterName: virtualRouter.VirtualRouterName,
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
		awsRoute.Data.Spec.HttpRoute.Action.WeightedTargets = newAWSWeightedTargets(targets)
	}
	return awsRoute
}

func newAWSHttp2Route(routeName string, prefix string, targets []appmeshv1beta1.WeightedTarget) aws.Route {
	awsRoute := aws.Route{
		Data: appmesh.RouteData{
			Spec: &appmesh.RouteSpec{
				Http2Route: &appmesh.HttpRoute{
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
		awsRoute.Data.Spec.Http2Route.Action.WeightedTargets = newAWSWeightedTargets(targets)
	}
	return awsRoute
}

func newAWSGrpcRoute(routeName string, serviceName string, methodName string, targets []appmeshv1beta1.WeightedTarget) aws.Route {
	awsRoute := aws.Route{
		Data: appmesh.RouteData{
			Spec: &appmesh.RouteSpec{
				GrpcRoute: &appmesh.GrpcRoute{
					Action: &appmesh.GrpcRouteAction{},
					Match: &appmesh.GrpcRouteMatch{
						ServiceName: awssdk.String(serviceName),
						MethodName:  awssdk.String(methodName),
					},
				},
			},
			RouteName: awssdk.String(routeName),
		},
	}

	if targets != nil && len(targets) > 0 {
		awsRoute.Data.Spec.GrpcRoute.Action.WeightedTargets = newAWSWeightedTargets(targets)
	}
	return awsRoute
}

func newAWSWeightedTargets(targets []appmeshv1beta1.WeightedTarget) []*appmesh.WeightedTarget {
	var awstargets []*appmesh.WeightedTarget
	for _, t := range targets {
		awstargets = append(awstargets, &appmesh.WeightedTarget{
			VirtualNode: awssdk.String(t.VirtualNodeName),
			Weight:      awssdk.Int64(t.Weight),
		})
	}
	return awstargets
}

func newAWSTcpRoute(routeName string, targets []appmeshv1beta1.WeightedTarget) aws.Route {
	awsRoute := aws.Route{
		Data: appmesh.RouteData{
			Spec: &appmesh.RouteSpec{
				TcpRoute: &appmesh.TcpRoute{
					Action: &appmesh.TcpRouteAction{},
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
		awsRoute.Data.Spec.TcpRoute.Action.WeightedTargets = awstargets
	}
	return awsRoute
}

func TestVServiceNeedsUpdate(t *testing.T) {

	var (
		// defaults
		defaultMeshName = "example-mesh"
		defaultRouter   = &appmeshv1beta1.VirtualRouter{
			Name: "example-router",
		}
		defaultRouteName = "example-route"
		defaultPrefix    = "/"
		defaultTargets   = []appmeshv1beta1.WeightedTarget{}

		// Spec with default values
		defaultServiceSpec = newAPIVirtualService(defaultMeshName,
			defaultRouter,
			[]appmeshv1beta1.Route{
				newAPIHttpRoute(defaultRouteName, defaultPrefix, defaultTargets),
			},
		)

		// result with the same values as spec1_default
		defaultServiceResult = newAWSVirtualService(&appmesh.VirtualRouterData{
			MeshName:          awssdk.String(defaultMeshName),
			VirtualRouterName: awssdk.String(defaultRouter.Name),
			Spec:              &appmesh.VirtualRouterSpec{},
		})

		serviceResultDifferentRouterName = newAWSVirtualService(&appmesh.VirtualRouterData{
			MeshName:          awssdk.String(defaultMeshName),
			VirtualRouterName: awssdk.String(defaultRouter.Name + "-2"),
			Spec:              &appmesh.VirtualRouterSpec{},
		})
	)

	var vservicetests = []struct {
		name        string
		spec        appmeshv1beta1.VirtualService
		aws         aws.VirtualService
		needsUpdate bool
	}{
		{"vservices are the same", defaultServiceSpec, defaultServiceResult, false},
		{"result has different router name", defaultServiceSpec, serviceResultDifferentRouterName, true},
	}

	for _, tt := range vservicetests {
		t.Run(tt.name, func(t *testing.T) {
			if res := vserviceNeedsUpdate(&tt.spec, &tt.aws); res != tt.needsUpdate {
				t.Errorf("got %v, want %v", res, tt.needsUpdate)
			}
		})
	}
}

func TestVirtualRouterNeedsUpdate(t *testing.T) {

	var (
		// defaults
		defaultRouter = &appmeshv1beta1.VirtualRouter{
			Name: "example-router",
			Listeners: []appmeshv1beta1.VirtualRouterListener{
				appmeshv1beta1.VirtualRouterListener{
					PortMapping: appmeshv1beta1.PortMapping{
						Port:     9080,
						Protocol: "http",
					},
				},
			},
		}

		// result with the same values as spec1_default
		defaultResult = &aws.VirtualRouter{
			Data: appmesh.VirtualRouterData{
				VirtualRouterName: awssdk.String(defaultRouter.Name),
				Spec: &appmesh.VirtualRouterSpec{
					Listeners: []*appmesh.VirtualRouterListener{
						&appmesh.VirtualRouterListener{
							PortMapping: &appmesh.PortMapping{
								Port:     awssdk.Int64(9080),
								Protocol: awssdk.String(appmesh.PortProtocolHttp),
							},
						},
					},
				},
			},
		}

		resultDifferentRouterName = &aws.VirtualRouter{
			Data: appmesh.VirtualRouterData{
				VirtualRouterName: awssdk.String(defaultRouter.Name + "2"),
				Spec: &appmesh.VirtualRouterSpec{
					Listeners: []*appmesh.VirtualRouterListener{
						&appmesh.VirtualRouterListener{
							PortMapping: &appmesh.PortMapping{
								Port:     awssdk.Int64(9080),
								Protocol: awssdk.String(appmesh.PortProtocolHttp),
							},
						},
					},
				},
			},
		}

		resultDifferentRouterListener = &aws.VirtualRouter{
			Data: appmesh.VirtualRouterData{
				VirtualRouterName: awssdk.String(defaultRouter.Name),
				Spec: &appmesh.VirtualRouterSpec{
					Listeners: []*appmesh.VirtualRouterListener{
						&appmesh.VirtualRouterListener{
							PortMapping: &appmesh.PortMapping{
								Port:     awssdk.Int64(9999),
								Protocol: awssdk.String(appmesh.PortProtocolHttp),
							},
						},
					},
				},
			},
		}
	)

	var tests = []struct {
		name        string
		spec        *appmeshv1beta1.VirtualRouter
		aws         *aws.VirtualRouter
		needsUpdate bool
	}{
		{"virtual-routers are the same", defaultRouter, defaultResult, false},
		{"virtual-routers has different router name", defaultRouter, resultDifferentRouterName, true},
		{"virtual-routers has different listener", defaultRouter, resultDifferentRouterListener, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if res := vrouterNeedsUpdate(tt.spec, tt.aws); res != tt.needsUpdate {
				t.Errorf("got %v, want %v", res, tt.needsUpdate)
			}
		})
	}
}

func TestHttpRouteNeedUpdate(t *testing.T) {

	var (
		// shared defaults
		defaultRouteName = "example-route"
		defaultPrefix    = "/"
		defaultNodeName  = "example-node"

		// Targets for default custom resource spec
		defaultTargets = []appmeshv1beta1.WeightedTarget{
			{Weight: int64(1), VirtualNodeName: defaultNodeName},
		}

		// Spec with default values
		defaultSpec = newAPIHttpRoute(defaultRouteName, defaultPrefix, defaultTargets)

		// Result with the equivalent values as defaultSpec
		defaultRouteResult = newAWSHttpRoute(defaultRouteName, defaultPrefix, defaultTargets)

		differentTargetResult = newAWSHttpRoute(defaultRouteName, defaultPrefix, []appmeshv1beta1.WeightedTarget{
			{Weight: int64(1), VirtualNodeName: "diff-node"},
		})

		extraTarget = []appmeshv1beta1.WeightedTarget{
			{Weight: int64(1), VirtualNodeName: defaultNodeName},
			{Weight: int64(1), VirtualNodeName: "extra-node"},
		}

		// Extra target spec and result
		extraTargetSpec   = newAPIHttpRoute(defaultRouteName, defaultPrefix, extraTarget)
		extraTargetResult = newAWSHttpRoute(defaultRouteName, defaultPrefix, extraTarget)

		// No targets spec and result
		noTargetSpec    = newAPIHttpRoute(defaultRouteName, defaultPrefix, []appmeshv1beta1.WeightedTarget{})
		noTargetsResult = newAWSHttpRoute(defaultRouteName, defaultPrefix, []appmeshv1beta1.WeightedTarget{})

		// Default result with different prefix match
		differentPrefixResult = newAWSHttpRoute(defaultRouteName, "/foo", defaultTargets)

		// Varying weight targets spec and result
		varyingWeightTargets = []appmeshv1beta1.WeightedTarget{
			{Weight: int64(1), VirtualNodeName: "foo-bar"},
			{Weight: int64(2), VirtualNodeName: "foo-bar-zoo"},
			{Weight: int64(3), VirtualNodeName: "foo-dummyNamespace"},
		}
		varyingWeightSpec = newAPIHttpRoute(defaultRouteName, defaultPrefix, varyingWeightTargets)

		varyingWeightResult = newAWSHttpRoute(defaultRouteName, defaultPrefix, varyingWeightTargets)
	)

	var routetests = []struct {
		name        string
		spec        appmeshv1beta1.Route
		routes      aws.Route
		needsUpdate bool
	}{
		{"routes are the same", defaultSpec, defaultRouteResult, false},
		{"different weighted target in result", defaultSpec, differentTargetResult, true},
		{"extra weighted target in result", defaultSpec, extraTargetResult, true},
		{"extra weighted target in spec", extraTargetSpec, defaultRouteResult, true},
		{"no targets in result", extraTargetSpec, noTargetsResult, true},
		{"no targets in spec", noTargetSpec, defaultRouteResult, true},
		{"different prefix", defaultSpec, differentPrefixResult, true},
		{"routes with varying weights are the same", varyingWeightSpec, varyingWeightResult, false},
	}

	for _, tt := range routetests {
		t.Run(tt.name, func(t *testing.T) {
			if res := routeNeedsUpdate(tt.spec, tt.routes); res != tt.needsUpdate {
				t.Errorf("got %v, want %v", res, tt.needsUpdate)
			}
		})
	}
}

func TestTcpRouteNeedUpdate(t *testing.T) {

	var (
		// shared defaults
		defaultRouteName = "example-tcp-route"
		defaultNodeName  = "example-node"

		// Targets for default custom resource spec
		defaultTargets = []appmeshv1beta1.WeightedTarget{
			{Weight: int64(1), VirtualNodeName: defaultNodeName},
		}

		// Spec with default values
		defaultSpec = newAPITcpRoute(defaultRouteName, defaultTargets)

		// Result with the equivalent values as defaultSpec
		defaultRouteResult = newAWSTcpRoute(defaultRouteName, defaultTargets)

		extraTarget = []appmeshv1beta1.WeightedTarget{
			{Weight: int64(1), VirtualNodeName: defaultNodeName},
			{Weight: int64(1), VirtualNodeName: "extra-node"},
		}

		// Extra target spec and result
		extraTargetSpec   = newAPITcpRoute(defaultRouteName, extraTarget)
		extraTargetResult = newAWSTcpRoute(defaultRouteName, extraTarget)

		// No targets spec and result
		noTargetSpec    = newAPITcpRoute(defaultRouteName, []appmeshv1beta1.WeightedTarget{})
		noTargetsResult = newAWSTcpRoute(defaultRouteName, []appmeshv1beta1.WeightedTarget{})

		// Varying weight targets spec and result
		varyingWeightTargets = []appmeshv1beta1.WeightedTarget{
			{Weight: int64(1), VirtualNodeName: "foo-bar"},
			{Weight: int64(2), VirtualNodeName: "foo-bar-zoo"},
			{Weight: int64(3), VirtualNodeName: "foo-dummyNamespace"},
		}
		varyingWeightSpec = newAPITcpRoute(defaultRouteName, varyingWeightTargets)

		varyingWeightResult = newAWSTcpRoute(defaultRouteName, varyingWeightTargets)
	)

	var routetests = []struct {
		name        string
		spec        appmeshv1beta1.Route
		routes      aws.Route
		needsUpdate bool
	}{
		{"routes are the same", defaultSpec, defaultRouteResult, false},
		{"extra weighted target in result", defaultSpec, extraTargetResult, true},
		{"extra weighted target in spec", extraTargetSpec, defaultRouteResult, true},
		{"no targets in result", extraTargetSpec, noTargetsResult, true},
		{"no targets in spec", noTargetSpec, defaultRouteResult, true},
		{"routes with varying weights are the same", varyingWeightSpec, varyingWeightResult, false},
	}

	for _, tt := range routetests {
		t.Run(tt.name, func(t *testing.T) {
			if res := routeNeedsUpdate(tt.spec, tt.routes); res != tt.needsUpdate {
				t.Errorf("got %v, want %v", res, tt.needsUpdate)
			}
		})
	}
}

func TestHttp2RouteNeedUpdate(t *testing.T) {

	var (
		// shared defaults
		defaultRouteName = "example-http2-route"
		defaultNodeName  = "example-node"
		defaultPrefix    = "/"

		// Targets for default custom resource spec
		defaultTargets = []appmeshv1beta1.WeightedTarget{
			{Weight: int64(1), VirtualNodeName: defaultNodeName},
		}

		// Spec with default values
		defaultSpec = newAPIHttp2Route(defaultRouteName, defaultPrefix, defaultTargets)

		// Result with the equivalent values as defaultSpec
		defaultRouteResult = newAWSHttp2Route(defaultRouteName, defaultPrefix, defaultTargets)

		extraTarget = []appmeshv1beta1.WeightedTarget{
			{Weight: int64(1), VirtualNodeName: defaultNodeName},
			{Weight: int64(1), VirtualNodeName: "extra-node"},
		}

		// Extra target spec and result
		extraTargetSpec   = newAPIHttp2Route(defaultRouteName, defaultPrefix, extraTarget)
		extraTargetResult = newAWSHttp2Route(defaultRouteName, defaultPrefix, extraTarget)

		// No targets spec and result
		noTargetSpec    = newAPIHttp2Route(defaultRouteName, defaultPrefix, []appmeshv1beta1.WeightedTarget{})
		noTargetsResult = newAWSHttp2Route(defaultRouteName, defaultPrefix, []appmeshv1beta1.WeightedTarget{})

		// Varying weight targets spec and result
		varyingWeightTargets = []appmeshv1beta1.WeightedTarget{
			{Weight: int64(1), VirtualNodeName: "foo-bar"},
			{Weight: int64(2), VirtualNodeName: "foo-bar-zoo"},
			{Weight: int64(3), VirtualNodeName: "foo-dummyNamespace"},
		}
		varyingWeightSpec = newAPIHttp2Route(defaultRouteName, defaultPrefix, varyingWeightTargets)

		varyingWeightResult = newAWSHttp2Route(defaultRouteName, defaultPrefix, varyingWeightTargets)
	)

	var routetests = []struct {
		name        string
		spec        appmeshv1beta1.Route
		routes      aws.Route
		needsUpdate bool
	}{
		{"routes are the same", defaultSpec, defaultRouteResult, false},
		{"extra weighted target in result", defaultSpec, extraTargetResult, true},
		{"extra weighted target in spec", extraTargetSpec, defaultRouteResult, true},
		{"no targets in result", extraTargetSpec, noTargetsResult, true},
		{"no targets in spec", noTargetSpec, defaultRouteResult, true},
		{"routes with varying weights are the same", varyingWeightSpec, varyingWeightResult, false},
	}

	for _, tt := range routetests {
		t.Run(tt.name, func(t *testing.T) {
			if res := routeNeedsUpdate(tt.spec, tt.routes); res != tt.needsUpdate {
				t.Errorf("got %v, want %v", res, tt.needsUpdate)
			}
		})
	}
}

func TestGrpcRouteNeedUpdate(t *testing.T) {

	var (
		// shared defaults
		defaultRouteName   = "example-http2-route"
		defaultNodeName    = "example-node"
		defaultServiceName = "service-name"
		defaultMethodName  = "method-name"

		// Targets for default custom resource spec
		defaultTargets = []appmeshv1beta1.WeightedTarget{
			{Weight: int64(1), VirtualNodeName: defaultNodeName},
		}

		// Spec with default values
		defaultSpec = newAPIGrpcRoute(defaultRouteName, defaultServiceName, defaultMethodName, defaultTargets)

		// Result with the equivalent values as defaultSpec
		defaultRouteResult = newAWSGrpcRoute(defaultRouteName, defaultServiceName, defaultMethodName, defaultTargets)

		extraTarget = []appmeshv1beta1.WeightedTarget{
			{Weight: int64(1), VirtualNodeName: defaultNodeName},
			{Weight: int64(1), VirtualNodeName: "extra-node"},
		}

		// Extra target spec and result
		extraTargetSpec   = newAPIGrpcRoute(defaultRouteName, defaultServiceName, defaultMethodName, extraTarget)
		extraTargetResult = newAWSGrpcRoute(defaultRouteName, defaultServiceName, defaultMethodName, extraTarget)

		// No targets spec and result
		noTargetSpec    = newAPIGrpcRoute(defaultRouteName, defaultServiceName, defaultMethodName, []appmeshv1beta1.WeightedTarget{})
		noTargetsResult = newAWSGrpcRoute(defaultRouteName, defaultServiceName, defaultMethodName, []appmeshv1beta1.WeightedTarget{})

		// Varying weight targets spec and result
		varyingWeightTargets = []appmeshv1beta1.WeightedTarget{
			{Weight: int64(1), VirtualNodeName: "foo-bar"},
			{Weight: int64(2), VirtualNodeName: "foo-bar-zoo"},
			{Weight: int64(3), VirtualNodeName: "foo-dummyNamespace"},
		}
		varyingWeightSpec = newAPIGrpcRoute(defaultRouteName, defaultServiceName, defaultMethodName, varyingWeightTargets)

		varyingWeightResult = newAWSGrpcRoute(defaultRouteName, defaultServiceName, defaultMethodName, varyingWeightTargets)
	)

	var routetests = []struct {
		name        string
		spec        appmeshv1beta1.Route
		routes      aws.Route
		needsUpdate bool
	}{
		{"routes are the same", defaultSpec, defaultRouteResult, false},
		{"extra weighted target in result", defaultSpec, extraTargetResult, true},
		{"extra weighted target in spec", extraTargetSpec, defaultRouteResult, true},
		{"no targets in result", extraTargetSpec, noTargetsResult, true},
		{"no targets in spec", noTargetSpec, defaultRouteResult, true},
		{"routes with varying weights are the same", varyingWeightSpec, varyingWeightResult, false},
	}

	for _, tt := range routetests {
		t.Run(tt.name, func(t *testing.T) {
			if res := routeNeedsUpdate(tt.spec, tt.routes); res != tt.needsUpdate {
				t.Errorf("got %v, want %v", res, tt.needsUpdate)
			}
		})
	}
}

func TestHttpRouteWithHeaderNeedUpdate(t *testing.T) {
	var (
		// shared defaults
		defaultRouteName = "example-route"
		defaultPrefix    = "/"
		defaultNodeName  = "example-node"

		// Targets for default custom resource spec
		defaultTargets = []appmeshv1beta1.WeightedTarget{
			{Weight: int64(1), VirtualNodeName: defaultNodeName},
		}

		specWithNoMatch = appmeshv1beta1.HttpRouteHeader{
			Name:  "header1",
			Match: &appmeshv1beta1.HeaderMatchMethod{},
		}
		resultWithNoMatch = appmesh.HttpRouteHeader{
			Name:  awssdk.String("header1"),
			Match: &appmesh.HeaderMatchMethod{},
		}

		//Exact
		specWithExactHeaderMatch = appmeshv1beta1.HttpRouteHeader{
			Name: "header1",
			Match: &appmeshv1beta1.HeaderMatchMethod{
				Exact: awssdk.String("value1"),
			},
		}
		resultWithExactHeaderMatch = appmesh.HttpRouteHeader{
			Name: awssdk.String("header1"),
			Match: &appmesh.HeaderMatchMethod{
				Exact: awssdk.String("value1"),
			},
		}
		resultWithExactHeaderMatchDifferent = appmesh.HttpRouteHeader{
			Name: awssdk.String("header1"),
			Match: &appmesh.HeaderMatchMethod{
				Exact: awssdk.String("diff1"),
			},
		}

		//Prefix
		specWithPrefixHeaderMatch = appmeshv1beta1.HttpRouteHeader{
			Name: "header1",
			Match: &appmeshv1beta1.HeaderMatchMethod{
				Prefix: awssdk.String("value1"),
			},
		}
		resultWithPrefixHeaderMatch = appmesh.HttpRouteHeader{
			Name: awssdk.String("header1"),
			Match: &appmesh.HeaderMatchMethod{
				Prefix: awssdk.String("value1"),
			},
		}
		resultWithPrefixHeaderMatchDifferent = appmesh.HttpRouteHeader{
			Name: awssdk.String("header1"),
			Match: &appmesh.HeaderMatchMethod{
				Prefix: awssdk.String("diff1"),
			},
		}

		//Suffix
		specWithSuffixHeaderMatch = appmeshv1beta1.HttpRouteHeader{
			Name: "header1",
			Match: &appmeshv1beta1.HeaderMatchMethod{
				Suffix: awssdk.String("value1"),
			},
		}
		resultWithSuffixHeaderMatch = appmesh.HttpRouteHeader{
			Name: awssdk.String("header1"),
			Match: &appmesh.HeaderMatchMethod{
				Suffix: awssdk.String("value1"),
			},
		}
		resultWithSuffixHeaderMatchDifferent = appmesh.HttpRouteHeader{
			Name: awssdk.String("header1"),
			Match: &appmesh.HeaderMatchMethod{
				Suffix: awssdk.String("diff1"),
			},
		}

		//Regex
		specWithRegexHeaderMatch = appmeshv1beta1.HttpRouteHeader{
			Name: "header1",
			Match: &appmeshv1beta1.HeaderMatchMethod{
				Regex: awssdk.String("value1"),
			},
		}
		resultWithRegexHeaderMatch = appmesh.HttpRouteHeader{
			Name: awssdk.String("header1"),
			Match: &appmesh.HeaderMatchMethod{
				Regex: awssdk.String("value1"),
			},
		}
		resultWithRegexHeaderMatchDifferent = appmesh.HttpRouteHeader{
			Name: awssdk.String("header1"),
			Match: &appmesh.HeaderMatchMethod{
				Regex: awssdk.String("diff1"),
			},
		}

		//Range
		specWithRangeHeaderMatch = appmeshv1beta1.HttpRouteHeader{
			Name: "header1",
			Match: &appmeshv1beta1.HeaderMatchMethod{
				Range: &appmeshv1beta1.MatchRange{
					Start: awssdk.Int64(100),
					End:   awssdk.Int64(200),
				},
			},
		}
		resultWithRangeHeaderMatch = appmesh.HttpRouteHeader{
			Name: awssdk.String("header1"),
			Match: &appmesh.HeaderMatchMethod{
				Range: &appmesh.MatchRange{
					Start: awssdk.Int64(100),
					End:   awssdk.Int64(200),
				},
			},
		}
		resultWithRangeHeaderMatchDifferent = appmesh.HttpRouteHeader{
			Name: awssdk.String("header1"),
			Match: &appmesh.HeaderMatchMethod{
				Range: &appmesh.MatchRange{
					Start: awssdk.Int64(10),
					End:   awssdk.Int64(20),
				},
			},
		}
	)

	var tests = []struct {
		name      string
		desired   appmeshv1beta1.HttpRouteHeader
		target    appmesh.HttpRouteHeader
		different bool
	}{
		{"Desired and target have no match defined", specWithNoMatch, resultWithNoMatch, false},

		{"Exact: headers are the same", specWithExactHeaderMatch, resultWithExactHeaderMatch, false},
		{"Exact: target is missing", specWithExactHeaderMatch, resultWithNoMatch, true},
		{"Exact: target is diff", specWithExactHeaderMatch, resultWithExactHeaderMatchDifferent, true},
		{"Exact: desired is missing", specWithNoMatch, resultWithExactHeaderMatch, true},

		{"Prefix: headers are the same", specWithPrefixHeaderMatch, resultWithPrefixHeaderMatch, false},
		{"Prefix: target is missing", specWithPrefixHeaderMatch, resultWithNoMatch, true},
		{"Prefix: target is diff", specWithPrefixHeaderMatch, resultWithPrefixHeaderMatchDifferent, true},
		{"Prefix: desired is missing", specWithNoMatch, resultWithPrefixHeaderMatch, true},

		{"Suffix: headers are the same", specWithSuffixHeaderMatch, resultWithSuffixHeaderMatch, false},
		{"Suffix: target is missing", specWithSuffixHeaderMatch, resultWithNoMatch, true},
		{"Suffix: target is diff", specWithSuffixHeaderMatch, resultWithSuffixHeaderMatchDifferent, true},
		{"Suffix: desired is missing", specWithNoMatch, resultWithSuffixHeaderMatch, true},

		{"Regex: headers are the same", specWithRegexHeaderMatch, resultWithRegexHeaderMatch, false},
		{"Regex: target is missing", specWithRegexHeaderMatch, resultWithNoMatch, true},
		{"Regex: target is diff", specWithRegexHeaderMatch, resultWithRegexHeaderMatchDifferent, true},
		{"Regex: desired is missing", specWithNoMatch, resultWithRegexHeaderMatch, true},

		{"Range: headers are the same", specWithRangeHeaderMatch, resultWithRangeHeaderMatch, false},
		{"Range: target is missing", specWithRangeHeaderMatch, resultWithNoMatch, true},
		{"Range: target is diff", specWithRangeHeaderMatch, resultWithRangeHeaderMatchDifferent, true},
		{"Range: desired is missing", specWithNoMatch, resultWithRangeHeaderMatch, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := newAPIHttpRoute(defaultRouteName, defaultPrefix, defaultTargets)
			spec.Http.Match.Headers = []appmeshv1beta1.HttpRouteHeader{tt.desired}
			result := newAWSHttpRoute(defaultRouteName, defaultPrefix, defaultTargets)
			result.Data.Spec.HttpRoute.Match.Headers = []*appmesh.HttpRouteHeader{&tt.target}
			if res := routeNeedsUpdate(spec, result); res != tt.different {
				t.Errorf("got %v, want %v", res, tt.different)
			}
		})
	}
}

func TestHttp2RouteWithHeaderNeedUpdate(t *testing.T) {
	var (
		// shared defaults
		defaultRouteName = "example-route"
		defaultPrefix    = "/"
		defaultNodeName  = "example-node"

		// Targets for default custom resource spec
		defaultTargets = []appmeshv1beta1.WeightedTarget{
			{Weight: int64(1), VirtualNodeName: defaultNodeName},
		}

		specWithNoMatch = appmeshv1beta1.HttpRouteHeader{
			Name:  "header1",
			Match: &appmeshv1beta1.HeaderMatchMethod{},
		}
		resultWithNoMatch = appmesh.HttpRouteHeader{
			Name:  awssdk.String("header1"),
			Match: &appmesh.HeaderMatchMethod{},
		}

		//Exact
		specWithExactHeaderMatch = appmeshv1beta1.HttpRouteHeader{
			Name: "header1",
			Match: &appmeshv1beta1.HeaderMatchMethod{
				Exact: awssdk.String("value1"),
			},
		}
		resultWithExactHeaderMatch = appmesh.HttpRouteHeader{
			Name: awssdk.String("header1"),
			Match: &appmesh.HeaderMatchMethod{
				Exact: awssdk.String("value1"),
			},
		}
		resultWithExactHeaderMatchDifferent = appmesh.HttpRouteHeader{
			Name: awssdk.String("header1"),
			Match: &appmesh.HeaderMatchMethod{
				Exact: awssdk.String("diff1"),
			},
		}

		//Prefix
		specWithPrefixHeaderMatch = appmeshv1beta1.HttpRouteHeader{
			Name: "header1",
			Match: &appmeshv1beta1.HeaderMatchMethod{
				Prefix: awssdk.String("value1"),
			},
		}
		resultWithPrefixHeaderMatch = appmesh.HttpRouteHeader{
			Name: awssdk.String("header1"),
			Match: &appmesh.HeaderMatchMethod{
				Prefix: awssdk.String("value1"),
			},
		}
		resultWithPrefixHeaderMatchDifferent = appmesh.HttpRouteHeader{
			Name: awssdk.String("header1"),
			Match: &appmesh.HeaderMatchMethod{
				Prefix: awssdk.String("diff1"),
			},
		}

		//Suffix
		specWithSuffixHeaderMatch = appmeshv1beta1.HttpRouteHeader{
			Name: "header1",
			Match: &appmeshv1beta1.HeaderMatchMethod{
				Suffix: awssdk.String("value1"),
			},
		}
		resultWithSuffixHeaderMatch = appmesh.HttpRouteHeader{
			Name: awssdk.String("header1"),
			Match: &appmesh.HeaderMatchMethod{
				Suffix: awssdk.String("value1"),
			},
		}
		resultWithSuffixHeaderMatchDifferent = appmesh.HttpRouteHeader{
			Name: awssdk.String("header1"),
			Match: &appmesh.HeaderMatchMethod{
				Suffix: awssdk.String("diff1"),
			},
		}

		//Regex
		specWithRegexHeaderMatch = appmeshv1beta1.HttpRouteHeader{
			Name: "header1",
			Match: &appmeshv1beta1.HeaderMatchMethod{
				Regex: awssdk.String("value1"),
			},
		}
		resultWithRegexHeaderMatch = appmesh.HttpRouteHeader{
			Name: awssdk.String("header1"),
			Match: &appmesh.HeaderMatchMethod{
				Regex: awssdk.String("value1"),
			},
		}
		resultWithRegexHeaderMatchDifferent = appmesh.HttpRouteHeader{
			Name: awssdk.String("header1"),
			Match: &appmesh.HeaderMatchMethod{
				Regex: awssdk.String("diff1"),
			},
		}

		//Range
		specWithRangeHeaderMatch = appmeshv1beta1.HttpRouteHeader{
			Name: "header1",
			Match: &appmeshv1beta1.HeaderMatchMethod{
				Range: &appmeshv1beta1.MatchRange{
					Start: awssdk.Int64(100),
					End:   awssdk.Int64(200),
				},
			},
		}
		resultWithRangeHeaderMatch = appmesh.HttpRouteHeader{
			Name: awssdk.String("header1"),
			Match: &appmesh.HeaderMatchMethod{
				Range: &appmesh.MatchRange{
					Start: awssdk.Int64(100),
					End:   awssdk.Int64(200),
				},
			},
		}
		resultWithRangeHeaderMatchDifferent = appmesh.HttpRouteHeader{
			Name: awssdk.String("header1"),
			Match: &appmesh.HeaderMatchMethod{
				Range: &appmesh.MatchRange{
					Start: awssdk.Int64(10),
					End:   awssdk.Int64(20),
				},
			},
		}
	)

	var tests = []struct {
		name      string
		desired   appmeshv1beta1.HttpRouteHeader
		target    appmesh.HttpRouteHeader
		different bool
	}{
		{"Desired and target have no match defined", specWithNoMatch, resultWithNoMatch, false},

		{"Exact: headers are the same", specWithExactHeaderMatch, resultWithExactHeaderMatch, false},
		{"Exact: target is missing", specWithExactHeaderMatch, resultWithNoMatch, true},
		{"Exact: target is diff", specWithExactHeaderMatch, resultWithExactHeaderMatchDifferent, true},
		{"Exact: desired is missing", specWithNoMatch, resultWithExactHeaderMatch, true},

		{"Prefix: headers are the same", specWithPrefixHeaderMatch, resultWithPrefixHeaderMatch, false},
		{"Prefix: target is missing", specWithPrefixHeaderMatch, resultWithNoMatch, true},
		{"Prefix: target is diff", specWithPrefixHeaderMatch, resultWithPrefixHeaderMatchDifferent, true},
		{"Prefix: desired is missing", specWithNoMatch, resultWithPrefixHeaderMatch, true},

		{"Suffix: headers are the same", specWithSuffixHeaderMatch, resultWithSuffixHeaderMatch, false},
		{"Suffix: target is missing", specWithSuffixHeaderMatch, resultWithNoMatch, true},
		{"Suffix: target is diff", specWithSuffixHeaderMatch, resultWithSuffixHeaderMatchDifferent, true},
		{"Suffix: desired is missing", specWithNoMatch, resultWithSuffixHeaderMatch, true},

		{"Regex: headers are the same", specWithRegexHeaderMatch, resultWithRegexHeaderMatch, false},
		{"Regex: target is missing", specWithRegexHeaderMatch, resultWithNoMatch, true},
		{"Regex: target is diff", specWithRegexHeaderMatch, resultWithRegexHeaderMatchDifferent, true},
		{"Regex: desired is missing", specWithNoMatch, resultWithRegexHeaderMatch, true},

		{"Range: headers are the same", specWithRangeHeaderMatch, resultWithRangeHeaderMatch, false},
		{"Range: target is missing", specWithRangeHeaderMatch, resultWithNoMatch, true},
		{"Range: target is diff", specWithRangeHeaderMatch, resultWithRangeHeaderMatchDifferent, true},
		{"Range: desired is missing", specWithNoMatch, resultWithRangeHeaderMatch, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := newAPIHttp2Route(defaultRouteName, defaultPrefix, defaultTargets)
			spec.Http2.Match.Headers = []appmeshv1beta1.HttpRouteHeader{tt.desired}
			result := newAWSHttp2Route(defaultRouteName, defaultPrefix, defaultTargets)
			result.Data.Spec.Http2Route.Match.Headers = []*appmesh.HttpRouteHeader{&tt.target}
			if res := routeNeedsUpdate(spec, result); res != tt.different {
				t.Errorf("got %v, want %v", res, tt.different)
			}
		})
	}
}

func TestGrpcRouteWithMetadataNeedUpdate(t *testing.T) {
	var (
		// shared defaults
		defaultRouteName   = "example-route"
		defaultServiceName = "example-serviceName"
		defaultMethodName  = "example-methodName"
		defaultNodeName    = "example-node"

		// Targets for default custom resource spec
		defaultTargets = []appmeshv1beta1.WeightedTarget{
			{Weight: int64(1), VirtualNodeName: defaultNodeName},
		}

		specWithNoMatch = appmeshv1beta1.GrpcRouteMetadata{
			Name:  "header1",
			Match: &appmeshv1beta1.MetadataMatchMethod{},
		}
		resultWithNoMatch = appmesh.GrpcRouteMetadata{
			Name:  awssdk.String("header1"),
			Match: &appmesh.GrpcRouteMetadataMatchMethod{},
		}

		//Exact
		specWithExactHeaderMatch = appmeshv1beta1.GrpcRouteMetadata{
			Name: "header1",
			Match: &appmeshv1beta1.MetadataMatchMethod{
				Exact: awssdk.String("value1"),
			},
		}
		resultWithExactHeaderMatch = appmesh.GrpcRouteMetadata{
			Name: awssdk.String("header1"),
			Match: &appmesh.GrpcRouteMetadataMatchMethod{
				Exact: awssdk.String("value1"),
			},
		}
		resultWithExactHeaderMatchDifferent = appmesh.GrpcRouteMetadata{
			Name: awssdk.String("header1"),
			Match: &appmesh.GrpcRouteMetadataMatchMethod{
				Exact: awssdk.String("diff1"),
			},
		}

		//Prefix
		specWithPrefixHeaderMatch = appmeshv1beta1.GrpcRouteMetadata{
			Name: "header1",
			Match: &appmeshv1beta1.MetadataMatchMethod{
				Prefix: awssdk.String("value1"),
			},
		}
		resultWithPrefixHeaderMatch = appmesh.GrpcRouteMetadata{
			Name: awssdk.String("header1"),
			Match: &appmesh.GrpcRouteMetadataMatchMethod{
				Prefix: awssdk.String("value1"),
			},
		}
		resultWithPrefixHeaderMatchDifferent = appmesh.GrpcRouteMetadata{
			Name: awssdk.String("header1"),
			Match: &appmesh.GrpcRouteMetadataMatchMethod{
				Prefix: awssdk.String("diff1"),
			},
		}

		//Suffix
		specWithSuffixHeaderMatch = appmeshv1beta1.GrpcRouteMetadata{
			Name: "header1",
			Match: &appmeshv1beta1.MetadataMatchMethod{
				Suffix: awssdk.String("value1"),
			},
		}
		resultWithSuffixHeaderMatch = appmesh.GrpcRouteMetadata{
			Name: awssdk.String("header1"),
			Match: &appmesh.GrpcRouteMetadataMatchMethod{
				Suffix: awssdk.String("value1"),
			},
		}
		resultWithSuffixHeaderMatchDifferent = appmesh.GrpcRouteMetadata{
			Name: awssdk.String("header1"),
			Match: &appmesh.GrpcRouteMetadataMatchMethod{
				Suffix: awssdk.String("diff1"),
			},
		}

		//Regex
		specWithRegexHeaderMatch = appmeshv1beta1.GrpcRouteMetadata{
			Name: "header1",
			Match: &appmeshv1beta1.MetadataMatchMethod{
				Regex: awssdk.String("value1"),
			},
		}
		resultWithRegexHeaderMatch = appmesh.GrpcRouteMetadata{
			Name: awssdk.String("header1"),
			Match: &appmesh.GrpcRouteMetadataMatchMethod{
				Regex: awssdk.String("value1"),
			},
		}
		resultWithRegexHeaderMatchDifferent = appmesh.GrpcRouteMetadata{
			Name: awssdk.String("header1"),
			Match: &appmesh.GrpcRouteMetadataMatchMethod{
				Regex: awssdk.String("diff1"),
			},
		}

		//Range
		specWithRangeHeaderMatch = appmeshv1beta1.GrpcRouteMetadata{
			Name: "header1",
			Match: &appmeshv1beta1.MetadataMatchMethod{
				Range: &appmeshv1beta1.MatchRange{
					Start: awssdk.Int64(100),
					End:   awssdk.Int64(200),
				},
			},
		}
		resultWithRangeHeaderMatch = appmesh.GrpcRouteMetadata{
			Name: awssdk.String("header1"),
			Match: &appmesh.GrpcRouteMetadataMatchMethod{
				Range: &appmesh.MatchRange{
					Start: awssdk.Int64(100),
					End:   awssdk.Int64(200),
				},
			},
		}
		resultWithRangeHeaderMatchDifferent = appmesh.GrpcRouteMetadata{
			Name: awssdk.String("header1"),
			Match: &appmesh.GrpcRouteMetadataMatchMethod{
				Range: &appmesh.MatchRange{
					Start: awssdk.Int64(10),
					End:   awssdk.Int64(20),
				},
			},
		}
	)

	var tests = []struct {
		name      string
		desired   appmeshv1beta1.GrpcRouteMetadata
		target    appmesh.GrpcRouteMetadata
		different bool
	}{
		{"Desired and target have no match defined", specWithNoMatch, resultWithNoMatch, false},

		{"Exact: headers are the same", specWithExactHeaderMatch, resultWithExactHeaderMatch, false},
		{"Exact: target is missing", specWithExactHeaderMatch, resultWithNoMatch, true},
		{"Exact: target is diff", specWithExactHeaderMatch, resultWithExactHeaderMatchDifferent, true},
		{"Exact: desired is missing", specWithNoMatch, resultWithExactHeaderMatch, true},

		{"Prefix: headers are the same", specWithPrefixHeaderMatch, resultWithPrefixHeaderMatch, false},
		{"Prefix: target is missing", specWithPrefixHeaderMatch, resultWithNoMatch, true},
		{"Prefix: target is diff", specWithPrefixHeaderMatch, resultWithPrefixHeaderMatchDifferent, true},
		{"Prefix: desired is missing", specWithNoMatch, resultWithPrefixHeaderMatch, true},

		{"Suffix: headers are the same", specWithSuffixHeaderMatch, resultWithSuffixHeaderMatch, false},
		{"Suffix: target is missing", specWithSuffixHeaderMatch, resultWithNoMatch, true},
		{"Suffix: target is diff", specWithSuffixHeaderMatch, resultWithSuffixHeaderMatchDifferent, true},
		{"Suffix: desired is missing", specWithNoMatch, resultWithSuffixHeaderMatch, true},

		{"Regex: headers are the same", specWithRegexHeaderMatch, resultWithRegexHeaderMatch, false},
		{"Regex: target is missing", specWithRegexHeaderMatch, resultWithNoMatch, true},
		{"Regex: target is diff", specWithRegexHeaderMatch, resultWithRegexHeaderMatchDifferent, true},
		{"Regex: desired is missing", specWithNoMatch, resultWithRegexHeaderMatch, true},

		{"Range: headers are the same", specWithRangeHeaderMatch, resultWithRangeHeaderMatch, false},
		{"Range: target is missing", specWithRangeHeaderMatch, resultWithNoMatch, true},
		{"Range: target is diff", specWithRangeHeaderMatch, resultWithRangeHeaderMatchDifferent, true},
		{"Range: desired is missing", specWithNoMatch, resultWithRangeHeaderMatch, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := newAPIGrpcRoute(defaultRouteName, defaultServiceName, defaultMethodName, defaultTargets)
			spec.Grpc.Match.Metadata = []appmeshv1beta1.GrpcRouteMetadata{tt.desired}
			result := newAWSGrpcRoute(defaultRouteName, defaultServiceName, defaultMethodName, defaultTargets)
			result.Data.Spec.GrpcRoute.Match.Metadata = []*appmesh.GrpcRouteMetadata{&tt.target}
			if res := routeNeedsUpdate(spec, result); res != tt.different {
				t.Errorf("got %v, want %v", res, tt.different)
			}
		})
	}
}

func TestGetVirtualRouter(t *testing.T) {
	var (
		defaultMeshName    = "example-mesh"
		defaultRouteName   = "example-route"
		defaultPrefix      = "/"
		defaultServiceName = "serviceName"
		defaultMethodName  = "methodName"

		defaultHttpPort = appmeshv1beta1.PortMapping{
			Port:     8080,
			Protocol: appmeshv1beta1.PortProtocolHttp,
		}
		defaultTcpPort = appmeshv1beta1.PortMapping{
			Port:     6379,
			Protocol: appmeshv1beta1.PortProtocolTcp,
		}
		defaultHttp2Port = appmeshv1beta1.PortMapping{
			Port:     8081,
			Protocol: appmeshv1beta1.PortProtocolHttp2,
		}
		defaultGrpcPort = appmeshv1beta1.PortMapping{
			Port:     8082,
			Protocol: appmeshv1beta1.PortProtocolGrpc,
		}

		defaultHttpRouterListener = appmeshv1beta1.VirtualRouterListener{
			PortMapping: defaultHttpPort,
		}
		defaultTcpRouterListener = appmeshv1beta1.VirtualRouterListener{
			PortMapping: defaultTcpPort,
		}
		defaultHttp2RouterListener = appmeshv1beta1.VirtualRouterListener{
			PortMapping: defaultHttp2Port,
		}
		defaultGrpcRouterListener = appmeshv1beta1.VirtualRouterListener{
			PortMapping: defaultGrpcPort,
		}

		defaultHttpListener = appmeshv1beta1.Listener{
			PortMapping: appmeshv1beta1.PortMapping{
				Port:     8080,
				Protocol: appmeshv1beta1.PortProtocolHttp,
			},
		}
		defaultTcpListener = appmeshv1beta1.Listener{
			PortMapping: appmeshv1beta1.PortMapping{
				Port:     6379,
				Protocol: appmeshv1beta1.PortProtocolTcp,
			},
		}
		defaultHttp2Listener = appmeshv1beta1.Listener{
			PortMapping: appmeshv1beta1.PortMapping{
				Port:     8081,
				Protocol: appmeshv1beta1.PortProtocolHttp2,
			},
		}
		defaultGrpcListener = appmeshv1beta1.Listener{
			PortMapping: appmeshv1beta1.PortMapping{
				Port:     8082,
				Protocol: appmeshv1beta1.PortProtocolGrpc,
			},
		}

		defaultHttpRoute  = newAPIHttpRoute(defaultRouteName, defaultPrefix, []appmeshv1beta1.WeightedTarget{})
		defaultTcpRoute   = newAPITcpRoute(defaultRouteName, []appmeshv1beta1.WeightedTarget{})
		defaultHttp2Route = newAPIHttp2Route(defaultRouteName, defaultPrefix, []appmeshv1beta1.WeightedTarget{})
		defaultGrpcRoute  = newAPIGrpcRoute(defaultRouteName, defaultServiceName, defaultMethodName, []appmeshv1beta1.WeightedTarget{})

		virtualRouterWithNoListener = appmeshv1beta1.VirtualRouter{
			Name: "example-router",
		}
		virtualRouterWithHttpListener = appmeshv1beta1.VirtualRouter{
			Name:      "example-http-router",
			Listeners: []appmeshv1beta1.VirtualRouterListener{defaultHttpRouterListener},
		}
		virtualRouterWithTcpListener = appmeshv1beta1.VirtualRouter{
			Name:      "example-tcp-router",
			Listeners: []appmeshv1beta1.VirtualRouterListener{defaultTcpRouterListener},
		}
		virtualRouterWithHttp2Listener = appmeshv1beta1.VirtualRouter{
			Name:      "example-http2-router",
			Listeners: []appmeshv1beta1.VirtualRouterListener{defaultHttp2RouterListener},
		}
		virtualRouterWithGrpcListener = appmeshv1beta1.VirtualRouter{
			Name:      "example-grpc-router",
			Listeners: []appmeshv1beta1.VirtualRouterListener{defaultGrpcRouterListener},
		}
	)

	var vservicetests = []struct {
		id                   string
		name                 string
		expectedListener     *appmeshv1beta1.VirtualRouterListener
		virtualRouter        *appmeshv1beta1.VirtualRouter
		route                *appmeshv1beta1.Route
		errorOnGetTargetNode bool
		virtualNodeListener  *appmeshv1beta1.Listener
	}{
		{"0",
			"virtual-service with router HTTP listener should be preserved",
			&defaultHttpRouterListener,
			&virtualRouterWithHttpListener,
			nil,
			false,
			nil,
		},

		{"1",
			"virtual-service with router TCP listener should be preserved",
			&defaultTcpRouterListener,
			&virtualRouterWithTcpListener,
			nil,
			false,
			nil,
		},

		{"2",
			"virtual-service with router HTTP2 listener should be preserved",
			&defaultHttp2RouterListener,
			&virtualRouterWithHttp2Listener,
			nil,
			false,
			nil,
		},

		{"3",
			"virtual-service with router GRPC listener should be preserved",
			&defaultGrpcRouterListener,
			&virtualRouterWithGrpcListener,
			nil,
			false,
			nil,
		},

		{"4",
			"virtual-service with router missing listener should get listener from target node of HTTP route",
			&defaultHttpRouterListener,
			&virtualRouterWithNoListener,
			&defaultHttpRoute,
			false,
			&defaultHttpListener,
		},

		{"5",
			"virtual-service with router missing listener and failed to load virtual-node of HTTP route",
			nil,
			&virtualRouterWithNoListener,
			&defaultHttpRoute,
			true,
			&defaultHttpListener,
		},

		{"6",
			"virtual-service with router missing listener should get listener from target node of TCP route",
			&defaultTcpRouterListener,
			&virtualRouterWithNoListener,
			&defaultTcpRoute,
			false,
			&defaultTcpListener,
		},

		{"7",
			"virtual-service with router missing listener and failed to load virtual-node of TCP route",
			nil,
			&virtualRouterWithNoListener,
			&defaultTcpRoute,
			true,
			&defaultTcpListener,
		},

		{"8",
			"virtual-service with router missing listener should get listener from target node of HTTP2 route",
			&defaultHttp2RouterListener,
			&virtualRouterWithNoListener,
			&defaultHttp2Route,
			false,
			&defaultHttp2Listener,
		},

		{"9",
			"virtual-service with router missing listener and failed to load virtual-node of HTTP2 route",
			nil,
			&virtualRouterWithNoListener,
			&defaultHttp2Route,
			true,
			&defaultHttp2Listener,
		},

		{"10",
			"virtual-service with router missing listener should get listener from target node of GRPC route",
			&defaultGrpcRouterListener,
			&virtualRouterWithNoListener,
			&defaultGrpcRoute,
			false,
			&defaultGrpcListener,
		},

		{"11",
			"virtual-service with router missing listener and failed to load virtual-node of GRPC route",
			nil,
			&virtualRouterWithNoListener,
			&defaultGrpcRoute,
			true,
			&defaultGrpcListener,
		},

		{"12",
			"virtual-service with router missing listener and no routes",
			nil,
			&virtualRouterWithNoListener,
			nil,
			false,
			nil,
		},

		{"13",
			"virtual-service with router missing listener and target node missing listener",
			nil,
			&virtualRouterWithNoListener,
			&defaultHttpRoute,
			false,
			nil,
		},

		{"14",
			"virtual-service with no router should get listener from target node of HTTP route",
			&defaultHttpRouterListener,
			nil,
			&defaultHttpRoute,
			false,
			&defaultHttpListener,
		},
	}

	for _, tt := range vservicetests {
		t.Run(tt.name, func(t *testing.T) {
			mockCloudAPI := new(ctrlawsmocks.CloudAPI)
			mockMeshClientSet := new(appmeshv1beta1mocks.Interface)
			mockAppmeshv1beta1Client := new(appmeshv1beta1typedmocks.AppmeshV1beta1Interface)
			mockMeshClientSet.On(
				"AppmeshV1beta1",
			).Return(mockAppmeshv1beta1Client)
			mockVirtualServiceInterface := new(appmeshv1beta1typedmocks.VirtualServiceInterface)
			mockAppmeshv1beta1Client.On(
				"VirtualServices",
				mock.Anything,
			).Return(mockVirtualServiceInterface)
			mockVirtualNodeInterface := new(appmeshv1beta1typedmocks.VirtualNodeInterface)
			mockAppmeshv1beta1Client.On(
				"VirtualNodes",
				mock.Anything,
			).Return(mockVirtualNodeInterface)

			virtualService := newAPIVirtualService(defaultMeshName,
				nil,
				[]appmeshv1beta1.Route{},
			)
			virtualService.Name = fmt.Sprintf("%s-vsvc", tt.id)
			virtualService.Namespace = fmt.Sprintf("%s-ns", tt.id)

			if tt.virtualRouter != nil {
				virtualService.Spec.VirtualRouter = tt.virtualRouter.DeepCopy()
			}

			if tt.route != nil {
				targetVirtualNodeName := fmt.Sprintf("%s-vnode", tt.id)
				weightedTarget := appmeshv1beta1.WeightedTarget{
					VirtualNodeName: targetVirtualNodeName,
					Weight:          100,
				}

				if tt.route.Http != nil {
					copy := tt.route.DeepCopy()
					copy.Http.Action.WeightedTargets = append(copy.Http.Action.WeightedTargets, weightedTarget)
					virtualService.Spec.Routes = append(virtualService.Spec.Routes, *copy)
				} else if tt.virtualNodeListener.PortMapping.Protocol == appmeshv1beta1.PortProtocolTcp {
					copy := tt.route.DeepCopy()
					copy.Tcp.Action.WeightedTargets = append(copy.Tcp.Action.WeightedTargets, weightedTarget)
					virtualService.Spec.Routes = append(virtualService.Spec.Routes, *copy)
				} else if tt.virtualNodeListener.PortMapping.Protocol == appmeshv1beta1.PortProtocolHttp2 {
					copy := tt.route.DeepCopy()
					copy.Http2.Action.WeightedTargets = append(copy.Http2.Action.WeightedTargets, weightedTarget)
					virtualService.Spec.Routes = append(virtualService.Spec.Routes, *copy)
				} else if tt.virtualNodeListener.PortMapping.Protocol == appmeshv1beta1.PortProtocolGrpc {
					copy := tt.route.DeepCopy()
					copy.Grpc.Action.WeightedTargets = append(copy.Grpc.Action.WeightedTargets, weightedTarget)
					virtualService.Spec.Routes = append(virtualService.Spec.Routes, *copy)
				}

				if tt.errorOnGetTargetNode {
					mockVirtualNodeInterface.On(
						"Get",
						targetVirtualNodeName,
						metav1.GetOptions{},
					).Return(nil, fmt.Errorf("Error loading virtual-node"))
				} else {
					var targetVirtualNode *appmeshv1beta1.VirtualNode
					if tt.virtualNodeListener != nil {
						targetVirtualNode = newAPIVirtualNode([]int64{tt.virtualNodeListener.PortMapping.Port},
							[]string{tt.virtualNodeListener.PortMapping.Protocol},
							[]string{},
							"sample.com",
							nil)
					} else {
						targetVirtualNode = newAPIVirtualNode([]int64{},
							[]string{},
							[]string{},
							"",
							nil)
					}
					targetVirtualNode.Name = targetVirtualNodeName
					targetVirtualNode.Namespace = virtualService.Namespace
					mockVirtualNodeInterface.On(
						"Get",
						targetVirtualNodeName,
						metav1.GetOptions{},
					).Return(targetVirtualNode, nil)
				}
			} else {
				mockVirtualNodeInterface.AssertNotCalled(t, tt.name)
			}

			mockVirtualServiceInterface.On(
				"Get",
				virtualService.Name,
				metav1.GetOptions{},
			).Return(&virtualService, nil)

			c := &Controller{
				name:          "test",
				cloud:         mockCloudAPI,
				meshclientset: mockMeshClientSet,
			}
			vrouter := c.getVirtualRouter(&virtualService)
			if tt.expectedListener != nil {
				if len(vrouter.Listeners) == 0 {
					t.Errorf("Expected listener %+v but got none", tt.expectedListener)
				} else if !reflect.DeepEqual(vrouter.Listeners[0], *tt.expectedListener) {
					t.Errorf("Expected listener %+v but got %+v", tt.expectedListener, vrouter.Listeners[0])
				}
			} else if len(vrouter.Listeners) > 0 {
				t.Errorf("Expecting empty list of listeners on virtual-router but found %+v", vrouter.Listeners)
			}
		})
	}
}

func TestHttpRouteWithRetryPolicyNeedUpdate(t *testing.T) {
	var (
		// shared defaults
		defaultRouteName             = "example-route"
		defaultPrefix                = "/"
		defaultNodeName              = "example-node"
		defaultPerRetryTimeoutMillis = int64(1000)
		defaultMaxRetries            = int64(5)
		defaultHttpRetryPolicyEvent  = "http-error"
		defaultTcpRetryPolicyEvent   = appmesh.TcpRetryPolicyEventConnectionError

		// Targets for default custom resource spec
		defaultTargets = []appmeshv1beta1.WeightedTarget{
			{Weight: int64(1), VirtualNodeName: defaultNodeName},
		}

		nilSpec   *appmeshv1beta1.HttpRetryPolicy
		nilResult *appmesh.HttpRetryPolicy

		emptySpec   = &appmeshv1beta1.HttpRetryPolicy{}
		emptyResult = &appmesh.HttpRetryPolicy{}

		specWithHttpEvent = &appmeshv1beta1.HttpRetryPolicy{
			HttpRetryPolicyEvents: []appmeshv1beta1.HttpRetryPolicyEvent{
				appmeshv1beta1.HttpRetryPolicyEvent(defaultHttpRetryPolicyEvent),
			},
		}
		resultWithHttpEvent = &appmesh.HttpRetryPolicy{
			HttpRetryEvents: []*string{
				awssdk.String(defaultHttpRetryPolicyEvent),
			},
		}
		resultWithDifferentHttpEvent = &appmesh.HttpRetryPolicy{
			HttpRetryEvents: []*string{
				awssdk.String("diff-http-error"),
			},
		}

		specWithPerTryTimeout = &appmeshv1beta1.HttpRetryPolicy{
			PerRetryTimeoutMillis: awssdk.Int64(defaultPerRetryTimeoutMillis),
		}
		resultWithPerTryTimeout = &appmesh.HttpRetryPolicy{
			PerRetryTimeout: &appmesh.Duration{
				Unit:  awssdk.String(appmesh.DurationUnitMs),
				Value: awssdk.Int64(defaultPerRetryTimeoutMillis),
			},
		}
		resultWithDifferentPerTryTimeout = &appmesh.HttpRetryPolicy{
			PerRetryTimeout: &appmesh.Duration{
				Unit:  awssdk.String(appmesh.DurationUnitMs),
				Value: awssdk.Int64(defaultPerRetryTimeoutMillis + 1),
			},
		}
		resultWithSamePerTryTimeoutAndDiffUnit = &appmesh.HttpRetryPolicy{
			PerRetryTimeout: &appmesh.Duration{
				Unit:  awssdk.String(appmesh.DurationUnitS),
				Value: awssdk.Int64(defaultPerRetryTimeoutMillis / 1000),
			},
		}

		specWithMaxRetries = &appmeshv1beta1.HttpRetryPolicy{
			MaxRetries: awssdk.Int64(defaultMaxRetries),
		}
		resultWithMaxRetries = &appmesh.HttpRetryPolicy{
			MaxRetries: awssdk.Int64(defaultMaxRetries),
		}
		resultWithDifferentMaxRetries = &appmesh.HttpRetryPolicy{
			MaxRetries: awssdk.Int64(defaultMaxRetries + 1),
		}

		specWithTcpEvent = &appmeshv1beta1.HttpRetryPolicy{
			TcpRetryPolicyEvents: []appmeshv1beta1.TcpRetryPolicyEvent{
				appmeshv1beta1.TcpRetryPolicyEvent(defaultTcpRetryPolicyEvent),
			},
		}
		resultWithTcpEvent = &appmesh.HttpRetryPolicy{
			TcpRetryEvents: []*string{
				awssdk.String(defaultTcpRetryPolicyEvent),
			},
		}
		resultWithDifferentTcpEvent = &appmesh.HttpRetryPolicy{
			TcpRetryEvents: []*string{
				awssdk.String("diff-tcp-error"),
			},
		}
	)

	var tests = []struct {
		name      string
		desired   *appmeshv1beta1.HttpRetryPolicy
		target    *appmesh.HttpRetryPolicy
		different bool
	}{
		{"Nil spec", nilSpec, nilResult, false},
		{"Empty spec", emptySpec, emptyResult, false},

		{"PerTryTimeout: match", specWithPerTryTimeout, resultWithPerTryTimeout, false},
		{"PerTryTimeout: match with different unit", specWithPerTryTimeout, resultWithSamePerTryTimeoutAndDiffUnit, false},
		{"PerTryTimeout: missing in desired", emptySpec, resultWithPerTryTimeout, true},
		{"PerTryTimeout: missing in target", specWithPerTryTimeout, emptyResult, true},
		{"PerTryTimeout: diff", specWithPerTryTimeout, resultWithDifferentPerTryTimeout, true},

		{"MaxRetries: match", specWithMaxRetries, resultWithMaxRetries, false},
		{"MaxRetries: missing in desired", emptySpec, resultWithMaxRetries, true},
		{"MaxRetries: missing in target", specWithMaxRetries, emptyResult, true},
		{"MaxRetries: diff", specWithMaxRetries, resultWithDifferentMaxRetries, true},

		{"HttpRetryPolicyEvent: match", specWithHttpEvent, resultWithHttpEvent, false},
		{"HttpRetryPolicyEvent: missing in desired", emptySpec, resultWithHttpEvent, true},
		{"HttpRetryPolicyEvent: missing in target", specWithHttpEvent, emptyResult, true},
		{"HttpRetryPolicyEvent: diff", specWithHttpEvent, resultWithDifferentHttpEvent, true},

		{"TcpRetryPolicyEvent: match", specWithTcpEvent, resultWithTcpEvent, false},
		{"TcpRetryPolicyEvent: missing in desired", emptySpec, resultWithTcpEvent, true},
		{"TcpRetryPolicyEvent: missing in target", specWithTcpEvent, emptyResult, true},
		{"TcpRetryPolicyEvent: diff", specWithTcpEvent, resultWithDifferentTcpEvent, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := newAPIHttpRoute(defaultRouteName, defaultPrefix, defaultTargets)
			spec.Http.RetryPolicy = tt.desired
			result := newAWSHttpRoute(defaultRouteName, defaultPrefix, defaultTargets)
			result.Data.Spec.HttpRoute.RetryPolicy = tt.target
			if res := routeNeedsUpdate(spec, result); res != tt.different {
				t.Errorf("got %v, want %v", res, tt.different)
			}
		})
	}
}

func TestHttp2RouteWithRetryPolicyNeedUpdate(t *testing.T) {
	var (
		// shared defaults
		defaultRouteName             = "example-route"
		defaultPrefix                = "/"
		defaultNodeName              = "example-node"
		defaultPerRetryTimeoutMillis = int64(1000)
		defaultMaxRetries            = int64(5)
		defaultHttpRetryPolicyEvent  = "http-error"
		defaultTcpRetryPolicyEvent   = appmesh.TcpRetryPolicyEventConnectionError

		// Targets for default custom resource spec
		defaultTargets = []appmeshv1beta1.WeightedTarget{
			{Weight: int64(1), VirtualNodeName: defaultNodeName},
		}

		nilSpec   *appmeshv1beta1.HttpRetryPolicy
		nilResult *appmesh.HttpRetryPolicy

		emptySpec   = &appmeshv1beta1.HttpRetryPolicy{}
		emptyResult = &appmesh.HttpRetryPolicy{}

		specWithHttpEvent = &appmeshv1beta1.HttpRetryPolicy{
			HttpRetryPolicyEvents: []appmeshv1beta1.HttpRetryPolicyEvent{
				appmeshv1beta1.HttpRetryPolicyEvent(defaultHttpRetryPolicyEvent),
			},
		}
		resultWithHttpEvent = &appmesh.HttpRetryPolicy{
			HttpRetryEvents: []*string{
				awssdk.String(defaultHttpRetryPolicyEvent),
			},
		}
		resultWithDifferentHttpEvent = &appmesh.HttpRetryPolicy{
			HttpRetryEvents: []*string{
				awssdk.String("diff-http-error"),
			},
		}

		specWithPerTryTimeout = &appmeshv1beta1.HttpRetryPolicy{
			PerRetryTimeoutMillis: awssdk.Int64(defaultPerRetryTimeoutMillis),
		}
		resultWithPerTryTimeout = &appmesh.HttpRetryPolicy{
			PerRetryTimeout: &appmesh.Duration{
				Unit:  awssdk.String(appmesh.DurationUnitMs),
				Value: awssdk.Int64(defaultPerRetryTimeoutMillis),
			},
		}
		resultWithDifferentPerTryTimeout = &appmesh.HttpRetryPolicy{
			PerRetryTimeout: &appmesh.Duration{
				Unit:  awssdk.String(appmesh.DurationUnitMs),
				Value: awssdk.Int64(defaultPerRetryTimeoutMillis + 1),
			},
		}
		resultWithSamePerTryTimeoutAndDiffUnit = &appmesh.HttpRetryPolicy{
			PerRetryTimeout: &appmesh.Duration{
				Unit:  awssdk.String(appmesh.DurationUnitS),
				Value: awssdk.Int64(defaultPerRetryTimeoutMillis / 1000),
			},
		}

		specWithMaxRetries = &appmeshv1beta1.HttpRetryPolicy{
			MaxRetries: awssdk.Int64(defaultMaxRetries),
		}
		resultWithMaxRetries = &appmesh.HttpRetryPolicy{
			MaxRetries: awssdk.Int64(defaultMaxRetries),
		}
		resultWithDifferentMaxRetries = &appmesh.HttpRetryPolicy{
			MaxRetries: awssdk.Int64(defaultMaxRetries + 1),
		}

		specWithTcpEvent = &appmeshv1beta1.HttpRetryPolicy{
			TcpRetryPolicyEvents: []appmeshv1beta1.TcpRetryPolicyEvent{
				appmeshv1beta1.TcpRetryPolicyEvent(defaultTcpRetryPolicyEvent),
			},
		}
		resultWithTcpEvent = &appmesh.HttpRetryPolicy{
			TcpRetryEvents: []*string{
				awssdk.String(defaultTcpRetryPolicyEvent),
			},
		}
		resultWithDifferentTcpEvent = &appmesh.HttpRetryPolicy{
			TcpRetryEvents: []*string{
				awssdk.String("diff-tcp-error"),
			},
		}
	)

	var tests = []struct {
		name      string
		desired   *appmeshv1beta1.HttpRetryPolicy
		target    *appmesh.HttpRetryPolicy
		different bool
	}{
		{"Nil spec", nilSpec, nilResult, false},
		{"Empty spec", emptySpec, emptyResult, false},

		{"PerTryTimeout: match", specWithPerTryTimeout, resultWithPerTryTimeout, false},
		{"PerTryTimeout: match with different unit", specWithPerTryTimeout, resultWithSamePerTryTimeoutAndDiffUnit, false},
		{"PerTryTimeout: missing in desired", emptySpec, resultWithPerTryTimeout, true},
		{"PerTryTimeout: missing in target", specWithPerTryTimeout, emptyResult, true},
		{"PerTryTimeout: diff", specWithPerTryTimeout, resultWithDifferentPerTryTimeout, true},

		{"MaxRetries: match", specWithMaxRetries, resultWithMaxRetries, false},
		{"MaxRetries: missing in desired", emptySpec, resultWithMaxRetries, true},
		{"MaxRetries: missing in target", specWithMaxRetries, emptyResult, true},
		{"MaxRetries: diff", specWithMaxRetries, resultWithDifferentMaxRetries, true},

		{"HttpRetryPolicyEvent: match", specWithHttpEvent, resultWithHttpEvent, false},
		{"HttpRetryPolicyEvent: missing in desired", emptySpec, resultWithHttpEvent, true},
		{"HttpRetryPolicyEvent: missing in target", specWithHttpEvent, emptyResult, true},
		{"HttpRetryPolicyEvent: diff", specWithHttpEvent, resultWithDifferentHttpEvent, true},

		{"TcpRetryPolicyEvent: match", specWithTcpEvent, resultWithTcpEvent, false},
		{"TcpRetryPolicyEvent: missing in desired", emptySpec, resultWithTcpEvent, true},
		{"TcpRetryPolicyEvent: missing in target", specWithTcpEvent, emptyResult, true},
		{"TcpRetryPolicyEvent: diff", specWithTcpEvent, resultWithDifferentTcpEvent, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := newAPIHttp2Route(defaultRouteName, defaultPrefix, defaultTargets)
			spec.Http2.RetryPolicy = tt.desired
			result := newAWSHttp2Route(defaultRouteName, defaultPrefix, defaultTargets)
			result.Data.Spec.Http2Route.RetryPolicy = tt.target
			if res := routeNeedsUpdate(spec, result); res != tt.different {
				t.Errorf("got %v, want %v", res, tt.different)
			}
		})
	}
}

func TestGrpcRouteWithRetryPolicyNeedUpdate(t *testing.T) {
	var (
		// shared defaults
		defaultRouteName             = "example-route"
		defaultServiceName           = "service-name"
		defaultMethodName            = "method-name"
		defaultNodeName              = "example-node"
		defaultPerRetryTimeoutMillis = int64(1000)
		defaultMaxRetries            = int64(5)
		defaultGrpcRetryPolicyEvent  = "unavailable"
		defaultHttpRetryPolicyEvent  = "http-error"
		defaultTcpRetryPolicyEvent   = appmesh.TcpRetryPolicyEventConnectionError

		// Targets for default custom resource spec
		defaultTargets = []appmeshv1beta1.WeightedTarget{
			{Weight: int64(1), VirtualNodeName: defaultNodeName},
		}

		nilSpec   *appmeshv1beta1.GrpcRetryPolicy
		nilResult *appmesh.GrpcRetryPolicy

		emptySpec   = &appmeshv1beta1.GrpcRetryPolicy{}
		emptyResult = &appmesh.GrpcRetryPolicy{}

		specWithHttpEvent = &appmeshv1beta1.GrpcRetryPolicy{
			HttpRetryPolicyEvents: []appmeshv1beta1.HttpRetryPolicyEvent{
				appmeshv1beta1.HttpRetryPolicyEvent(defaultHttpRetryPolicyEvent),
			},
		}
		resultWithHttpEvent = &appmesh.GrpcRetryPolicy{
			HttpRetryEvents: []*string{
				awssdk.String(defaultHttpRetryPolicyEvent),
			},
		}
		resultWithDifferentHttpEvent = &appmesh.GrpcRetryPolicy{
			HttpRetryEvents: []*string{
				awssdk.String("diff-http-error"),
			},
		}

		specWithGrpcEvent = &appmeshv1beta1.GrpcRetryPolicy{
			GrpcRetryPolicyEvents: []appmeshv1beta1.GrpcRetryPolicyEvent{
				appmeshv1beta1.GrpcRetryPolicyEvent(defaultGrpcRetryPolicyEvent),
			},
		}
		resultWithGrpcEvent = &appmesh.GrpcRetryPolicy{
			GrpcRetryEvents: []*string{
				awssdk.String(defaultGrpcRetryPolicyEvent),
			},
		}
		resultWithDifferentGrpcEvent = &appmesh.GrpcRetryPolicy{
			GrpcRetryEvents: []*string{
				awssdk.String("cancelled"),
			},
		}

		specWithGrpcPerTryTimeout = &appmeshv1beta1.GrpcRetryPolicy{
			PerRetryTimeoutMillis: awssdk.Int64(defaultPerRetryTimeoutMillis),
		}
		resultWithGrpcPerTryTimeout = &appmesh.GrpcRetryPolicy{
			PerRetryTimeout: &appmesh.Duration{
				Unit:  awssdk.String(appmesh.DurationUnitMs),
				Value: awssdk.Int64(defaultPerRetryTimeoutMillis),
			},
		}
		resultWithDifferentGrpcPerTryTimeout = &appmesh.GrpcRetryPolicy{
			PerRetryTimeout: &appmesh.Duration{
				Unit:  awssdk.String(appmesh.DurationUnitMs),
				Value: awssdk.Int64(defaultPerRetryTimeoutMillis + 1),
			},
		}
		resultWithSameGrpcPerTryTimeoutAndDiffUnit = &appmesh.GrpcRetryPolicy{
			PerRetryTimeout: &appmesh.Duration{
				Unit:  awssdk.String(appmesh.DurationUnitS),
				Value: awssdk.Int64(defaultPerRetryTimeoutMillis / 1000),
			},
		}

		specWithGrpcMaxRetries = &appmeshv1beta1.GrpcRetryPolicy{
			MaxRetries: awssdk.Int64(defaultMaxRetries),
		}
		resultWithGrpcMaxRetries = &appmesh.GrpcRetryPolicy{
			MaxRetries: awssdk.Int64(defaultMaxRetries),
		}
		resultWithDifferentGrpcMaxRetries = &appmesh.GrpcRetryPolicy{
			MaxRetries: awssdk.Int64(defaultMaxRetries + 1),
		}

		specWithTcpEvent = &appmeshv1beta1.GrpcRetryPolicy{
			TcpRetryPolicyEvents: []appmeshv1beta1.TcpRetryPolicyEvent{
				appmeshv1beta1.TcpRetryPolicyEvent(defaultTcpRetryPolicyEvent),
			},
		}
		resultWithTcpEvent = &appmesh.GrpcRetryPolicy{
			TcpRetryEvents: []*string{
				awssdk.String(defaultTcpRetryPolicyEvent),
			},
		}
		resultWithDifferentTcpEvent = &appmesh.GrpcRetryPolicy{
			TcpRetryEvents: []*string{
				awssdk.String("diff-tcp-error"),
			},
		}
	)

	var tests = []struct {
		name      string
		desired   *appmeshv1beta1.GrpcRetryPolicy
		target    *appmesh.GrpcRetryPolicy
		different bool
	}{
		{"Nil spec", nilSpec, nilResult, false},
		{"Empty spec", emptySpec, emptyResult, false},

		{"HttpRetryPolicyEvent: match", specWithHttpEvent, resultWithHttpEvent, false},
		{"HttpRetryPolicyEvent: missing in desired", emptySpec, resultWithHttpEvent, true},
		{"HttpRetryPolicyEvent: missing in target", specWithHttpEvent, emptyResult, true},
		{"HttpRetryPolicyEvent: diff", specWithHttpEvent, resultWithDifferentHttpEvent, true},

		{"PerTryTimeout: match", specWithGrpcPerTryTimeout, resultWithGrpcPerTryTimeout, false},
		{"PerTryTimeout: match with different unit", specWithGrpcPerTryTimeout, resultWithSameGrpcPerTryTimeoutAndDiffUnit, false},
		{"PerTryTimeout: missing in desired", emptySpec, resultWithGrpcPerTryTimeout, true},
		{"PerTryTimeout: missing in target", specWithGrpcPerTryTimeout, emptyResult, true},
		{"PerTryTimeout: diff", specWithGrpcPerTryTimeout, resultWithDifferentGrpcPerTryTimeout, true},

		{"MaxRetries: match", specWithGrpcMaxRetries, resultWithGrpcMaxRetries, false},
		{"MaxRetries: missing in desired", emptySpec, resultWithGrpcMaxRetries, true},
		{"MaxRetries: missing in target", specWithGrpcMaxRetries, emptyResult, true},
		{"MaxRetries: diff", specWithGrpcMaxRetries, resultWithDifferentGrpcMaxRetries, true},

		{"GrpcRetryPolicyEvent: match", specWithGrpcEvent, resultWithGrpcEvent, false},
		{"GrpcRetryPolicyEvent: missing in desired", emptySpec, resultWithGrpcEvent, true},
		{"GrpcRetryPolicyEvent: missing in target", specWithGrpcEvent, emptyResult, true},
		{"GrpcRetryPolicyEvent: diff", specWithGrpcEvent, resultWithDifferentGrpcEvent, true},

		{"TcpRetryPolicyEvent: match", specWithTcpEvent, resultWithTcpEvent, false},
		{"TcpRetryPolicyEvent: missing in desired", emptySpec, resultWithTcpEvent, true},
		{"TcpRetryPolicyEvent: missing in target", specWithTcpEvent, emptyResult, true},
		{"TcpRetryPolicyEvent: diff", specWithTcpEvent, resultWithDifferentTcpEvent, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := newAPIGrpcRoute(defaultRouteName, defaultServiceName, defaultMethodName, defaultTargets)
			spec.Grpc.RetryPolicy = tt.desired
			result := newAWSGrpcRoute(defaultRouteName, defaultServiceName, defaultMethodName, defaultTargets)
			result.Data.Spec.GrpcRoute.RetryPolicy = tt.target
			if res := routeNeedsUpdate(spec, result); res != tt.different {
				t.Errorf("got %v, want %v", res, tt.different)
			}
		})
	}
}
