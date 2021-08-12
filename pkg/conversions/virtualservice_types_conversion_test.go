package conversions

import (
	"fmt"
	"testing"

	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	mock_conversion "github.com/aws/aws-app-mesh-controller-for-k8s/mocks/apimachinery/pkg/conversion"
	"github.com/aws/aws-sdk-go/aws"
	appmeshsdk "github.com/aws/aws-sdk-go/service/appmesh"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestConvert_CRD_VirtualNodeServiceProvider_To_SDK_VirtualNodeServiceProvider(t *testing.T) {
	type args struct {
		crdObj           *appmesh.VirtualNodeServiceProvider
		sdkObj           *appmeshsdk.VirtualNodeServiceProvider
		scopeConvertFunc func(src, dest interface{}) error
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.VirtualNodeServiceProvider
		wantErr    error
	}{
		{
			name: "use virtualNodeRef",
			args: args{
				crdObj: &appmesh.VirtualNodeServiceProvider{
					VirtualNodeRef: &appmesh.VirtualNodeReference{
						Namespace: aws.String("ns-1"),
						Name:      "vn-1",
					},
				},
				sdkObj: &appmeshsdk.VirtualNodeServiceProvider{},
				scopeConvertFunc: func(src, dest interface{}) error {
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
		{
			name: "use virtualNodeARN",
			args: args{
				crdObj: &appmesh.VirtualNodeServiceProvider{
					VirtualNodeARN: aws.String("arn:aws:appmesh:us-west-2:000000000000:mesh/mesh-name/virtualNode/vn-name"),
				},
				sdkObj: &appmeshsdk.VirtualNodeServiceProvider{},
			},
			wantSDKObj: &appmeshsdk.VirtualNodeServiceProvider{
				VirtualNodeName: aws.String("vn-name"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			scope := mock_conversion.NewMockScope(ctrl)
			if tt.args.scopeConvertFunc != nil {
				scope.EXPECT().Convert(gomock.Any(), gomock.Any()).DoAndReturn(tt.args.scopeConvertFunc)
			}

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
		scopeConvertFunc func(src, dest interface{}) error
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.VirtualRouterServiceProvider
		wantErr    error
	}{
		{
			name: "use virtualRouterRef",
			args: args{
				crdObj: &appmesh.VirtualRouterServiceProvider{
					VirtualRouterRef: &appmesh.VirtualRouterReference{
						Namespace: aws.String("ns-1"),
						Name:      "vr-1",
					},
				},
				sdkObj: &appmeshsdk.VirtualRouterServiceProvider{},
				scopeConvertFunc: func(src, dest interface{}) error {
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
		{
			name: "use virtualRouterARN",
			args: args{
				crdObj: &appmesh.VirtualRouterServiceProvider{
					VirtualRouterARN: aws.String("arn:aws:appmesh:us-west-2:000000000000:mesh/mesh-name/virtualRouter/vr-name"),
				},
				sdkObj: &appmeshsdk.VirtualRouterServiceProvider{},
			},
			wantSDKObj: &appmeshsdk.VirtualRouterServiceProvider{
				VirtualRouterName: aws.String("vr-name"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			scope := mock_conversion.NewMockScope(ctrl)
			if tt.args.scopeConvertFunc != nil {
				scope.EXPECT().Convert(gomock.Any(), gomock.Any()).DoAndReturn(tt.args.scopeConvertFunc)
			}

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

func TestConvert_CRD_VirtualServiceProvider_To_SDK_VirtualServiceProvider(t *testing.T) {
	type args struct {
		crdObj           *appmesh.VirtualServiceProvider
		sdkObj           *appmeshsdk.VirtualServiceProvider
		scopeConvertFunc func(src, dest interface{}) error
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.VirtualServiceProvider
		wantErr    error
	}{
		{
			name: "virtual node + nil virtual router",
			args: args{
				crdObj: &appmesh.VirtualServiceProvider{
					VirtualNode: &appmesh.VirtualNodeServiceProvider{
						VirtualNodeRef: &appmesh.VirtualNodeReference{
							Namespace: aws.String("ns-1"),
							Name:      "vn-1",
						},
					},
					VirtualRouter: nil,
				},
				sdkObj: &appmeshsdk.VirtualServiceProvider{},
				scopeConvertFunc: func(src, dest interface{}) error {
					vnRef := src.(*appmesh.VirtualNodeReference)
					vnNamePtr := dest.(*string)
					*vnNamePtr = fmt.Sprintf("%s.%s", vnRef.Name, aws.StringValue(vnRef.Namespace))
					return nil
				},
			},
			wantSDKObj: &appmeshsdk.VirtualServiceProvider{
				VirtualNode: &appmeshsdk.VirtualNodeServiceProvider{
					VirtualNodeName: aws.String("vn-1.ns-1"),
				},
				VirtualRouter: nil,
			},
		},
		{
			name: "virtual router + nil virtual node",
			args: args{
				crdObj: &appmesh.VirtualServiceProvider{
					VirtualRouter: &appmesh.VirtualRouterServiceProvider{
						VirtualRouterRef: &appmesh.VirtualRouterReference{
							Namespace: aws.String("ns-1"),
							Name:      "vr-1",
						},
					},
					VirtualNode: nil,
				},
				sdkObj: &appmeshsdk.VirtualServiceProvider{},
				scopeConvertFunc: func(src, dest interface{}) error {
					vrRef := src.(*appmesh.VirtualRouterReference)
					vrNamePtr := dest.(*string)
					*vrNamePtr = fmt.Sprintf("%s.%s", vrRef.Name, aws.StringValue(vrRef.Namespace))
					return nil
				},
			},
			wantSDKObj: &appmeshsdk.VirtualServiceProvider{
				VirtualRouter: &appmeshsdk.VirtualRouterServiceProvider{
					VirtualRouterName: aws.String("vr-1.ns-1"),
				},
				VirtualNode: nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			scope := mock_conversion.NewMockScope(ctrl)
			if tt.args.scopeConvertFunc != nil {
				scope.EXPECT().Convert(gomock.Any(), gomock.Any()).DoAndReturn(tt.args.scopeConvertFunc)
			}

			err := Convert_CRD_VirtualServiceProvider_To_SDK_VirtualServiceProvider(tt.args.crdObj, tt.args.sdkObj, scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_VirtualServiceSpec_To_SDK_VirtualServiceSpec(t *testing.T) {
	type args struct {
		crdObj           *appmesh.VirtualServiceSpec
		sdkObj           *appmeshsdk.VirtualServiceSpec
		scopeConvertFunc func(src, dest interface{}) error
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.VirtualServiceSpec
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.VirtualServiceSpec{
					AWSName: aws.String("app1"),
					Provider: &appmesh.VirtualServiceProvider{
						VirtualNode: &appmesh.VirtualNodeServiceProvider{
							VirtualNodeRef: &appmesh.VirtualNodeReference{
								Namespace: aws.String("ns-1"),
								Name:      "vn-1",
							},
						},
						VirtualRouter: nil,
					},
				},
				sdkObj: &appmeshsdk.VirtualServiceSpec{},
				scopeConvertFunc: func(src, dest interface{}) error {
					vnRef := src.(*appmesh.VirtualNodeReference)
					vnNamePtr := dest.(*string)
					*vnNamePtr = fmt.Sprintf("%s.%s", vnRef.Name, aws.StringValue(vnRef.Namespace))
					return nil
				},
			},
			wantSDKObj: &appmeshsdk.VirtualServiceSpec{
				Provider: &appmeshsdk.VirtualServiceProvider{
					VirtualNode: &appmeshsdk.VirtualNodeServiceProvider{
						VirtualNodeName: aws.String("vn-1.ns-1"),
					},
					VirtualRouter: nil,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			scope := mock_conversion.NewMockScope(ctrl)
			if tt.args.scopeConvertFunc != nil {
				scope.EXPECT().Convert(gomock.Any(), gomock.Any()).DoAndReturn(tt.args.scopeConvertFunc)
			}

			err := Convert_CRD_VirtualServiceSpec_To_SDK_VirtualServiceSpec(tt.args.crdObj, tt.args.sdkObj, scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}
