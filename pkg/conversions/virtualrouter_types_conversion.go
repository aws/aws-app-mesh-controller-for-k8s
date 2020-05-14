package conversions

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-sdk-go/aws"
	appmeshsdk "github.com/aws/aws-sdk-go/service/appmesh"
	"k8s.io/apimachinery/pkg/conversion"
)

func Convert_CRD_VirtualRouterListener_To_SDK_VirtualRouterListener(crdObj *appmesh.VirtualRouterListener,
	sdkObj *appmeshsdk.VirtualRouterListener, scope conversion.Scope) error {

	sdkObj.PortMapping = &appmeshsdk.PortMapping{}
	return Convert_CRD_PortMapping_To_SDK_PortMapping(&crdObj.PortMapping, sdkObj.PortMapping, scope)
}

func Convert_CRD_WeightedTarget_To_SDK_WeightedTarget(crdObj *appmesh.WeightedTarget,
	sdkObj *appmeshsdk.WeightedTarget, scope conversion.Scope) error {

	sdkObj.VirtualNode = aws.String("")
	if err := scope.Convert(&crdObj.VirtualNodeRef, sdkObj.VirtualNode, scope.Flags()); err != nil {
		return err
	}
	sdkObj.Weight = aws.Int64(crdObj.Weight)
	return nil
}

func Convert_CRD_MatchRange_To_SDK_MatchRange(crdObj *appmesh.MatchRange,
	sdkObj *appmeshsdk.MatchRange, scope conversion.Scope) error {
	sdkObj.Start = aws.Int64(crdObj.Start)
	sdkObj.End = aws.Int64(crdObj.End)
	return nil
}

func Convert_CRD_HeaderMatchMethod_To_SDK_HeaderMatchMethod(crdObj *appmesh.HeaderMatchMethod,
	sdkObj *appmeshsdk.HeaderMatchMethod, scope conversion.Scope) error {

	sdkObj.Exact = crdObj.Exact
	sdkObj.Prefix = crdObj.Prefix

	if crdObj.Range != nil {

		sdkObj.Range = &appmeshsdk.MatchRange{}
		if err := Convert_CRD_MatchRange_To_SDK_MatchRange(crdObj.Range, sdkObj.Range, scope); err != nil {
			return err
		}
	} else {
		sdkObj.Range = nil
	}

	sdkObj.Regex = crdObj.Regex
	sdkObj.Suffix = crdObj.Suffix
	return nil
}

func Convert_CRD_HTTPRouteHeader_To_SDK_HttpRouteHeader(crdObj *appmesh.HTTPRouteHeader,
	sdkObj *appmeshsdk.HttpRouteHeader, scope conversion.Scope) error {

	sdkObj.Name = aws.String(crdObj.Name)

	if crdObj.Match != nil {
		sdkObj.Match = &appmeshsdk.HeaderMatchMethod{}
		if err := Convert_CRD_HeaderMatchMethod_To_SDK_HeaderMatchMethod(crdObj.Match, sdkObj.Match, scope); err != nil {
			return err
		}
	} else {
		sdkObj.Match = nil
	}

	sdkObj.Invert = crdObj.Invert
	return nil
}

func Convert_CRD_HTTPRouteMatch_To_SDK_HttpRouteMatch(crdObj *appmesh.HTTPRouteMatch,
	sdkObj *appmeshsdk.HttpRouteMatch, scope conversion.Scope) error {

	var sdkHeaders []*appmeshsdk.HttpRouteHeader
	if len(crdObj.Headers) != 0 {
		sdkHeaders = make([]*appmeshsdk.HttpRouteHeader, 0, len(crdObj.Headers))
		for _, crdHeader := range crdObj.Headers {
			sdkHeader := &appmeshsdk.HttpRouteHeader{}
			if err := Convert_CRD_HTTPRouteHeader_To_SDK_HttpRouteHeader(&crdHeader, sdkHeader, scope); err != nil {
				return err
			}
			sdkHeaders = append(sdkHeaders, sdkHeader)
		}
	}

	sdkObj.Headers = sdkHeaders
	sdkObj.Method = crdObj.Method
	sdkObj.Prefix = aws.String(crdObj.Prefix)
	sdkObj.Scheme = crdObj.Scheme
	return nil

}

