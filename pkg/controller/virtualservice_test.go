package controller

import (
	"fmt"
	"reflect"
	"testing"

	appmeshv1beta1 "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/apis/appmesh/v1beta1"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws"
	ctrlawsmocks "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws/mocks"
	appmeshv1beta1mocks "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/client/clientset/versioned/mocks"
	appmeshv1beta1typedmocks "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/client/clientset/versioned/typed/appmesh/v1beta1/mocks"
	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/appmesh"
	"github.com/stretchr/testify/mock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func TestRouteNeedUpdate(t *testing.T) {

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

func TestGetVirtualRouter(t *testing.T) {
	var (
		defaultMeshName  = "example-mesh"
		defaultRouteName = "example-route"
		defaultPrefix    = "/"

		defaultHttpPort = appmeshv1beta1.PortMapping{
			Port:     8080,
			Protocol: appmeshv1beta1.PortProtocolHttp,
		}
		defaultTcpPort = appmeshv1beta1.PortMapping{
			Port:     6379,
			Protocol: appmeshv1beta1.PortProtocolTcp,
		}
		defaultHttpRouterListener = appmeshv1beta1.VirtualRouterListener{
			PortMapping: defaultHttpPort,
		}
		defaultTcpRouterListener = appmeshv1beta1.VirtualRouterListener{
			PortMapping: defaultTcpPort,
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
		defaultHttpRoute            = newAPIHttpRoute(defaultRouteName, defaultPrefix, []appmeshv1beta1.WeightedTarget{})
		defaultTcpRoute             = newAPITcpRoute(defaultRouteName, []appmeshv1beta1.WeightedTarget{})
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
			"virtual-service with router missing listener should get listener from target node of HTTP route",
			&defaultHttpRouterListener,
			&virtualRouterWithNoListener,
			&defaultHttpRoute,
			false,
			&defaultHttpListener,
		},

		{"3",
			"virtual-service with router missing listener and failed to load virtual-node of HTTP route",
			nil,
			&virtualRouterWithNoListener,
			&defaultHttpRoute,
			true,
			&defaultHttpListener,
		},

		{"4",
			"virtual-service with router missing listener should get listener from target node of TCP route",
			&defaultTcpRouterListener,
			&virtualRouterWithNoListener,
			&defaultTcpRoute,
			false,
			&defaultTcpListener,
		},

		{"5",
			"virtual-service with router missing listener and failed to load virtual-node of TCP route",
			nil,
			&virtualRouterWithNoListener,
			&defaultTcpRoute,
			true,
			&defaultTcpListener,
		},

		{"6",
			"virtual-service with router missing listener and no routes",
			nil,
			&virtualRouterWithNoListener,
			nil,
			false,
			nil,
		},

		{"7",
			"virtual-service with router missing listener and target node missing listener",
			nil,
			&virtualRouterWithNoListener,
			&defaultHttpRoute,
			false,
			nil,
		},

		{"8",
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
