package references

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/types"
)

// SDKVirtualGatewayReferenceConvertFunc is func that can convert VirtualGatewayReference to its AppMesh VirtualGateway name.
type SDKVirtualGatewayReferenceConvertFunc func(vgRef *appmesh.VirtualGatewayReference, vgAWSName *string, scope conversion.Scope) error

// SDKVirtualNodeReferenceConvertFunc is func that can convert VirtualNodeReference to its AppMesh VirtualNode name.
type SDKVirtualNodeReferenceConvertFunc func(vnRef *appmesh.VirtualNodeReference, vnAWSName *string, scope conversion.Scope) error

// SDKVirtualServiceReferenceConvertFunc is func that can convert VirtualServiceReference to its AppMesh VirtualService name.
type SDKVirtualServiceReferenceConvertFunc func(vsRef *appmesh.VirtualServiceReference, vsAWSName *string, scope conversion.Scope) error

// SDKVirtualRouterReferenceConvertFunc is func that can convert VirtualRouterReference to its AppMesh VirtualRouter name.
type SDKVirtualRouterReferenceConvertFunc func(vrRef *appmesh.VirtualRouterReference, vrAWSName *string, scope conversion.Scope) error

// BuildSDKVirtualGatewayReferenceConvertFunc constructs new SDKVirtualGatewayReferenceConvertFunc by given referencing object and VirtualGateway mapping.
func BuildSDKVirtualGatewayReferenceConvertFunc(obj metav1.Object, vgByKey map[types.NamespacedName]*appmesh.VirtualGateway) SDKVirtualGatewayReferenceConvertFunc {
	return func(vgRef *appmesh.VirtualGatewayReference, vgAWSName *string, scope conversion.Scope) error {
		vgKey := ObjectKeyForVirtualGatewayReference(obj, *vgRef)
		vg, ok := vgByKey[vgKey]
		if !ok {
			return errors.Errorf("unexpected VirtualGatewayReference: %v", vgKey)
		}
		*vgAWSName = aws.StringValue(vg.Spec.AWSName)
		return nil
	}
}

// BuildSDKVirtualNodeReferenceConvertFunc constructs new SDKVirtualNodeReferenceConvertFunc by given referencing object and VirtualNode mapping.
func BuildSDKVirtualNodeReferenceConvertFunc(obj metav1.Object, vnByKey map[types.NamespacedName]*appmesh.VirtualNode) SDKVirtualNodeReferenceConvertFunc {
	return func(vnRef *appmesh.VirtualNodeReference, vnAWSName *string, scope conversion.Scope) error {
		vnKey := ObjectKeyForVirtualNodeReference(obj, *vnRef)
		vn, ok := vnByKey[vnKey]
		if !ok {
			return errors.Errorf("unexpected VirtualNodeReference: %v", vnKey)
		}
		*vnAWSName = aws.StringValue(vn.Spec.AWSName)
		return nil
	}
}

// BuildSDKVirtualServiceReferenceConvertFunc constructs new SDKVirtualServiceReferenceConvertFunc by given referencing object and VirtualService mapping.
func BuildSDKVirtualServiceReferenceConvertFunc(obj metav1.Object, vsByKey map[types.NamespacedName]*appmesh.VirtualService) SDKVirtualServiceReferenceConvertFunc {
	return func(vsRef *appmesh.VirtualServiceReference, vsAWSName *string, scope conversion.Scope) error {
		vsKey := ObjectKeyForVirtualServiceReference(obj, *vsRef)
		vs, ok := vsByKey[vsKey]
		if !ok {
			return errors.Errorf("unexpected VirtualServiceReference: %v", vsKey)
		}
		*vsAWSName = aws.StringValue(vs.Spec.AWSName)
		return nil
	}
}

// BuildSDKVirtualRouterReferenceConvertFunc constructs new SDKVirtualRouterReferenceConvertFunc by given referencing object and VirtualRouter mapping.
func BuildSDKVirtualRouterReferenceConvertFunc(obj metav1.Object, vrByKey map[types.NamespacedName]*appmesh.VirtualRouter) SDKVirtualRouterReferenceConvertFunc {
	return func(vrRef *appmesh.VirtualRouterReference, vrAWSName *string, scope conversion.Scope) error {
		vrKey := ObjectKeyForVirtualRouterReference(obj, *vrRef)
		vr, ok := vrByKey[vrKey]
		if !ok {
			return errors.Errorf("unexpected VirtualRouterReference: %v", vrKey)
		}
		*vrAWSName = aws.StringValue(vr.Spec.AWSName)
		return nil
	}
}
