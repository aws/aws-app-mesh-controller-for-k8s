package conversions

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-sdk-go/aws"
	appmeshsdk "github.com/aws/aws-sdk-go/service/appmesh"
	"k8s.io/apimachinery/pkg/conversion"
)

func Convert_CRD_PortMapping_To_SDK_PortMapping(crdObj *appmesh.PortMapping, sdkObj *appmeshsdk.PortMapping, scope conversion.Scope) error {
	sdkObj.Port = aws.Int64((int64)(crdObj.Port))
	sdkObj.Protocol = aws.String((string)(crdObj.Protocol))
	return nil
}

func Convert_CRD_Duration_To_SDK_Duration(crdObj *appmesh.Duration, sdkObj *appmeshsdk.Duration,
	scope conversion.Scope) error {
	sdkObj.Unit = aws.String((string)(crdObj.Unit))
	sdkObj.Value = aws.Int64((int64)(crdObj.Value))
	return nil
}

func Convert_CRD_HTTPHeaderMatchMethod_To_SDK_HttpHeaderMatchMethod(crdObj *appmesh.HeaderMatchMethod, sdkObj *appmeshsdk.HeaderMatchMethod) error {
	sdkObj.Exact = crdObj.Exact
	sdkObj.Prefix = crdObj.Prefix

	if crdObj.Range != nil {
		sdkObj.Range = &appmeshsdk.MatchRange{}
		if err := Convert_CRD_MatchRange_To_SDK_MatchRange(crdObj.Range, sdkObj.Range); err != nil {
			return err
		}
	} else {
		sdkObj.Range = nil
	}
	sdkObj.Regex = crdObj.Regex
	sdkObj.Suffix = crdObj.Suffix
	return nil
}

func Convert_CRD_GrpcMetdataMatchMethod_To_SDK_GrpcMetadataMatchMethod(crdObj *appmesh.GRPCRouteMetadataMatchMethod, sdkObj *appmeshsdk.GrpcMetadataMatchMethod) error {
	sdkObj.Exact = crdObj.Exact
	sdkObj.Prefix = crdObj.Prefix

	if crdObj.Range != nil {
		sdkObj.Range = &appmeshsdk.MatchRange{}
		if err := Convert_CRD_MatchRange_To_SDK_MatchRange(crdObj.Range, sdkObj.Range); err != nil {
			return err
		}
	}

	sdkObj.Regex = crdObj.Regex
	sdkObj.Suffix = crdObj.Suffix
	return nil
}

func Convert_CRD_MatchRange_To_SDK_MatchRange(crdObj *appmesh.MatchRange, sdkObj *appmeshsdk.MatchRange) error {
	sdkObj.Start = aws.Int64(crdObj.Start)
	sdkObj.End = aws.Int64(crdObj.End)
	return nil
}
