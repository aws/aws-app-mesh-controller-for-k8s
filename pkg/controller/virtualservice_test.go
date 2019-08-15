package controller

import (
	"testing"

	appmeshv1beta1 "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/apis/appmesh/v1beta1"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws"
	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/appmesh"
)

// newAWSVirtualService is a helper function to generate an Kubernetes Custom Resource API object.
func newAPIVirtualService(meshName string, virtualRouter *appmeshv1beta1.VirtualRouter, routes []appmeshv1beta1.Route) appmeshv1beta1.VirtualService {
	vs := appmeshv1beta1.VirtualService{
		Spec: appmeshv1beta1.VirtualServiceSpec{
			MeshName:      meshName,
			VirtualRouter: virtualRouter,
		},
	}

	if routes != nil && len(routes) > 0 {
		vs.Spec.Routes = routes
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
			Listeners: []appmeshv1beta1.Listener{
				appmeshv1beta1.Listener{
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
