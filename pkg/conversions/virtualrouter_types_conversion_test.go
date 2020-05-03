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

func TestConvert_CRD_VirtualRouterListener_To_SDK_VirtualRouterListener(t *testing.T) {
	port80 := appmesh.PortNumber(80)
	protocolHTTP := appmesh.PortProtocolHTTP
	type args struct {
		crdObj *appmesh.VirtualRouterListener
		sdkObj *appmeshsdk.VirtualRouterListener
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.VirtualRouterListener
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.VirtualRouterListener{
					PortMapping: appmesh.PortMapping{
						Port:     port80,
						Protocol: protocolHTTP,
					},
				},
				sdkObj: &appmeshsdk.VirtualRouterListener{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.VirtualRouterListener{
				PortMapping: &appmeshsdk.PortMapping{
					Port:     aws.Int64(80),
					Protocol: aws.String("http"),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_VirtualRouterListener_To_SDK_VirtualRouterListener(tt.args.crdObj, tt.args.sdkObj, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_WeightedTarget_To_SDK_WeightedTarget(t *testing.T) {
	type args struct {
		crdObj           *appmesh.WeightedTarget
		sdkObj           *appmeshsdk.WeightedTarget
		scopeConvertFunc func(src, dest interface{}, flags conversion.FieldMatchingFlags) error
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.WeightedTarget
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.WeightedTarget{
					VirtualNodeRef: appmesh.VirtualNodeReference{
						Namespace: aws.String("ns-1"),
						Name:      "vn-1",
					},
					Weight: int64(100),
				},
				sdkObj: &appmeshsdk.WeightedTarget{},
				scopeConvertFunc: func(src, dest interface{}, flags conversion.FieldMatchingFlags) error {
					vnRef := src.(*appmesh.VirtualNodeReference)
					vnNamePtr := dest.(*string)
					*vnNamePtr = fmt.Sprintf("%s.%s", vnRef.Name, aws.StringValue(vnRef.Namespace))
					return nil
				},
			},
			wantSDKObj: &appmeshsdk.WeightedTarget{
				VirtualNode: aws.String("vn-1.ns-1"),
				Weight:      aws.Int64(100),
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

			err := Convert_CRD_WeightedTarget_To_SDK_WeightedTarget(tt.args.crdObj, tt.args.sdkObj, scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_HeaderMatchMethod_To_SDK_HeaderMatchMethod(t *testing.T) {
	type args struct {
		crdObj *appmesh.HeaderMatchMethod
		sdkObj *appmeshsdk.HeaderMatchMethod
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.HeaderMatchMethod
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.HeaderMatchMethod{
					Exact: aws.String("header1"),
					Range: &appmesh.MatchRange{
						Start: int64(20),
						End:   int64(80),
					},
					Prefix: aws.String("prefix-1"),
					Regex:  aws.String("am*zon"),
					Suffix: aws.String("suffix-1"),
				},
				sdkObj: &appmeshsdk.HeaderMatchMethod{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.HeaderMatchMethod{
				Exact: aws.String("header1"),
				Range: &appmeshsdk.MatchRange{
					Start: aws.Int64(20),
					End:   aws.Int64(80),
				},
				Prefix: aws.String("prefix-1"),
				Regex:  aws.String("am*zon"),
				Suffix: aws.String("suffix-1"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_HeaderMatchMethod_To_SDK_HeaderMatchMethod(tt.args.crdObj, tt.args.sdkObj, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_HTTPRouteHeader_To_SDK_HttpRouteHeader(t *testing.T) {
	type args struct {
		crdObj *appmesh.HTTPRouteHeader
		sdkObj *appmeshsdk.HttpRouteHeader
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.HttpRouteHeader
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.HTTPRouteHeader{
					Name: "User-Agent: X",
					Match: &appmesh.HeaderMatchMethod{
						Exact: aws.String("User-Agent: X"),
						Range: &appmesh.MatchRange{
							Start: int64(20),
							End:   int64(80),
						},
						Prefix: aws.String("prefix-1"),
						Regex:  aws.String("am*zon"),
						Suffix: aws.String("suffix-1"),
					},
					Invert: aws.Bool(false),
				},
				sdkObj: &appmeshsdk.HttpRouteHeader{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.HttpRouteHeader{
				Name: aws.String("User-Agent: X"),
				Match: &appmeshsdk.HeaderMatchMethod{
					Exact: aws.String("User-Agent: X"),
					Range: &appmeshsdk.MatchRange{
						Start: aws.Int64(20),
						End:   aws.Int64(80),
					},
					Prefix: aws.String("prefix-1"),
					Regex:  aws.String("am*zon"),
					Suffix: aws.String("suffix-1"),
				},
				Invert: aws.Bool(false),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_HTTPRouteHeader_To_SDK_HttpRouteHeader(tt.args.crdObj, tt.args.sdkObj, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_HTTPRouteMatch_To_SDK_HttpRouteMatch(t *testing.T) {
	type args struct {
		crdObj *appmesh.HTTPRouteMatch
		sdkObj *appmeshsdk.HttpRouteMatch
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.HttpRouteMatch
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.HTTPRouteMatch{
					Headers: []appmesh.HTTPRouteHeader{
						{
							Name: "User-Agent: X",
							Match: &appmesh.HeaderMatchMethod{
								Exact: aws.String("User-Agent: X"),
								Range: &appmesh.MatchRange{
									Start: int64(20),
									End:   int64(80),
								},
								Prefix: aws.String("prefix-1"),
								Regex:  aws.String("am*zon"),
								Suffix: aws.String("suffix-1"),
							},
							Invert: aws.Bool(false),
						},
						{
							Name: "User-Agent: Y",
							Match: &appmesh.HeaderMatchMethod{
								Exact: aws.String("User-Agent: Y"),
								Range: &appmesh.MatchRange{
									Start: int64(20),
									End:   int64(80),
								},
								Prefix: aws.String("prefix-2"),
								Regex:  aws.String("am*zon"),
								Suffix: aws.String("suffix-2"),
							},
							Invert: aws.Bool(true),
						},
					},
					Method: aws.String("GET"),
					Prefix: "/appmesh",
					Scheme: aws.String("https"),
				},

				sdkObj: &appmeshsdk.HttpRouteMatch{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.HttpRouteMatch{
				Headers: []*appmeshsdk.HttpRouteHeader{
					{
						Name: aws.String("User-Agent: X"),
						Match: &appmeshsdk.HeaderMatchMethod{
							Exact: aws.String("User-Agent: X"),
							Range: &appmeshsdk.MatchRange{
								Start: aws.Int64(20),
								End:   aws.Int64(80),
							},
							Prefix: aws.String("prefix-1"),
							Regex:  aws.String("am*zon"),
							Suffix: aws.String("suffix-1"),
						},
						Invert: aws.Bool(false),
					},
					{
						Name: aws.String("User-Agent: Y"),
						Match: &appmeshsdk.HeaderMatchMethod{
							Exact: aws.String("User-Agent: Y"),
							Range: &appmeshsdk.MatchRange{
								Start: aws.Int64(20),
								End:   aws.Int64(80),
							},
							Prefix: aws.String("prefix-2"),
							Regex:  aws.String("am*zon"),
							Suffix: aws.String("suffix-2"),
						},
						Invert: aws.Bool(true),
					},
				},
				Method: aws.String("GET"),
				Prefix: aws.String("/appmesh"),
				Scheme: aws.String("https"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_HTTPRouteMatch_To_SDK_HttpRouteMatch(tt.args.crdObj, tt.args.sdkObj, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_HTTPRouteAction_To_SDK_HttpRouteAction(t *testing.T) {
	type args struct {
		crdObj           *appmesh.HTTPRouteAction
		sdkObj           *appmeshsdk.HttpRouteAction
		scopeConvertFunc func(src, dest interface{}, flags conversion.FieldMatchingFlags) error
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.HttpRouteAction
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.HTTPRouteAction{
					WeightedTargets: []appmesh.WeightedTarget{
						{
							VirtualNodeRef: appmesh.VirtualNodeReference{
								Namespace: aws.String("ns-1"),
								Name:      "vn-1",
							},
							Weight: int64(100),
						},
						{
							VirtualNodeRef: appmesh.VirtualNodeReference{
								Namespace: aws.String("ns-2"),
								Name:      "vn-2",
							},
							Weight: int64(90),
						},
					},
				},
				sdkObj: &appmeshsdk.HttpRouteAction{},
				scopeConvertFunc: func(src, dest interface{}, flags conversion.FieldMatchingFlags) error {
					vnRef := src.(*appmesh.VirtualNodeReference)
					vnNamePtr := dest.(*string)
					*vnNamePtr = fmt.Sprintf("%s.%s", vnRef.Name, aws.StringValue(vnRef.Namespace))
					return nil
				},
			},
			wantSDKObj: &appmeshsdk.HttpRouteAction{
				WeightedTargets: []*appmeshsdk.WeightedTarget{
					{
						VirtualNode: aws.String("vn-1.ns-1"),
						Weight:      aws.Int64(100),
					},
					{
						VirtualNode: aws.String("vn-2.ns-2"),
						Weight:      aws.Int64(90),
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

			err := Convert_CRD_HTTPRouteAction_To_SDK_HttpRouteAction(tt.args.crdObj, tt.args.sdkObj, scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_HTTPRetryPolicy_To_SDK_HttpRetryPolicy(t *testing.T) {
	type args struct {
		crdObj *appmesh.HTTPRetryPolicy
		sdkObj *appmeshsdk.HttpRetryPolicy
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.HttpRetryPolicy
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.HTTPRetryPolicy{
					HTTPRetryEvents: []appmesh.HTTPRetryPolicyEvent{"server-error", "client-error"},
					TCPRetryEvents:  []appmesh.TCPRetryPolicyEvent{"connection-error"},
					MaxRetries:      int64(5),
					PerRetryTimeout: appmesh.Duration{
						Unit:  "ms",
						Value: int64(200),
					},
				},
				sdkObj: &appmeshsdk.HttpRetryPolicy{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.HttpRetryPolicy{
				HttpRetryEvents: []*string{aws.String("server-error"), aws.String("client-error")},
				TcpRetryEvents:  []*string{aws.String("connection-error")},
				MaxRetries:      aws.Int64(5),
				PerRetryTimeout: &appmeshsdk.Duration{
					Unit:  aws.String("ms"),
					Value: aws.Int64(200),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_HTTPRetryPolicy_To_SDK_HttpRetryPolicy(tt.args.crdObj, tt.args.sdkObj, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_HTTPRoute_To_SDK_HttpRoute(t *testing.T) {
	type args struct {
		crdObj           *appmesh.HTTPRoute
		sdkObj           *appmeshsdk.HttpRoute
		scopeConvertFunc func(src, dest interface{}, flags conversion.FieldMatchingFlags) error
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.HttpRoute
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.HTTPRoute{
					Match: appmesh.HTTPRouteMatch{
						Headers: []appmesh.HTTPRouteHeader{
							{
								Name: "User-Agent: X",
								Match: &appmesh.HeaderMatchMethod{
									Exact: aws.String("User-Agent: X"),
									Range: &appmesh.MatchRange{
										Start: int64(20),
										End:   int64(80),
									},
									Prefix: aws.String("prefix-1"),
									Regex:  aws.String("am*zon"),
									Suffix: aws.String("suffix-1"),
								},
								Invert: aws.Bool(false),
							},
							{
								Name: "User-Agent: Y",
								Match: &appmesh.HeaderMatchMethod{
									Exact: aws.String("User-Agent: Y"),
									Range: &appmesh.MatchRange{
										Start: int64(20),
										End:   int64(80),
									},
									Prefix: aws.String("prefix-2"),
									Regex:  aws.String("am*zon"),
									Suffix: aws.String("suffix-2"),
								},
								Invert: aws.Bool(true),
							},
						},
						Method: aws.String("GET"),
						Prefix: "/appmesh",
						Scheme: aws.String("https"),
					},
					Action: appmesh.HTTPRouteAction{
						WeightedTargets: []appmesh.WeightedTarget{
							{
								VirtualNodeRef: appmesh.VirtualNodeReference{
									Namespace: aws.String("ns-1"),
									Name:      "vn-1",
								},
								Weight: int64(100),
							},
							{
								VirtualNodeRef: appmesh.VirtualNodeReference{
									Namespace: aws.String("ns-2"),
									Name:      "vn-2",
								},
								Weight: int64(90),
							},
						},
					},
					RetryPolicy: &appmesh.HTTPRetryPolicy{
						HTTPRetryEvents: []appmesh.HTTPRetryPolicyEvent{"server-error", "client-error"},
						TCPRetryEvents:  []appmesh.TCPRetryPolicyEvent{"connection-error"},
						MaxRetries:      int64(5),
						PerRetryTimeout: appmesh.Duration{
							Unit:  "ms",
							Value: int64(200),
						},
					},
				},
				sdkObj: &appmeshsdk.HttpRoute{},
				scopeConvertFunc: func(src, dest interface{}, flags conversion.FieldMatchingFlags) error {
					vnRef := src.(*appmesh.VirtualNodeReference)
					vnNamePtr := dest.(*string)
					*vnNamePtr = fmt.Sprintf("%s.%s", vnRef.Name, aws.StringValue(vnRef.Namespace))
					return nil
				},
			},
			wantSDKObj: &appmeshsdk.HttpRoute{
				Match: &appmeshsdk.HttpRouteMatch{
					Headers: []*appmeshsdk.HttpRouteHeader{
						{
							Name: aws.String("User-Agent: X"),
							Match: &appmeshsdk.HeaderMatchMethod{
								Exact: aws.String("User-Agent: X"),
								Range: &appmeshsdk.MatchRange{
									Start: aws.Int64(20),
									End:   aws.Int64(80),
								},
								Prefix: aws.String("prefix-1"),
								Regex:  aws.String("am*zon"),
								Suffix: aws.String("suffix-1"),
							},
							Invert: aws.Bool(false),
						},
						{
							Name: aws.String("User-Agent: Y"),
							Match: &appmeshsdk.HeaderMatchMethod{
								Exact: aws.String("User-Agent: Y"),
								Range: &appmeshsdk.MatchRange{
									Start: aws.Int64(20),
									End:   aws.Int64(80),
								},
								Prefix: aws.String("prefix-2"),
								Regex:  aws.String("am*zon"),
								Suffix: aws.String("suffix-2"),
							},
							Invert: aws.Bool(true),
						},
					},
					Method: aws.String("GET"),
					Prefix: aws.String("/appmesh"),
					Scheme: aws.String("https"),
				},
				Action: &appmeshsdk.HttpRouteAction{
					WeightedTargets: []*appmeshsdk.WeightedTarget{
						{
							VirtualNode: aws.String("vn-1.ns-1"),
							Weight:      aws.Int64(100),
						},
						{
							VirtualNode: aws.String("vn-2.ns-2"),
							Weight:      aws.Int64(90),
						},
					},
				},
				RetryPolicy: &appmeshsdk.HttpRetryPolicy{
					HttpRetryEvents: []*string{aws.String("server-error"), aws.String("client-error")},
					TcpRetryEvents:  []*string{aws.String("connection-error")},
					MaxRetries:      aws.Int64(5),
					PerRetryTimeout: &appmeshsdk.Duration{
						Unit:  aws.String("ms"),
						Value: aws.Int64(200),
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

			err := Convert_CRD_HTTPRoute_To_SDK_HttpRoute(tt.args.crdObj, tt.args.sdkObj, scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_TCPRouteAction_To_SDK_TcpRouteAction(t *testing.T) {
	type args struct {
		crdObj           *appmesh.TCPRouteAction
		sdkObj           *appmeshsdk.TcpRouteAction
		scopeConvertFunc func(src, dest interface{}, flags conversion.FieldMatchingFlags) error
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.TcpRouteAction
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.TCPRouteAction{
					WeightedTargets: []appmesh.WeightedTarget{
						{
							VirtualNodeRef: appmesh.VirtualNodeReference{
								Namespace: aws.String("ns-1"),
								Name:      "vn-1",
							},
							Weight: int64(100),
						},
						{
							VirtualNodeRef: appmesh.VirtualNodeReference{
								Namespace: aws.String("ns-2"),
								Name:      "vn-2",
							},
							Weight: int64(90),
						},
					},
				},
				sdkObj: &appmeshsdk.TcpRouteAction{},
				scopeConvertFunc: func(src, dest interface{}, flags conversion.FieldMatchingFlags) error {
					vnRef := src.(*appmesh.VirtualNodeReference)
					vnNamePtr := dest.(*string)
					*vnNamePtr = fmt.Sprintf("%s.%s", vnRef.Name, aws.StringValue(vnRef.Namespace))
					return nil
				},
			},
			wantSDKObj: &appmeshsdk.TcpRouteAction{
				WeightedTargets: []*appmeshsdk.WeightedTarget{
					{
						VirtualNode: aws.String("vn-1.ns-1"),
						Weight:      aws.Int64(100),
					},
					{
						VirtualNode: aws.String("vn-2.ns-2"),
						Weight:      aws.Int64(90),
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

			err := Convert_CRD_TCPRouteAction_To_SDK_TcpRouteAction(tt.args.crdObj, tt.args.sdkObj, scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_TCPRoute_To_SDK_TcpRoute(t *testing.T) {
	type args struct {
		crdObj           *appmesh.TCPRoute
		sdkObj           *appmeshsdk.TcpRoute
		scopeConvertFunc func(src, dest interface{}, flags conversion.FieldMatchingFlags) error
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.TcpRoute
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.TCPRoute{
					Action: appmesh.TCPRouteAction{
						WeightedTargets: []appmesh.WeightedTarget{
							{
								VirtualNodeRef: appmesh.VirtualNodeReference{
									Namespace: aws.String("ns-1"),
									Name:      "vn-1",
								},
								Weight: int64(100),
							},
							{
								VirtualNodeRef: appmesh.VirtualNodeReference{
									Namespace: aws.String("ns-2"),
									Name:      "vn-2",
								},
								Weight: int64(90),
							},
						},
					},
				},
				sdkObj: &appmeshsdk.TcpRoute{},
				scopeConvertFunc: func(src, dest interface{}, flags conversion.FieldMatchingFlags) error {
					vnRef := src.(*appmesh.VirtualNodeReference)
					vnNamePtr := dest.(*string)
					*vnNamePtr = fmt.Sprintf("%s.%s", vnRef.Name, aws.StringValue(vnRef.Namespace))
					return nil
				},
			},
			wantSDKObj: &appmeshsdk.TcpRoute{
				Action: &appmeshsdk.TcpRouteAction{
					WeightedTargets: []*appmeshsdk.WeightedTarget{
						{
							VirtualNode: aws.String("vn-1.ns-1"),
							Weight:      aws.Int64(100),
						},
						{
							VirtualNode: aws.String("vn-2.ns-2"),
							Weight:      aws.Int64(90),
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

			err := Convert_CRD_TCPRoute_To_SDK_TcpRoute(tt.args.crdObj, tt.args.sdkObj, scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_GRPCRouteMetadataMatchMethod_To_SDK_GrpcRouteMetadataMatchMethod(t *testing.T) {
	type args struct {
		crdObj *appmesh.GRPCRouteMetadataMatchMethod
		sdkObj *appmeshsdk.GrpcRouteMetadataMatchMethod
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.GrpcRouteMetadataMatchMethod
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.GRPCRouteMetadataMatchMethod{
					Exact: aws.String("header1"),
					Range: &appmesh.MatchRange{
						Start: int64(20),
						End:   int64(80),
					},
					Prefix: aws.String("prefix-1"),
					Regex:  aws.String("am*zon"),
					Suffix: aws.String("suffix-1"),
				},
				sdkObj: &appmeshsdk.GrpcRouteMetadataMatchMethod{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.GrpcRouteMetadataMatchMethod{
				Exact: aws.String("header1"),
				Range: &appmeshsdk.MatchRange{
					Start: aws.Int64(20),
					End:   aws.Int64(80),
				},
				Prefix: aws.String("prefix-1"),
				Regex:  aws.String("am*zon"),
				Suffix: aws.String("suffix-1"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_GRPCRouteMetadataMatchMethod_To_SDK_GrpcRouteMetadataMatchMethod(tt.args.crdObj, tt.args.sdkObj, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_GRPCRouteMetadata_To_SDK_GrpcRouteMetadata(t *testing.T) {
	type args struct {
		crdObj *appmesh.GRPCRouteMetadata
		sdkObj *appmeshsdk.GrpcRouteMetadata
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.GrpcRouteMetadata
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.GRPCRouteMetadata{
					Name: "User-Agent: X",
					Match: &appmesh.GRPCRouteMetadataMatchMethod{
						Exact: aws.String("User-Agent: X"),
						Range: &appmesh.MatchRange{
							Start: int64(20),
							End:   int64(80),
						},
						Prefix: aws.String("prefix-1"),
						Regex:  aws.String("am*zon"),
						Suffix: aws.String("suffix-1"),
					},
					Invert: aws.Bool(false),
				},
				sdkObj: &appmeshsdk.GrpcRouteMetadata{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.GrpcRouteMetadata{
				Name: aws.String("User-Agent: X"),
				Match: &appmeshsdk.GrpcRouteMetadataMatchMethod{
					Exact: aws.String("User-Agent: X"),
					Range: &appmeshsdk.MatchRange{
						Start: aws.Int64(20),
						End:   aws.Int64(80),
					},
					Prefix: aws.String("prefix-1"),
					Regex:  aws.String("am*zon"),
					Suffix: aws.String("suffix-1"),
				},
				Invert: aws.Bool(false),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_GRPCRouteMetadata_To_SDK_GrpcRouteMetadata(tt.args.crdObj, tt.args.sdkObj, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_GRPCRouteMatch_To_SDK_GrpcRouteMatch(t *testing.T) {
	type args struct {
		crdObj *appmesh.GRPCRouteMatch
		sdkObj *appmeshsdk.GrpcRouteMatch
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.GrpcRouteMatch
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.GRPCRouteMatch{
					Metadata: []appmesh.GRPCRouteMetadata{
						{
							Name: "User-Agent: X",
							Match: &appmesh.GRPCRouteMetadataMatchMethod{
								Exact: aws.String("User-Agent: X"),
								Range: &appmesh.MatchRange{
									Start: int64(20),
									End:   int64(80),
								},
								Prefix: aws.String("prefix-1"),
								Regex:  aws.String("am*zon"),
								Suffix: aws.String("suffix-1"),
							},
							Invert: aws.Bool(false),
						},
						{
							Name: "User-Agent: Y",
							Match: &appmesh.GRPCRouteMetadataMatchMethod{
								Exact: aws.String("User-Agent: Y"),
								Range: &appmesh.MatchRange{
									Start: int64(20),
									End:   int64(80),
								},
								Prefix: aws.String("prefix-2"),
								Regex:  aws.String("am*zon"),
								Suffix: aws.String("suffix-2"),
							},
							Invert: aws.Bool(true),
						},
					},
					MethodName:  aws.String("stream"),
					ServiceName: aws.String("foo.foodomain.local"),
				},

				sdkObj: &appmeshsdk.GrpcRouteMatch{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.GrpcRouteMatch{
				Metadata: []*appmeshsdk.GrpcRouteMetadata{
					{
						Name: aws.String("User-Agent: X"),
						Match: &appmeshsdk.GrpcRouteMetadataMatchMethod{
							Exact: aws.String("User-Agent: X"),
							Range: &appmeshsdk.MatchRange{
								Start: aws.Int64(20),
								End:   aws.Int64(80),
							},
							Prefix: aws.String("prefix-1"),
							Regex:  aws.String("am*zon"),
							Suffix: aws.String("suffix-1"),
						},
						Invert: aws.Bool(false),
					},
					{
						Name: aws.String("User-Agent: Y"),
						Match: &appmeshsdk.GrpcRouteMetadataMatchMethod{
							Exact: aws.String("User-Agent: Y"),
							Range: &appmeshsdk.MatchRange{
								Start: aws.Int64(20),
								End:   aws.Int64(80),
							},
							Prefix: aws.String("prefix-2"),
							Regex:  aws.String("am*zon"),
							Suffix: aws.String("suffix-2"),
						},
						Invert: aws.Bool(true),
					},
				},
				MethodName:  aws.String("stream"),
				ServiceName: aws.String("foo.foodomain.local"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_GRPCRouteMatch_To_SDK_GrpcRouteMatch(tt.args.crdObj, tt.args.sdkObj, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_GRPCRouteAction_To_SDK_GrpcRouteAction(t *testing.T) {
	type args struct {
		crdObj           *appmesh.GRPCRouteAction
		sdkObj           *appmeshsdk.GrpcRouteAction
		scopeConvertFunc func(src, dest interface{}, flags conversion.FieldMatchingFlags) error
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.GrpcRouteAction
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.GRPCRouteAction{
					WeightedTargets: []appmesh.WeightedTarget{
						{
							VirtualNodeRef: appmesh.VirtualNodeReference{
								Namespace: aws.String("ns-1"),
								Name:      "vn-1",
							},
							Weight: int64(100),
						},
						{
							VirtualNodeRef: appmesh.VirtualNodeReference{
								Namespace: aws.String("ns-2"),
								Name:      "vn-2",
							},
							Weight: int64(90),
						},
					},
				},
				sdkObj: &appmeshsdk.GrpcRouteAction{},
				scopeConvertFunc: func(src, dest interface{}, flags conversion.FieldMatchingFlags) error {
					vnRef := src.(*appmesh.VirtualNodeReference)
					vnNamePtr := dest.(*string)
					*vnNamePtr = fmt.Sprintf("%s.%s", vnRef.Name, aws.StringValue(vnRef.Namespace))
					return nil
				},
			},
			wantSDKObj: &appmeshsdk.GrpcRouteAction{
				WeightedTargets: []*appmeshsdk.WeightedTarget{
					{
						VirtualNode: aws.String("vn-1.ns-1"),
						Weight:      aws.Int64(100),
					},
					{
						VirtualNode: aws.String("vn-2.ns-2"),
						Weight:      aws.Int64(90),
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

			err := Convert_CRD_GRPCRouteAction_To_SDK_GrpcRouteAction(tt.args.crdObj, tt.args.sdkObj, scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_GRPCRetryPolicy_To_SDK_GrpcRetryPolicy(t *testing.T) {
	type args struct {
		crdObj *appmesh.GRPCRetryPolicy
		sdkObj *appmeshsdk.GrpcRetryPolicy
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.GrpcRetryPolicy
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.GRPCRetryPolicy{
					GRPCRetryEvents: []appmesh.GRPCRetryPolicyEvent{"cancelled", "deadline-exceeded"},
					HTTPRetryEvents: []appmesh.HTTPRetryPolicyEvent{"server-error", "client-error"},
					TCPRetryEvents:  []appmesh.TCPRetryPolicyEvent{"connection-error"},
					MaxRetries:      int64(5),
					PerRetryTimeout: appmesh.Duration{
						Unit:  "ms",
						Value: int64(200),
					},
				},
				sdkObj: &appmeshsdk.GrpcRetryPolicy{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.GrpcRetryPolicy{
				GrpcRetryEvents: []*string{aws.String("cancelled"), aws.String("deadline-exceeded")},
				HttpRetryEvents: []*string{aws.String("server-error"), aws.String("client-error")},
				TcpRetryEvents:  []*string{aws.String("connection-error")},
				MaxRetries:      aws.Int64(5),
				PerRetryTimeout: &appmeshsdk.Duration{
					Unit:  aws.String("ms"),
					Value: aws.Int64(200),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_GRPCRetryPolicy_To_SDK_GrpcRetryPolicy(tt.args.crdObj, tt.args.sdkObj, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_GRPCRoute_To_SDK_GrpcRoute(t *testing.T) {
	type args struct {
		crdObj           *appmesh.GRPCRoute
		sdkObj           *appmeshsdk.GrpcRoute
		scopeConvertFunc func(src, dest interface{}, flags conversion.FieldMatchingFlags) error
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.GrpcRoute
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.GRPCRoute{
					Match: appmesh.GRPCRouteMatch{
						Metadata: []appmesh.GRPCRouteMetadata{
							{
								Name: "User-Agent: X",
								Match: &appmesh.GRPCRouteMetadataMatchMethod{
									Exact: aws.String("User-Agent: X"),
									Range: &appmesh.MatchRange{
										Start: int64(20),
										End:   int64(80),
									},
									Prefix: aws.String("prefix-1"),
									Regex:  aws.String("am*zon"),
									Suffix: aws.String("suffix-1"),
								},
								Invert: aws.Bool(false),
							},
							{
								Name: "User-Agent: Y",
								Match: &appmesh.GRPCRouteMetadataMatchMethod{
									Exact: aws.String("User-Agent: Y"),
									Range: &appmesh.MatchRange{
										Start: int64(20),
										End:   int64(80),
									},
									Prefix: aws.String("prefix-2"),
									Regex:  aws.String("am*zon"),
									Suffix: aws.String("suffix-2"),
								},
								Invert: aws.Bool(true),
							},
						},
						MethodName:  aws.String("stream"),
						ServiceName: aws.String("foo.foodomain.local"),
					},
					Action: appmesh.GRPCRouteAction{
						WeightedTargets: []appmesh.WeightedTarget{
							{
								VirtualNodeRef: appmesh.VirtualNodeReference{
									Namespace: aws.String("ns-1"),
									Name:      "vn-1",
								},
								Weight: int64(100),
							},
							{
								VirtualNodeRef: appmesh.VirtualNodeReference{
									Namespace: aws.String("ns-2"),
									Name:      "vn-2",
								},
								Weight: int64(90),
							},
						},
					},
					RetryPolicy: &appmesh.GRPCRetryPolicy{
						GRPCRetryEvents: []appmesh.GRPCRetryPolicyEvent{"cancelled", "deadline-exceeded"},
						HTTPRetryEvents: []appmesh.HTTPRetryPolicyEvent{"server-error", "client-error"},
						TCPRetryEvents:  []appmesh.TCPRetryPolicyEvent{"connection-error"},
						MaxRetries:      int64(5),
						PerRetryTimeout: appmesh.Duration{
							Unit:  "ms",
							Value: int64(200),
						},
					},
				},
				sdkObj: &appmeshsdk.GrpcRoute{},
				scopeConvertFunc: func(src, dest interface{}, flags conversion.FieldMatchingFlags) error {
					vnRef := src.(*appmesh.VirtualNodeReference)
					vnNamePtr := dest.(*string)
					*vnNamePtr = fmt.Sprintf("%s.%s", vnRef.Name, aws.StringValue(vnRef.Namespace))
					return nil
				},
			},
			wantSDKObj: &appmeshsdk.GrpcRoute{
				Match: &appmeshsdk.GrpcRouteMatch{
					Metadata: []*appmeshsdk.GrpcRouteMetadata{
						{
							Name: aws.String("User-Agent: X"),
							Match: &appmeshsdk.GrpcRouteMetadataMatchMethod{
								Exact: aws.String("User-Agent: X"),
								Range: &appmeshsdk.MatchRange{
									Start: aws.Int64(20),
									End:   aws.Int64(80),
								},
								Prefix: aws.String("prefix-1"),
								Regex:  aws.String("am*zon"),
								Suffix: aws.String("suffix-1"),
							},
							Invert: aws.Bool(false),
						},
						{
							Name: aws.String("User-Agent: Y"),
							Match: &appmeshsdk.GrpcRouteMetadataMatchMethod{
								Exact: aws.String("User-Agent: Y"),
								Range: &appmeshsdk.MatchRange{
									Start: aws.Int64(20),
									End:   aws.Int64(80),
								},
								Prefix: aws.String("prefix-2"),
								Regex:  aws.String("am*zon"),
								Suffix: aws.String("suffix-2"),
							},
							Invert: aws.Bool(true),
						},
					},
					MethodName:  aws.String("stream"),
					ServiceName: aws.String("foo.foodomain.local"),
				},
				Action: &appmeshsdk.GrpcRouteAction{
					WeightedTargets: []*appmeshsdk.WeightedTarget{
						{
							VirtualNode: aws.String("vn-1.ns-1"),
							Weight:      aws.Int64(100),
						},
						{
							VirtualNode: aws.String("vn-2.ns-2"),
							Weight:      aws.Int64(90),
						},
					},
				},
				RetryPolicy: &appmeshsdk.GrpcRetryPolicy{
					GrpcRetryEvents: []*string{aws.String("cancelled"), aws.String("deadline-exceeded")},
					HttpRetryEvents: []*string{aws.String("server-error"), aws.String("client-error")},
					TcpRetryEvents:  []*string{aws.String("connection-error")},
					MaxRetries:      aws.Int64(5),
					PerRetryTimeout: &appmeshsdk.Duration{
						Unit:  aws.String("ms"),
						Value: aws.Int64(200),
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

			err := Convert_CRD_GRPCRoute_To_SDK_GrpcRoute(tt.args.crdObj, tt.args.sdkObj, scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_Route_To_SDK_RouteSpec(t *testing.T) {
	type args struct {
		crdObj           *appmesh.Route
		sdkObj           *appmeshsdk.RouteSpec
		scopeConvertFunc func(src, dest interface{}, flags conversion.FieldMatchingFlags) error
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.RouteSpec
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.Route{
					Name: "route1",
					GRPCRoute: &appmesh.GRPCRoute{
						Match: appmesh.GRPCRouteMatch{
							Metadata: []appmesh.GRPCRouteMetadata{
								{
									Name: "User-Agent: X",
									Match: &appmesh.GRPCRouteMetadataMatchMethod{
										Exact: aws.String("User-Agent: X"),
										Range: &appmesh.MatchRange{
											Start: int64(20),
											End:   int64(80),
										},
										Prefix: aws.String("prefix-1"),
										Regex:  aws.String("am*zon"),
										Suffix: aws.String("suffix-1"),
									},
									Invert: aws.Bool(false),
								},
								{
									Name: "User-Agent: Y",
									Match: &appmesh.GRPCRouteMetadataMatchMethod{
										Exact: aws.String("User-Agent: Y"),
										Range: &appmesh.MatchRange{
											Start: int64(20),
											End:   int64(80),
										},
										Prefix: aws.String("prefix-2"),
										Regex:  aws.String("am*zon"),
										Suffix: aws.String("suffix-2"),
									},
									Invert: aws.Bool(true),
								},
							},
							MethodName:  aws.String("stream"),
							ServiceName: aws.String("foo.foodomain.local"),
						},
						Action: appmesh.GRPCRouteAction{
							WeightedTargets: []appmesh.WeightedTarget{
								{
									VirtualNodeRef: appmesh.VirtualNodeReference{
										Namespace: aws.String("ns-1"),
										Name:      "vn-1",
									},
									Weight: int64(100),
								},
								{
									VirtualNodeRef: appmesh.VirtualNodeReference{
										Namespace: aws.String("ns-2"),
										Name:      "vn-2",
									},
									Weight: int64(90),
								},
							},
						},
						RetryPolicy: &appmesh.GRPCRetryPolicy{
							GRPCRetryEvents: []appmesh.GRPCRetryPolicyEvent{"cancelled", "deadline-exceeded"},
							HTTPRetryEvents: []appmesh.HTTPRetryPolicyEvent{"server-error", "client-error"},
							TCPRetryEvents:  []appmesh.TCPRetryPolicyEvent{"connection-error"},
							MaxRetries:      int64(5),
							PerRetryTimeout: appmesh.Duration{
								Unit:  "ms",
								Value: int64(200),
							},
						},
					},
					HTTPRoute: &appmesh.HTTPRoute{
						Match: appmesh.HTTPRouteMatch{
							Headers: []appmesh.HTTPRouteHeader{
								{
									Name: "User-Agent: X",
									Match: &appmesh.HeaderMatchMethod{
										Exact: aws.String("User-Agent: X"),
										Range: &appmesh.MatchRange{
											Start: int64(20),
											End:   int64(80),
										},
										Prefix: aws.String("prefix-1"),
										Regex:  aws.String("am*zon"),
										Suffix: aws.String("suffix-1"),
									},
									Invert: aws.Bool(false),
								},
								{
									Name: "User-Agent: Y",
									Match: &appmesh.HeaderMatchMethod{
										Exact: aws.String("User-Agent: Y"),
										Range: &appmesh.MatchRange{
											Start: int64(20),
											End:   int64(80),
										},
										Prefix: aws.String("prefix-2"),
										Regex:  aws.String("am*zon"),
										Suffix: aws.String("suffix-2"),
									},
									Invert: aws.Bool(true),
								},
							},
							Method: aws.String("GET"),
							Prefix: "/appmesh",
							Scheme: aws.String("https"),
						},
						Action: appmesh.HTTPRouteAction{
							WeightedTargets: []appmesh.WeightedTarget{
								{
									VirtualNodeRef: appmesh.VirtualNodeReference{
										Namespace: aws.String("ns-1"),
										Name:      "vn-1",
									},
									Weight: int64(100),
								},
								{
									VirtualNodeRef: appmesh.VirtualNodeReference{
										Namespace: aws.String("ns-2"),
										Name:      "vn-2",
									},
									Weight: int64(90),
								},
							},
						},
						RetryPolicy: &appmesh.HTTPRetryPolicy{
							HTTPRetryEvents: []appmesh.HTTPRetryPolicyEvent{"server-error", "client-error"},
							TCPRetryEvents:  []appmesh.TCPRetryPolicyEvent{"connection-error"},
							MaxRetries:      int64(5),
							PerRetryTimeout: appmesh.Duration{
								Unit:  "ms",
								Value: int64(200),
							},
						},
					},

					HTTP2Route: &appmesh.HTTPRoute{
						Match: appmesh.HTTPRouteMatch{
							Headers: []appmesh.HTTPRouteHeader{
								{
									Name: "User-Agent: X",
									Match: &appmesh.HeaderMatchMethod{
										Exact: aws.String("User-Agent: X"),
										Range: &appmesh.MatchRange{
											Start: int64(20),
											End:   int64(80),
										},
										Prefix: aws.String("prefix-1"),
										Regex:  aws.String("am*zon"),
										Suffix: aws.String("suffix-1"),
									},
									Invert: aws.Bool(false),
								},
								{
									Name: "User-Agent: Y",
									Match: &appmesh.HeaderMatchMethod{
										Exact: aws.String("User-Agent: Y"),
										Range: &appmesh.MatchRange{
											Start: int64(20),
											End:   int64(80),
										},
										Prefix: aws.String("prefix-2"),
										Regex:  aws.String("am*zon"),
										Suffix: aws.String("suffix-2"),
									},
									Invert: aws.Bool(true),
								},
							},
							Method: aws.String("GET"),
							Prefix: "/appmesh",
							Scheme: aws.String("https"),
						},
						Action: appmesh.HTTPRouteAction{
							WeightedTargets: []appmesh.WeightedTarget{
								{
									VirtualNodeRef: appmesh.VirtualNodeReference{
										Namespace: aws.String("ns-1"),
										Name:      "vn-1",
									},
									Weight: int64(100),
								},
								{
									VirtualNodeRef: appmesh.VirtualNodeReference{
										Namespace: aws.String("ns-2"),
										Name:      "vn-2",
									},
									Weight: int64(90),
								},
							},
						},
						RetryPolicy: &appmesh.HTTPRetryPolicy{
							HTTPRetryEvents: []appmesh.HTTPRetryPolicyEvent{"server-error", "client-error"},
							TCPRetryEvents:  []appmesh.TCPRetryPolicyEvent{"connection-error"},
							MaxRetries:      int64(5),
							PerRetryTimeout: appmesh.Duration{
								Unit:  "ms",
								Value: int64(200),
							},
						},
					},
					TCPRoute: &appmesh.TCPRoute{
						Action: appmesh.TCPRouteAction{
							WeightedTargets: []appmesh.WeightedTarget{
								{
									VirtualNodeRef: appmesh.VirtualNodeReference{
										Namespace: aws.String("ns-1"),
										Name:      "vn-1",
									},
									Weight: int64(100),
								},
								{
									VirtualNodeRef: appmesh.VirtualNodeReference{
										Namespace: aws.String("ns-2"),
										Name:      "vn-2",
									},
									Weight: int64(90),
								},
							},
						},
					},
					Priority: aws.Int64(400),
				},
				sdkObj: &appmeshsdk.RouteSpec{},
				scopeConvertFunc: func(src, dest interface{}, flags conversion.FieldMatchingFlags) error {
					vnRef := src.(*appmesh.VirtualNodeReference)
					vnNamePtr := dest.(*string)
					*vnNamePtr = fmt.Sprintf("%s.%s", vnRef.Name, aws.StringValue(vnRef.Namespace))
					return nil
				},
			},
			wantSDKObj: &appmeshsdk.RouteSpec{
				GrpcRoute: &appmeshsdk.GrpcRoute{
					Match: &appmeshsdk.GrpcRouteMatch{
						Metadata: []*appmeshsdk.GrpcRouteMetadata{
							{
								Name: aws.String("User-Agent: X"),
								Match: &appmeshsdk.GrpcRouteMetadataMatchMethod{
									Exact: aws.String("User-Agent: X"),
									Range: &appmeshsdk.MatchRange{
										Start: aws.Int64(20),
										End:   aws.Int64(80),
									},
									Prefix: aws.String("prefix-1"),
									Regex:  aws.String("am*zon"),
									Suffix: aws.String("suffix-1"),
								},
								Invert: aws.Bool(false),
							},
							{
								Name: aws.String("User-Agent: Y"),
								Match: &appmeshsdk.GrpcRouteMetadataMatchMethod{
									Exact: aws.String("User-Agent: Y"),
									Range: &appmeshsdk.MatchRange{
										Start: aws.Int64(20),
										End:   aws.Int64(80),
									},
									Prefix: aws.String("prefix-2"),
									Regex:  aws.String("am*zon"),
									Suffix: aws.String("suffix-2"),
								},
								Invert: aws.Bool(true),
							},
						},
						MethodName:  aws.String("stream"),
						ServiceName: aws.String("foo.foodomain.local"),
					},
					Action: &appmeshsdk.GrpcRouteAction{
						WeightedTargets: []*appmeshsdk.WeightedTarget{
							{
								VirtualNode: aws.String("vn-1.ns-1"),
								Weight:      aws.Int64(100),
							},
							{
								VirtualNode: aws.String("vn-2.ns-2"),
								Weight:      aws.Int64(90),
							},
						},
					},
					RetryPolicy: &appmeshsdk.GrpcRetryPolicy{
						GrpcRetryEvents: []*string{aws.String("cancelled"), aws.String("deadline-exceeded")},
						HttpRetryEvents: []*string{aws.String("server-error"), aws.String("client-error")},
						TcpRetryEvents:  []*string{aws.String("connection-error")},
						MaxRetries:      aws.Int64(5),
						PerRetryTimeout: &appmeshsdk.Duration{
							Unit:  aws.String("ms"),
							Value: aws.Int64(200),
						},
					},
				},
				HttpRoute: &appmeshsdk.HttpRoute{
					Match: &appmeshsdk.HttpRouteMatch{
						Headers: []*appmeshsdk.HttpRouteHeader{
							{
								Name: aws.String("User-Agent: X"),
								Match: &appmeshsdk.HeaderMatchMethod{
									Exact: aws.String("User-Agent: X"),
									Range: &appmeshsdk.MatchRange{
										Start: aws.Int64(20),
										End:   aws.Int64(80),
									},
									Prefix: aws.String("prefix-1"),
									Regex:  aws.String("am*zon"),
									Suffix: aws.String("suffix-1"),
								},
								Invert: aws.Bool(false),
							},
							{
								Name: aws.String("User-Agent: Y"),
								Match: &appmeshsdk.HeaderMatchMethod{
									Exact: aws.String("User-Agent: Y"),
									Range: &appmeshsdk.MatchRange{
										Start: aws.Int64(20),
										End:   aws.Int64(80),
									},
									Prefix: aws.String("prefix-2"),
									Regex:  aws.String("am*zon"),
									Suffix: aws.String("suffix-2"),
								},
								Invert: aws.Bool(true),
							},
						},
						Method: aws.String("GET"),
						Prefix: aws.String("/appmesh"),
						Scheme: aws.String("https"),
					},
					Action: &appmeshsdk.HttpRouteAction{
						WeightedTargets: []*appmeshsdk.WeightedTarget{
							{
								VirtualNode: aws.String("vn-1.ns-1"),
								Weight:      aws.Int64(100),
							},
							{
								VirtualNode: aws.String("vn-2.ns-2"),
								Weight:      aws.Int64(90),
							},
						},
					},
					RetryPolicy: &appmeshsdk.HttpRetryPolicy{
						HttpRetryEvents: []*string{aws.String("server-error"), aws.String("client-error")},
						TcpRetryEvents:  []*string{aws.String("connection-error")},
						MaxRetries:      aws.Int64(5),
						PerRetryTimeout: &appmeshsdk.Duration{
							Unit:  aws.String("ms"),
							Value: aws.Int64(200),
						},
					},
				},
				Http2Route: &appmeshsdk.HttpRoute{
					Match: &appmeshsdk.HttpRouteMatch{
						Headers: []*appmeshsdk.HttpRouteHeader{
							{
								Name: aws.String("User-Agent: X"),
								Match: &appmeshsdk.HeaderMatchMethod{
									Exact: aws.String("User-Agent: X"),
									Range: &appmeshsdk.MatchRange{
										Start: aws.Int64(20),
										End:   aws.Int64(80),
									},
									Prefix: aws.String("prefix-1"),
									Regex:  aws.String("am*zon"),
									Suffix: aws.String("suffix-1"),
								},
								Invert: aws.Bool(false),
							},
							{
								Name: aws.String("User-Agent: Y"),
								Match: &appmeshsdk.HeaderMatchMethod{
									Exact: aws.String("User-Agent: Y"),
									Range: &appmeshsdk.MatchRange{
										Start: aws.Int64(20),
										End:   aws.Int64(80),
									},
									Prefix: aws.String("prefix-2"),
									Regex:  aws.String("am*zon"),
									Suffix: aws.String("suffix-2"),
								},
								Invert: aws.Bool(true),
							},
						},
						Method: aws.String("GET"),
						Prefix: aws.String("/appmesh"),
						Scheme: aws.String("https"),
					},
					Action: &appmeshsdk.HttpRouteAction{
						WeightedTargets: []*appmeshsdk.WeightedTarget{
							{
								VirtualNode: aws.String("vn-1.ns-1"),
								Weight:      aws.Int64(100),
							},
							{
								VirtualNode: aws.String("vn-2.ns-2"),
								Weight:      aws.Int64(90),
							},
						},
					},
					RetryPolicy: &appmeshsdk.HttpRetryPolicy{
						HttpRetryEvents: []*string{aws.String("server-error"), aws.String("client-error")},
						TcpRetryEvents:  []*string{aws.String("connection-error")},
						MaxRetries:      aws.Int64(5),
						PerRetryTimeout: &appmeshsdk.Duration{
							Unit:  aws.String("ms"),
							Value: aws.Int64(200),
						},
					},
				},
				TcpRoute: &appmeshsdk.TcpRoute{
					Action: &appmeshsdk.TcpRouteAction{
						WeightedTargets: []*appmeshsdk.WeightedTarget{
							{
								VirtualNode: aws.String("vn-1.ns-1"),
								Weight:      aws.Int64(100),
							},
							{
								VirtualNode: aws.String("vn-2.ns-2"),
								Weight:      aws.Int64(90),
							},
						},
					},
				},
				Priority: aws.Int64(400),
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

			err := Convert_CRD_Route_To_SDK_RouteSpec(tt.args.crdObj, tt.args.sdkObj, scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}
