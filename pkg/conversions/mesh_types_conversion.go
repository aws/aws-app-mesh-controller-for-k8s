package conversions

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-sdk-go/aws"
	appmeshsdk "github.com/aws/aws-sdk-go/service/appmesh"
	"k8s.io/apimachinery/pkg/conversion"
)

func Convert_CRD_EgressFilter_To_SDK_EgressFilter(crdObj *appmesh.EgressFilter, sdkObj *appmeshsdk.EgressFilter, scope conversion.Scope) error {
	sdkObj.Type = aws.String((string)(crdObj.Type))
	return nil
}

func Convert_CRD_MeshSpec_To_SDK_MeshSpec(crdObj *appmesh.MeshSpec, sdkObj *appmeshsdk.MeshSpec, scope conversion.Scope) error {
	if crdObj.EgressFilter != nil {
		sdkObj.EgressFilter = &appmeshsdk.EgressFilter{}
		if err := Convert_CRD_EgressFilter_To_SDK_EgressFilter(crdObj.EgressFilter, sdkObj.EgressFilter, scope); err != nil {
			return err
		}
	} else {
		sdkObj.EgressFilter = nil
	}
	return nil
}
