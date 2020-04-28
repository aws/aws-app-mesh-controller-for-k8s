package conversions

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-sdk-go/aws"
	appmeshsdk "github.com/aws/aws-sdk-go/service/appmesh"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/conversion"
	"testing"
)

func TestConvert_CRD_TLSValidationContextACMTrust_To_SDK_TLSValidationContextACMTrust(t *testing.T) {
	type args struct {
		crdObj *appmesh.TLSValidationContextACMTrust
		sdkObj *appmeshsdk.TlsValidationContextAcmTrust
		scope  conversion.Scope
	}
	tests := []struct {
		name    string
		args    args
		want    *appmeshsdk.TlsValidationContextAcmTrust
		wantErr error
	}{
		{
			name: "single arn",
			args: args{
				crdObj: &appmesh.TLSValidationContextACMTrust{
					CertificateAuthorityARNs: []string{"arn-1"},
				},
				sdkObj: &appmeshsdk.TlsValidationContextAcmTrust{},
			},
			want: &appmeshsdk.TlsValidationContextAcmTrust{
				CertificateAuthorityArns: []*string{aws.String("arn-1")},
			},
		},
		{
			name: "multiple arn",
			args: args{
				crdObj: &appmesh.TLSValidationContextACMTrust{
					CertificateAuthorityARNs: []string{"arn-1", "arn-2"},
				},
				sdkObj: &appmeshsdk.TlsValidationContextAcmTrust{},
			},
			want: &appmeshsdk.TlsValidationContextAcmTrust{
				CertificateAuthorityArns: []*string{aws.String("arn-1"), aws.String("arn-2")},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_TLSValidationContextACMTrust_To_SDK_TLSValidationContextACMTrust(tt.args.crdObj, tt.args.sdkObj, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, tt.args.sdkObj)
			}
		})
	}
}
