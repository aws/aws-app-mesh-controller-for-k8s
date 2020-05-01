package conversions

import (
	"fmt"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	mock_conversion "github.com/aws/aws-app-mesh-controller-for-k8s/mocks/apimachinery/pkg/conversion"
	"github.com/aws/aws-sdk-go/aws"
	appmeshsdk "github.com/aws/aws-sdk-go/service/appmesh"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/conversion"
	"testing"
)

func TestConvert_CRD_VirtualNodeServiceProvider_To_SDK_VirtualNodeServiceProvider(t *testing.T) {
	type args struct {
		crdObj           *appmesh.VirtualNodeServiceProvider
		sdkObj           *appmeshsdk.VirtualNodeServiceProvider
		scopeConvertFunc func(src, dest interface{}, flags conversion.FieldMatchingFlags) error
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.VirtualNodeServiceProvider
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.VirtualNodeServiceProvider{
					VirtualNodeRef: appmesh.VirtualNodeReference{
						Namespace: aws.String("ns-1"),
						Name:      "vn-1",
					},
				},
				sdkObj: &appmeshsdk.VirtualNodeServiceProvider{},
				scopeConvertFunc: func(src, dest interface{}, flags conversion.FieldMatchingFlags) error {
					vnRef := src.(*appmesh.VirtualNodeReference)
					vnNamePtr := dest.(*string)
					*vnNamePtr = fmt.Sprintf("%s.%s", vnRef.Name, aws.StringValue(vnRef.Namespace))
					return nil
				},
			},
			wantSDKObj: &appmeshsdk.VirtualNodeServiceProvider{
				VirtualNodeName: aws.String("vn-1.ns-1"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			scope := mock_conversion.NewMockScope(ctrl)
			if tt.args.scopeConvertFunc != nil {
				scope.EXPECT().Convert(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(tt.args.scopeConvertFunc)
			}
			scope.EXPECT().Flags().Return(conversion.DestFromSource)

			err := Convert_CRD_VirtualNodeServiceProvider_To_SDK_VirtualNodeServiceProvider(tt.args.crdObj, tt.args.sdkObj, scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_VirtualRouterServiceProvider_To_SDK_VirtualRouterServiceProvider(t *testing.T) {
	type args struct {
		crdObj           *appmesh.VirtualRouterServiceProvider
		sdkObj           *appmeshsdk.VirtualRouterServiceProvider
		scopeConvertFunc func(src, dest interface{}, flags conversion.FieldMatchingFlags) error
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.VirtualRouterServiceProvider
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.VirtualRouterServiceProvider{
					VirtualRouterRef: appmesh.VirtualRouterReference{
						Namespace: aws.String("ns-1"),
						Name:      "vr-1",
					},
				},
				sdkObj: &appmeshsdk.VirtualRouterServiceProvider{},
				scopeConvertFunc: func(src, dest interface{}, flags conversion.FieldMatchingFlags) error {
					vrRef := src.(*appmesh.VirtualRouterReference)
					vrNamePtr := dest.(*string)
					*vrNamePtr = fmt.Sprintf("%s.%s", vrRef.Name, aws.StringValue(vrRef.Namespace))
					return nil
				},
			},
			wantSDKObj: &appmeshsdk.VirtualRouterServiceProvider{
				VirtualRouterName: aws.String("vr-1.ns-1"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			scope := mock_conversion.NewMockScope(ctrl)
			if tt.args.scopeConvertFunc != nil {
				scope.EXPECT().Convert(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(tt.args.scopeConvertFunc)
			}
			scope.EXPECT().Flags().Return(conversion.DestFromSource)

			err := Convert_CRD_VirtualRouterServiceProvider_To_SDK_VirtualRouterServiceProvider(tt.args.crdObj, tt.args.sdkObj, scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}