func Convert_CRD_HTTPRouteAction_To_SDK_HttpRouteAction(crdObj *appmesh.HTTPRouteAction,
	sdkObj *appmeshsdk.HttpRouteAction, scope conversion.Scope) error {

	var sdkWeightedTargets []*appmeshsdk.WeightedTarget
	if len(crdObj.WeightedTargets) != 0 {
		sdkWeightedTargets = make([]*appmeshsdk.WeightedTarget, 0, len(crdObj.WeightedTargets))
		for _, crdWeightedTarget := range crdObj.WeightedTargets {
			sdkWeightedTarget := &appmeshsdk.WeightedTarget{}
			if err := Convert_CRD_WeightedTarget_To_SDK_WeightedTarget(&crdWeightedTarget, sdkWeightedTarget, scope); err != nil {
				return err
			}
			sdkWeightedTargets = append(sdkWeightedTargets, sdkWeightedTarget)
		}
	}
	sdkObj.WeightedTargets = sdkWeightedTargets
	return nil
}

func Convert_CRD_HTTPRetryPolicy_To_SDK_HttpRetryPolicy(crdObj *appmesh.HTTPRetryPolicy,
	sdkObj *appmeshsdk.HttpRetryPolicy, scope conversion.Scope) error {

	var sdkHttpRetryEvents []*string
	if len(crdObj.HTTPRetryEvents) != 0 {
		sdkHttpRetryEvents = make([]*string, 0, len(crdObj.HTTPRetryEvents))
		for _, crdHTTPRetryEvent := range crdObj.HTTPRetryEvents {
			sdkHttpRetryEvents = append(sdkHttpRetryEvents, aws.String((string)(crdHTTPRetryEvent)))
		}
	}
	sdkObj.HttpRetryEvents = sdkHttpRetryEvents

	var sdkTcpRetryEvents []*string
	if len(crdObj.TCPRetryEvents) != 0 {
		sdkTcpRetryEvents = make([]*string, 0, len(crdObj.TCPRetryEvents))
		for _, crdTCPRetryEvent := range crdObj.TCPRetryEvents {
			sdkTcpRetryEvents = append(sdkTcpRetryEvents, aws.String((string)(crdTCPRetryEvent)))
		}
	}
	sdkObj.TcpRetryEvents = sdkTcpRetryEvents

	sdkObj.PerRetryTimeout = &appmeshsdk.Duration{}
	if err := Convert_CRD_Duration_To_SDK_Duration(&crdObj.PerRetryTimeout, sdkObj.PerRetryTimeout, scope); err != nil {
		return err
	}

	sdkObj.MaxRetries = aws.Int64((int64)(crdObj.MaxRetries))
	return nil
}

func Convert_CRD_HTTPTimeout_To_SDK_HttpTimeout(crdObj *appmesh.HTTPTimeout,
	sdkObj *appmeshsdk.HttpTimeout, scope conversion.Scope) error {

	if crdObj.PerRequest != nil {
		sdkObj.PerRequest = &appmeshsdk.Duration{}
		if err := Convert_CRD_Duration_To_SDK_Duration(crdObj.PerRequest, sdkObj.PerRequest, scope); err != nil {
			return err
		}
	} else {
		sdkObj.PerRequest = nil
	}

	if crdObj.Idle != nil {
		sdkObj.Idle = &appmeshsdk.Duration{}
		if err := Convert_CRD_Duration_To_SDK_Duration(crdObj.Idle, sdkObj.Idle, scope); err != nil {
			return err
		}
	} else {
		sdkObj.Idle = nil
	}

	return nil
}

func Convert_CRD_HTTPRoute_To_SDK_HttpRoute(crdObj *appmesh.HTTPRoute,
	sdkObj *appmeshsdk.HttpRoute, scope conversion.Scope) error {

	sdkObj.Match = &appmeshsdk.HttpRouteMatch{}
	if err := Convert_CRD_HTTPRouteMatch_To_SDK_HttpRouteMatch(&crdObj.Match, sdkObj.Match, scope); err != nil {
		return err
	}

	sdkObj.Action = &appmeshsdk.HttpRouteAction{}
	if err := Convert_CRD_HTTPRouteAction_To_SDK_HttpRouteAction(&crdObj.Action, sdkObj.Action, scope); err != nil {
		return err
	}

	if crdObj.RetryPolicy != nil {
		sdkObj.RetryPolicy = &appmeshsdk.HttpRetryPolicy{}
		if err := Convert_CRD_HTTPRetryPolicy_To_SDK_HttpRetryPolicy(crdObj.RetryPolicy, sdkObj.RetryPolicy, scope); err != nil {
			return err
		}
	} else {
		sdkObj.RetryPolicy = nil

	}
	return nil
}

