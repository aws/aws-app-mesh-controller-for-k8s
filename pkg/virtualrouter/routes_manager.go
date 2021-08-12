package virtualrouter

import (
	"context"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws/services"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/conversions"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/k8s"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/references"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	appmeshsdk "github.com/aws/aws-sdk-go/service/appmesh"
	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
)

// routesManager is responsible for manage routes for virtualRouter.
type routesManager interface {
	// create will create routes on AppMesh virtualRouter to match k8s virtualRouter spec.
	create(ctx context.Context, ms *appmesh.Mesh, vr *appmesh.VirtualRouter, vnByRefHash map[types.NamespacedName]*appmesh.VirtualNode) (map[string]*appmeshsdk.RouteData, error)
	// update will update routes on AppMesh virtualRouter to match k8s virtualRouter spec.
	update(ctx context.Context, ms *appmesh.Mesh, vr *appmesh.VirtualRouter, vnByRefHash map[types.NamespacedName]*appmesh.VirtualNode) (map[string]*appmeshsdk.RouteData, error)
	// cleanup will cleanup routes on AppMesh virtualRouter
	cleanup(ctx context.Context, ms *appmesh.Mesh, vr *appmesh.VirtualRouter) error
}

// newDefaultRoutesManager constructs new routesManager
func newDefaultRoutesManager(appMeshSDK services.AppMesh, log logr.Logger) routesManager {
	return &defaultRoutesManager{
		appMeshSDK: appMeshSDK,
		log:        log,
	}
}

type defaultRoutesManager struct {
	appMeshSDK services.AppMesh
	log        logr.Logger
}

func (m *defaultRoutesManager) create(ctx context.Context, ms *appmesh.Mesh, vr *appmesh.VirtualRouter, vnByKey map[types.NamespacedName]*appmesh.VirtualNode) (map[string]*appmeshsdk.RouteData, error) {
	return m.reconcile(ctx, ms, vr, vnByKey, vr.Spec.Routes, nil)
}

func (m *defaultRoutesManager) update(ctx context.Context, ms *appmesh.Mesh, vr *appmesh.VirtualRouter, vnByKey map[types.NamespacedName]*appmesh.VirtualNode) (map[string]*appmeshsdk.RouteData, error) {
	sdkRouteRefs, err := m.listSDKRouteRefs(ctx, ms, vr)
	if err != nil {
		return nil, err
	}
	return m.reconcile(ctx, ms, vr, vnByKey, vr.Spec.Routes, sdkRouteRefs)
}

func (m *defaultRoutesManager) cleanup(ctx context.Context, ms *appmesh.Mesh, vr *appmesh.VirtualRouter) error {
	sdkRouteRefs, err := m.listSDKRouteRefs(ctx, ms, vr)
	if err != nil {
		return err
	}
	_, err = m.reconcile(ctx, ms, vr, nil, nil, sdkRouteRefs)
	return err
}

// reconcile will make AppMesh routes(sdkRouteRefs) matches routes.
func (m *defaultRoutesManager) reconcile(ctx context.Context, ms *appmesh.Mesh, vr *appmesh.VirtualRouter, vnByKey map[types.NamespacedName]*appmesh.VirtualNode,
	routes []appmesh.Route, sdkRouteRefs []*appmeshsdk.RouteRef) (map[string]*appmeshsdk.RouteData, error) {

	matchedRouteAndSDKRouteRefs, unmatchedRoutes, unmatchedSDKRouteRefs := matchRoutesAgainstSDKRouteRefs(routes, sdkRouteRefs)
	sdkRouteByName := make(map[string]*appmeshsdk.RouteData, len(matchedRouteAndSDKRouteRefs)+len(unmatchedRoutes))

	for _, route := range unmatchedRoutes {
		sdkRoute, err := m.createSDKRoute(ctx, ms, vr, route, vnByKey)
		if err != nil {
			return nil, err
		}
		sdkRouteByName[route.Name] = sdkRoute
	}

	for _, routeAndSDKRouteRef := range matchedRouteAndSDKRouteRefs {
		route := routeAndSDKRouteRef.route
		sdkRouteRef := routeAndSDKRouteRef.sdkRouteRef
		sdkRoute, err := m.findSDKRoute(ctx, sdkRouteRef)
		if err != nil {
			return nil, err
		}
		if sdkRoute == nil {
			return nil, errors.Errorf("route not found: %v", aws.StringValue(sdkRouteRef.RouteName))
		}
		sdkRoute, err = m.updateSDKRoute(ctx, sdkRoute, vr, route, vnByKey)
		if err != nil {
			return nil, err
		}
		sdkRouteByName[route.Name] = sdkRoute
	}

	for _, sdkRouteRef := range unmatchedSDKRouteRefs {
		sdkRoute, err := m.findSDKRoute(ctx, sdkRouteRef)
		if err != nil {
			return nil, err
		}
		if sdkRoute == nil {
			return nil, errors.Errorf("route not found: %v", aws.StringValue(sdkRouteRef.RouteName))
		}
		if err = m.deleteSDKRoute(ctx, sdkRoute); err != nil {
			return nil, err
		}
	}
	return sdkRouteByName, nil
}

