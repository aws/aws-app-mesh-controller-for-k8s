package conversions

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/conversion"
	"testing"
)

func TestConvert_CRD_VirtualNodeARN_To_SDK_VirtualNodeName(t *testing.T) {
	type args struct {
		vnARN  *string
		vnName *string
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantVNName string
		wantErr    error
	}{
		{
			name: "valid virtualNode ARN",
			args: args{
				vnARN:  aws.String("arn:aws:appmesh:us-west-2:000000000000:mesh/mesh-name/virtualNode/vn-name"),
				vnName: aws.String(""),
				scope:  nil,
			},
			wantVNName: "vn-name",
		},
		{
			name: "valid virtualNode ARN - shared mesh",
			args: args{
				vnARN:  aws.String("arn:aws:appmesh:us-west-2:000000000000:mesh/mesh-name@111111111111/virtualNode/vn-name"),
				vnName: aws.String(""),
				scope:  nil,
			},
			wantVNName: "vn-name",
		},
		{
			name: "valid virtualService ARN",
			args: args{
				vnARN:  aws.String("arn:aws:appmesh:us-west-2:000000000000:mesh/mesh-name/virtualService/vs-name"),
				vnName: aws.String(""),
				scope:  nil,
			},
			wantErr: errors.New("expects virtualNode ARN, got virtualService"),
		},
		{
			name: "valid mesh ARN",
			args: args{
				vnARN:  aws.String("arn:aws:appmesh:us-west-2:000000000000:mesh/mesh-name"),
				vnName: aws.String(""),
				scope:  nil,
			},
			wantErr: errors.New("invalid resource in appMesh ARN: mesh/mesh-name"),
		},
		{
			name: "invalid ARN",
			args: args{
				vnARN:  aws.String("xxxxx"),
				vnName: aws.String(""),
				scope:  nil,
			},
			wantErr: errors.New("invalid arn: arn: invalid prefix"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_VirtualNodeARN_To_SDK_VirtualNodeName(tt.args.vnARN, tt.args.vnName, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantVNName, *tt.args.vnName)
			}
		})
	}
}

func TestConvert_CRD_VirtualServiceARN_To_SDK_VirtualServiceName(t *testing.T) {
	type args struct {
		vsARN  *string
		vsName *string
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantVSName string
		wantErr    error
	}{
		{
			name: "valid virtualService ARN",
			args: args{
				vsARN:  aws.String("arn:aws:appmesh:us-west-2:000000000000:mesh/mesh-name/virtualService/vs-name"),
				vsName: aws.String(""),
				scope:  nil,
			},
			wantVSName: "vs-name",
		},
		{
			name: "valid virtualService ARN - shared mesh",
			args: args{
				vsARN:  aws.String("arn:aws:appmesh:us-west-2:000000000000:mesh/mesh-name@111111111111/virtualService/vs-name"),
				vsName: aws.String(""),
				scope:  nil,
			},
			wantVSName: "vs-name",
		},
		{
			name: "valid virtualNode ARN",
			args: args{
				vsARN:  aws.String("arn:aws:appmesh:us-west-2:000000000000:mesh/mesh-name/virtualNode/vn-name"),
				vsName: aws.String(""),
				scope:  nil,
			},
			wantErr: errors.New("expects virtualService ARN, got virtualNode"),
		},
		{
			name: "valid mesh ARN",
			args: args{
				vsARN:  aws.String("arn:aws:appmesh:us-west-2:000000000000:mesh/mesh-name"),
				vsName: aws.String(""),
				scope:  nil,
			},
			wantErr: errors.New("invalid resource in appMesh ARN: mesh/mesh-name"),
		},
		{
			name: "invalid ARN",
			args: args{
				vsARN:  aws.String("xxxxx"),
				vsName: aws.String(""),
				scope:  nil,
			},
			wantErr: errors.New("invalid arn: arn: invalid prefix"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_VirtualServiceARN_To_SDK_VirtualServiceName(tt.args.vsARN, tt.args.vsName, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantVSName, *tt.args.vsName)
			}
		})
	}
}

func TestConvert_CRD_VirtualRouterARN_To_SDK_VirtualRouterName(t *testing.T) {
	type args struct {
		vrARN  *string
		vrName *string
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantVRName string
		wantErr    error
	}{
		{
			name: "valid virtualService ARN",
			args: args{
				vrARN:  aws.String("arn:aws:appmesh:us-west-2:000000000000:mesh/mesh-name/virtualRouter/vr-name"),
				vrName: aws.String(""),
				scope:  nil,
			},
			wantVRName: "vr-name",
		},
		{
			name: "valid virtualService ARN - shared mesh",
			args: args{
				vrARN:  aws.String("arn:aws:appmesh:us-west-2:000000000000:mesh/mesh-name@111111111111/virtualRouter/vr-name"),
				vrName: aws.String(""),
				scope:  nil,
			},
			wantVRName: "vr-name",
		},
		{
			name: "valid virtualNode ARN",
			args: args{
				vrARN:  aws.String("arn:aws:appmesh:us-west-2:000000000000:mesh/mesh-name/virtualNode/vn-name"),
				vrName: aws.String(""),
				scope:  nil,
			},
			wantErr: errors.New("expects virtualRouter ARN, got virtualNode"),
		},
		{
			name: "valid mesh ARN",
			args: args{
				vrARN:  aws.String("arn:aws:appmesh:us-west-2:000000000000:mesh/mesh-name"),
				vrName: aws.String(""),
				scope:  nil,
			},
			wantErr: errors.New("invalid resource in appMesh ARN: mesh/mesh-name"),
		},
		{
			name: "invalid ARN",
			args: args{
				vrARN:  aws.String("xxxxx"),
				vrName: aws.String(""),
				scope:  nil,
			},
			wantErr: errors.New("invalid arn: arn: invalid prefix"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_VirtualRouterARN_To_SDK_VirtualRouterName(tt.args.vrARN, tt.args.vrName, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantVRName, *tt.args.vrName)
			}
		})
	}
}
