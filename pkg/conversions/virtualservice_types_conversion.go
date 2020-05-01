package conversions

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-sdk-go/aws"
	appmeshsdk "github.com/aws/aws-sdk-go/service/appmesh"
	"k8s.io/apimachinery/pkg/conversion"
)

func Convert_CRD_VirtualNodeServiceProvider_To_SDK_VirtualNodeServiceProvider(crdObj *appmesh.VirtualNodeServiceProvider,
	sdkObj *appmeshsdk.VirtualNodeServiceProvider, scope conversion.Scope) error {

	sdkObj.VirtualNodeName = aws.String("")
	if err := scope.Convert(&crdObj.VirtualNodeRef, sdkObj.VirtualNodeName, scope.Flags()); err != nil {
		return err
	}
	return nil
}

func Convert_CRD_VirtualRouterServiceProvider_To_SDK_VirtualRouterServiceProvider(crdObj *appmesh.VirtualRouterServiceProvider,
	sdkObj *appmeshsdk.VirtualRouterServiceProvider, scope conversion.Scope) error {

	sdkObj.VirtualRouterName = aws.String("")
	if err := scope.Convert(&crdObj.VirtualRouterRef, sdkObj.VirtualRouterName, scope.Flags()); err != nil {
		return err
	}
	return nil
}

func Convert_CRD_VirtualServiceProvider_To_SDK_VirtualServiceProvider(crdObj *appmesh.VirtualServiceProvider,
	sdkObj *appmeshsdk.VirtualServiceProvider, scope conversion.Scope) error {

	if crdObj.VirtualNode != nil {

		sdkObj.VirtualNode = &appmeshsdk.VirtualNodeServiceProvider{}
		if err := Convert_CRD_VirtualNodeServiceProvider_To_SDK_VirtualNodeServiceProvider(crdObj.VirtualNode, sdkObj.VirtualNode, scope); err != nil {
			return err
		}
	} else {
		sdkObj.VirtualNode = nil
	}

	if crdObj.VirtualRouter != nil {

		sdkObj.VirtualRouter = &appmeshsdk.VirtualRouterServiceProvider{}
		if err := Convert_CRD_VirtualRouterServiceProvider_To_SDK_VirtualRouterServiceProvider(crdObj.VirtualRouter, sdkObj.VirtualRouter, scope); err != nil {
			return err
		}
	} else {
		sdkObj.VirtualRouter = nil
	}
	return nil

}

func Convert_CRD_VirtualServiceSpec_To_SDK_VirtualServiceSpec(crdObj *appmesh.VirtualServiceSpec,
	sdkObj *appmeshsdk.VirtualServiceSpec, scope conversion.Scope) error {

	return Convert_CRD_VirtualServiceProvider_To_SDK_VirtualServiceProvider(&crdObj.Provider, sdkObj.Provider, scope)

}