func (m *defaultRoutesManager) listSDKRouteRefs(ctx context.Context, ms *appmesh.Mesh, vr *appmesh.VirtualRouter) ([]*appmeshsdk.RouteRef, error) {
	var sdkRouteRefs []*appmeshsdk.RouteRef
	if err := m.appMeshSDK.ListRoutesPagesWithContext(ctx, &appmeshsdk.ListRoutesInput{
		MeshName:          ms.Spec.AWSName,
		MeshOwner:         ms.Spec.MeshOwner,
		VirtualRouterName: vr.Spec.AWSName,
	}, func(output *appmeshsdk.ListRoutesOutput, b bool) bool {
		sdkRouteRefs = append(sdkRouteRefs, output.Routes...)
		return true
	}); err != nil {
		return nil, err
	}
	return sdkRouteRefs, nil
}

func (m *defaultRoutesManager) findSDKRoute(ctx context.Context, sdkRouteRef *appmeshsdk.RouteRef) (*appmeshsdk.RouteData, error) {
	resp, err := m.appMeshSDK.DescribeRouteWithContext(ctx, &appmeshsdk.DescribeRouteInput{
		MeshName:          sdkRouteRef.MeshName,
		MeshOwner:         sdkRouteRef.MeshOwner,
		VirtualRouterName: sdkRouteRef.VirtualRouterName,
		RouteName:         sdkRouteRef.RouteName,
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "NotFoundException" {
			return nil, nil
		}
		return nil, err
	}
	return resp.Route, nil
}

func (m *defaultRoutesManager) createSDKRoute(ctx context.Context, ms *appmesh.Mesh, vr *appmesh.VirtualRouter, route appmesh.Route, vnByKey map[types.NamespacedName]*appmesh.VirtualNode) (*appmeshsdk.RouteData, error) {
	sdkRouteSpec, err := BuildSDKRouteSpec(vr, route, vnByKey)
	if err != nil {
		return nil, err
	}

	resp, err := m.appMeshSDK.CreateRouteWithContext(ctx, &appmeshsdk.CreateRouteInput{
		MeshName:          ms.Spec.AWSName,
		MeshOwner:         ms.Spec.MeshOwner,
		VirtualRouterName: vr.Spec.AWSName,
		RouteName:         aws.String(route.Name),
		Spec:              sdkRouteSpec,
	})
	if err != nil {
		return nil, err
	}
	return resp.Route, nil
}

func (m *defaultRoutesManager) updateSDKRoute(ctx context.Context, sdkRoute *appmeshsdk.RouteData, vr *appmesh.VirtualRouter, route appmesh.Route, vnByKey map[types.NamespacedName]*appmesh.VirtualNode) (*appmeshsdk.RouteData, error) {
	actualSDKRouteSpec := sdkRoute.Spec
	desiredSDKRouteSpec, err := BuildSDKRouteSpec(vr, route, vnByKey)
	if err != nil {
		return nil, err
	}

	opts := cmpopts.EquateEmpty()
	if cmp.Equal(desiredSDKRouteSpec, actualSDKRouteSpec, opts) {
		return sdkRoute, nil
	}
	diff := cmp.Diff(desiredSDKRouteSpec, actualSDKRouteSpec, opts)
	m.log.V(1).Info("routeSpec changed",
		"virtualRouter", k8s.NamespacedName(vr),
		"route", route.Name,
		"actualSDKRouteSpec", actualSDKRouteSpec,
		"desiredSDKRouteSpec", desiredSDKRouteSpec,
		"diff", diff,
	)
	resp, err := m.appMeshSDK.UpdateRouteWithContext(ctx, &appmeshsdk.UpdateRouteInput{
		MeshName:          sdkRoute.MeshName,
		MeshOwner:         sdkRoute.Metadata.MeshOwner,
		VirtualRouterName: sdkRoute.VirtualRouterName,
		RouteName:         sdkRoute.RouteName,
		Spec:              desiredSDKRouteSpec,
	})
	if err != nil {
		return nil, err
	}
	return resp.Route, nil
}

func (m *defaultRoutesManager) deleteSDKRoute(ctx context.Context, sdkRoute *appmeshsdk.RouteData) error {
	_, err := m.appMeshSDK.DeleteRouteWithContext(ctx, &appmeshsdk.DeleteRouteInput{
		MeshName:          sdkRoute.MeshName,
		MeshOwner:         sdkRoute.Metadata.MeshOwner,
		VirtualRouterName: sdkRoute.VirtualRouterName,
		RouteName:         sdkRoute.RouteName,
	})
	if err != nil {
		return err
	}
	return nil
}

type routeAndSDKRouteRef struct {
	route       appmesh.Route
	sdkRouteRef *appmeshsdk.RouteRef
}

// matchRoutesAgainstSDKRouteRefs will match routes against sdkRouteRefs.
// return matched routeAndSDKRouteRef, not matched routes and not matched sdkRouteRefs
func matchRoutesAgainstSDKRouteRefs(routes []appmesh.Route, sdkRouteRefs []*appmeshsdk.RouteRef) ([]routeAndSDKRouteRef, []appmesh.Route, []*appmeshsdk.RouteRef) {
	routeByName := make(map[string]appmesh.Route, len(routes))
	sdkRouteRefByName := make(map[string]*appmeshsdk.RouteRef, len(sdkRouteRefs))
	for _, route := range routes {
		routeByName[route.Name] = route
	}
	for _, sdkRouteRef := range sdkRouteRefs {
		sdkRouteRefByName[aws.StringValue(sdkRouteRef.RouteName)] = sdkRouteRef
	}
	routeNameSet := sets.StringKeySet(routeByName)
	sdkRouteRefNameSet := sets.StringKeySet(sdkRouteRefByName)
	matchedNameSet := routeNameSet.Intersection(sdkRouteRefNameSet)
	unmatchedRouteNameSet := routeNameSet.Difference(sdkRouteRefNameSet)
	unmatchedSDKRouteRefNameSet := sdkRouteRefNameSet.Difference(routeNameSet)

	matchedRouteAndSDKRouteRef := make([]routeAndSDKRouteRef, 0, len(matchedNameSet))
	for _, name := range matchedNameSet.List() {
		matchedRouteAndSDKRouteRef = append(matchedRouteAndSDKRouteRef, routeAndSDKRouteRef{
			route:       routeByName[name],
			sdkRouteRef: sdkRouteRefByName[name],
		})
	}

	unmatchedRoutes := make([]appmesh.Route, 0, len(unmatchedRouteNameSet))
	for _, name := range unmatchedRouteNameSet.List() {
		unmatchedRoutes = append(unmatchedRoutes, routeByName[name])
	}

	unmatchedSDKRouteRefs := make([]*appmeshsdk.RouteRef, 0, len(unmatchedSDKRouteRefNameSet))
	for _, name := range unmatchedSDKRouteRefNameSet.List() {
		unmatchedSDKRouteRefs = append(unmatchedSDKRouteRefs, sdkRouteRefByName[name])
	}

	return matchedRouteAndSDKRouteRef, unmatchedRoutes, unmatchedSDKRouteRefs
}

func BuildSDKRouteSpec(vr *appmesh.VirtualRouter, route appmesh.Route, vnByKey map[types.NamespacedName]*appmesh.VirtualNode) (*appmeshsdk.RouteSpec, error) {
	sdkVNRefConvertFunc := references.BuildSDKVirtualNodeReferenceConvertFunc(vr, vnByKey)
	converter := conversion.NewConverter(conversion.DefaultNameFunc)
	converter.RegisterUntypedConversionFunc((*appmesh.Route)(nil), (*appmeshsdk.RouteSpec)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return conversions.Convert_CRD_Route_To_SDK_RouteSpec(a.(*appmesh.Route), b.(*appmeshsdk.RouteSpec), scope)
	})
	converter.RegisterUntypedConversionFunc((*appmesh.VirtualNodeReference)(nil), (*string)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return sdkVNRefConvertFunc(a.(*appmesh.VirtualNodeReference), b.(*string), scope)
	})
	sdkRouteSpec := &appmeshsdk.RouteSpec{}
	if err := converter.Convert(&route, sdkRouteSpec, nil); err != nil {
		return nil, err
	}
	return sdkRouteSpec, nil
}
