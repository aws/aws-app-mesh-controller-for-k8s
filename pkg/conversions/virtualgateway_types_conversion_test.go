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
	"k8s.io/apimachinery/pkg/conversion"
)

func TestConvert_CRD_VirtualGatewayTLSValidationContextACMTrust_To_SDK_VirtualGatewayTLSValidationContextACMTrust(t *testing.T) {
	type args struct {
		crdObj *appmesh.VirtualGatewayTLSValidationContextACMTrust
		sdkObj *appmeshsdk.VirtualGatewayTlsValidationContextAcmTrust
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.VirtualGatewayTlsValidationContextAcmTrust
		wantErr    error
	}{
		{
			name: "single arn",
			args: args{
				crdObj: &appmesh.VirtualGatewayTLSValidationContextACMTrust{
					CertificateAuthorityARNs: []string{"arn-1"},
				},
				sdkObj: &appmeshsdk.VirtualGatewayTlsValidationContextAcmTrust{},
			},
			wantSDKObj: &appmeshsdk.VirtualGatewayTlsValidationContextAcmTrust{
				CertificateAuthorityArns: []*string{aws.String("arn-1")},
			},
		},
		{
			name: "multiple arn",
			args: args{
				crdObj: &appmesh.VirtualGatewayTLSValidationContextACMTrust{
					CertificateAuthorityARNs: []string{"arn-1", "arn-2"},
				},
				sdkObj: &appmeshsdk.VirtualGatewayTlsValidationContextAcmTrust{},
			},
			wantSDKObj: &appmeshsdk.VirtualGatewayTlsValidationContextAcmTrust{
				CertificateAuthorityArns: []*string{aws.String("arn-1"), aws.String("arn-2")},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_VirtualGatewayTLSValidationContextACMTrust_To_SDK_VirtualGatewayTLSValidationContextACMTrust(tt.args.crdObj, tt.args.sdkObj, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_VirtualGatewayTLSValidationContextFileTrust_To_SDK_VirtualGatewayTLSValidationContextFileTrust(t *testing.T) {
	type args struct {
		crdObj *appmesh.VirtualGatewayTLSValidationContextFileTrust
		sdkObj *appmeshsdk.VirtualGatewayTlsValidationContextFileTrust
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.VirtualGatewayTlsValidationContextFileTrust
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.VirtualGatewayTLSValidationContextFileTrust{
					CertificateChain: "dummy",
				},
				sdkObj: &appmeshsdk.VirtualGatewayTlsValidationContextFileTrust{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.VirtualGatewayTlsValidationContextFileTrust{
				CertificateChain: aws.String("dummy"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_VirtualGatewayTLSValidationContextFileTrust_To_SDK_VirtualGatewayTLSValidationContextFileTrust(tt.args.crdObj, tt.args.sdkObj, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_VirtualGatewayTLSValidationContextTrust_To_SDK_VirtualGatewayTLSValidationContextTrust(t *testing.T) {
	validationContext := "sds://certAuthority"
	type args struct {
		crdObj *appmesh.VirtualGatewayTLSValidationContextTrust
		sdkObj *appmeshsdk.VirtualGatewayTlsValidationContextTrust
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.VirtualGatewayTlsValidationContextTrust
		wantErr    error
	}{
		{
			name: "acm validation context",
			args: args{
				crdObj: &appmesh.VirtualGatewayTLSValidationContextTrust{
					ACM: &appmesh.VirtualGatewayTLSValidationContextACMTrust{
						CertificateAuthorityARNs: []string{"arn-1", "arn-2"},
					},
				},
				sdkObj: &appmeshsdk.VirtualGatewayTlsValidationContextTrust{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.VirtualGatewayTlsValidationContextTrust{
				Acm: &appmeshsdk.VirtualGatewayTlsValidationContextAcmTrust{
					CertificateAuthorityArns: []*string{aws.String("arn-1"), aws.String("arn-2")},
				},
			},
		},
		{
			name: "file validation context",
			args: args{
				crdObj: &appmesh.VirtualGatewayTLSValidationContextTrust{
					File: &appmesh.VirtualGatewayTLSValidationContextFileTrust{
						CertificateChain: "dummy",
					},
				},
				sdkObj: &appmeshsdk.VirtualGatewayTlsValidationContextTrust{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.VirtualGatewayTlsValidationContextTrust{
				File: &appmeshsdk.VirtualGatewayTlsValidationContextFileTrust{
					CertificateChain: aws.String("dummy"),
				},
			},
		},
		{
			name: "sds validation context",
			args: args{
				crdObj: &appmesh.VirtualGatewayTLSValidationContextTrust{
					SDS: &appmesh.VirtualGatewayTLSValidationContextSDSTrust{
						SecretName: &validationContext,
					},
				},
				sdkObj: &appmeshsdk.VirtualGatewayTlsValidationContextTrust{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.VirtualGatewayTlsValidationContextTrust{
				Sds: &appmeshsdk.VirtualGatewayTlsValidationContextSdsTrust{
					SecretName: aws.String(validationContext),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_VirtualGatewayTLSValidationContextTrust_To_SDK_VirtualGatewayTLSValidationContextTrust(tt.args.crdObj, tt.args.sdkObj, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_VirtualGatewayTLSValidationContext_To_SDK_VirtualGatewayTLSValidationContext(t *testing.T) {
	validationContext := "sds://certAuthority"
	serverSAN := "sds://server"
	type args struct {
		crdObj *appmesh.VirtualGatewayTLSValidationContext
		sdkObj *appmeshsdk.VirtualGatewayTlsValidationContext
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.VirtualGatewayTlsValidationContext
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.VirtualGatewayTLSValidationContext{
					Trust: appmesh.VirtualGatewayTLSValidationContextTrust{
						ACM: &appmesh.VirtualGatewayTLSValidationContextACMTrust{
							CertificateAuthorityARNs: []string{"arn-1", "arn-2"},
						},
					},
				},
				sdkObj: &appmeshsdk.VirtualGatewayTlsValidationContext{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.VirtualGatewayTlsValidationContext{
				Trust: &appmeshsdk.VirtualGatewayTlsValidationContextTrust{
					Acm: &appmeshsdk.VirtualGatewayTlsValidationContextAcmTrust{
						CertificateAuthorityArns: []*string{aws.String("arn-1"), aws.String("arn-2")},
					},
				},
			},
		},
		{
			name: "SDS Validation + no SAN",
			args: args{
				crdObj: &appmesh.VirtualGatewayTLSValidationContext{
					Trust: appmesh.VirtualGatewayTLSValidationContextTrust{
						SDS: &appmesh.VirtualGatewayTLSValidationContextSDSTrust{
							SecretName: &validationContext,
						},
					},
				},
				sdkObj: &appmeshsdk.VirtualGatewayTlsValidationContext{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.VirtualGatewayTlsValidationContext{
				Trust: &appmeshsdk.VirtualGatewayTlsValidationContextTrust{
					Sds: &appmeshsdk.VirtualGatewayTlsValidationContextSdsTrust{
						SecretName: &validationContext,
					},
				},
			},
		},
		{
			name: "SDS Validation + SAN",
			args: args{
				crdObj: &appmesh.VirtualGatewayTLSValidationContext{
					Trust: appmesh.VirtualGatewayTLSValidationContextTrust{
						SDS: &appmesh.VirtualGatewayTLSValidationContextSDSTrust{
							SecretName: &validationContext,
						},
					},
					SubjectAlternativeNames: &appmesh.SubjectAlternativeNames{
						Match: &appmesh.SubjectAlternativeNameMatchers{
							Exact: []*string{
								&serverSAN,
							},
						},
					},
				},
				sdkObj: &appmeshsdk.VirtualGatewayTlsValidationContext{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.VirtualGatewayTlsValidationContext{
				Trust: &appmeshsdk.VirtualGatewayTlsValidationContextTrust{
					Sds: &appmeshsdk.VirtualGatewayTlsValidationContextSdsTrust{
						SecretName: &validationContext,
					},
				},
				SubjectAlternativeNames: &appmeshsdk.SubjectAlternativeNames{
					Match: &appmeshsdk.SubjectAlternativeNameMatchers{
						Exact: []*string{
							&serverSAN,
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_VirtualGatewayTLSValidationContext_To_SDK_VirtualGatewayTLSValidationContext(tt.args.crdObj, tt.args.sdkObj, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_VirtualGatewayClientPolicyTLS_To_SDK_VirtualGatewayClientPolicyTLS(t *testing.T) {
	validationContext := "sds://certAuthority"
	appCert := "sds://appCert"
	type args struct {
		crdObj *appmesh.VirtualGatewayClientPolicyTLS
		sdkObj *appmeshsdk.VirtualGatewayClientPolicyTls
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.VirtualGatewayClientPolicyTls
		wantErr    error
	}{
		{
			name: "normal tls case",
			args: args{
				crdObj: &appmesh.VirtualGatewayClientPolicyTLS{
					Enforce: aws.Bool(true),
					Ports:   []appmesh.PortNumber{80, 443},
					Validation: appmesh.VirtualGatewayTLSValidationContext{
						Trust: appmesh.VirtualGatewayTLSValidationContextTrust{
							ACM: &appmesh.VirtualGatewayTLSValidationContextACMTrust{
								CertificateAuthorityARNs: []string{"arn-1", "arn-2"},
							},
						},
					},
				},
				sdkObj: &appmeshsdk.VirtualGatewayClientPolicyTls{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.VirtualGatewayClientPolicyTls{
				Enforce: aws.Bool(true),
				Ports:   []*int64{aws.Int64(80), aws.Int64(443)},
				Validation: &appmeshsdk.VirtualGatewayTlsValidationContext{
					Trust: &appmeshsdk.VirtualGatewayTlsValidationContextTrust{
						Acm: &appmeshsdk.VirtualGatewayTlsValidationContextAcmTrust{
							CertificateAuthorityArns: []*string{aws.String("arn-1"), aws.String("arn-2")},
						},
					},
				},
			},
		},
		{
			name: "file based mtls",
			args: args{
				crdObj: &appmesh.VirtualGatewayClientPolicyTLS{
					Enforce: aws.Bool(true),
					Ports:   []appmesh.PortNumber{80, 443},
					Validation: appmesh.VirtualGatewayTLSValidationContext{
						Trust: appmesh.VirtualGatewayTLSValidationContextTrust{
							File: &appmesh.VirtualGatewayTLSValidationContextFileTrust{
								CertificateChain: "certficateAuthority",
							},
						},
					},
					Certificate: &appmesh.VirtualGatewayClientTLSCertificate{
						File: &appmesh.VirtualGatewayListenerTLSFileCertificate{
							CertificateChain: "certChain",
							PrivateKey:       "secret",
						},
					},
				},
				sdkObj: &appmeshsdk.VirtualGatewayClientPolicyTls{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.VirtualGatewayClientPolicyTls{
				Enforce: aws.Bool(true),
				Ports:   []*int64{aws.Int64(80), aws.Int64(443)},
				Validation: &appmeshsdk.VirtualGatewayTlsValidationContext{
					Trust: &appmeshsdk.VirtualGatewayTlsValidationContextTrust{
						File: &appmeshsdk.VirtualGatewayTlsValidationContextFileTrust{
							CertificateChain: aws.String("certficateAuthority"),
						},
					},
				},
				Certificate: &appmeshsdk.VirtualGatewayClientTlsCertificate{
					File: &appmeshsdk.VirtualGatewayListenerTlsFileCertificate{
						CertificateChain: aws.String("certChain"),
						PrivateKey:       aws.String("secret"),
					},
				},
			},
		},
		{
			name: "sds based mtls",
			args: args{
				crdObj: &appmesh.VirtualGatewayClientPolicyTLS{
					Enforce: aws.Bool(true),
					Ports:   []appmesh.PortNumber{80, 443},
					Validation: appmesh.VirtualGatewayTLSValidationContext{
						Trust: appmesh.VirtualGatewayTLSValidationContextTrust{
							SDS: &appmesh.VirtualGatewayTLSValidationContextSDSTrust{
								SecretName: &validationContext,
							},
						},
					},
					Certificate: &appmesh.VirtualGatewayClientTLSCertificate{
						SDS: &appmesh.VirtualGatewayListenerTLSSDSCertificate{
							SecretName: &appCert,
						},
					},
				},
				sdkObj: &appmeshsdk.VirtualGatewayClientPolicyTls{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.VirtualGatewayClientPolicyTls{
				Enforce: aws.Bool(true),
				Ports:   []*int64{aws.Int64(80), aws.Int64(443)},
				Validation: &appmeshsdk.VirtualGatewayTlsValidationContext{
					Trust: &appmeshsdk.VirtualGatewayTlsValidationContextTrust{
						Sds: &appmeshsdk.VirtualGatewayTlsValidationContextSdsTrust{
							SecretName: aws.String(validationContext),
						},
					},
				},
				Certificate: &appmeshsdk.VirtualGatewayClientTlsCertificate{
					Sds: &appmeshsdk.VirtualGatewayListenerTlsSdsCertificate{
						SecretName: aws.String(appCert),
					},
				},
			},
		},
		{
			name: "normal case + enforce false",
			args: args{
				crdObj: &appmesh.VirtualGatewayClientPolicyTLS{
					Enforce: aws.Bool(false),
					Ports:   []appmesh.PortNumber{80, 443},
					Validation: appmesh.VirtualGatewayTLSValidationContext{
						Trust: appmesh.VirtualGatewayTLSValidationContextTrust{
							ACM: &appmesh.VirtualGatewayTLSValidationContextACMTrust{
								CertificateAuthorityARNs: []string{"arn-1", "arn-2"},
							},
						},
					},
				},
				sdkObj: &appmeshsdk.VirtualGatewayClientPolicyTls{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.VirtualGatewayClientPolicyTls{
				Enforce: aws.Bool(false),
				Ports:   []*int64{aws.Int64(80), aws.Int64(443)},
				Validation: &appmeshsdk.VirtualGatewayTlsValidationContext{
					Trust: &appmeshsdk.VirtualGatewayTlsValidationContextTrust{
						Acm: &appmeshsdk.VirtualGatewayTlsValidationContextAcmTrust{
							CertificateAuthorityArns: []*string{aws.String("arn-1"), aws.String("arn-2")},
						},
					},
				},
			},
		},
		{
			name: "normal case + enforce unset",
			args: args{
				crdObj: &appmesh.VirtualGatewayClientPolicyTLS{
					Enforce: nil,
					Ports:   []appmesh.PortNumber{80, 443},
					Validation: appmesh.VirtualGatewayTLSValidationContext{
						Trust: appmesh.VirtualGatewayTLSValidationContextTrust{
							ACM: &appmesh.VirtualGatewayTLSValidationContextACMTrust{
								CertificateAuthorityARNs: []string{"arn-1", "arn-2"},
							},
						},
					},
				},
				sdkObj: &appmeshsdk.VirtualGatewayClientPolicyTls{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.VirtualGatewayClientPolicyTls{
				Enforce: nil,
				Ports:   []*int64{aws.Int64(80), aws.Int64(443)},
				Validation: &appmeshsdk.VirtualGatewayTlsValidationContext{
					Trust: &appmeshsdk.VirtualGatewayTlsValidationContextTrust{
						Acm: &appmeshsdk.VirtualGatewayTlsValidationContextAcmTrust{
							CertificateAuthorityArns: []*string{aws.String("arn-1"), aws.String("arn-2")},
						},
					},
				},
			},
		},
		{
			name: "normal case + nil ports",
			args: args{
				crdObj: &appmesh.VirtualGatewayClientPolicyTLS{
					Enforce: aws.Bool(true),
					Ports:   nil,
					Validation: appmesh.VirtualGatewayTLSValidationContext{
						Trust: appmesh.VirtualGatewayTLSValidationContextTrust{
							ACM: &appmesh.VirtualGatewayTLSValidationContextACMTrust{
								CertificateAuthorityARNs: []string{"arn-1", "arn-2"},
							},
						},
					},
				},
				sdkObj: &appmeshsdk.VirtualGatewayClientPolicyTls{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.VirtualGatewayClientPolicyTls{
				Enforce: aws.Bool(true),
				Ports:   nil,
				Validation: &appmeshsdk.VirtualGatewayTlsValidationContext{
					Trust: &appmeshsdk.VirtualGatewayTlsValidationContextTrust{
						Acm: &appmeshsdk.VirtualGatewayTlsValidationContextAcmTrust{
							CertificateAuthorityArns: []*string{aws.String("arn-1"), aws.String("arn-2")},
						},
					},
				},
			},
		},
		{
			name: "normal case + empty ports",
			args: args{
				crdObj: &appmesh.VirtualGatewayClientPolicyTLS{
					Enforce: aws.Bool(true),
					Ports:   []appmesh.PortNumber{},
					Validation: appmesh.VirtualGatewayTLSValidationContext{
						Trust: appmesh.VirtualGatewayTLSValidationContextTrust{
							ACM: &appmesh.VirtualGatewayTLSValidationContextACMTrust{
								CertificateAuthorityARNs: []string{"arn-1", "arn-2"},
							},
						},
					},
				},
				sdkObj: &appmeshsdk.VirtualGatewayClientPolicyTls{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.VirtualGatewayClientPolicyTls{
				Enforce: aws.Bool(true),
				Ports:   nil,
				Validation: &appmeshsdk.VirtualGatewayTlsValidationContext{
					Trust: &appmeshsdk.VirtualGatewayTlsValidationContextTrust{
						Acm: &appmeshsdk.VirtualGatewayTlsValidationContextAcmTrust{
							CertificateAuthorityArns: []*string{aws.String("arn-1"), aws.String("arn-2")},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_VirtualGatewayClientPolicyTLS_To_SDK_VirtualGatewayClientPolicyTLS(tt.args.crdObj, tt.args.sdkObj, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_VirtualGatewayClientPolicy_To_SDK_VirtualGatewayClientPolicy(t *testing.T) {
	validationContext := "sds://certAuthority"
	appCert := "appCert"
	type args struct {
		crdObj *appmesh.VirtualGatewayClientPolicy
		sdkObj *appmeshsdk.VirtualGatewayClientPolicy
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.VirtualGatewayClientPolicy
		wantErr    error
	}{
		{
			name: "non nil TLS + File based cert",
			args: args{
				crdObj: &appmesh.VirtualGatewayClientPolicy{
					TLS: &appmesh.VirtualGatewayClientPolicyTLS{
						Enforce: aws.Bool(true),
						Ports:   []appmesh.PortNumber{80, 443},
						Validation: appmesh.VirtualGatewayTLSValidationContext{
							Trust: appmesh.VirtualGatewayTLSValidationContextTrust{
								ACM: &appmesh.VirtualGatewayTLSValidationContextACMTrust{
									CertificateAuthorityARNs: []string{"arn-1", "arn-2"},
								},
							},
						},
						Certificate: &appmesh.VirtualGatewayClientTLSCertificate{
							File: &appmesh.VirtualGatewayListenerTLSFileCertificate{
								CertificateChain: "certChain",
								PrivateKey:       "secret",
							},
						},
					},
				},
				sdkObj: &appmeshsdk.VirtualGatewayClientPolicy{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.VirtualGatewayClientPolicy{
				Tls: &appmeshsdk.VirtualGatewayClientPolicyTls{
					Enforce: aws.Bool(true),
					Ports:   []*int64{aws.Int64(80), aws.Int64(443)},
					Validation: &appmeshsdk.VirtualGatewayTlsValidationContext{
						Trust: &appmeshsdk.VirtualGatewayTlsValidationContextTrust{
							Acm: &appmeshsdk.VirtualGatewayTlsValidationContextAcmTrust{
								CertificateAuthorityArns: []*string{aws.String("arn-1"), aws.String("arn-2")},
							},
						},
					},
					Certificate: &appmeshsdk.VirtualGatewayClientTlsCertificate{
						File: &appmeshsdk.VirtualGatewayListenerTlsFileCertificate{
							CertificateChain: aws.String("certChain"),
							PrivateKey:       aws.String("secret"),
						},
					},
				},
			},
		},
		{
			name: "non nil TLS + sds based cert",
			args: args{
				crdObj: &appmesh.VirtualGatewayClientPolicy{
					TLS: &appmesh.VirtualGatewayClientPolicyTLS{
						Enforce: aws.Bool(true),
						Ports:   []appmesh.PortNumber{80, 443},
						Validation: appmesh.VirtualGatewayTLSValidationContext{
							Trust: appmesh.VirtualGatewayTLSValidationContextTrust{
								SDS: &appmesh.VirtualGatewayTLSValidationContextSDSTrust{
									SecretName: &validationContext,
								},
							},
						},
						Certificate: &appmesh.VirtualGatewayClientTLSCertificate{
							SDS: &appmesh.VirtualGatewayListenerTLSSDSCertificate{
								SecretName: &appCert,
							},
						},
					},
				},
				sdkObj: &appmeshsdk.VirtualGatewayClientPolicy{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.VirtualGatewayClientPolicy{
				Tls: &appmeshsdk.VirtualGatewayClientPolicyTls{
					Enforce: aws.Bool(true),
					Ports:   []*int64{aws.Int64(80), aws.Int64(443)},
					Validation: &appmeshsdk.VirtualGatewayTlsValidationContext{
						Trust: &appmeshsdk.VirtualGatewayTlsValidationContextTrust{
							Sds: &appmeshsdk.VirtualGatewayTlsValidationContextSdsTrust{
								SecretName: &validationContext,
							},
						},
					},
					Certificate: &appmeshsdk.VirtualGatewayClientTlsCertificate{
						Sds: &appmeshsdk.VirtualGatewayListenerTlsSdsCertificate{
							SecretName: aws.String(appCert),
						},
					},
				},
			},
		},
		{
			name: "nil TLS",
			args: args{
				crdObj: &appmesh.VirtualGatewayClientPolicy{
					TLS: nil,
				},
				sdkObj: &appmeshsdk.VirtualGatewayClientPolicy{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.VirtualGatewayClientPolicy{
				Tls: nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_VirtualGatewayClientPolicy_To_SDK_VirtualGatewayClientPolicy(tt.args.crdObj, tt.args.sdkObj, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_VirtualGatewayBackendDefaults_To_SDK_VirtualGatewayBackendDefaults(t *testing.T) {
	type args struct {
		crdObj *appmesh.VirtualGatewayBackendDefaults
		sdkObj *appmeshsdk.VirtualGatewayBackendDefaults
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.VirtualGatewayBackendDefaults
		wantErr    error
	}{
		{
			name: "non-nil clientPolicy",
			args: args{
				crdObj: &appmesh.VirtualGatewayBackendDefaults{
					ClientPolicy: &appmesh.VirtualGatewayClientPolicy{
						TLS: &appmesh.VirtualGatewayClientPolicyTLS{
							Enforce: aws.Bool(true),
							Ports:   []appmesh.PortNumber{80, 443},
							Validation: appmesh.VirtualGatewayTLSValidationContext{
								Trust: appmesh.VirtualGatewayTLSValidationContextTrust{
									ACM: &appmesh.VirtualGatewayTLSValidationContextACMTrust{
										CertificateAuthorityARNs: []string{"arn-1", "arn-2"},
									},
								},
							},
						},
					},
				},
				sdkObj: &appmeshsdk.VirtualGatewayBackendDefaults{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.VirtualGatewayBackendDefaults{
				ClientPolicy: &appmeshsdk.VirtualGatewayClientPolicy{
					Tls: &appmeshsdk.VirtualGatewayClientPolicyTls{
						Enforce: aws.Bool(true),
						Ports:   []*int64{aws.Int64(80), aws.Int64(443)},
						Validation: &appmeshsdk.VirtualGatewayTlsValidationContext{
							Trust: &appmeshsdk.VirtualGatewayTlsValidationContextTrust{
								Acm: &appmeshsdk.VirtualGatewayTlsValidationContextAcmTrust{
									CertificateAuthorityArns: []*string{aws.String("arn-1"), aws.String("arn-2")},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "nil clientPolicy",
			args: args{
				crdObj: &appmesh.VirtualGatewayBackendDefaults{
					ClientPolicy: nil,
				},
				sdkObj: &appmeshsdk.VirtualGatewayBackendDefaults{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.VirtualGatewayBackendDefaults{
				ClientPolicy: nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_VirtualGatewayBackendDefaults_To_SDK_VirtualGatewayBackendDefaults(tt.args.crdObj, tt.args.sdkObj, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_VirtualGatewayHealthCheckPolicy_To_SDK_VirtualGatewayHealthCheckPolicy(t *testing.T) {
	port80 := appmesh.PortNumber(80)
	protocolHTTP := appmesh.VirtualGatewayPortProtocolHTTP
	type args struct {
		crdObj *appmesh.VirtualGatewayHealthCheckPolicy
		sdkObj *appmeshsdk.VirtualGatewayHealthCheckPolicy
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.VirtualGatewayHealthCheckPolicy
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.VirtualGatewayHealthCheckPolicy{
					HealthyThreshold:   3,
					IntervalMillis:     60,
					Path:               aws.String("/"),
					Port:               &port80,
					Protocol:           protocolHTTP,
					TimeoutMillis:      30,
					UnhealthyThreshold: 2,
				},
				sdkObj: &appmeshsdk.VirtualGatewayHealthCheckPolicy{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.VirtualGatewayHealthCheckPolicy{
				HealthyThreshold:   aws.Int64(3),
				IntervalMillis:     aws.Int64(60),
				Path:               aws.String("/"),
				Port:               aws.Int64(80),
				Protocol:           aws.String("http"),
				TimeoutMillis:      aws.Int64(30),
				UnhealthyThreshold: aws.Int64(2),
			},
		},
		{
			name: "normal case + nil port",
			args: args{
				crdObj: &appmesh.VirtualGatewayHealthCheckPolicy{
					HealthyThreshold:   3,
					IntervalMillis:     60,
					Path:               aws.String("/"),
					Port:               nil,
					Protocol:           protocolHTTP,
					TimeoutMillis:      30,
					UnhealthyThreshold: 2,
				},
				sdkObj: &appmeshsdk.VirtualGatewayHealthCheckPolicy{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.VirtualGatewayHealthCheckPolicy{
				HealthyThreshold:   aws.Int64(3),
				IntervalMillis:     aws.Int64(60),
				Path:               aws.String("/"),
				Port:               nil,
				Protocol:           aws.String("http"),
				TimeoutMillis:      aws.Int64(30),
				UnhealthyThreshold: aws.Int64(2),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_VirtualGatewayHealthCheckPolicy_To_SDK_VirtualGatewayHealthCheckPolicy(tt.args.crdObj, tt.args.sdkObj, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_VirtualGatewayListenerTLSACMCertificate_To_SDK_VirtualGatewayListenerTLSACMCertificate(t *testing.T) {
	type args struct {
		crdObj *appmesh.VirtualGatewayListenerTLSACMCertificate
		sdkObj *appmeshsdk.VirtualGatewayListenerTlsAcmCertificate
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.VirtualGatewayListenerTlsAcmCertificate
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.VirtualGatewayListenerTLSACMCertificate{
					CertificateARN: "arn-1",
				},
				sdkObj: &appmeshsdk.VirtualGatewayListenerTlsAcmCertificate{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.VirtualGatewayListenerTlsAcmCertificate{
				CertificateArn: aws.String("arn-1"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_VirtualGatewayListenerTLSACMCertificate_To_SDK_VirtualGatewayListenerTLSACMCertificate(tt.args.crdObj, tt.args.sdkObj, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_VirtualGatewayListenerTLSFileCertificate_To_SDK_VirtualGatewayListenerTLSFileCertificate(t *testing.T) {
	type args struct {
		crdObj *appmesh.VirtualGatewayListenerTLSFileCertificate
		sdkObj *appmeshsdk.VirtualGatewayListenerTlsFileCertificate
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.VirtualGatewayListenerTlsFileCertificate
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.VirtualGatewayListenerTLSFileCertificate{
					CertificateChain: "certificateChain",
					PrivateKey:       "privateKey",
				},
				sdkObj: &appmeshsdk.VirtualGatewayListenerTlsFileCertificate{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.VirtualGatewayListenerTlsFileCertificate{
				CertificateChain: aws.String("certificateChain"),
				PrivateKey:       aws.String("privateKey"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_VirtualGatewayListenerTLSFileCertificate_To_SDK_VirtualGatewayListenerTLSFileCertificate(tt.args.crdObj, tt.args.sdkObj, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_VirtualGatewayListenerTLSCertificate_To_SDK_VirtualGatewayListenerTLSCertificate(t *testing.T) {
	type args struct {
		crdObj *appmesh.VirtualGatewayListenerTLSCertificate
		sdkObj *appmeshsdk.VirtualGatewayListenerTlsCertificate
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.VirtualGatewayListenerTlsCertificate
		wantErr    error
	}{
		{
			name: "acm certificate",
			args: args{
				crdObj: &appmesh.VirtualGatewayListenerTLSCertificate{
					ACM: &appmesh.VirtualGatewayListenerTLSACMCertificate{
						CertificateARN: "arn-1",
					},
				},
				sdkObj: &appmeshsdk.VirtualGatewayListenerTlsCertificate{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.VirtualGatewayListenerTlsCertificate{
				Acm: &appmeshsdk.VirtualGatewayListenerTlsAcmCertificate{
					CertificateArn: aws.String("arn-1"),
				},
			},
		},
		{
			name: "file certificate",
			args: args{
				crdObj: &appmesh.VirtualGatewayListenerTLSCertificate{
					File: &appmesh.VirtualGatewayListenerTLSFileCertificate{
						CertificateChain: "certificateChain",
						PrivateKey:       "privateKey",
					},
				},
				sdkObj: &appmeshsdk.VirtualGatewayListenerTlsCertificate{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.VirtualGatewayListenerTlsCertificate{
				File: &appmeshsdk.VirtualGatewayListenerTlsFileCertificate{
					CertificateChain: aws.String("certificateChain"),
					PrivateKey:       aws.String("privateKey"),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_VirtualGatewayListenerTLSCertificate_To_SDK_VirtualGatewayListenerTLSCertificate(tt.args.crdObj, tt.args.sdkObj, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_VirtualGatewayListenerTLSValidationContext_To_SDK_VirtualGatewayListenerTLSValidationContext(t *testing.T) {
	validationContext := "sds://certAuthority"
	type args struct {
		crdObj *appmesh.VirtualGatewayListenerTLSValidationContext
		sdkObj *appmeshsdk.VirtualGatewayListenerTlsValidationContext
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.VirtualGatewayListenerTlsValidationContext
		wantErr    error
	}{
		{
			name: "file based validation",
			args: args{
				crdObj: &appmesh.VirtualGatewayListenerTLSValidationContext{
					Trust: appmesh.VirtualGatewayListenerTLSValidationContextTrust{
						File: &appmesh.VirtualGatewayTLSValidationContextFileTrust{
							CertificateChain: "CACert",
						},
					},
				},
				sdkObj: &appmeshsdk.VirtualGatewayListenerTlsValidationContext{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.VirtualGatewayListenerTlsValidationContext{
				Trust: &appmeshsdk.VirtualGatewayListenerTlsValidationContextTrust{
					File: &appmeshsdk.VirtualGatewayTlsValidationContextFileTrust{
						CertificateChain: aws.String("CACert"),
					},
				},
			},
		},
		{
			name: "sds based validation",
			args: args{
				crdObj: &appmesh.VirtualGatewayListenerTLSValidationContext{
					Trust: appmesh.VirtualGatewayListenerTLSValidationContextTrust{
						SDS: &appmesh.VirtualGatewayTLSValidationContextSDSTrust{
							SecretName: &validationContext,
						},
					},
				},
				sdkObj: &appmeshsdk.VirtualGatewayListenerTlsValidationContext{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.VirtualGatewayListenerTlsValidationContext{
				Trust: &appmeshsdk.VirtualGatewayListenerTlsValidationContextTrust{
					Sds: &appmeshsdk.VirtualGatewayTlsValidationContextSdsTrust{
						SecretName: &validationContext,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_VirtualGatewayListenerTLSValidationContext_To_SDK_VirtualGatewayListenerTLSValidationContext(tt.args.crdObj, tt.args.sdkObj, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_VirtualGatewayListenerTLS_To_SDK_VirtualGatewayListenerTLS(t *testing.T) {
	validationContext := "sds://certAuthority"
	appCert := "sds://appCert"
	type args struct {
		crdObj *appmesh.VirtualGatewayListenerTLS
		sdkObj *appmeshsdk.VirtualGatewayListenerTls
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.VirtualGatewayListenerTls
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.VirtualGatewayListenerTLS{
					Certificate: appmesh.VirtualGatewayListenerTLSCertificate{
						File: &appmesh.VirtualGatewayListenerTLSFileCertificate{
							CertificateChain: "certificateChain",
							PrivateKey:       "privateKey",
						},
					},
					Mode: appmesh.VirtualGatewayListenerTLSModeStrict,
				},
				sdkObj: &appmeshsdk.VirtualGatewayListenerTls{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.VirtualGatewayListenerTls{
				Certificate: &appmeshsdk.VirtualGatewayListenerTlsCertificate{
					File: &appmeshsdk.VirtualGatewayListenerTlsFileCertificate{
						CertificateChain: aws.String("certificateChain"),
						PrivateKey:       aws.String("privateKey"),
					},
				},
				Mode: aws.String("STRICT"),
			},
		},
		{
			name: "file cert + validation",
			args: args{
				crdObj: &appmesh.VirtualGatewayListenerTLS{
					Certificate: appmesh.VirtualGatewayListenerTLSCertificate{
						File: &appmesh.VirtualGatewayListenerTLSFileCertificate{
							CertificateChain: "certificateChain",
							PrivateKey:       "privateKey",
						},
					},
					Mode: appmesh.VirtualGatewayListenerTLSModeStrict,
					Validation: &appmesh.VirtualGatewayListenerTLSValidationContext{
						Trust: appmesh.VirtualGatewayListenerTLSValidationContextTrust{
							File: &appmesh.VirtualGatewayTLSValidationContextFileTrust{
								CertificateChain: "CACert",
							},
						},
					},
				},
				sdkObj: &appmeshsdk.VirtualGatewayListenerTls{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.VirtualGatewayListenerTls{
				Certificate: &appmeshsdk.VirtualGatewayListenerTlsCertificate{
					File: &appmeshsdk.VirtualGatewayListenerTlsFileCertificate{
						CertificateChain: aws.String("certificateChain"),
						PrivateKey:       aws.String("privateKey"),
					},
				},
				Mode: aws.String("STRICT"),
				Validation: &appmeshsdk.VirtualGatewayListenerTlsValidationContext{
					Trust: &appmeshsdk.VirtualGatewayListenerTlsValidationContextTrust{
						File: &appmeshsdk.VirtualGatewayTlsValidationContextFileTrust{
							CertificateChain: aws.String("CACert"),
						},
					},
				},
			},
		},
		{
			name: "sds cert + validation",
			args: args{
				crdObj: &appmesh.VirtualGatewayListenerTLS{
					Certificate: appmesh.VirtualGatewayListenerTLSCertificate{
						SDS: &appmesh.VirtualGatewayListenerTLSSDSCertificate{
							SecretName: &appCert,
						},
					},
					Mode: appmesh.VirtualGatewayListenerTLSModeStrict,
					Validation: &appmesh.VirtualGatewayListenerTLSValidationContext{
						Trust: appmesh.VirtualGatewayListenerTLSValidationContextTrust{
							SDS: &appmesh.VirtualGatewayTLSValidationContextSDSTrust{
								SecretName: &validationContext,
							},
						},
					},
				},
				sdkObj: &appmeshsdk.VirtualGatewayListenerTls{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.VirtualGatewayListenerTls{
				Certificate: &appmeshsdk.VirtualGatewayListenerTlsCertificate{
					Sds: &appmeshsdk.VirtualGatewayListenerTlsSdsCertificate{
						SecretName: &appCert,
					},
				},
				Mode: aws.String("STRICT"),
				Validation: &appmeshsdk.VirtualGatewayListenerTlsValidationContext{
					Trust: &appmeshsdk.VirtualGatewayListenerTlsValidationContextTrust{
						Sds: &appmeshsdk.VirtualGatewayTlsValidationContextSdsTrust{
							SecretName: &validationContext,
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_VirtualGatewayListenerTLS_To_SDK_VirtualGatewayListenerTLS(tt.args.crdObj, tt.args.sdkObj, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_VirtualGatewayFileAccessLog_To_SDK_VirtualGatewayFileAccessLog(t *testing.T) {
	type args struct {
		crdObj *appmesh.VirtualGatewayFileAccessLog
		sdkObj *appmeshsdk.VirtualGatewayFileAccessLog
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.VirtualGatewayFileAccessLog
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.VirtualGatewayFileAccessLog{
					Path: "/",
				},
				sdkObj: &appmeshsdk.VirtualGatewayFileAccessLog{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.VirtualGatewayFileAccessLog{
				Path: aws.String("/"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_VirtualGatewayFileAccessLog_To_SDK_VirtualGatewayFileAccessLog(tt.args.crdObj, tt.args.sdkObj, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_VirtualGatewayAccessLog_To_SDK_VirtualGatewayAccessLog(t *testing.T) {
	type args struct {
		crdObj *appmesh.VirtualGatewayAccessLog
		sdkObj *appmeshsdk.VirtualGatewayAccessLog
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.VirtualGatewayAccessLog
		wantErr    error
	}{
		{
			name: "non-nil file access log",
			args: args{
				crdObj: &appmesh.VirtualGatewayAccessLog{
					File: &appmesh.VirtualGatewayFileAccessLog{
						Path: "/",
					},
				},
				sdkObj: &appmeshsdk.VirtualGatewayAccessLog{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.VirtualGatewayAccessLog{
				File: &appmeshsdk.VirtualGatewayFileAccessLog{
					Path: aws.String("/"),
				},
			},
		},
		{
			name: "nil file access log",
			args: args{
				crdObj: &appmesh.VirtualGatewayAccessLog{
					File: nil,
				},
				sdkObj: &appmeshsdk.VirtualGatewayAccessLog{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.VirtualGatewayAccessLog{
				File: nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_VirtualGatewayAccessLog_To_SDK_VirtualGatewayAccessLog(tt.args.crdObj, tt.args.sdkObj, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_VirtualGatewayLogging_To_SDK_VirtualGatewayLogging(t *testing.T) {
	type args struct {
		crdObj *appmesh.VirtualGatewayLogging
		sdkObj *appmeshsdk.VirtualGatewayLogging
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.VirtualGatewayLogging
		wantErr    error
	}{
		{
			name: "non-nil AccessLog",
			args: args{
				crdObj: &appmesh.VirtualGatewayLogging{
					AccessLog: &appmesh.VirtualGatewayAccessLog{
						File: &appmesh.VirtualGatewayFileAccessLog{
							Path: "/",
						},
					},
				},
				sdkObj: &appmeshsdk.VirtualGatewayLogging{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.VirtualGatewayLogging{
				AccessLog: &appmeshsdk.VirtualGatewayAccessLog{
					File: &appmeshsdk.VirtualGatewayFileAccessLog{
						Path: aws.String("/"),
					},
				},
			},
		},
		{
			name: "nil AccessLog",
			args: args{
				crdObj: &appmesh.VirtualGatewayLogging{
					AccessLog: nil,
				},
				sdkObj: &appmeshsdk.VirtualGatewayLogging{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.VirtualGatewayLogging{
				AccessLog: nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_VirtualGatewayLogging_To_SDK_VirtualGatewayLogging(tt.args.crdObj, tt.args.sdkObj, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_VirtualGatewayHTTPConnectionPool_To_SDK_VirtualGatewayHttpConnectionPool(t *testing.T) {
	type args struct {
		crdObj *appmesh.HTTPConnectionPool
		sdkObj *appmeshsdk.VirtualGatewayHttpConnectionPool
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.VirtualGatewayHttpConnectionPool
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.HTTPConnectionPool{
					MaxConnections:     50,
					MaxPendingRequests: aws.Int64(20),
				},
				sdkObj: &appmeshsdk.VirtualGatewayHttpConnectionPool{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.VirtualGatewayHttpConnectionPool{
				MaxConnections:     aws.Int64(50),
				MaxPendingRequests: aws.Int64(20),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_VirtualGatewayHTTPConnectionPool_To_SDK_VirtualGatewayHttpConnectionPool(tt.args.crdObj, tt.args.sdkObj, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_VirtualGatewayHTTP2ConnectionPool_To_SDK_VirtualGatewayHttp2ConnectionPool(t *testing.T) {
	type args struct {
		crdObj *appmesh.HTTP2ConnectionPool
		sdkObj *appmeshsdk.VirtualGatewayHttp2ConnectionPool
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.VirtualGatewayHttp2ConnectionPool
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.HTTP2ConnectionPool{
					MaxRequests: 200,
				},
				sdkObj: &appmeshsdk.VirtualGatewayHttp2ConnectionPool{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.VirtualGatewayHttp2ConnectionPool{
				MaxRequests: aws.Int64(200),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_VirtualGatewayHTTP2ConnectionPool_To_SDK_VirtualGatewayHttp2ConnectionPool(tt.args.crdObj, tt.args.sdkObj, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_VirtualGatewayGRPCConnectionPool_To_SDK_VirtualGatewayGrpcConnectionPool(t *testing.T) {
	type args struct {
		crdObj *appmesh.GRPCConnectionPool
		sdkObj *appmeshsdk.VirtualGatewayGrpcConnectionPool
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.VirtualGatewayGrpcConnectionPool
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.GRPCConnectionPool{
					MaxRequests: 200,
				},
				sdkObj: &appmeshsdk.VirtualGatewayGrpcConnectionPool{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.VirtualGatewayGrpcConnectionPool{
				MaxRequests: aws.Int64(200),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_VirtualGatewayGRPCConnectionPool_To_SDK_VirtualGatewayGrpcConnectionPool(tt.args.crdObj, tt.args.sdkObj, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_VirtualGatewayConnectionPool_To_SDK_VirtualGatewayConnectionPool(t *testing.T) {
	type args struct {
		crdObj *appmesh.VirtualGatewayConnectionPool
		sdkObj *appmeshsdk.VirtualGatewayConnectionPool
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.VirtualGatewayConnectionPool
		wantErr    error
	}{
		{
			name: "http connection pool case",
			args: args{
				crdObj: &appmesh.VirtualGatewayConnectionPool{
					HTTP: &appmesh.HTTPConnectionPool{
						MaxConnections:     50,
						MaxPendingRequests: aws.Int64(40),
					},
				},
				sdkObj: &appmeshsdk.VirtualGatewayConnectionPool{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.VirtualGatewayConnectionPool{
				Http: &appmeshsdk.VirtualGatewayHttpConnectionPool{
					MaxConnections:     aws.Int64(50),
					MaxPendingRequests: aws.Int64(40),
				},
			},
		},
		{
			name: "http2 connection pool case",
			args: args{
				crdObj: &appmesh.VirtualGatewayConnectionPool{
					HTTP2: &appmesh.HTTP2ConnectionPool{
						MaxRequests: 50,
					},
				},
				sdkObj: &appmeshsdk.VirtualGatewayConnectionPool{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.VirtualGatewayConnectionPool{
				Http2: &appmeshsdk.VirtualGatewayHttp2ConnectionPool{
					MaxRequests: aws.Int64(50),
				},
			},
		},
		{
			name: "grpc connection pool case",
			args: args{
				crdObj: &appmesh.VirtualGatewayConnectionPool{
					GRPC: &appmesh.GRPCConnectionPool{
						MaxRequests: 50,
					},
				},
				sdkObj: &appmeshsdk.VirtualGatewayConnectionPool{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.VirtualGatewayConnectionPool{
				Grpc: &appmeshsdk.VirtualGatewayGrpcConnectionPool{
					MaxRequests: aws.Int64(50),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_VirtualGatewayConnectionPool_To_SDK_VirtualGatewayConnectionPool(tt.args.crdObj, tt.args.sdkObj, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_VirtualGatewayListener_To_SDK_VirtualGatewayListener(t *testing.T) {
	port80 := appmesh.PortNumber(80)
	protocolHTTP := appmesh.VirtualGatewayPortProtocolHTTP
	type args struct {
		crdObj *appmesh.VirtualGatewayListener
		sdkObj *appmeshsdk.VirtualGatewayListener
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.VirtualGatewayListener
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.VirtualGatewayListener{
					PortMapping: appmesh.VirtualGatewayPortMapping{
						Port:     port80,
						Protocol: protocolHTTP,
					},
					HealthCheck: &appmesh.VirtualGatewayHealthCheckPolicy{
						HealthyThreshold:   3,
						IntervalMillis:     60,
						Path:               aws.String("/"),
						Port:               &port80,
						Protocol:           protocolHTTP,
						TimeoutMillis:      30,
						UnhealthyThreshold: 2,
					},
					ConnectionPool: &appmesh.VirtualGatewayConnectionPool{
						HTTP: &appmesh.HTTPConnectionPool{
							MaxConnections:     50,
							MaxPendingRequests: aws.Int64(40),
						},
					},
					TLS: &appmesh.VirtualGatewayListenerTLS{
						Certificate: appmesh.VirtualGatewayListenerTLSCertificate{
							ACM: &appmesh.VirtualGatewayListenerTLSACMCertificate{
								CertificateARN: "arn-1",
							},
						},
						Mode: appmesh.VirtualGatewayListenerTLSModeStrict,
					},
				},
				sdkObj: &appmeshsdk.VirtualGatewayListener{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.VirtualGatewayListener{
				PortMapping: &appmeshsdk.VirtualGatewayPortMapping{
					Port:     aws.Int64(80),
					Protocol: aws.String("http"),
				},
				HealthCheck: &appmeshsdk.VirtualGatewayHealthCheckPolicy{
					HealthyThreshold:   aws.Int64(3),
					IntervalMillis:     aws.Int64(60),
					Path:               aws.String("/"),
					Port:               aws.Int64(80),
					Protocol:           aws.String("http"),
					TimeoutMillis:      aws.Int64(30),
					UnhealthyThreshold: aws.Int64(2),
				},
				ConnectionPool: &appmeshsdk.VirtualGatewayConnectionPool{
					Http: &appmeshsdk.VirtualGatewayHttpConnectionPool{
						MaxConnections:     aws.Int64(50),
						MaxPendingRequests: aws.Int64(40),
					},
				},
				Tls: &appmeshsdk.VirtualGatewayListenerTls{
					Certificate: &appmeshsdk.VirtualGatewayListenerTlsCertificate{
						Acm: &appmeshsdk.VirtualGatewayListenerTlsAcmCertificate{
							CertificateArn: aws.String("arn-1"),
						},
					},
					Mode: aws.String("STRICT"),
				},
			},
		},
		{
			name: "normal case + nil HealthCheck",
			args: args{
				crdObj: &appmesh.VirtualGatewayListener{
					PortMapping: appmesh.VirtualGatewayPortMapping{
						Port:     port80,
						Protocol: protocolHTTP,
					},
					HealthCheck: nil,
					ConnectionPool: &appmesh.VirtualGatewayConnectionPool{
						HTTP: &appmesh.HTTPConnectionPool{
							MaxConnections:     50,
							MaxPendingRequests: aws.Int64(40),
						},
					},
					TLS: &appmesh.VirtualGatewayListenerTLS{
						Certificate: appmesh.VirtualGatewayListenerTLSCertificate{
							ACM: &appmesh.VirtualGatewayListenerTLSACMCertificate{
								CertificateARN: "arn-1",
							},
						},
						Mode: appmesh.VirtualGatewayListenerTLSModeStrict,
					},
				},
				sdkObj: &appmeshsdk.VirtualGatewayListener{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.VirtualGatewayListener{
				PortMapping: &appmeshsdk.VirtualGatewayPortMapping{
					Port:     aws.Int64(80),
					Protocol: aws.String("http"),
				},
				HealthCheck: nil,
				ConnectionPool: &appmeshsdk.VirtualGatewayConnectionPool{
					Http: &appmeshsdk.VirtualGatewayHttpConnectionPool{
						MaxConnections:     aws.Int64(50),
						MaxPendingRequests: aws.Int64(40),
					},
				},
				Tls: &appmeshsdk.VirtualGatewayListenerTls{
					Certificate: &appmeshsdk.VirtualGatewayListenerTlsCertificate{
						Acm: &appmeshsdk.VirtualGatewayListenerTlsAcmCertificate{
							CertificateArn: aws.String("arn-1"),
						},
					},
					Mode: aws.String("STRICT"),
				},
			},
		},
		{
			name: "normal case + nil TLS",
			args: args{
				crdObj: &appmesh.VirtualGatewayListener{
					PortMapping: appmesh.VirtualGatewayPortMapping{
						Port:     port80,
						Protocol: protocolHTTP,
					},
					HealthCheck: &appmesh.VirtualGatewayHealthCheckPolicy{
						HealthyThreshold:   3,
						IntervalMillis:     60,
						Path:               aws.String("/"),
						Port:               &port80,
						Protocol:           protocolHTTP,
						TimeoutMillis:      30,
						UnhealthyThreshold: 2,
					},
					ConnectionPool: &appmesh.VirtualGatewayConnectionPool{
						HTTP: &appmesh.HTTPConnectionPool{
							MaxConnections:     50,
							MaxPendingRequests: aws.Int64(40),
						},
					},
					TLS: nil,
				},
				sdkObj: &appmeshsdk.VirtualGatewayListener{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.VirtualGatewayListener{
				PortMapping: &appmeshsdk.VirtualGatewayPortMapping{
					Port:     aws.Int64(80),
					Protocol: aws.String("http"),
				},
				HealthCheck: &appmeshsdk.VirtualGatewayHealthCheckPolicy{
					HealthyThreshold:   aws.Int64(3),
					IntervalMillis:     aws.Int64(60),
					Path:               aws.String("/"),
					Port:               aws.Int64(80),
					Protocol:           aws.String("http"),
					TimeoutMillis:      aws.Int64(30),
					UnhealthyThreshold: aws.Int64(2),
				},
				ConnectionPool: &appmeshsdk.VirtualGatewayConnectionPool{
					Http: &appmeshsdk.VirtualGatewayHttpConnectionPool{
						MaxConnections:     aws.Int64(50),
						MaxPendingRequests: aws.Int64(40),
					},
				},
				Tls: nil,
			},
		},
		{
			name: "normal case + nil HealthCheck + nil TLS + nil connection pool",
			args: args{
				crdObj: &appmesh.VirtualGatewayListener{
					PortMapping: appmesh.VirtualGatewayPortMapping{
						Port:     port80,
						Protocol: protocolHTTP,
					},
					HealthCheck: nil,
					TLS:         nil,
				},
				sdkObj: &appmeshsdk.VirtualGatewayListener{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.VirtualGatewayListener{
				PortMapping: &appmeshsdk.VirtualGatewayPortMapping{
					Port:     aws.Int64(80),
					Protocol: aws.String("http"),
				},
				HealthCheck: nil,
				Tls:         nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_VirtualGatewayListener_To_SDK_VirtualGatewayListener(tt.args.crdObj, tt.args.sdkObj, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_VirtualGatewaySpec_To_SDK_VirtualGatewaySpec(t *testing.T) {
	port80 := appmesh.PortNumber(80)
	port443 := appmesh.PortNumber(443)
	protocolHTTP := appmesh.VirtualGatewayPortProtocolHTTP
	protocolHTTP2 := appmesh.VirtualGatewayPortProtocolHTTP2
	type args struct {
		crdObj           *appmesh.VirtualGatewaySpec
		sdkObj           *appmeshsdk.VirtualGatewaySpec
		scopeConvertFunc func(src, dest interface{}) error
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.VirtualGatewaySpec
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.VirtualGatewaySpec{
					Listeners: []appmesh.VirtualGatewayListener{
						{
							PortMapping: appmesh.VirtualGatewayPortMapping{
								Port:     port80,
								Protocol: protocolHTTP,
							},
							HealthCheck: &appmesh.VirtualGatewayHealthCheckPolicy{
								HealthyThreshold:   3,
								IntervalMillis:     60,
								Path:               aws.String("/"),
								Port:               &port80,
								Protocol:           protocolHTTP,
								TimeoutMillis:      30,
								UnhealthyThreshold: 2,
							},
							TLS: &appmesh.VirtualGatewayListenerTLS{
								Certificate: appmesh.VirtualGatewayListenerTLSCertificate{
									ACM: &appmesh.VirtualGatewayListenerTLSACMCertificate{
										CertificateARN: "arn-1",
									},
								},
								Mode: appmesh.VirtualGatewayListenerTLSModeStrict,
							},
						},
						{
							PortMapping: appmesh.VirtualGatewayPortMapping{
								Port:     port443,
								Protocol: protocolHTTP2,
							},
						},
					},
					Logging: &appmesh.VirtualGatewayLogging{
						AccessLog: &appmesh.VirtualGatewayAccessLog{
							File: &appmesh.VirtualGatewayFileAccessLog{
								Path: "/",
							},
						},
					},
					BackendDefaults: &appmesh.VirtualGatewayBackendDefaults{
						ClientPolicy: &appmesh.VirtualGatewayClientPolicy{
							TLS: &appmesh.VirtualGatewayClientPolicyTLS{
								Enforce: aws.Bool(true),
								Ports:   []appmesh.PortNumber{80, 443},
								Validation: appmesh.VirtualGatewayTLSValidationContext{
									Trust: appmesh.VirtualGatewayTLSValidationContextTrust{
										ACM: &appmesh.VirtualGatewayTLSValidationContextACMTrust{
											CertificateAuthorityARNs: []string{"arn-1", "arn-2"},
										},
									},
								},
							},
						},
					},
					MeshRef: nil,
				},
				sdkObj: &appmeshsdk.VirtualGatewaySpec{},
				scopeConvertFunc: func(src, dest interface{}) error {
					vsRef := src.(*appmesh.VirtualServiceReference)
					vsNamePtr := dest.(*string)
					*vsNamePtr = fmt.Sprintf("%s.%s", vsRef.Name, aws.StringValue(vsRef.Namespace))
					return nil
				},
			},
			wantSDKObj: &appmeshsdk.VirtualGatewaySpec{
				Listeners: []*appmeshsdk.VirtualGatewayListener{
					{
						PortMapping: &appmeshsdk.VirtualGatewayPortMapping{
							Port:     aws.Int64(80),
							Protocol: aws.String("http"),
						},
						HealthCheck: &appmeshsdk.VirtualGatewayHealthCheckPolicy{
							HealthyThreshold:   aws.Int64(3),
							IntervalMillis:     aws.Int64(60),
							Path:               aws.String("/"),
							Port:               aws.Int64(80),
							Protocol:           aws.String("http"),
							TimeoutMillis:      aws.Int64(30),
							UnhealthyThreshold: aws.Int64(2),
						},
						Tls: &appmeshsdk.VirtualGatewayListenerTls{
							Certificate: &appmeshsdk.VirtualGatewayListenerTlsCertificate{
								Acm: &appmeshsdk.VirtualGatewayListenerTlsAcmCertificate{
									CertificateArn: aws.String("arn-1"),
								},
							},
							Mode: aws.String("STRICT"),
						},
					},
					{
						PortMapping: &appmeshsdk.VirtualGatewayPortMapping{
							Port:     aws.Int64(443),
							Protocol: aws.String("http2"),
						},
					},
				},
				Logging: &appmeshsdk.VirtualGatewayLogging{
					AccessLog: &appmeshsdk.VirtualGatewayAccessLog{
						File: &appmeshsdk.VirtualGatewayFileAccessLog{
							Path: aws.String("/"),
						},
					},
				},
				BackendDefaults: &appmeshsdk.VirtualGatewayBackendDefaults{
					ClientPolicy: &appmeshsdk.VirtualGatewayClientPolicy{
						Tls: &appmeshsdk.VirtualGatewayClientPolicyTls{
							Enforce: aws.Bool(true),
							Ports:   []*int64{aws.Int64(80), aws.Int64(443)},
							Validation: &appmeshsdk.VirtualGatewayTlsValidationContext{
								Trust: &appmeshsdk.VirtualGatewayTlsValidationContextTrust{
									Acm: &appmeshsdk.VirtualGatewayTlsValidationContextAcmTrust{
										CertificateAuthorityArns: []*string{aws.String("arn-1"), aws.String("arn-2")},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "normal case + nil listener",
			args: args{
				crdObj: &appmesh.VirtualGatewaySpec{
					Listeners: nil,
					BackendDefaults: &appmesh.VirtualGatewayBackendDefaults{
						ClientPolicy: &appmesh.VirtualGatewayClientPolicy{
							TLS: &appmesh.VirtualGatewayClientPolicyTLS{
								Enforce: aws.Bool(true),
								Ports:   []appmesh.PortNumber{80, 443},
								Validation: appmesh.VirtualGatewayTLSValidationContext{
									Trust: appmesh.VirtualGatewayTLSValidationContextTrust{
										ACM: &appmesh.VirtualGatewayTLSValidationContextACMTrust{
											CertificateAuthorityARNs: []string{"arn-1", "arn-2"},
										},
									},
								},
							},
						},
					},
					MeshRef: nil,
				},
				sdkObj: &appmeshsdk.VirtualGatewaySpec{},
				scopeConvertFunc: func(src, dest interface{}) error {
					vsRef := src.(*appmesh.VirtualServiceReference)
					vsNamePtr := dest.(*string)
					*vsNamePtr = fmt.Sprintf("%s.%s", vsRef.Name, aws.StringValue(vsRef.Namespace))
					return nil
				},
			},
			wantSDKObj: &appmeshsdk.VirtualGatewaySpec{
				Listeners: nil,
				BackendDefaults: &appmeshsdk.VirtualGatewayBackendDefaults{
					ClientPolicy: &appmeshsdk.VirtualGatewayClientPolicy{
						Tls: &appmeshsdk.VirtualGatewayClientPolicyTls{
							Enforce: aws.Bool(true),
							Ports:   []*int64{aws.Int64(80), aws.Int64(443)},
							Validation: &appmeshsdk.VirtualGatewayTlsValidationContext{
								Trust: &appmeshsdk.VirtualGatewayTlsValidationContextTrust{
									Acm: &appmeshsdk.VirtualGatewayTlsValidationContextAcmTrust{
										CertificateAuthorityArns: []*string{aws.String("arn-1"), aws.String("arn-2")},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "normal case + nil BackendDefaults",
			args: args{
				crdObj: &appmesh.VirtualGatewaySpec{
					Listeners: []appmesh.VirtualGatewayListener{
						{
							PortMapping: appmesh.VirtualGatewayPortMapping{
								Port:     port80,
								Protocol: protocolHTTP,
							},
							HealthCheck: &appmesh.VirtualGatewayHealthCheckPolicy{
								HealthyThreshold:   3,
								IntervalMillis:     60,
								Path:               aws.String("/"),
								Port:               &port80,
								Protocol:           protocolHTTP,
								TimeoutMillis:      30,
								UnhealthyThreshold: 2,
							},
							TLS: &appmesh.VirtualGatewayListenerTLS{
								Certificate: appmesh.VirtualGatewayListenerTLSCertificate{
									ACM: &appmesh.VirtualGatewayListenerTLSACMCertificate{
										CertificateARN: "arn-1",
									},
								},
								Mode: appmesh.VirtualGatewayListenerTLSModeStrict,
							},
						},
						{
							PortMapping: appmesh.VirtualGatewayPortMapping{
								Port:     port443,
								Protocol: protocolHTTP2,
							},
						},
					},
					Logging: &appmesh.VirtualGatewayLogging{
						AccessLog: &appmesh.VirtualGatewayAccessLog{
							File: &appmesh.VirtualGatewayFileAccessLog{
								Path: "/",
							},
						},
					},
					BackendDefaults: nil,
					MeshRef:         nil,
				},
				sdkObj: &appmeshsdk.VirtualGatewaySpec{},
				scopeConvertFunc: func(src, dest interface{}) error {
					vsRef := src.(*appmesh.VirtualServiceReference)
					vsNamePtr := dest.(*string)
					*vsNamePtr = fmt.Sprintf("%s.%s", vsRef.Name, aws.StringValue(vsRef.Namespace))
					return nil
				},
			},
			wantSDKObj: &appmeshsdk.VirtualGatewaySpec{
				Listeners: []*appmeshsdk.VirtualGatewayListener{
					{
						PortMapping: &appmeshsdk.VirtualGatewayPortMapping{
							Port:     aws.Int64(80),
							Protocol: aws.String("http"),
						},
						HealthCheck: &appmeshsdk.VirtualGatewayHealthCheckPolicy{
							HealthyThreshold:   aws.Int64(3),
							IntervalMillis:     aws.Int64(60),
							Path:               aws.String("/"),
							Port:               aws.Int64(80),
							Protocol:           aws.String("http"),
							TimeoutMillis:      aws.Int64(30),
							UnhealthyThreshold: aws.Int64(2),
						},
						Tls: &appmeshsdk.VirtualGatewayListenerTls{
							Certificate: &appmeshsdk.VirtualGatewayListenerTlsCertificate{
								Acm: &appmeshsdk.VirtualGatewayListenerTlsAcmCertificate{
									CertificateArn: aws.String("arn-1"),
								},
							},
							Mode: aws.String("STRICT"),
						},
					},
					{
						PortMapping: &appmeshsdk.VirtualGatewayPortMapping{
							Port:     aws.Int64(443),
							Protocol: aws.String("http2"),
						},
					},
				},
				Logging: &appmeshsdk.VirtualGatewayLogging{
					AccessLog: &appmeshsdk.VirtualGatewayAccessLog{
						File: &appmeshsdk.VirtualGatewayFileAccessLog{
							Path: aws.String("/"),
						},
					},
				},
				BackendDefaults: nil,
			},
		},
		{
			name: "normal case + nil logging",
			args: args{
				crdObj: &appmesh.VirtualGatewaySpec{
					Listeners: []appmesh.VirtualGatewayListener{
						{
							PortMapping: appmesh.VirtualGatewayPortMapping{
								Port:     port80,
								Protocol: protocolHTTP,
							},
							HealthCheck: &appmesh.VirtualGatewayHealthCheckPolicy{
								HealthyThreshold:   3,
								IntervalMillis:     60,
								Path:               aws.String("/"),
								Port:               &port80,
								Protocol:           protocolHTTP,
								TimeoutMillis:      30,
								UnhealthyThreshold: 2,
							},
							TLS: &appmesh.VirtualGatewayListenerTLS{
								Certificate: appmesh.VirtualGatewayListenerTLSCertificate{
									ACM: &appmesh.VirtualGatewayListenerTLSACMCertificate{
										CertificateARN: "arn-1",
									},
								},
								Mode: appmesh.VirtualGatewayListenerTLSModeStrict,
							},
						},
						{
							PortMapping: appmesh.VirtualGatewayPortMapping{
								Port:     port443,
								Protocol: protocolHTTP2,
							},
						},
					},
					Logging: nil,
					BackendDefaults: &appmesh.VirtualGatewayBackendDefaults{
						ClientPolicy: &appmesh.VirtualGatewayClientPolicy{
							TLS: &appmesh.VirtualGatewayClientPolicyTLS{
								Enforce: aws.Bool(true),
								Ports:   []appmesh.PortNumber{80, 443},
								Validation: appmesh.VirtualGatewayTLSValidationContext{
									Trust: appmesh.VirtualGatewayTLSValidationContextTrust{
										ACM: &appmesh.VirtualGatewayTLSValidationContextACMTrust{
											CertificateAuthorityARNs: []string{"arn-1", "arn-2"},
										},
									},
								},
							},
						},
					},
					MeshRef: nil,
				},
				sdkObj: &appmeshsdk.VirtualGatewaySpec{},
				scopeConvertFunc: func(src, dest interface{}) error {
					vsRef := src.(*appmesh.VirtualServiceReference)
					vsNamePtr := dest.(*string)
					*vsNamePtr = fmt.Sprintf("%s.%s", vsRef.Name, aws.StringValue(vsRef.Namespace))
					return nil
				},
			},
			wantSDKObj: &appmeshsdk.VirtualGatewaySpec{
				Listeners: []*appmeshsdk.VirtualGatewayListener{
					{
						PortMapping: &appmeshsdk.VirtualGatewayPortMapping{
							Port:     aws.Int64(80),
							Protocol: aws.String("http"),
						},
						HealthCheck: &appmeshsdk.VirtualGatewayHealthCheckPolicy{
							HealthyThreshold:   aws.Int64(3),
							IntervalMillis:     aws.Int64(60),
							Path:               aws.String("/"),
							Port:               aws.Int64(80),
							Protocol:           aws.String("http"),
							TimeoutMillis:      aws.Int64(30),
							UnhealthyThreshold: aws.Int64(2),
						},
						Tls: &appmeshsdk.VirtualGatewayListenerTls{
							Certificate: &appmeshsdk.VirtualGatewayListenerTlsCertificate{
								Acm: &appmeshsdk.VirtualGatewayListenerTlsAcmCertificate{
									CertificateArn: aws.String("arn-1"),
								},
							},
							Mode: aws.String("STRICT"),
						},
					},
					{
						PortMapping: &appmeshsdk.VirtualGatewayPortMapping{
							Port:     aws.Int64(443),
							Protocol: aws.String("http2"),
						},
					},
				},
				Logging: nil,
				BackendDefaults: &appmeshsdk.VirtualGatewayBackendDefaults{
					ClientPolicy: &appmeshsdk.VirtualGatewayClientPolicy{
						Tls: &appmeshsdk.VirtualGatewayClientPolicyTls{
							Enforce: aws.Bool(true),
							Ports:   []*int64{aws.Int64(80), aws.Int64(443)},
							Validation: &appmeshsdk.VirtualGatewayTlsValidationContext{
								Trust: &appmeshsdk.VirtualGatewayTlsValidationContextTrust{
									Acm: &appmeshsdk.VirtualGatewayTlsValidationContextAcmTrust{
										CertificateAuthorityArns: []*string{aws.String("arn-1"), aws.String("arn-2")},
									},
								},
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
			err := Convert_CRD_VirtualGatewaySpec_To_SDK_VirtualGatewaySpec(tt.args.crdObj, tt.args.sdkObj, scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}
