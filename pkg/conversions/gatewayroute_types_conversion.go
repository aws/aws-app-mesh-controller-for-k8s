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
	if crdObj.VirtualServiceRef != nil {
		if err := scope.Convert(crdObj.VirtualServiceRef, sdkObj.VirtualServiceName, scope.Flags()); err != nil {
			return err
		}
	}
	if crdObj.VirtualServiceARN != nil {
		if err := Convert_CRD_VirtualServiceARN_To_SDK_VirtualServiceName(crdObj.VirtualServiceARN, sdkObj.VirtualServiceName, scope); err != nil {
			return err
		}
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

	if crdObj.Rewrite != nil {
		sdkObj.Rewrite = &appmeshsdk.GrpcGatewayRouteRewrite{}
		Convert_CRD_GRPCHostnameRewrite_To_SDK_GrpcHostnameRewrite(crdObj.Rewrite, sdkObj.Rewrite)
	}
	return nil
}

func Convert_CRD_GRPCHostnameRewrite_To_SDK_GrpcHostnameRewrite(crdObj *appmesh.GrpcGatewayRouteRewrite, sdkObj *appmeshsdk.GrpcGatewayRouteRewrite) {
	sdkObj.Hostname = &appmeshsdk.GatewayRouteHostnameRewrite{
		DefaultTargetHostname: crdObj.Hostname.DefaultTargetHostname,
	}
}

func Convert_CRD_GRPCGatewayRouteMatch_To_SDK_GrpcGatewayRouteMatch(crdObj *appmesh.GRPCGatewayRouteMatch, sdkObj *appmeshsdk.GrpcGatewayRouteMatch) error {
	sdkObj.ServiceName = crdObj.ServiceName
	if crdObj.Hostname != nil {
		sdkObj.Hostname = &appmeshsdk.GatewayRouteHostnameMatch{}
		Convert_CRD_GatewayRouteHostnameMatch_To_SDK_GatewayRouteHostnameMatch(crdObj.Hostname, sdkObj.Hostname)
	}
	if crdObj.Metadata != nil && len(crdObj.Metadata) != 0 {
		if err := Convert_CRD_GRPCGatewayRouteMetadata_To_SDK_GrpcGatewayRouteMetadata(crdObj, sdkObj); err != nil {
			return err
		}
	}
	return nil
}

func Convert_CRD_GRPCGatewayRouteMetadata_To_SDK_GrpcGatewayRouteMetadata(crdObj *appmesh.GRPCGatewayRouteMatch, sdkObj *appmeshsdk.GrpcGatewayRouteMatch) error {
	length := len(crdObj.Metadata)
	sdkMetadataList := make([]*appmeshsdk.GrpcGatewayRouteMetadata, 0, length)
	for _, metadata := range crdObj.Metadata {
		sdkMetadata := &appmeshsdk.GrpcGatewayRouteMetadata{}
		sdkMetadata.Name = metadata.Name
		if metadata.Match != nil {
			sdkMetadata.Match = &appmeshsdk.GrpcMetadataMatchMethod{}
			if err := Convert_CRD_GrpcMetdataMatchMethod_To_SDK_GrpcMetadataMatchMethod(metadata.Match, sdkMetadata.Match); err != nil {
				return err
			}
		}
		sdkMetadata.Invert = metadata.Invert
		sdkMetadataList = append(sdkMetadataList, sdkMetadata)
	}
	sdkObj.Metadata = sdkMetadataList
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
	if crdObj.Rewrite != nil {
		Convert_CRD_HTTPGatewayRouteRewrite_To_SDK_HttpGatewayRouteRewrite(crdObj, sdkObj)
	}
	return nil
}

func Convert_CRD_HTTPGatewayRouteRewrite_To_SDK_HttpGatewayRouteRewrite(crdObj *appmesh.HTTPGatewayRouteAction, sdkObj *appmeshsdk.HttpGatewayRouteAction) {
	sdkObj.Rewrite = &appmeshsdk.HttpGatewayRouteRewrite{}

	if crdObj.Rewrite.Prefix != nil {
		Convert_CRD_HTTPGatewayRouteRewritePrefix_To_SDK_HttpGatewayRouteRewritePrefix(crdObj.Rewrite, sdkObj.Rewrite)
	}

	if crdObj.Rewrite.Hostname != nil {
		Convert_CRD_HTTPGatewayRouteRewriteHostname_To_SDK_HttpGatewayRouteRewriteHostname(crdObj.Rewrite, sdkObj.Rewrite)
	}

	if crdObj.Rewrite.Path != nil {
		Convert_CRD_HTTPGatewayRouteRewritePath_To_SDK_HttpGatewayRouteRewritePath(crdObj.Rewrite, sdkObj.Rewrite)
	}
}

func Convert_CRD_HTTPGatewayRouteRewritePrefix_To_SDK_HttpGatewayRouteRewritePrefix(crdObj *appmesh.HTTPGatewayRouteRewrite, sdkObj *appmeshsdk.HttpGatewayRouteRewrite) {
	sdkObj.Prefix = &appmeshsdk.HttpGatewayRoutePrefixRewrite{
		DefaultPrefix: crdObj.Prefix.DefaultPrefix,
		Value:         crdObj.Prefix.Value,
	}
}

func Convert_CRD_HTTPGatewayRouteRewriteHostname_To_SDK_HttpGatewayRouteRewriteHostname(crdObj *appmesh.HTTPGatewayRouteRewrite, sdkObj *appmeshsdk.HttpGatewayRouteRewrite) {
	sdkObj.Hostname = &appmeshsdk.GatewayRouteHostnameRewrite{
		DefaultTargetHostname: crdObj.Hostname.DefaultTargetHostname,
	}
}

