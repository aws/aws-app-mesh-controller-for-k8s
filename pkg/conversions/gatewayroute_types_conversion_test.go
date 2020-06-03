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

func TestConvert_CRD_GatewayRouteVirtualService_To_SDK_GatewayRouteVirtualService(t *testing.T) {
	type args struct {
		crdObj           *appmesh.GatewayRouteVirtualService
		sdkObj           *appmeshsdk.GatewayRouteVirtualService
		scopeConvertFunc func(src, dest interface{}, flags conversion.FieldMatchingFlags) error
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.GatewayRouteVirtualService
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.GatewayRouteVirtualService{
					VirtualServiceRef: appmesh.VirtualServiceReference{
						Namespace: aws.String("ns-1"),
						Name:      "vs-1",
					},
				},
				sdkObj: &appmeshsdk.GatewayRouteVirtualService{},
				scopeConvertFunc: func(src, dest interface{}, flags conversion.FieldMatchingFlags) error {
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			scope := mock_conversion.NewMockScope(ctrl)
			if tt.args.scopeConvertFunc != nil {
				scope.EXPECT().Convert(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(tt.args.scopeConvertFunc).AnyTimes()
			}
			scope.EXPECT().Flags().Return(conversion.DestFromSource).AnyTimes()

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

func TestConvert_CRD_GatewayRouteTarget_To_SDK_GatewayRouteTarget(t *testing.T) {
	type args struct {
		crdObj           *appmesh.GatewayRouteTarget
		sdkObj           *appmeshsdk.GatewayRouteTarget
		scopeConvertFunc func(src, dest interface{}, flags conversion.FieldMatchingFlags) error
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
						VirtualServiceRef: appmesh.VirtualServiceReference{
							Namespace: aws.String("ns-1"),
							Name:      "vs-1",
						},
					},
				},
				sdkObj: &appmeshsdk.GatewayRouteTarget{},
				scopeConvertFunc: func(src, dest interface{}, flags conversion.FieldMatchingFlags) error {
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
				scope.EXPECT().Convert(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(tt.args.scopeConvertFunc).AnyTimes()
			}
			scope.EXPECT().Flags().Return(conversion.DestFromSource).AnyTimes()

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
			name: "normal case",
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
			name: "normal case + nil prefix",
			args: args{
				crdObj: &appmesh.HTTPGatewayRouteMatch{
					Prefix: nil,
				},
				sdkObj: &appmeshsdk.HttpGatewayRouteMatch{},
			},
			wantSDKObj: &appmeshsdk.HttpGatewayRouteMatch{
				Prefix: nil,
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
		scopeConvertFunc func(src, dest interface{}, flags conversion.FieldMatchingFlags) error
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
							VirtualServiceRef: appmesh.VirtualServiceReference{
								Namespace: aws.String("ns-1"),
								Name:      "vs-1",
							},
						},
					},
				},
				sdkObj: &appmeshsdk.HttpGatewayRouteAction{},
				scopeConvertFunc: func(src, dest interface{}, flags conversion.FieldMatchingFlags) error {
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
				scope.EXPECT().Convert(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(tt.args.scopeConvertFunc).AnyTimes()
			}
			scope.EXPECT().Flags().Return(conversion.DestFromSource).AnyTimes()

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
		scopeConvertFunc func(src, dest interface{}, flags conversion.FieldMatchingFlags) error
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
								VirtualServiceRef: appmesh.VirtualServiceReference{
									Namespace: aws.String("ns-1"),
									Name:      "vs-1",
								},
							},
						},
					},
				},
				sdkObj: &appmeshsdk.HttpGatewayRoute{},
				scopeConvertFunc: func(src, dest interface{}, flags conversion.FieldMatchingFlags) error {
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
				scope.EXPECT().Convert(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(tt.args.scopeConvertFunc).AnyTimes()
			}
			scope.EXPECT().Flags().Return(conversion.DestFromSource).AnyTimes()

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

func TestConvert_CRD_GRPCGatewayRouteAction_To_SDK_GrpcGatewayRouteAction(t *testing.T) {
	type args struct {
		crdObj           *appmesh.GRPCGatewayRouteAction
		sdkObj           *appmeshsdk.GrpcGatewayRouteAction
		scopeConvertFunc func(src, dest interface{}, flags conversion.FieldMatchingFlags) error
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
							VirtualServiceRef: appmesh.VirtualServiceReference{
								Namespace: aws.String("ns-1"),
								Name:      "vs-1",
							},
						},
					},
				},
				sdkObj: &appmeshsdk.GrpcGatewayRouteAction{},
				scopeConvertFunc: func(src, dest interface{}, flags conversion.FieldMatchingFlags) error {
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			scope := mock_conversion.NewMockScope(ctrl)
			if tt.args.scopeConvertFunc != nil {
				scope.EXPECT().Convert(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(tt.args.scopeConvertFunc).AnyTimes()
			}
			scope.EXPECT().Flags().Return(conversion.DestFromSource).AnyTimes()

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

func TestConvert_CRD_GRPCGatewayRoute_To_SDK_GrpcGatewayRoute(t *testing.T) {
	type args struct {
		crdObj           *appmesh.GRPCGatewayRoute
		sdkObj           *appmeshsdk.GrpcGatewayRoute
		scopeConvertFunc func(src, dest interface{}, flags conversion.FieldMatchingFlags) error
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
								VirtualServiceRef: appmesh.VirtualServiceReference{
									Namespace: aws.String("ns-1"),
									Name:      "vs-1",
								},
							},
						},
					},
				},
				sdkObj: &appmeshsdk.GrpcGatewayRoute{},
				scopeConvertFunc: func(src, dest interface{}, flags conversion.FieldMatchingFlags) error {
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
				scope.EXPECT().Convert(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(tt.args.scopeConvertFunc).AnyTimes()
			}
			scope.EXPECT().Flags().Return(conversion.DestFromSource).AnyTimes()

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
		scopeConvertFunc func(src, dest interface{}, flags conversion.FieldMatchingFlags) error
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
									VirtualServiceRef: appmesh.VirtualServiceReference{
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
									VirtualServiceRef: appmesh.VirtualServiceReference{
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
									VirtualServiceRef: appmesh.VirtualServiceReference{
										Namespace: aws.String("ns-1"),
										Name:      "vs-1",
									},
								},
							},
						},
					},
				},
				sdkObj: &appmeshsdk.GatewayRouteSpec{},
				scopeConvertFunc: func(src, dest interface{}, flags conversion.FieldMatchingFlags) error {
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
									VirtualServiceRef: appmesh.VirtualServiceReference{
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
									VirtualServiceRef: appmesh.VirtualServiceReference{
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
				scopeConvertFunc: func(src, dest interface{}, flags conversion.FieldMatchingFlags) error {
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
									VirtualServiceRef: appmesh.VirtualServiceReference{
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
									VirtualServiceRef: appmesh.VirtualServiceReference{
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
				scopeConvertFunc: func(src, dest interface{}, flags conversion.FieldMatchingFlags) error {
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
									VirtualServiceRef: appmesh.VirtualServiceReference{
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
									VirtualServiceRef: appmesh.VirtualServiceReference{
										Namespace: aws.String("ns-1"),
										Name:      "vs-1",
									},
								},
							},
						},
					},
				},
				sdkObj: &appmeshsdk.GatewayRouteSpec{},
				scopeConvertFunc: func(src, dest interface{}, flags conversion.FieldMatchingFlags) error {
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
				scope.EXPECT().Convert(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(tt.args.scopeConvertFunc).AnyTimes()
			}
			scope.EXPECT().Flags().Return(conversion.DestFromSource).AnyTimes()

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
