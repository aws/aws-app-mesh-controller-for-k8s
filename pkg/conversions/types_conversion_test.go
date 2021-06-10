package conversions

import (
	"testing"

	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-sdk-go/aws"
	appmeshsdk "github.com/aws/aws-sdk-go/service/appmesh"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/conversion"
)

func TestConvert_CRD_PortMapping_To_SDK_PortMapping(t *testing.T) {
	type args struct {
		crdObj *appmesh.PortMapping
		sdkObj *appmeshsdk.PortMapping
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.PortMapping
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.PortMapping{
					Port:     8080,
					Protocol: "http",
				},
				sdkObj: &appmeshsdk.PortMapping{},
			},
			wantSDKObj: &appmeshsdk.PortMapping{
				Port:     aws.Int64(8080),
				Protocol: aws.String("http"),
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_PortMapping_To_SDK_PortMapping(tt.args.crdObj, tt.args.sdkObj, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_Duration_To_SDK_Duration(t *testing.T) {
	type args struct {
		crdObj *appmesh.Duration
		sdkObj *appmeshsdk.Duration
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.Duration
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.Duration{
					Unit:  "ms",
					Value: int64(200),
				},
				sdkObj: &appmeshsdk.Duration{},
			},
			wantSDKObj: &appmeshsdk.Duration{
				Unit:  aws.String("ms"),
				Value: aws.Int64(200),
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_Duration_To_SDK_Duration(tt.args.crdObj, tt.args.sdkObj, tt.args.scope)
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
		{
			name: "normal case + nil exact",
			args: args{
				crdObj: &appmesh.HeaderMatchMethod{
					Exact: nil,
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
				Exact: nil,
				Range: &appmeshsdk.MatchRange{
					Start: aws.Int64(20),
					End:   aws.Int64(80),
				},
				Prefix: aws.String("prefix-1"),
				Regex:  aws.String("am*zon"),
				Suffix: aws.String("suffix-1"),
			},
		},
		{
			name: "normal case + nil range",
			args: args{
				crdObj: &appmesh.HeaderMatchMethod{
					Exact:  aws.String("header1"),
					Range:  nil,
					Prefix: aws.String("prefix-1"),
					Regex:  aws.String("am*zon"),
					Suffix: aws.String("suffix-1"),
				},
				sdkObj: &appmeshsdk.HeaderMatchMethod{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.HeaderMatchMethod{
				Exact:  aws.String("header1"),
				Range:  nil,
				Prefix: aws.String("prefix-1"),
				Regex:  aws.String("am*zon"),
				Suffix: aws.String("suffix-1"),
			},
		},
		{
			name: "normal case + nil prefix",
			args: args{
				crdObj: &appmesh.HeaderMatchMethod{
					Exact: aws.String("header1"),
					Range: &appmesh.MatchRange{
						Start: int64(20),
						End:   int64(80),
					},
					Prefix: nil,
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
				Prefix: nil,
				Regex:  aws.String("am*zon"),
				Suffix: aws.String("suffix-1"),
			},
		},
		{
			name: "normal case + nil regex",
			args: args{
				crdObj: &appmesh.HeaderMatchMethod{
					Exact: aws.String("header1"),
					Range: &appmesh.MatchRange{
						Start: int64(20),
						End:   int64(80),
					},
					Prefix: aws.String("prefix-1"),
					Regex:  nil,
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
				Regex:  nil,
				Suffix: aws.String("suffix-1"),
			},
		},
		{
			name: "normal case + nil suffix",
			args: args{
				crdObj: &appmesh.HeaderMatchMethod{
					Exact: aws.String("header1"),
					Range: &appmesh.MatchRange{
						Start: int64(20),
						End:   int64(80),
					},
					Prefix: aws.String("prefix-1"),
					Regex:  aws.String("am*zon"),
					Suffix: nil,
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
				Suffix: nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_HTTPHeaderMatchMethod_To_SDK_HttpHeaderMatchMethod(tt.args.crdObj, tt.args.sdkObj)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}