func Convert_CRD_HTTPGatewayRouteRewritePath_To_SDK_HttpGatewayRouteRewritePath(crdObj *appmesh.HTTPGatewayRouteRewrite, sdkObj *appmeshsdk.HttpGatewayRouteRewrite) {
	sdkObj.Path = &appmeshsdk.HttpGatewayRoutePathRewrite{Exact: crdObj.Path.Exact}
}

func Convert_CRD_HTTPGatewayRouteMatch_To_SDK_HttpGatewayRouteMatch(crdObj *appmesh.HTTPGatewayRouteMatch, sdkObj *appmeshsdk.HttpGatewayRouteMatch) error {
	sdkObj.Prefix = crdObj.Prefix
	if crdObj.Hostname != nil {
		sdkObj.Hostname = &appmeshsdk.GatewayRouteHostnameMatch{}
		Convert_CRD_GatewayRouteHostnameMatch_To_SDK_GatewayRouteHostnameMatch(crdObj.Hostname, sdkObj.Hostname)
	}
	sdkObj.Method = crdObj.Method
	if crdObj.Headers != nil && len(crdObj.Headers) != 0 {
		if err := Convert_CRD_HTTPGatewayRouteHeaders_To_SDK_HttpGatewayRouteHeaders(crdObj, sdkObj); err != nil {
			return err
		}
	}
	if crdObj.Path != nil {
		Convert_CRD_HTTPGatewayPath_To_SDK_HttpGatewayPath(crdObj, sdkObj)
	}
	if crdObj.QueryParameters != nil && len(crdObj.QueryParameters) != 0 {
		Convert_CRD_HTTPGatewayRouteQueryParams_To_SDK_HttpGatewayRouteQueryParams(crdObj, sdkObj)
	}
	return nil
}

func Convert_CRD_HTTPGatewayRouteQueryParams_To_SDK_HttpGatewayRouteQueryParams(crdObj *appmesh.HTTPGatewayRouteMatch, sdkObj *appmeshsdk.HttpGatewayRouteMatch) {
	length := len(crdObj.QueryParameters)
	sdkQueryParams := make([]*appmeshsdk.HttpQueryParameter, 0, length)
	for _, queryParam := range crdObj.QueryParameters {
		sdkQueryParam := &appmeshsdk.HttpQueryParameter{}
		sdkQueryParam.Name = queryParam.Name
		if queryParam.Match != nil {
			sdkQueryParam.Match = &appmeshsdk.QueryParameterMatch{Exact: queryParam.Match.Exact}
		}
		sdkQueryParams = append(sdkQueryParams, sdkQueryParam)
	}
	sdkObj.QueryParameters = sdkQueryParams
}

func Convert_CRD_HTTPGatewayPath_To_SDK_HttpGatewayPath(crdObj *appmesh.HTTPGatewayRouteMatch, sdkObj *appmeshsdk.HttpGatewayRouteMatch) {
	sdkObj.Path = &appmeshsdk.HttpPathMatch{
		Exact: crdObj.Path.Exact,
		Regex: crdObj.Path.Regex,
	}
}

func Convert_CRD_HTTPGatewayRouteHeaders_To_SDK_HttpGatewayRouteHeaders(crdObj *appmesh.HTTPGatewayRouteMatch, sdkObj *appmeshsdk.HttpGatewayRouteMatch) error {
	length := len(crdObj.Headers)
	sdkHeaders := make([]*appmeshsdk.HttpGatewayRouteHeader, 0, length)
	for _, header := range crdObj.Headers {
		sdkHeader := &appmeshsdk.HttpGatewayRouteHeader{}
		sdkHeader.Name = aws.String(header.Name)
		if header.Match != nil {
			sdkHeader.Match = &appmeshsdk.HeaderMatchMethod{}
			if err := Convert_CRD_HTTPHeaderMatchMethod_To_SDK_HttpHeaderMatchMethod(header.Match, sdkHeader.Match); err != nil {
				return err
			}
		}
		sdkHeader.Invert = header.Invert
		sdkHeaders = append(sdkHeaders, sdkHeader)
	}
	sdkObj.Headers = sdkHeaders
	return nil
}

func Convert_CRD_GatewayRouteHostnameMatch_To_SDK_GatewayRouteHostnameMatch(crdObj *appmesh.GatewayRouteHostnameMatch, sdkObj *appmeshsdk.GatewayRouteHostnameMatch) {
	sdkObj.Exact = crdObj.Exact
	sdkObj.Suffix = crdObj.Suffix
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
	}

	if crdObj.HTTP2Route != nil {
		sdkObj.Http2Route = &appmeshsdk.HttpGatewayRoute{}
		if err := Convert_CRD_HTTPGatewayRoute_To_SDK_HttpGatewayRoute(crdObj.HTTP2Route, sdkObj.Http2Route, scope); err != nil {
			return err
		}
	}

	if crdObj.GRPCRoute != nil {
		sdkObj.GrpcRoute = &appmeshsdk.GrpcGatewayRoute{}
		if err := Convert_CRD_GRPCGatewayRoute_To_SDK_GrpcGatewayRoute(crdObj.GRPCRoute, sdkObj.GrpcRoute, scope); err != nil {
			return err
		}
	}

	if crdObj.Priority != nil {
		sdkObj.Priority = crdObj.Priority
	}
	return nil
}