func Convert_CRD_TCPRouteAction_To_SDK_TcpRouteAction(crdObj *appmesh.TCPRouteAction,
	sdkObj *appmeshsdk.TcpRouteAction, scope conversion.Scope) error {

	var sdkWeightedTargets []*appmeshsdk.WeightedTarget
	if len(crdObj.WeightedTargets) != 0 {
		sdkWeightedTargets = make([]*appmeshsdk.WeightedTarget, 0, len(crdObj.WeightedTargets))
		for _, crdWeightedTarget := range crdObj.WeightedTargets {
			sdkWeightedTarget := &appmeshsdk.WeightedTarget{}
			if err := Convert_CRD_WeightedTarget_To_SDK_WeightedTarget(&crdWeightedTarget, sdkWeightedTarget, scope); err != nil {
				return err
			}
			sdkWeightedTargets = append(sdkWeightedTargets, sdkWeightedTarget)
		}
	}
	sdkObj.WeightedTargets = sdkWeightedTargets
	return nil
}

func Convert_CRD_TCPTimeout_To_SDK_TcpTimeout(crdObj *appmesh.TCPTimeout,
	sdkObj *appmeshsdk.TcpTimeout, scope conversion.Scope) error {

	if crdObj.Idle != nil {
		sdkObj.Idle = &appmeshsdk.Duration{}
		if err := Convert_CRD_Duration_To_SDK_Duration(crdObj.Idle, sdkObj.Idle, scope); err != nil {
			return err
		}
	} else {
		sdkObj.Idle = nil
	}

	return nil
}

func Convert_CRD_TCPRoute_To_SDK_TcpRoute(crdObj *appmesh.TCPRoute,
	sdkObj *appmeshsdk.TcpRoute, scope conversion.Scope) error {

	sdkObj.Action = &appmeshsdk.TcpRouteAction{}
	return Convert_CRD_TCPRouteAction_To_SDK_TcpRouteAction(&crdObj.Action, sdkObj.Action, scope)
}

func Convert_CRD_GRPCRouteMetadataMatchMethod_To_SDK_GrpcRouteMetadataMatchMethod(crdObj *appmesh.GRPCRouteMetadataMatchMethod,
	sdkObj *appmeshsdk.GrpcRouteMetadataMatchMethod, scope conversion.Scope) error {

	sdkObj.Exact = crdObj.Exact
	sdkObj.Prefix = crdObj.Prefix

	if crdObj.Range != nil {
		sdkObj.Range = &appmeshsdk.MatchRange{}
		if err := Convert_CRD_MatchRange_To_SDK_MatchRange(crdObj.Range, sdkObj.Range, scope); err != nil {
			return err
		}
	} else {
		sdkObj.Range = nil
	}

	sdkObj.Regex = crdObj.Regex
	sdkObj.Suffix = crdObj.Suffix
	return nil
}

func Convert_CRD_GRPCRouteMetadata_To_SDK_GrpcRouteMetadata(crdObj *appmesh.GRPCRouteMetadata,
	sdkObj *appmeshsdk.GrpcRouteMetadata, scope conversion.Scope) error {

	sdkObj.Name = aws.String(crdObj.Name)

	if crdObj.Match != nil {
		sdkObj.Match = &appmeshsdk.GrpcRouteMetadataMatchMethod{}
		if err := Convert_CRD_GRPCRouteMetadataMatchMethod_To_SDK_GrpcRouteMetadataMatchMethod(crdObj.Match, sdkObj.Match, scope); err != nil {
			return err
		}
	} else {
		sdkObj.Match = nil
	}

	sdkObj.Invert = crdObj.Invert
	return nil
}

