package conversions

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-sdk-go/aws"
	appmeshsdk "github.com/aws/aws-sdk-go/service/appmesh"
	"k8s.io/apimachinery/pkg/conversion"
)

func Convert_CRD_GatewayRouteVirtualService_To_SDK_GatewayRouteVirtualService(crdObj *appmesh.GatewayRouteVirtualService,
	sdkObj *appmeshsdk.GatewayRouteVirtualService, scope conversion.Scope) error {

	sdkObj.VirtualServiceName = aws.String("")
	if err := scope.Convert(crdObj.VirtualServiceRef, sdkObj.VirtualServiceName, scope.Flags()); err != nil {
		return err
	}
	return nil
}

func Convert_CRD_GatewayRouteTarget_To_SDK_GatewayRouteTarget(crdObj *appmesh.GatewayRouteTarget, sdkObj *appmeshsdk.GatewayRouteTarget, scope conversion.Scope) error {
	sdkObj.VirtualService = &appmeshsdk.GatewayRouteVirtualService{}
	if err := Convert_CRD_GatewayRouteVirtualService_To_SDK_GatewayRouteVirtualService(&crdObj.VirtualService, sdkObj.VirtualService, scope); err != nil {
		return err
	}
	return nil
}

func Convert_CRD_GRPCGatewayRouteAction_To_SDK_GrpcGatewayRouteAction(crdObj *appmesh.GRPCGatewayRouteAction,
	sdkObj *appmeshsdk.GrpcGatewayRouteAction, scope conversion.Scope) error {

	sdkObj.Target = &appmeshsdk.GatewayRouteTarget{}
	if err := Convert_CRD_GatewayRouteTarget_To_SDK_GatewayRouteTarget(&crdObj.Target, sdkObj.Target, scope); err != nil {
		return err
	}
	return nil
}

func Convert_CRD_GRPCGatewayRouteMatch_To_SDK_GrpcGatewayRouteMatch(crdObj *appmesh.GRPCGatewayRouteMatch, sdkObj *appmeshsdk.GrpcGatewayRouteMatch) error {
	sdkObj.ServiceName = crdObj.ServiceName
	return nil
}

func Convert_CRD_GRPCGatewayRoute_To_SDK_GrpcGatewayRoute(crdObj *appmesh.GRPCGatewayRoute, sdkObj *appmeshsdk.GrpcGatewayRoute, scope conversion.Scope) error {
	sdkObj.Match = &appmeshsdk.GrpcGatewayRouteMatch{}
	if err := Convert_CRD_GRPCGatewayRouteMatch_To_SDK_GrpcGatewayRouteMatch(&crdObj.Match, sdkObj.Match); err != nil {
		return err
	}
	sdkObj.Action = &appmeshsdk.GrpcGatewayRouteAction{}
	if err := Convert_CRD_GRPCGatewayRouteAction_To_SDK_GrpcGatewayRouteAction(&crdObj.Action, sdkObj.Action, scope); err != nil {
		return err
	}
	return nil
}

func Convert_CRD_HTTPGatewayRouteAction_To_SDK_HttpGatewayRouteAction(crdObj *appmesh.HTTPGatewayRouteAction,
	sdkObj *appmeshsdk.HttpGatewayRouteAction, scope conversion.Scope) error {

	sdkObj.Target = &appmeshsdk.GatewayRouteTarget{}
	if err := Convert_CRD_GatewayRouteTarget_To_SDK_GatewayRouteTarget(&crdObj.Target, sdkObj.Target, scope); err != nil {
		return err
	}
	return nil
}

func Convert_CRD_HTTPGatewayRouteMatch_To_SDK_HttpGatewayRouteMatch(crdObj *appmesh.HTTPGatewayRouteMatch, sdkObj *appmeshsdk.HttpGatewayRouteMatch) error {
	sdkObj.Prefix = crdObj.Prefix
	return nil
}

func Convert_CRD_HTTPGatewayRoute_To_SDK_HttpGatewayRoute(crdObj *appmesh.HTTPGatewayRoute, sdkObj *appmeshsdk.HttpGatewayRoute, scope conversion.Scope) error {
	sdkObj.Match = &appmeshsdk.HttpGatewayRouteMatch{}
	if err := Convert_CRD_HTTPGatewayRouteMatch_To_SDK_HttpGatewayRouteMatch(&crdObj.Match, sdkObj.Match); err != nil {
		return err
	}
	sdkObj.Action = &appmeshsdk.HttpGatewayRouteAction{}
	if err := Convert_CRD_HTTPGatewayRouteAction_To_SDK_HttpGatewayRouteAction(&crdObj.Action, sdkObj.Action, scope); err != nil {
		return err
	}
	return nil
}

func Convert_CRD_GatewayRouteSpec_To_SDK_GatewayRouteSpec(crdObj *appmesh.GatewayRouteSpec, sdkObj *appmeshsdk.GatewayRouteSpec, scope conversion.Scope) error {
	if crdObj.HTTPRoute != nil {
		sdkObj.HttpRoute = &appmeshsdk.HttpGatewayRoute{}
		if err := Convert_CRD_HTTPGatewayRoute_To_SDK_HttpGatewayRoute(crdObj.HTTPRoute, sdkObj.HttpRoute, scope); err != nil {
			return err
		}
	} else {
		sdkObj.HttpRoute = nil
	}

	if crdObj.HTTP2Route != nil {
		sdkObj.Http2Route = &appmeshsdk.HttpGatewayRoute{}
		if err := Convert_CRD_HTTPGatewayRoute_To_SDK_HttpGatewayRoute(crdObj.HTTP2Route, sdkObj.Http2Route, scope); err != nil {
			return err
		}
	} else {
		sdkObj.Http2Route = nil
	}

	if crdObj.GRPCRoute != nil {
		sdkObj.GrpcRoute = &appmeshsdk.GrpcGatewayRoute{}
		if err := Convert_CRD_GRPCGatewayRoute_To_SDK_GrpcGatewayRoute(crdObj.GRPCRoute, sdkObj.GrpcRoute, scope); err != nil {
			return err
		}
	} else {
		sdkObj.GrpcRoute = nil
	}

	return nil
}
