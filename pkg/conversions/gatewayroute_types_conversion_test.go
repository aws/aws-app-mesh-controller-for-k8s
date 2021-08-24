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

func TestConvert_CRD_GatewayRouteVirtualService_To_SDK_GatewayRouteVirtualService(t *testing.T) {
	type args struct {
		crdObj           *appmesh.GatewayRouteVirtualService
		sdkObj           *appmeshsdk.GatewayRouteVirtualService
		scopeConvertFunc func(src, dest interface{}) error
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.GatewayRouteVirtualService
		wantErr    error
	}{
		{
			name: "use virtualServiceRef",
			args: args{
				crdObj: &appmesh.GatewayRouteVirtualService{
					VirtualServiceRef: &appmesh.VirtualServiceReference{
						Namespace: aws.String("ns-1"),
						Name:      "vs-1",
					},
				},
				sdkObj: &appmeshsdk.GatewayRouteVirtualService{},
				scopeConvertFunc: func(src, dest interface{}) error {
					vsRef := src.(*appmesh.VirtualServiceReference)
					vsNamePtr := dest.(*string)
					*vsNamePtr = fmt.Sprintf("%s.%s", vsRef.Name, aws.StringValue(vsRef.Namespace))
					return nil
				},
			},
			wantSDKObj: &appmeshsdk.GatewayRouteVirtualService{
				VirtualServiceName: aws.String("vs-1.ns-1"),
			},
		},
		{
			name: "use virtualServiceArn",
			args: args{
				crdObj: &appmesh.GatewayRouteVirtualService{
					VirtualServiceARN: aws.String("arn:aws:appmesh:us-west-2:000000000000:mesh/mesh-name/virtualService/vs-name"),
				},
				sdkObj: &appmeshsdk.GatewayRouteVirtualService{},
			},
			wantSDKObj: &appmeshsdk.GatewayRouteVirtualService{
				VirtualServiceName: aws.String("vs-name"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			scope := mock_conversion.NewMockScope(ctrl)
			if tt.args.scopeConvertFunc != nil {
				scope.EXPECT().Convert(gomock.Any(), gomock.Any()).DoAndReturn(tt.args.scopeConvertFunc).AnyTimes()
			}

			err := Convert_CRD_GatewayRouteVirtualService_To_SDK_GatewayRouteVirtualService(tt.args.crdObj, tt.args.sdkObj, scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_GRPCHostnameRewrite_To_SDK_GrpcHostnameRewrite(t *testing.T) {
	tests := []struct {
		name          string
		crdObjRewrite *appmesh.GrpcGatewayRouteRewrite
		sdkObj        *appmeshsdk.GrpcGatewayRouteRewrite
		wantSDKObj    *appmeshsdk.GrpcGatewayRouteRewrite
	}{
		{
			name: "Only DefaultHostname specified",
			crdObjRewrite: &appmesh.GrpcGatewayRouteRewrite{
				Hostname: &appmesh.GatewayRouteHostnameRewrite{
					DefaultTargetHostname: aws.String("DISABLED"),
				},
			},
			sdkObj: &appmeshsdk.GrpcGatewayRouteRewrite{},
			wantSDKObj: &appmeshsdk.GrpcGatewayRouteRewrite{
				Hostname: &appmeshsdk.GatewayRouteHostnameRewrite{
					DefaultTargetHostname: aws.String("DISABLED"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Convert_CRD_GRPCHostnameRewrite_To_SDK_GrpcHostnameRewrite(tt.crdObjRewrite, tt.sdkObj)
			assert.Equal(t, tt.wantSDKObj, tt.sdkObj)
		})
	}

}

func TestConvert_CRD_GatewayRouteTarget_To_SDK_GatewayRouteTarget(t *testing.T) {
	type args struct {
		crdObj           *appmesh.GatewayRouteTarget
		sdkObj           *appmeshsdk.GatewayRouteTarget
		scopeConvertFunc func(src, dest interface{}) error
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.GatewayRouteTarget
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.GatewayRouteTarget{
					VirtualService: appmesh.GatewayRouteVirtualService{
						VirtualServiceRef: &appmesh.VirtualServiceReference{
							Namespace: aws.String("ns-1"),
							Name:      "vs-1",
						},
					},
				},
				sdkObj: &appmeshsdk.GatewayRouteTarget{},
				scopeConvertFunc: func(src, dest interface{}) error {
					vsRef := src.(*appmesh.VirtualServiceReference)
					vsNamePtr := dest.(*string)
					*vsNamePtr = fmt.Sprintf("%s.%s", vsRef.Name, aws.StringValue(vsRef.Namespace))
					return nil
				},
			},
			wantSDKObj: &appmeshsdk.GatewayRouteTarget{
				VirtualService: &appmeshsdk.GatewayRouteVirtualService{
					VirtualServiceName: aws.String("vs-1.ns-1"),
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
				scope.EXPECT().Convert(gomock.Any(), gomock.Any()).DoAndReturn(tt.args.scopeConvertFunc).AnyTimes()
			}

			err := Convert_CRD_GatewayRouteTarget_To_SDK_GatewayRouteTarget(tt.args.crdObj, tt.args.sdkObj, scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_HTTPGatewayRouteMatch_To_SDK_HttpGatewayRouteMatch(t *testing.T) {
	type args struct {
		crdObj *appmesh.HTTPGatewayRouteMatch
		sdkObj *appmeshsdk.HttpGatewayRouteMatch
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.HttpGatewayRouteMatch
		wantErr    error
	}{
		{
			name: "prefix match",
			args: args{
				crdObj: &appmesh.HTTPGatewayRouteMatch{
					Prefix: aws.String("prefix"),
				},
				sdkObj: &appmeshsdk.HttpGatewayRouteMatch{},
			},
			wantSDKObj: &appmeshsdk.HttpGatewayRouteMatch{
				Prefix: aws.String("prefix"),
			},
		},
		{
			name: "path match",
			args: args{
				crdObj: &appmesh.HTTPGatewayRouteMatch{
					Path: &appmesh.HTTPPathMatch{
						Exact: aws.String("/color-paths/green/"),
					},
				},
				sdkObj: &appmeshsdk.HttpGatewayRouteMatch{},
			},
			wantSDKObj: &appmeshsdk.HttpGatewayRouteMatch{
				Path: &appmeshsdk.HttpPathMatch{
					Exact: aws.String("/color-paths/green/"),
				},
			},
		},
		{
			name: "queryparameter+method match",
			args: args{
				crdObj: &appmesh.HTTPGatewayRouteMatch{
					QueryParameters: []appmesh.HTTPQueryParameters{
						{
							Name: aws.String("color"),
							Match: &appmesh.QueryMatchMethod{
								Exact: aws.String("red"),
							},
						},
						{
							Name: aws.String("device"),
						},
					},
					Method: aws.String("POST"),
				},
				sdkObj: &appmeshsdk.HttpGatewayRouteMatch{},
			},
			wantSDKObj: &appmeshsdk.HttpGatewayRouteMatch{
				QueryParameters: []*appmeshsdk.HttpQueryParameter{
					{
						Name: aws.String("color"),
						Match: &appmeshsdk.QueryParameterMatch{
							Exact: aws.String("red"),
						},
					},
					{
						Name: aws.String("device"),
					},
				},
				Method: aws.String("POST"),
			},
		},
		{
			name: "Header+method+prefix match",
			args: args{
				crdObj: &appmesh.HTTPGatewayRouteMatch{
					Prefix: aws.String("/users"),
					Headers: []appmesh.HTTPGatewayRouteHeader{
						{
							Name: "username",
							Match: &appmesh.HeaderMatchMethod{
								Suffix: aws.String("admin"),
							},
						},
						{
							Name: "username",
							Match: &appmesh.HeaderMatchMethod{
								Regex: aws.String(".*test.*"),
							},
							Invert: aws.Bool(true),
						},
					},
					Method: aws.String("POST"),
				},
				sdkObj: &appmeshsdk.HttpGatewayRouteMatch{},
			},
			wantSDKObj: &appmeshsdk.HttpGatewayRouteMatch{
				Prefix: aws.String("/users"),
				Headers: []*appmeshsdk.HttpGatewayRouteHeader{
					{
						Name: aws.String("username"),
						Match: &appmeshsdk.HeaderMatchMethod{
							Suffix: aws.String("admin"),
						},
					},
					{
						Name: aws.String("username"),
						Match: &appmeshsdk.HeaderMatchMethod{
							Regex: aws.String(".*test.*"),
						},
						Invert: aws.Bool(true),
					},
				},
				Method: aws.String("POST"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_HTTPGatewayRouteMatch_To_SDK_HttpGatewayRouteMatch(tt.args.crdObj, tt.args.sdkObj)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_HTTPGatewayRouteAction_To_SDK_HttpGatewayRouteAction(t *testing.T) {
	type args struct {
		crdObj           *appmesh.HTTPGatewayRouteAction
		sdkObj           *appmeshsdk.HttpGatewayRouteAction
		scopeConvertFunc func(src, dest interface{}) error
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.HttpGatewayRouteAction
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.HTTPGatewayRouteAction{
					Target: appmesh.GatewayRouteTarget{
						VirtualService: appmesh.GatewayRouteVirtualService{
							VirtualServiceRef: &appmesh.VirtualServiceReference{
								Namespace: aws.String("ns-1"),
								Name:      "vs-1",
							},
						},
					},
				},
				sdkObj: &appmeshsdk.HttpGatewayRouteAction{},
				scopeConvertFunc: func(src, dest interface{}) error {
					vsRef := src.(*appmesh.VirtualServiceReference)
					vsNamePtr := dest.(*string)
					*vsNamePtr = fmt.Sprintf("%s.%s", vsRef.Name, aws.StringValue(vsRef.Namespace))
					return nil
				},
			},
			wantSDKObj: &appmeshsdk.HttpGatewayRouteAction{
				Target: &appmeshsdk.GatewayRouteTarget{
					VirtualService: &appmeshsdk.GatewayRouteVirtualService{
						VirtualServiceName: aws.String("vs-1.ns-1"),
					},
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
				scope.EXPECT().Convert(gomock.Any(), gomock.Any()).DoAndReturn(tt.args.scopeConvertFunc).AnyTimes()
			}

			err := Convert_CRD_HTTPGatewayRouteAction_To_SDK_HttpGatewayRouteAction(tt.args.crdObj, tt.args.sdkObj, scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_HTTPGatewayRoute_To_SDK_HttpGatewayRoute(t *testing.T) {
	type args struct {
		crdObj           *appmesh.HTTPGatewayRoute
		sdkObj           *appmeshsdk.HttpGatewayRoute
		scopeConvertFunc func(src, dest interface{}) error
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.HttpGatewayRoute
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.HTTPGatewayRoute{
					Match: appmesh.HTTPGatewayRouteMatch{
						Prefix: aws.String("prefix"),
					},
					Action: appmesh.HTTPGatewayRouteAction{
						Target: appmesh.GatewayRouteTarget{
							VirtualService: appmesh.GatewayRouteVirtualService{
								VirtualServiceRef: &appmesh.VirtualServiceReference{
									Namespace: aws.String("ns-1"),
									Name:      "vs-1",
								},
							},
						},
					},
				},
				sdkObj: &appmeshsdk.HttpGatewayRoute{},
				scopeConvertFunc: func(src, dest interface{}) error {
					vsRef := src.(*appmesh.VirtualServiceReference)
					vsNamePtr := dest.(*string)
					*vsNamePtr = fmt.Sprintf("%s.%s", vsRef.Name, aws.StringValue(vsRef.Namespace))
					return nil
				},
			},
			wantSDKObj: &appmeshsdk.HttpGatewayRoute{
				Match: &appmeshsdk.HttpGatewayRouteMatch{
					Prefix: aws.String("prefix"),
				},
				Action: &appmeshsdk.HttpGatewayRouteAction{
					Target: &appmeshsdk.GatewayRouteTarget{
						VirtualService: &appmeshsdk.GatewayRouteVirtualService{
							VirtualServiceName: aws.String("vs-1.ns-1"),
						},
					},
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
				scope.EXPECT().Convert(gomock.Any(), gomock.Any()).DoAndReturn(tt.args.scopeConvertFunc).AnyTimes()
			}

			err := Convert_CRD_HTTPGatewayRoute_To_SDK_HttpGatewayRoute(tt.args.crdObj, tt.args.sdkObj, scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_GRPCGatewayRouteMatch_To_SDK_GrpcGatewayRouteMatch(t *testing.T) {
	type args struct {
		crdObj *appmesh.GRPCGatewayRouteMatch
		sdkObj *appmeshsdk.GrpcGatewayRouteMatch
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.GrpcGatewayRouteMatch
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.GRPCGatewayRouteMatch{
					ServiceName: aws.String("foo.foodomain.local"),
				},
				sdkObj: &appmeshsdk.GrpcGatewayRouteMatch{},
			},
			wantSDKObj: &appmeshsdk.GrpcGatewayRouteMatch{
				ServiceName: aws.String("foo.foodomain.local"),
			},
		},
		{
			name: "normal case + nil service name",
			args: args{
				crdObj: &appmesh.GRPCGatewayRouteMatch{
					ServiceName: nil,
				},
				sdkObj: &appmeshsdk.GrpcGatewayRouteMatch{},
			},
			wantSDKObj: &appmeshsdk.GrpcGatewayRouteMatch{
				ServiceName: nil,
			},
		},
		{
			name: "servicename+hostname+metadata",
			args: args{
				crdObj: &appmesh.GRPCGatewayRouteMatch{
					ServiceName: aws.String("color.ColorService"),
					Hostname: &appmesh.GatewayRouteHostnameMatch{
						Exact: aws.String("www.colorpicker.com"),
					},
					Metadata: []appmesh.GRPCGatewayRouteMetadata{
						{
							Name: aws.String("color_type"),
							Match: &appmesh.GRPCRouteMetadataMatchMethod{
								Prefix: aws.String("blue"),
							},
						},
						{
							Name: aws.String("userId"),
							Match: &appmesh.GRPCRouteMetadataMatchMethod{
								Range: &appmesh.MatchRange{
									Start: 30,
									End:   70,
								},
							},
						},
						{
							Name: aws.String("filter_color"),
							Match: &appmesh.GRPCRouteMetadataMatchMethod{
								Regex: aws.String("white"),
							},
						},
					},
				},
				sdkObj: &appmeshsdk.GrpcGatewayRouteMatch{},
			},
			wantSDKObj: &appmeshsdk.GrpcGatewayRouteMatch{
				ServiceName: aws.String("color.ColorService"),
				Hostname: &appmeshsdk.GatewayRouteHostnameMatch{
					Exact: aws.String("www.colorpicker.com"),
				},
				Metadata: []*appmeshsdk.GrpcGatewayRouteMetadata{
					{
						Name: aws.String("color_type"),
						Match: &appmeshsdk.GrpcMetadataMatchMethod{
							Prefix: aws.String("blue"),
						},
					},
					{
						Name: aws.String("userId"),
						Match: &appmeshsdk.GrpcMetadataMatchMethod{
							Range: &appmeshsdk.MatchRange{
								Start: aws.Int64(30),
								End:   aws.Int64(70),
							},
						},
					},
					{
						Name: aws.String("filter_color"),
						Match: &appmeshsdk.GrpcMetadataMatchMethod{
							Regex: aws.String("white"),
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_GRPCGatewayRouteMatch_To_SDK_GrpcGatewayRouteMatch(tt.args.crdObj, tt.args.sdkObj)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_HTTPGatewayRoutePathRewrite_To_SDK_HttpGatewayRoutePathRewrite(t *testing.T) {
	type args struct {
		crdObj *appmesh.HTTPGatewayRouteRewrite
		sdkObj *appmeshsdk.HttpGatewayRouteRewrite
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.HttpGatewayRouteRewrite
	}{
		{
			name: "Exact Path match",
			args: args{
				crdObj: &appmesh.HTTPGatewayRouteRewrite{
					Path: &appmesh.GatewayRoutePathRewrite{
						Exact: aws.String("www.domain.com/path"),
					},
				},
				sdkObj: &appmeshsdk.HttpGatewayRouteRewrite{},
			},
			wantSDKObj: &appmeshsdk.HttpGatewayRouteRewrite{
				Path: &appmeshsdk.HttpGatewayRoutePathRewrite{
					Exact: aws.String("www.domain.com/path"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Convert_CRD_HTTPGatewayRouteRewritePath_To_SDK_HttpGatewayRouteRewritePath(tt.args.crdObj, tt.args.sdkObj)
			assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
		})
	}
}

func TestConvert_CRD_HTTPGatewayRoutePrefixRewrite_To_SDK_HttpGatewayRoutePrefixRewrite(t *testing.T) {
	type args struct {
		crdObj *appmesh.HTTPGatewayRouteRewrite
		sdkObj *appmeshsdk.HttpGatewayRouteRewrite
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.HttpGatewayRouteRewrite
	}{
		{
			name: "Valid Case",
			args: args{
				crdObj: &appmesh.HTTPGatewayRouteRewrite{
					Prefix: &appmesh.GatewayRoutePrefixRewrite{
						DefaultPrefix: aws.String("ENABLED"),
					},
				},
				sdkObj: &appmeshsdk.HttpGatewayRouteRewrite{},
			},
			wantSDKObj: &appmeshsdk.HttpGatewayRouteRewrite{
				Prefix: &appmeshsdk.HttpGatewayRoutePrefixRewrite{
					DefaultPrefix: aws.String("ENABLED"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Convert_CRD_HTTPGatewayRouteRewritePrefix_To_SDK_HttpGatewayRouteRewritePrefix(tt.args.crdObj, tt.args.sdkObj)
			assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
		})
	}
}

func TestConvert_CRD_GRPCGatewayRouteAction_To_SDK_GrpcGatewayRouteAction(t *testing.T) {
	type args struct {
		crdObj           *appmesh.GRPCGatewayRouteAction
		sdkObj           *appmeshsdk.GrpcGatewayRouteAction
		scopeConvertFunc func(src, dest interface{}) error
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.GrpcGatewayRouteAction
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.GRPCGatewayRouteAction{
					Target: appmesh.GatewayRouteTarget{
						VirtualService: appmesh.GatewayRouteVirtualService{
							VirtualServiceRef: &appmesh.VirtualServiceReference{
								Namespace: aws.String("ns-1"),
								Name:      "vs-1",
							},
						},
					},
				},
				sdkObj: &appmeshsdk.GrpcGatewayRouteAction{},
				scopeConvertFunc: func(src, dest interface{}) error {
					vsRef := src.(*appmesh.VirtualServiceReference)
					vsNamePtr := dest.(*string)
					*vsNamePtr = fmt.Sprintf("%s.%s", vsRef.Name, aws.StringValue(vsRef.Namespace))
					return nil
				},
			},
			wantSDKObj: &appmeshsdk.GrpcGatewayRouteAction{
				Target: &appmeshsdk.GatewayRouteTarget{
					VirtualService: &appmeshsdk.GatewayRouteVirtualService{
						VirtualServiceName: aws.String("vs-1.ns-1"),
					},
				},
			},
		},
		{
			name: "Test Rewrite",
			args: args{
				crdObj: &appmesh.GRPCGatewayRouteAction{
					Target: appmesh.GatewayRouteTarget{
						VirtualService: appmesh.GatewayRouteVirtualService{
							VirtualServiceRef: &appmesh.VirtualServiceReference{
								Namespace: aws.String("ns-1"),
								Name:      "vs-1",
							},
						},
					},
					Rewrite: &appmesh.GrpcGatewayRouteRewrite{
						Hostname: &appmesh.GatewayRouteHostnameRewrite{
							DefaultTargetHostname: aws.String("DISABLED"),
						},
					},
				},
				sdkObj: &appmeshsdk.GrpcGatewayRouteAction{},
				scopeConvertFunc: func(src, dest interface{}) error {
					vsRef := src.(*appmesh.VirtualServiceReference)
					vsNamePtr := dest.(*string)
					*vsNamePtr = fmt.Sprintf("%s.%s", vsRef.Name, aws.StringValue(vsRef.Namespace))
					return nil
				},
			},
			wantSDKObj: &appmeshsdk.GrpcGatewayRouteAction{
				Target: &appmeshsdk.GatewayRouteTarget{
					VirtualService: &appmeshsdk.GatewayRouteVirtualService{
						VirtualServiceName: aws.String("vs-1.ns-1"),
					},
				},
				Rewrite: &appmeshsdk.GrpcGatewayRouteRewrite{
					Hostname: &appmeshsdk.GatewayRouteHostnameRewrite{
						DefaultTargetHostname: aws.String("DISABLED"),
					},
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
				scope.EXPECT().Convert(gomock.Any(), gomock.Any()).DoAndReturn(tt.args.scopeConvertFunc).AnyTimes()
			}

			err := Convert_CRD_GRPCGatewayRouteAction_To_SDK_GrpcGatewayRouteAction(tt.args.crdObj, tt.args.sdkObj, scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_GatewayRouteHostnameMatch_To_SDK_GatewayRouteHostnameMatch(t *testing.T) {
	type args struct {
		crdObj *appmesh.GatewayRouteHostnameMatch
		sdkObj *appmeshsdk.GatewayRouteHostnameMatch
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.GatewayRouteHostnameMatch
	}{
		{
			name: "Hostanme Exact match case",
			args: args{
				crdObj: &appmesh.GatewayRouteHostnameMatch{Exact: aws.String("test/payments")},
				sdkObj: &appmeshsdk.GatewayRouteHostnameMatch{},
			},
			wantSDKObj: &appmeshsdk.GatewayRouteHostnameMatch{Exact: aws.String("test/payments")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Convert_CRD_GatewayRouteHostnameMatch_To_SDK_GatewayRouteHostnameMatch(tt.args.crdObj, tt.args.sdkObj)
			assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
		})
	}
}

func TestConvert_CRD_HTTPGatewayRouteHeaders_To_SDK_HttpGatewayRouteHeaders(t *testing.T) {
	type args struct {
		crdObj *appmesh.HTTPGatewayRouteMatch
		sdkObj *appmeshsdk.HttpGatewayRouteMatch
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.HttpGatewayRouteMatch
	}{
		{
			name: "Match with 2 headers",
			args: args{
				crdObj: &appmesh.HTTPGatewayRouteMatch{
					Headers: []appmesh.HTTPGatewayRouteHeader{
						{
							Name: "scenario",
							Match: &appmesh.HeaderMatchMethod{
								Exact: aws.String("login"),
							},
						},
						{
							Name: "location",
							Match: &appmesh.HeaderMatchMethod{
								Suffix: aws.String(".us-east-1"),
							},
						},
					},
				},
				sdkObj: &appmeshsdk.HttpGatewayRouteMatch{},
			},
			wantSDKObj: &appmeshsdk.HttpGatewayRouteMatch{
				Headers: []*appmeshsdk.HttpGatewayRouteHeader{
					{
						Name: aws.String("scenario"),
						Match: &appmeshsdk.HeaderMatchMethod{
							Exact: aws.String("login"),
						},
					},
					{
						Name: aws.String("location"),
						Match: &appmeshsdk.HeaderMatchMethod{
							Suffix: aws.String(".us-east-1"),
						},
					},
				},
			},
		},
		{
			name: "Empty Headers list",
			args: args{
				crdObj: &appmesh.HTTPGatewayRouteMatch{
					Headers: []appmesh.HTTPGatewayRouteHeader{},
				},
				sdkObj: &appmeshsdk.HttpGatewayRouteMatch{},
			},
			wantSDKObj: &appmeshsdk.HttpGatewayRouteMatch{
				Headers: []*appmeshsdk.HttpGatewayRouteHeader{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Convert_CRD_HTTPGatewayRouteHeaders_To_SDK_HttpGatewayRouteHeaders(tt.args.crdObj, tt.args.sdkObj)
			assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
		})
	}
}

func TestConvert_CRD_HTTPGatewayPath_To_SDK_HttpGatewayPath(t *testing.T) {
	type args struct {
		crdObj *appmesh.HTTPGatewayRouteMatch
		sdkObj *appmeshsdk.HttpGatewayRouteMatch
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.HttpGatewayRouteMatch
	}{
		{
			name: "Exact Path Case",
			args: args{
				crdObj: &appmesh.HTTPGatewayRouteMatch{
					Path: &appmesh.HTTPPathMatch{
						Exact: aws.String("www.test.com/home.html"),
					},
				},
				sdkObj: &appmeshsdk.HttpGatewayRouteMatch{},
			},
			wantSDKObj: &appmeshsdk.HttpGatewayRouteMatch{
				Path: &appmeshsdk.HttpPathMatch{
					Exact: aws.String("www.test.com/home.html"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Convert_CRD_HTTPGatewayPath_To_SDK_HttpGatewayPath(tt.args.crdObj, tt.args.sdkObj)
			assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
		})
	}
}

func TestConvert_CRD_HTTPGatewayRouteQueryParams_To_SDK_HttpGatewayRouteQueryParams(t *testing.T) {
	type args struct {
		crdObj *appmesh.HTTPGatewayRouteMatch
		sdkObj *appmeshsdk.HttpGatewayRouteMatch
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.HttpGatewayRouteMatch
	}{
		{
			name: "Normal case 2 query parameters",
			args: args{
				crdObj: &appmesh.HTTPGatewayRouteMatch{
					QueryParameters: []appmesh.HTTPQueryParameters{
						{
							Name: aws.String("app"),
							Match: &appmesh.QueryMatchMethod{
								Exact: aws.String("backend"),
							},
						},
						{
							Name: aws.String("user"),
							Match: &appmesh.QueryMatchMethod{
								Exact: aws.String("test"),
							},
						},
					},
				},
				sdkObj: &appmeshsdk.HttpGatewayRouteMatch{},
			},
			wantSDKObj: &appmeshsdk.HttpGatewayRouteMatch{
				QueryParameters: []*appmeshsdk.HttpQueryParameter{
					{
						Name: aws.String("app"),
						Match: &appmeshsdk.QueryParameterMatch{
							Exact: aws.String("backend"),
						},
					},
					{
						Name: aws.String("user"),
						Match: &appmeshsdk.QueryParameterMatch{
							Exact: aws.String("test"),
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Convert_CRD_HTTPGatewayRouteQueryParams_To_SDK_HttpGatewayRouteQueryParams(tt.args.crdObj, tt.args.sdkObj)
			assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
		})
	}
}

func TestConvert_CRD_GRPCGatewayRouteMetadata_To_SDK_GrpcGatewayRouteMetadata(t *testing.T) {
	type args struct {
		crdObj *appmesh.GRPCGatewayRouteMatch
		sdkObj *appmeshsdk.GrpcGatewayRouteMatch
	}

	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.GrpcGatewayRouteMatch
	}{
		{
			name: "Normal case",
			args: args{
				crdObj: &appmesh.GRPCGatewayRouteMatch{
					Metadata: []appmesh.GRPCGatewayRouteMetadata{
						{
							Name: aws.String("scenario"),
							Match: &appmesh.GRPCRouteMetadataMatchMethod{
								Exact: aws.String("signup"),
							},
							Invert: aws.Bool(false),
						},
					},
				},
				sdkObj: &appmeshsdk.GrpcGatewayRouteMatch{},
			},
			wantSDKObj: &appmeshsdk.GrpcGatewayRouteMatch{
				Metadata: []*appmeshsdk.GrpcGatewayRouteMetadata{
					{
						Name: aws.String("scenario"),
						Match: &appmeshsdk.GrpcMetadataMatchMethod{
							Exact: aws.String("signup"),
						},
						Invert: aws.Bool(false),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Convert_CRD_GRPCGatewayRouteMetadata_To_SDK_GrpcGatewayRouteMetadata(tt.args.crdObj, tt.args.sdkObj)
			assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
		})
	}
}

func TestConvert_CRD_GRPCGatewayRoute_To_SDK_GrpcGatewayRoute(t *testing.T) {
	type args struct {
		crdObj           *appmesh.GRPCGatewayRoute
		sdkObj           *appmeshsdk.GrpcGatewayRoute
		scopeConvertFunc func(src, dest interface{}) error
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.GrpcGatewayRoute
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.GRPCGatewayRoute{
					Match: appmesh.GRPCGatewayRouteMatch{
						ServiceName: aws.String("foo.foodomain.local"),
					},
					Action: appmesh.GRPCGatewayRouteAction{
						Target: appmesh.GatewayRouteTarget{
							VirtualService: appmesh.GatewayRouteVirtualService{
								VirtualServiceRef: &appmesh.VirtualServiceReference{
									Namespace: aws.String("ns-1"),
									Name:      "vs-1",
								},
							},
						},
					},
				},
				sdkObj: &appmeshsdk.GrpcGatewayRoute{},
				scopeConvertFunc: func(src, dest interface{}) error {
					vsRef := src.(*appmesh.VirtualServiceReference)
					vsNamePtr := dest.(*string)
					*vsNamePtr = fmt.Sprintf("%s.%s", vsRef.Name, aws.StringValue(vsRef.Namespace))
					return nil
				},
			},
			wantSDKObj: &appmeshsdk.GrpcGatewayRoute{
				Match: &appmeshsdk.GrpcGatewayRouteMatch{
					ServiceName: aws.String("foo.foodomain.local"),
				},
				Action: &appmeshsdk.GrpcGatewayRouteAction{
					Target: &appmeshsdk.GatewayRouteTarget{
						VirtualService: &appmeshsdk.GatewayRouteVirtualService{
							VirtualServiceName: aws.String("vs-1.ns-1"),
						},
					},
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
				scope.EXPECT().Convert(gomock.Any(), gomock.Any()).DoAndReturn(tt.args.scopeConvertFunc).AnyTimes()
			}

			err := Convert_CRD_GRPCGatewayRoute_To_SDK_GrpcGatewayRoute(tt.args.crdObj, tt.args.sdkObj, scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_GatewayRouteSpec_To_SDK_GatewayRouteSpec(t *testing.T) {
	type args struct {
		crdObj           *appmesh.GatewayRouteSpec
		sdkObj           *appmeshsdk.GatewayRouteSpec
		scopeConvertFunc func(src, dest interface{}) error
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.GatewayRouteSpec
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.GatewayRouteSpec{
					GRPCRoute: &appmesh.GRPCGatewayRoute{
						Match: appmesh.GRPCGatewayRouteMatch{
							ServiceName: aws.String("foo.foodomain.local"),
						},
						Action: appmesh.GRPCGatewayRouteAction{
							Target: appmesh.GatewayRouteTarget{
								VirtualService: appmesh.GatewayRouteVirtualService{
									VirtualServiceRef: &appmesh.VirtualServiceReference{
										Namespace: aws.String("ns-1"),
										Name:      "vs-1",
									},
								},
							},
						},
					},
					HTTPRoute: &appmesh.HTTPGatewayRoute{
						Match: appmesh.HTTPGatewayRouteMatch{
							Prefix: aws.String("prefix"),
						},
						Action: appmesh.HTTPGatewayRouteAction{
							Target: appmesh.GatewayRouteTarget{
								VirtualService: appmesh.GatewayRouteVirtualService{
									VirtualServiceRef: &appmesh.VirtualServiceReference{
										Namespace: aws.String("ns-1"),
										Name:      "vs-1",
									},
								},
							},
						},
					},
					HTTP2Route: &appmesh.HTTPGatewayRoute{
						Match: appmesh.HTTPGatewayRouteMatch{
							Prefix: aws.String("prefix"),
						},
						Action: appmesh.HTTPGatewayRouteAction{
							Target: appmesh.GatewayRouteTarget{
								VirtualService: appmesh.GatewayRouteVirtualService{
									VirtualServiceRef: &appmesh.VirtualServiceReference{
										Namespace: aws.String("ns-1"),
										Name:      "vs-1",
									},
								},
							},
						},
					},
				},
				sdkObj: &appmeshsdk.GatewayRouteSpec{},
				scopeConvertFunc: func(src, dest interface{}) error {
					vsRef := src.(*appmesh.VirtualServiceReference)
					vsNamePtr := dest.(*string)
					*vsNamePtr = fmt.Sprintf("%s.%s", vsRef.Name, aws.StringValue(vsRef.Namespace))
					return nil
				},
			},
			wantSDKObj: &appmeshsdk.GatewayRouteSpec{
				GrpcRoute: &appmeshsdk.GrpcGatewayRoute{
					Match: &appmeshsdk.GrpcGatewayRouteMatch{
						ServiceName: aws.String("foo.foodomain.local"),
					},
					Action: &appmeshsdk.GrpcGatewayRouteAction{
						Target: &appmeshsdk.GatewayRouteTarget{
							VirtualService: &appmeshsdk.GatewayRouteVirtualService{
								VirtualServiceName: aws.String("vs-1.ns-1"),
							},
						},
					},
				},
				HttpRoute: &appmeshsdk.HttpGatewayRoute{
					Match: &appmeshsdk.HttpGatewayRouteMatch{
						Prefix: aws.String("prefix"),
					},
					Action: &appmeshsdk.HttpGatewayRouteAction{
						Target: &appmeshsdk.GatewayRouteTarget{
							VirtualService: &appmeshsdk.GatewayRouteVirtualService{
								VirtualServiceName: aws.String("vs-1.ns-1"),
							},
						},
					},
				},
				Http2Route: &appmeshsdk.HttpGatewayRoute{
					Match: &appmeshsdk.HttpGatewayRouteMatch{
						Prefix: aws.String("prefix"),
					},
					Action: &appmeshsdk.HttpGatewayRouteAction{
						Target: &appmeshsdk.GatewayRouteTarget{
							VirtualService: &appmeshsdk.GatewayRouteVirtualService{
								VirtualServiceName: aws.String("vs-1.ns-1"),
							},
						},
					},
				},
			},
		},
		{
			name: "nil http2 route",
			args: args{
				crdObj: &appmesh.GatewayRouteSpec{
					GRPCRoute: &appmesh.GRPCGatewayRoute{
						Match: appmesh.GRPCGatewayRouteMatch{
							ServiceName: aws.String("foo.foodomain.local"),
						},
						Action: appmesh.GRPCGatewayRouteAction{
							Target: appmesh.GatewayRouteTarget{
								VirtualService: appmesh.GatewayRouteVirtualService{
									VirtualServiceRef: &appmesh.VirtualServiceReference{
										Namespace: aws.String("ns-1"),
										Name:      "vs-1",
									},
								},
							},
						},
					},
					HTTPRoute: &appmesh.HTTPGatewayRoute{
						Match: appmesh.HTTPGatewayRouteMatch{
							Prefix: aws.String("prefix"),
						},
						Action: appmesh.HTTPGatewayRouteAction{
							Target: appmesh.GatewayRouteTarget{
								VirtualService: appmesh.GatewayRouteVirtualService{
									VirtualServiceRef: &appmesh.VirtualServiceReference{
										Namespace: aws.String("ns-1"),
										Name:      "vs-1",
									},
								},
							},
						},
					},
					HTTP2Route: nil,
				},
				sdkObj: &appmeshsdk.GatewayRouteSpec{},
				scopeConvertFunc: func(src, dest interface{}) error {
					vsRef := src.(*appmesh.VirtualServiceReference)
					vsNamePtr := dest.(*string)
					*vsNamePtr = fmt.Sprintf("%s.%s", vsRef.Name, aws.StringValue(vsRef.Namespace))
					return nil
				},
			},
			wantSDKObj: &appmeshsdk.GatewayRouteSpec{
				GrpcRoute: &appmeshsdk.GrpcGatewayRoute{
					Match: &appmeshsdk.GrpcGatewayRouteMatch{
						ServiceName: aws.String("foo.foodomain.local"),
					},
					Action: &appmeshsdk.GrpcGatewayRouteAction{
						Target: &appmeshsdk.GatewayRouteTarget{
							VirtualService: &appmeshsdk.GatewayRouteVirtualService{
								VirtualServiceName: aws.String("vs-1.ns-1"),
							},
						},
					},
				},
				HttpRoute: &appmeshsdk.HttpGatewayRoute{
					Match: &appmeshsdk.HttpGatewayRouteMatch{
						Prefix: aws.String("prefix"),
					},
					Action: &appmeshsdk.HttpGatewayRouteAction{
						Target: &appmeshsdk.GatewayRouteTarget{
							VirtualService: &appmeshsdk.GatewayRouteVirtualService{
								VirtualServiceName: aws.String("vs-1.ns-1"),
							},
						},
					},
				},
				Http2Route: nil,
			},
		},
		{
			name: "nil http route",
			args: args{
				crdObj: &appmesh.GatewayRouteSpec{
					GRPCRoute: &appmesh.GRPCGatewayRoute{
						Match: appmesh.GRPCGatewayRouteMatch{
							ServiceName: aws.String("foo.foodomain.local"),
						},
						Action: appmesh.GRPCGatewayRouteAction{
							Target: appmesh.GatewayRouteTarget{
								VirtualService: appmesh.GatewayRouteVirtualService{
									VirtualServiceRef: &appmesh.VirtualServiceReference{
										Namespace: aws.String("ns-1"),
										Name:      "vs-1",
									},
								},
							},
						},
					},
					HTTP2Route: &appmesh.HTTPGatewayRoute{
						Match: appmesh.HTTPGatewayRouteMatch{
							Prefix: aws.String("prefix"),
						},
						Action: appmesh.HTTPGatewayRouteAction{
							Target: appmesh.GatewayRouteTarget{
								VirtualService: appmesh.GatewayRouteVirtualService{
									VirtualServiceRef: &appmesh.VirtualServiceReference{
										Namespace: aws.String("ns-1"),
										Name:      "vs-1",
									},
								},
							},
						},
					},
					HTTPRoute: nil,
				},
				sdkObj: &appmeshsdk.GatewayRouteSpec{},
				scopeConvertFunc: func(src, dest interface{}) error {
					vsRef := src.(*appmesh.VirtualServiceReference)
					vsNamePtr := dest.(*string)
					*vsNamePtr = fmt.Sprintf("%s.%s", vsRef.Name, aws.StringValue(vsRef.Namespace))
					return nil
				},
			},
			wantSDKObj: &appmeshsdk.GatewayRouteSpec{
				GrpcRoute: &appmeshsdk.GrpcGatewayRoute{
					Match: &appmeshsdk.GrpcGatewayRouteMatch{
						ServiceName: aws.String("foo.foodomain.local"),
					},
					Action: &appmeshsdk.GrpcGatewayRouteAction{
						Target: &appmeshsdk.GatewayRouteTarget{
							VirtualService: &appmeshsdk.GatewayRouteVirtualService{
								VirtualServiceName: aws.String("vs-1.ns-1"),
							},
						},
					},
				},
				Http2Route: &appmeshsdk.HttpGatewayRoute{
					Match: &appmeshsdk.HttpGatewayRouteMatch{
						Prefix: aws.String("prefix"),
					},
					Action: &appmeshsdk.HttpGatewayRouteAction{
						Target: &appmeshsdk.GatewayRouteTarget{
							VirtualService: &appmeshsdk.GatewayRouteVirtualService{
								VirtualServiceName: aws.String("vs-1.ns-1"),
							},
						},
					},
				},
				HttpRoute: nil,
			},
		},
		{
			name: "nil grpc route",
			args: args{
				crdObj: &appmesh.GatewayRouteSpec{
					GRPCRoute: nil,
					HTTPRoute: &appmesh.HTTPGatewayRoute{
						Match: appmesh.HTTPGatewayRouteMatch{
							Prefix: aws.String("prefix"),
						},
						Action: appmesh.HTTPGatewayRouteAction{
							Target: appmesh.GatewayRouteTarget{
								VirtualService: appmesh.GatewayRouteVirtualService{
									VirtualServiceRef: &appmesh.VirtualServiceReference{
										Namespace: aws.String("ns-1"),
										Name:      "vs-1",
									},
								},
							},
						},
					},
					HTTP2Route: &appmesh.HTTPGatewayRoute{
						Match: appmesh.HTTPGatewayRouteMatch{
							Prefix: aws.String("prefix"),
						},
						Action: appmesh.HTTPGatewayRouteAction{
							Target: appmesh.GatewayRouteTarget{
								VirtualService: appmesh.GatewayRouteVirtualService{
									VirtualServiceRef: &appmesh.VirtualServiceReference{
										Namespace: aws.String("ns-1"),
										Name:      "vs-1",
									},
								},
							},
						},
					},
				},
				sdkObj: &appmeshsdk.GatewayRouteSpec{},
				scopeConvertFunc: func(src, dest interface{}) error {
					vsRef := src.(*appmesh.VirtualServiceReference)
					vsNamePtr := dest.(*string)
					*vsNamePtr = fmt.Sprintf("%s.%s", vsRef.Name, aws.StringValue(vsRef.Namespace))
					return nil
				},
			},
			wantSDKObj: &appmeshsdk.GatewayRouteSpec{
				GrpcRoute: nil,
				HttpRoute: &appmeshsdk.HttpGatewayRoute{
					Match: &appmeshsdk.HttpGatewayRouteMatch{
						Prefix: aws.String("prefix"),
					},
					Action: &appmeshsdk.HttpGatewayRouteAction{
						Target: &appmeshsdk.GatewayRouteTarget{
							VirtualService: &appmeshsdk.GatewayRouteVirtualService{
								VirtualServiceName: aws.String("vs-1.ns-1"),
							},
						},
					},
				},
				Http2Route: &appmeshsdk.HttpGatewayRoute{
					Match: &appmeshsdk.HttpGatewayRouteMatch{
						Prefix: aws.String("prefix"),
					},
					Action: &appmeshsdk.HttpGatewayRouteAction{
						Target: &appmeshsdk.GatewayRouteTarget{
							VirtualService: &appmeshsdk.GatewayRouteVirtualService{
								VirtualServiceName: aws.String("vs-1.ns-1"),
							},
						},
					},
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
				scope.EXPECT().Convert(gomock.Any(), gomock.Any()).DoAndReturn(tt.args.scopeConvertFunc).AnyTimes()
			}

			err := Convert_CRD_GatewayRouteSpec_To_SDK_GatewayRouteSpec(tt.args.crdObj, tt.args.sdkObj, scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}