func Convert_CRD_GRPCRouteMatch_To_SDK_GrpcRouteMatch(crdObj *appmesh.GRPCRouteMatch,
	sdkObj *appmeshsdk.GrpcRouteMatch, scope conversion.Scope) error {

	var sdkMetadataList []*appmeshsdk.GrpcRouteMetadata
	if len(crdObj.Metadata) != 0 {
		sdkMetadataList = make([]*appmeshsdk.GrpcRouteMetadata, 0, len(crdObj.Metadata))
		for _, crdMetadata := range crdObj.Metadata {
			sdkMetadata := &appmeshsdk.GrpcRouteMetadata{}
			if err := Convert_CRD_GRPCRouteMetadata_To_SDK_GrpcRouteMetadata(&crdMetadata, sdkMetadata, scope); err != nil {
				return err
			}
			sdkMetadataList = append(sdkMetadataList, sdkMetadata)
		}
	}

	sdkObj.Metadata = sdkMetadataList
	sdkObj.MethodName = crdObj.MethodName
	sdkObj.ServiceName = crdObj.ServiceName
	return nil

}

func Convert_CRD_GRPCRouteAction_To_SDK_GrpcRouteAction(crdObj *appmesh.GRPCRouteAction,
	sdkObj *appmeshsdk.GrpcRouteAction, scope conversion.Scope) error {

	var sdkWeightedTargets []*appmeshsdk.WeightedTarget
	if len(crdObj.WeightedTargets) != 0 {
		sdkWeightedTargets = make([]*appmeshsdk.WeightedTarget, 0, len(crdObj.WeightedTargets))
		for _, crdWeightedTarget := range crdObj.WeightedTargets {
			sdkWeightedTarget := &appmeshsdk.WeightedTarget{}
			if err := Convert_CRD_WeightedTarget_To_SDK_WeightedTarget(&crdWeightedTarget, sdkWeightedTarget, scope); err != nil {
				return err
			}
			sdkWeightedTargets = append(sdkWeightedTargets, sdkWeightedTarget)
		}
	}
	sdkObj.WeightedTargets = sdkWeightedTargets
	return nil
}

func Convert_CRD_GRPCRetryPolicy_To_SDK_GrpcRetryPolicy(crdObj *appmesh.GRPCRetryPolicy,
	sdkObj *appmeshsdk.GrpcRetryPolicy, scope conversion.Scope) error {

	var sdkGrpcRetryEvents []*string
	if len(crdObj.GRPCRetryEvents) != 0 {
		sdkGrpcRetryEvents = make([]*string, 0, len(crdObj.GRPCRetryEvents))
		for _, crdGRPCRetryEvent := range crdObj.GRPCRetryEvents {
			sdkGrpcRetryEvents = append(sdkGrpcRetryEvents, aws.String((string)(crdGRPCRetryEvent)))
		}
	}
	sdkObj.GrpcRetryEvents = sdkGrpcRetryEvents

	var sdkHttpRetryEvents []*string
	if len(crdObj.HTTPRetryEvents) != 0 {
		sdkHttpRetryEvents = make([]*string, 0, len(crdObj.HTTPRetryEvents))
		for _, crdHTTPRetryEvent := range crdObj.HTTPRetryEvents {
			sdkHttpRetryEvents = append(sdkHttpRetryEvents, aws.String((string)(crdHTTPRetryEvent)))
		}
	}
	sdkObj.HttpRetryEvents = sdkHttpRetryEvents

	var sdkTcpRetryEvents []*string
	if len(crdObj.TCPRetryEvents) != 0 {
		sdkTcpRetryEvents = make([]*string, 0, len(crdObj.TCPRetryEvents))
		for _, crdTCPRetryEvent := range crdObj.TCPRetryEvents {
			sdkTcpRetryEvents = append(sdkTcpRetryEvents, aws.String((string)(crdTCPRetryEvent)))
		}
	}
	sdkObj.TcpRetryEvents = sdkTcpRetryEvents

	sdkObj.PerRetryTimeout = &appmeshsdk.Duration{}
	if err := Convert_CRD_Duration_To_SDK_Duration(&crdObj.PerRetryTimeout, sdkObj.PerRetryTimeout, scope); err != nil {
		return err
	}

	sdkObj.MaxRetries = aws.Int64((int64)(crdObj.MaxRetries))
	return nil
}

