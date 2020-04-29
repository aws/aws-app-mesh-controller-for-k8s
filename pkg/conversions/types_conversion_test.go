package conversions

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-sdk-go/aws"
	appmeshsdk "github.com/aws/aws-sdk-go/service/appmesh"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/conversion"
	"testing"
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
