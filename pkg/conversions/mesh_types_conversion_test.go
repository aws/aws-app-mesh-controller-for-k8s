package conversions

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-sdk-go/aws"
	appmeshsdk "github.com/aws/aws-sdk-go/service/appmesh"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/conversion"
	"testing"
)

func TestConvert_CRD_EgressFilter_To_SDK_EgressFilter(t *testing.T) {
	type args struct {
		crdObj *appmesh.EgressFilter
		sdkObj *appmeshsdk.EgressFilter
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.EgressFilter
		wantErr    error
	}{
		{
			name: "allow all egress filter",
			args: args{
				crdObj: &appmesh.EgressFilter{
					Type: appmesh.EgressFilterTypeAllowAll,
				},
				sdkObj: &appmeshsdk.EgressFilter{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.EgressFilter{
				Type: aws.String("ALLOW_ALL"),
			},
		},
		{
			name: "drop all egress filter",
			args: args{
				crdObj: &appmesh.EgressFilter{
					Type: appmesh.EgressFilterTypeDropAll,
				},
				sdkObj: &appmeshsdk.EgressFilter{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.EgressFilter{
				Type: aws.String("DROP_ALL"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_EgressFilter_To_SDK_EgressFilter(tt.args.crdObj, tt.args.sdkObj, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_MeshSpec_To_SDK_MeshSpec(t *testing.T) {
	type args struct {
		crdObj *appmesh.MeshSpec
		sdkObj *appmeshsdk.MeshSpec
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.MeshSpec
		wantErr    error
	}{
		{
			name: "non-nil egress filter",
			args: args{
				crdObj: &appmesh.MeshSpec{
					EgressFilter: &appmesh.EgressFilter{
						Type: appmesh.EgressFilterTypeAllowAll,
					},
				},
				sdkObj: &appmeshsdk.MeshSpec{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.MeshSpec{
				EgressFilter: &appmeshsdk.EgressFilter{
					Type: aws.String("ALLOW_ALL"),
				},
			},
		},
		{
			name: "nil egress filter",
			args: args{
				crdObj: &appmesh.MeshSpec{
					EgressFilter: nil,
				},
				sdkObj: &appmeshsdk.MeshSpec{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.MeshSpec{
				EgressFilter: nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_MeshSpec_To_SDK_MeshSpec(tt.args.crdObj, tt.args.sdkObj, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}