func Convert_CRD_GRPCTimeout_To_SDK_GrpcTimeout(crdObj *appmesh.GRPCTimeout,
	sdkObj *appmeshsdk.GrpcTimeout, scope conversion.Scope) error {

	if crdObj.PerRequest != nil {
		sdkObj.PerRequest = &appmeshsdk.Duration{}
		if err := Convert_CRD_Duration_To_SDK_Duration(crdObj.PerRequest, sdkObj.PerRequest, scope); err != nil {
			return err
		}
	} else {
		sdkObj.PerRequest = nil
	}

	if crdObj.Idle != nil {
		sdkObj.Idle = &appmeshsdk.Duration{}
		if err := Convert_CRD_Duration_To_SDK_Duration(crdObj.Idle, sdkObj.Idle, scope); err != nil {
			return err
		}
	} else {
		sdkObj.Idle = nil
	}

	return nil
}

func Convert_CRD_GRPCRoute_To_SDK_GrpcRoute(crdObj *appmesh.GRPCRoute,
	sdkObj *appmeshsdk.GrpcRoute, scope conversion.Scope) error {

	sdkObj.Match = &appmeshsdk.GrpcRouteMatch{}
	if err := Convert_CRD_GRPCRouteMatch_To_SDK_GrpcRouteMatch(&crdObj.Match, sdkObj.Match, scope); err != nil {
		return err
	}

	sdkObj.Action = &appmeshsdk.GrpcRouteAction{}
	if err := Convert_CRD_GRPCRouteAction_To_SDK_GrpcRouteAction(&crdObj.Action, sdkObj.Action, scope); err != nil {
		return err
	}

	if crdObj.RetryPolicy != nil {
		sdkObj.RetryPolicy = &appmeshsdk.GrpcRetryPolicy{}
		if err := Convert_CRD_GRPCRetryPolicy_To_SDK_GrpcRetryPolicy(crdObj.RetryPolicy, sdkObj.RetryPolicy, scope); err != nil {
			return err
		}
	} else {
		sdkObj.RetryPolicy = nil
	}
	return nil
}

func Convert_CRD_Route_To_SDK_RouteSpec(crdObj *appmesh.Route, sdkObj *appmeshsdk.RouteSpec, scope conversion.Scope) error {

	if crdObj.GRPCRoute != nil {
		sdkObj.GrpcRoute = &appmeshsdk.GrpcRoute{}
		if err := Convert_CRD_GRPCRoute_To_SDK_GrpcRoute(crdObj.GRPCRoute, sdkObj.GrpcRoute, scope); err != nil {
			return err
		}
	} else {
		sdkObj.GrpcRoute = nil
	}

	if crdObj.HTTPRoute != nil {
		sdkObj.HttpRoute = &appmeshsdk.HttpRoute{}
		if err := Convert_CRD_HTTPRoute_To_SDK_HttpRoute(crdObj.HTTPRoute, sdkObj.HttpRoute, scope); err != nil {
			return err
		}
	} else {
		sdkObj.HttpRoute = nil
	}

	if crdObj.HTTP2Route != nil {
		sdkObj.Http2Route = &appmeshsdk.HttpRoute{}
		if err := Convert_CRD_HTTPRoute_To_SDK_HttpRoute(crdObj.HTTP2Route, sdkObj.Http2Route, scope); err != nil {
			return err
		}
	} else {
		sdkObj.Http2Route = nil
	}

	if crdObj.TCPRoute != nil {
		sdkObj.TcpRoute = &appmeshsdk.TcpRoute{}
		if err := Convert_CRD_TCPRoute_To_SDK_TcpRoute(crdObj.TCPRoute, sdkObj.TcpRoute, scope); err != nil {
			return err
		}
	} else {
		sdkObj.TcpRoute = nil
	}

	sdkObj.Priority = crdObj.Priority
	return nil
}

func Convert_CRD_VirtualRouterSpec_To_SDK_VirtualRouterSpec(crdObj *appmesh.VirtualRouterSpec, sdkObj *appmeshsdk.VirtualRouterSpec, scope conversion.Scope) error {
	var sdkListeners []*appmeshsdk.VirtualRouterListener
	for _, crdListener := range crdObj.Listeners {
		sdkListener := &appmeshsdk.VirtualRouterListener{}
		if err := Convert_CRD_VirtualRouterListener_To_SDK_VirtualRouterListener(&crdListener, sdkListener, scope); err != nil {
			return err
		}
		sdkListeners = append(sdkListeners, sdkListener)
	}
	sdkObj.Listeners = sdkListeners
	return nil
}
