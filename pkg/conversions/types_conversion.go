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
