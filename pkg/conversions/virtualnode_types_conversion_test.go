package conversions

import (
	"fmt"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	mock_conversion "github.com/aws/aws-app-mesh-controller-for-k8s/mocks/apimachinery/pkg/conversion"
	"github.com/aws/aws-sdk-go/aws"
	appmeshsdk "github.com/aws/aws-sdk-go/service/appmesh"
	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
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
		name       string
		args       args
		wantSDKObj *appmeshsdk.TlsValidationContextAcmTrust
		wantErr    error
	}{
		{
			name: "single arn",
			args: args{
				crdObj: &appmesh.TLSValidationContextACMTrust{
					CertificateAuthorityARNs: []string{"arn-1"},
				},
				sdkObj: &appmeshsdk.TlsValidationContextAcmTrust{},
			},
			wantSDKObj: &appmeshsdk.TlsValidationContextAcmTrust{
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
			wantSDKObj: &appmeshsdk.TlsValidationContextAcmTrust{
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
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_TLSValidationContextFileTrust_To_SDK_TLSValidationContextFileTrust(t *testing.T) {
	type args struct {
		crdObj *appmesh.TLSValidationContextFileTrust
		sdkObj *appmeshsdk.TlsValidationContextFileTrust
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.TlsValidationContextFileTrust
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.TLSValidationContextFileTrust{
					CertificateChain: "dummy",
				},
				sdkObj: &appmeshsdk.TlsValidationContextFileTrust{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.TlsValidationContextFileTrust{
				CertificateChain: aws.String("dummy"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_TLSValidationContextFileTrust_To_SDK_TLSValidationContextFileTrust(tt.args.crdObj, tt.args.sdkObj, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_TLSValidationContextTrust_To_SDK_TLSValidationContextTrust(t *testing.T) {
	type args struct {
		crdObj *appmesh.TLSValidationContextTrust
		sdkObj *appmeshsdk.TlsValidationContextTrust
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.TlsValidationContextTrust
		wantErr    error
	}{
		{
			name: "acm validation context",
			args: args{
				crdObj: &appmesh.TLSValidationContextTrust{
					ACM: &appmesh.TLSValidationContextACMTrust{
						CertificateAuthorityARNs: []string{"arn-1", "arn-2"},
					},
				},
				sdkObj: &appmeshsdk.TlsValidationContextTrust{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.TlsValidationContextTrust{
				Acm: &appmeshsdk.TlsValidationContextAcmTrust{
					CertificateAuthorityArns: []*string{aws.String("arn-1"), aws.String("arn-2")},
				},
			},
		},
		{
			name: "file validation context",
			args: args{
				crdObj: &appmesh.TLSValidationContextTrust{
					File: &appmesh.TLSValidationContextFileTrust{
						CertificateChain: "dummy",
					},
				},
				sdkObj: &appmeshsdk.TlsValidationContextTrust{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.TlsValidationContextTrust{
				File: &appmeshsdk.TlsValidationContextFileTrust{
					CertificateChain: aws.String("dummy"),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_TLSValidationContextTrust_To_SDK_TLSValidationContextTrust(tt.args.crdObj, tt.args.sdkObj, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_TLSValidationContext_To_SDK_TLSValidationContext(t *testing.T) {
	type args struct {
		crdObj *appmesh.TLSValidationContext
		sdkObj *appmeshsdk.TlsValidationContext
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.TlsValidationContext
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.TLSValidationContext{
					Trust: appmesh.TLSValidationContextTrust{
						ACM: &appmesh.TLSValidationContextACMTrust{
							CertificateAuthorityARNs: []string{"arn-1", "arn-2"},
						},
					},
				},
				sdkObj: &appmeshsdk.TlsValidationContext{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.TlsValidationContext{
				Trust: &appmeshsdk.TlsValidationContextTrust{
					Acm: &appmeshsdk.TlsValidationContextAcmTrust{
						CertificateAuthorityArns: []*string{aws.String("arn-1"), aws.String("arn-2")},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_TLSValidationContext_To_SDK_TLSValidationContext(tt.args.crdObj, tt.args.sdkObj, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_ClientPolicyTLS_To_SDK_ClientPolicyTLS(t *testing.T) {
	type args struct {
		crdObj *appmesh.ClientPolicyTLS
		sdkObj *appmeshsdk.ClientPolicyTls
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.ClientPolicyTls
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.ClientPolicyTLS{
					Enforce: aws.Bool(true),
					Ports:   []appmesh.PortNumber{80, 443},
					Validation: appmesh.TLSValidationContext{
						Trust: appmesh.TLSValidationContextTrust{
							ACM: &appmesh.TLSValidationContextACMTrust{
								CertificateAuthorityARNs: []string{"arn-1", "arn-2"},
							},
						},
					},
				},
				sdkObj: &appmeshsdk.ClientPolicyTls{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.ClientPolicyTls{
				Enforce: aws.Bool(true),
				Ports:   []*int64{aws.Int64(80), aws.Int64(443)},
				Validation: &appmeshsdk.TlsValidationContext{
					Trust: &appmeshsdk.TlsValidationContextTrust{
						Acm: &appmeshsdk.TlsValidationContextAcmTrust{
							CertificateAuthorityArns: []*string{aws.String("arn-1"), aws.String("arn-2")},
						},
					},
				},
			},
		},
		{
			name: "normal case + enforce false",
			args: args{
				crdObj: &appmesh.ClientPolicyTLS{
					Enforce: aws.Bool(false),
					Ports:   []appmesh.PortNumber{80, 443},
					Validation: appmesh.TLSValidationContext{
						Trust: appmesh.TLSValidationContextTrust{
							ACM: &appmesh.TLSValidationContextACMTrust{
								CertificateAuthorityARNs: []string{"arn-1", "arn-2"},
							},
						},
					},
				},
				sdkObj: &appmeshsdk.ClientPolicyTls{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.ClientPolicyTls{
				Enforce: aws.Bool(false),
				Ports:   []*int64{aws.Int64(80), aws.Int64(443)},
				Validation: &appmeshsdk.TlsValidationContext{
					Trust: &appmeshsdk.TlsValidationContextTrust{
						Acm: &appmeshsdk.TlsValidationContextAcmTrust{
							CertificateAuthorityArns: []*string{aws.String("arn-1"), aws.String("arn-2")},
						},
					},
				},
			},
		},
		{
			name: "normal case + enforce unset",
			args: args{
				crdObj: &appmesh.ClientPolicyTLS{
					Enforce: nil,
					Ports:   []appmesh.PortNumber{80, 443},
					Validation: appmesh.TLSValidationContext{
						Trust: appmesh.TLSValidationContextTrust{
							ACM: &appmesh.TLSValidationContextACMTrust{
								CertificateAuthorityARNs: []string{"arn-1", "arn-2"},
							},
						},
					},
				},
				sdkObj: &appmeshsdk.ClientPolicyTls{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.ClientPolicyTls{
				Enforce: nil,
				Ports:   []*int64{aws.Int64(80), aws.Int64(443)},
				Validation: &appmeshsdk.TlsValidationContext{
					Trust: &appmeshsdk.TlsValidationContextTrust{
						Acm: &appmeshsdk.TlsValidationContextAcmTrust{
							CertificateAuthorityArns: []*string{aws.String("arn-1"), aws.String("arn-2")},
						},
					},
				},
			},
		},
		{
			name: "normal case + nil ports",
			args: args{
				crdObj: &appmesh.ClientPolicyTLS{
					Enforce: aws.Bool(true),
					Ports:   nil,
					Validation: appmesh.TLSValidationContext{
						Trust: appmesh.TLSValidationContextTrust{
							ACM: &appmesh.TLSValidationContextACMTrust{
								CertificateAuthorityARNs: []string{"arn-1", "arn-2"},
							},
						},
					},
				},
				sdkObj: &appmeshsdk.ClientPolicyTls{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.ClientPolicyTls{
				Enforce: aws.Bool(true),
				Ports:   nil,
				Validation: &appmeshsdk.TlsValidationContext{
					Trust: &appmeshsdk.TlsValidationContextTrust{
						Acm: &appmeshsdk.TlsValidationContextAcmTrust{
							CertificateAuthorityArns: []*string{aws.String("arn-1"), aws.String("arn-2")},
						},
					},
				},
			},
		},
		{
			name: "normal case + empty ports",
			args: args{
				crdObj: &appmesh.ClientPolicyTLS{
					Enforce: aws.Bool(true),
					Ports:   []appmesh.PortNumber{},
					Validation: appmesh.TLSValidationContext{
						Trust: appmesh.TLSValidationContextTrust{
							ACM: &appmesh.TLSValidationContextACMTrust{
								CertificateAuthorityARNs: []string{"arn-1", "arn-2"},
							},
						},
					},
				},
				sdkObj: &appmeshsdk.ClientPolicyTls{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.ClientPolicyTls{
				Enforce: aws.Bool(true),
				Ports:   nil,
				Validation: &appmeshsdk.TlsValidationContext{
					Trust: &appmeshsdk.TlsValidationContextTrust{
						Acm: &appmeshsdk.TlsValidationContextAcmTrust{
							CertificateAuthorityArns: []*string{aws.String("arn-1"), aws.String("arn-2")},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_ClientPolicyTLS_To_SDK_ClientPolicyTLS(tt.args.crdObj, tt.args.sdkObj, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_ClientPolicy_To_SDK_ClientPolicy(t *testing.T) {
	type args struct {
		crdObj *appmesh.ClientPolicy
		sdkObj *appmeshsdk.ClientPolicy
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.ClientPolicy
		wantErr    error
	}{
		{
			name: "non nil TLS",
			args: args{
				crdObj: &appmesh.ClientPolicy{
					TLS: &appmesh.ClientPolicyTLS{
						Enforce: aws.Bool(true),
						Ports:   []appmesh.PortNumber{80, 443},
						Validation: appmesh.TLSValidationContext{
							Trust: appmesh.TLSValidationContextTrust{
								ACM: &appmesh.TLSValidationContextACMTrust{
									CertificateAuthorityARNs: []string{"arn-1", "arn-2"},
								},
							},
						},
					},
				},
				sdkObj: &appmeshsdk.ClientPolicy{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.ClientPolicy{
				Tls: &appmeshsdk.ClientPolicyTls{
					Enforce: aws.Bool(true),
					Ports:   []*int64{aws.Int64(80), aws.Int64(443)},
					Validation: &appmeshsdk.TlsValidationContext{
						Trust: &appmeshsdk.TlsValidationContextTrust{
							Acm: &appmeshsdk.TlsValidationContextAcmTrust{
								CertificateAuthorityArns: []*string{aws.String("arn-1"), aws.String("arn-2")},
							},
						},
					},
				},
			},
		},
		{
			name: "nil TLS",
			args: args{
				crdObj: &appmesh.ClientPolicy{
					TLS: nil,
				},
				sdkObj: &appmeshsdk.ClientPolicy{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.ClientPolicy{
				Tls: nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_ClientPolicy_To_SDK_ClientPolicy(tt.args.crdObj, tt.args.sdkObj, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_VirtualServiceBackend_To_SDK_VirtualServiceBackend(t *testing.T) {
	type args struct {
		crdObj           *appmesh.VirtualServiceBackend
		sdkObj           *appmeshsdk.VirtualServiceBackend
		scopeConvertFunc func(src, dest interface{}, flags conversion.FieldMatchingFlags) error
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.VirtualServiceBackend
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.VirtualServiceBackend{
					VirtualServiceRef: appmesh.VirtualServiceReference{
						Namespace: aws.String("my-ns"),
						Name:      "my-vs",
					},
					ClientPolicy: &appmesh.ClientPolicy{
						TLS: &appmesh.ClientPolicyTLS{
							Enforce: aws.Bool(true),
							Ports:   []appmesh.PortNumber{80, 443},
							Validation: appmesh.TLSValidationContext{
								Trust: appmesh.TLSValidationContextTrust{
									ACM: &appmesh.TLSValidationContextACMTrust{
										CertificateAuthorityARNs: []string{"arn-1", "arn-2"},
									},
								},
							},
						},
					},
				},
				sdkObj: &appmeshsdk.VirtualServiceBackend{},
				scopeConvertFunc: func(src, dest interface{}, flags conversion.FieldMatchingFlags) error {
					vsRef := src.(*appmesh.VirtualServiceReference)
					vsNamePtr := dest.(*string)
					*vsNamePtr = fmt.Sprintf("%s.%s", vsRef.Name, aws.StringValue(vsRef.Namespace))
					return nil
				},
			},
			wantSDKObj: &appmeshsdk.VirtualServiceBackend{
				VirtualServiceName: aws.String("my-vs.my-ns"),
				ClientPolicy: &appmeshsdk.ClientPolicy{
					Tls: &appmeshsdk.ClientPolicyTls{
						Enforce: aws.Bool(true),
						Ports:   []*int64{aws.Int64(80), aws.Int64(443)},
						Validation: &appmeshsdk.TlsValidationContext{
							Trust: &appmeshsdk.TlsValidationContextTrust{
								Acm: &appmeshsdk.TlsValidationContextAcmTrust{
									CertificateAuthorityArns: []*string{aws.String("arn-1"), aws.String("arn-2")},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "normal case + nil ClientPolicy",
			args: args{
				crdObj: &appmesh.VirtualServiceBackend{
					VirtualServiceRef: appmesh.VirtualServiceReference{
						Namespace: aws.String("my-ns"),
						Name:      "my-vs",
					},
					ClientPolicy: nil,
				},
				sdkObj: &appmeshsdk.VirtualServiceBackend{},
				scopeConvertFunc: func(src, dest interface{}, flags conversion.FieldMatchingFlags) error {
					vsRef := src.(*appmesh.VirtualServiceReference)
					vsNamePtr := dest.(*string)
					*vsNamePtr = fmt.Sprintf("%s.%s", vsRef.Name, aws.StringValue(vsRef.Namespace))
					return nil
				},
			},
			wantSDKObj: &appmeshsdk.VirtualServiceBackend{
				VirtualServiceName: aws.String("my-vs.my-ns"),
				ClientPolicy:       nil,
			},
		},
		{
			name: "error when convert VirtualServiceReference",
			args: args{
				crdObj: &appmesh.VirtualServiceBackend{
					VirtualServiceRef: appmesh.VirtualServiceReference{
						Namespace: aws.String("my-ns"),
						Name:      "my-vs",
					},
					ClientPolicy: nil,
				},
				sdkObj: &appmeshsdk.VirtualServiceBackend{},
				scopeConvertFunc: func(src, dest interface{}, flags conversion.FieldMatchingFlags) error {
					return errors.New("error convert VirtualServiceReference")
				},
			},
			wantErr: errors.New("error convert VirtualServiceReference"),
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

			err := Convert_CRD_VirtualServiceBackend_To_SDK_VirtualServiceBackend(tt.args.crdObj, tt.args.sdkObj, scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_Backend_To_SDK_Backend(t *testing.T) {
	type args struct {
		crdObj           *appmesh.Backend
		sdkObj           *appmeshsdk.Backend
		scopeConvertFunc func(src, dest interface{}, flags conversion.FieldMatchingFlags) error
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.Backend
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.Backend{
					VirtualService: appmesh.VirtualServiceBackend{
						VirtualServiceRef: appmesh.VirtualServiceReference{
							Namespace: aws.String("my-ns"),
							Name:      "my-vs",
						},
					},
				},
				sdkObj: &appmeshsdk.Backend{},
				scopeConvertFunc: func(src, dest interface{}, flags conversion.FieldMatchingFlags) error {
					vsRef := src.(*appmesh.VirtualServiceReference)
					vsNamePtr := dest.(*string)
					*vsNamePtr = fmt.Sprintf("%s.%s", vsRef.Name, aws.StringValue(vsRef.Namespace))
					return nil
				},
			},
			wantSDKObj: &appmeshsdk.Backend{
				VirtualService: &appmeshsdk.VirtualServiceBackend{
					VirtualServiceName: aws.String("my-vs.my-ns"),
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
				scope.EXPECT().Convert(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(tt.args.scopeConvertFunc)
			}
			scope.EXPECT().Flags().Return(conversion.DestFromSource)
			err := Convert_CRD_Backend_To_SDK_Backend(tt.args.crdObj, tt.args.sdkObj, scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_BackendDefaults_To_SDK_BackendDefaults(t *testing.T) {
	type args struct {
		crdObj *appmesh.BackendDefaults
		sdkObj *appmeshsdk.BackendDefaults
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.BackendDefaults
		wantErr    error
	}{
		{
			name: "non-nil clientPolicy",
			args: args{
				crdObj: &appmesh.BackendDefaults{
					ClientPolicy: &appmesh.ClientPolicy{
						TLS: &appmesh.ClientPolicyTLS{
							Enforce: aws.Bool(true),
							Ports:   []appmesh.PortNumber{80, 443},
							Validation: appmesh.TLSValidationContext{
								Trust: appmesh.TLSValidationContextTrust{
									ACM: &appmesh.TLSValidationContextACMTrust{
										CertificateAuthorityARNs: []string{"arn-1", "arn-2"},
									},
								},
							},
						},
					},
				},
				sdkObj: &appmeshsdk.BackendDefaults{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.BackendDefaults{
				ClientPolicy: &appmeshsdk.ClientPolicy{
					Tls: &appmeshsdk.ClientPolicyTls{
						Enforce: aws.Bool(true),
						Ports:   []*int64{aws.Int64(80), aws.Int64(443)},
						Validation: &appmeshsdk.TlsValidationContext{
							Trust: &appmeshsdk.TlsValidationContextTrust{
								Acm: &appmeshsdk.TlsValidationContextAcmTrust{
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
				crdObj: &appmesh.BackendDefaults{
					ClientPolicy: nil,
				},
				sdkObj: &appmeshsdk.BackendDefaults{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.BackendDefaults{
				ClientPolicy: nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_BackendDefaults_To_SDK_BackendDefaults(tt.args.crdObj, tt.args.sdkObj, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_HealthCheckPolicy_To_SDK_HealthCheckPolicy(t *testing.T) {
	port80 := appmesh.PortNumber(80)
	protocolHTTP := appmesh.PortProtocolHTTP
	type args struct {
		crdObj *appmesh.HealthCheckPolicy
		sdkObj *appmeshsdk.HealthCheckPolicy
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.HealthCheckPolicy
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.HealthCheckPolicy{
					HealthyThreshold:   3,
					IntervalMillis:     60,
					Path:               aws.String("/"),
					Port:               &port80,
					Protocol:           protocolHTTP,
					TimeoutMillis:      30,
					UnhealthyThreshold: 2,
				},
				sdkObj: &appmeshsdk.HealthCheckPolicy{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.HealthCheckPolicy{
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
				crdObj: &appmesh.HealthCheckPolicy{
					HealthyThreshold:   3,
					IntervalMillis:     60,
					Path:               aws.String("/"),
					Port:               nil,
					Protocol:           protocolHTTP,
					TimeoutMillis:      30,
					UnhealthyThreshold: 2,
				},
				sdkObj: &appmeshsdk.HealthCheckPolicy{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.HealthCheckPolicy{
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
			err := Convert_CRD_HealthCheckPolicy_To_SDK_HealthCheckPolicy(tt.args.crdObj, tt.args.sdkObj, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_ListenerTLSACMCertificate_To_SDK_ListenerTLSACMCertificate(t *testing.T) {
	type args struct {
		crdObj *appmesh.ListenerTLSACMCertificate
		sdkObj *appmeshsdk.ListenerTlsAcmCertificate
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.ListenerTlsAcmCertificate
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.ListenerTLSACMCertificate{
					CertificateARN: "arn-1",
				},
				sdkObj: &appmeshsdk.ListenerTlsAcmCertificate{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.ListenerTlsAcmCertificate{
				CertificateArn: aws.String("arn-1"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_ListenerTLSACMCertificate_To_SDK_ListenerTLSACMCertificate(tt.args.crdObj, tt.args.sdkObj, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_ListenerTLSFileCertificate_To_SDK_ListenerTLSFileCertificate(t *testing.T) {
	type args struct {
		crdObj *appmesh.ListenerTLSFileCertificate
		sdkObj *appmeshsdk.ListenerTlsFileCertificate
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.ListenerTlsFileCertificate
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.ListenerTLSFileCertificate{
					CertificateChain: "certificateChain",
					PrivateKey:       "privateKey",
				},
				sdkObj: &appmeshsdk.ListenerTlsFileCertificate{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.ListenerTlsFileCertificate{
				CertificateChain: aws.String("certificateChain"),
				PrivateKey:       aws.String("privateKey"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_ListenerTLSFileCertificate_To_SDK_ListenerTLSFileCertificate(tt.args.crdObj, tt.args.sdkObj, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_ListenerTLSCertificate_To_SDK_ListenerTLSCertificate(t *testing.T) {
	type args struct {
		crdObj *appmesh.ListenerTLSCertificate
		sdkObj *appmeshsdk.ListenerTlsCertificate
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.ListenerTlsCertificate
		wantErr    error
	}{
		{
			name: "acm certificate",
			args: args{
				crdObj: &appmesh.ListenerTLSCertificate{
					ACM: &appmesh.ListenerTLSACMCertificate{
						CertificateARN: "arn-1",
					},
				},
				sdkObj: &appmeshsdk.ListenerTlsCertificate{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.ListenerTlsCertificate{
				Acm: &appmeshsdk.ListenerTlsAcmCertificate{
					CertificateArn: aws.String("arn-1"),
				},
			},
		},
		{
			name: "file certificate",
			args: args{
				crdObj: &appmesh.ListenerTLSCertificate{
					File: &appmesh.ListenerTLSFileCertificate{
						CertificateChain: "certificateChain",
						PrivateKey:       "privateKey",
					},
				},
				sdkObj: &appmeshsdk.ListenerTlsCertificate{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.ListenerTlsCertificate{
				File: &appmeshsdk.ListenerTlsFileCertificate{
					CertificateChain: aws.String("certificateChain"),
					PrivateKey:       aws.String("privateKey"),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_ListenerTLSCertificate_To_SDK_ListenerTLSCertificate(tt.args.crdObj, tt.args.sdkObj, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_ListenerTLS_To_SDK_ListenerTLS(t *testing.T) {
	type args struct {
		crdObj *appmesh.ListenerTLS
		sdkObj *appmeshsdk.ListenerTls
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.ListenerTls
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.ListenerTLS{
					Certificate: appmesh.ListenerTLSCertificate{
						File: &appmesh.ListenerTLSFileCertificate{
							CertificateChain: "certificateChain",
							PrivateKey:       "privateKey",
						},
					},
					Mode: appmesh.ListenerTLSModeStrict,
				},
				sdkObj: &appmeshsdk.ListenerTls{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.ListenerTls{
				Certificate: &appmeshsdk.ListenerTlsCertificate{
					File: &appmeshsdk.ListenerTlsFileCertificate{
						CertificateChain: aws.String("certificateChain"),
						PrivateKey:       aws.String("privateKey"),
					},
				},
				Mode: aws.String("STRICT"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_ListenerTLS_To_SDK_ListenerTLS(tt.args.crdObj, tt.args.sdkObj, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_ListenerTimeoutTcp_To_SDK_ListenerTimeoutTcp(t *testing.T) {
	type args struct {
		crdObj *appmesh.TCPTimeout
		sdkObj *appmeshsdk.TcpTimeout
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.TcpTimeout
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.TCPTimeout{
					Idle: &appmesh.Duration{
						Unit:  "ms",
						Value: int64(200),
					},
				},
				sdkObj: &appmeshsdk.TcpTimeout{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.TcpTimeout{
				Idle: &appmeshsdk.Duration{
					Unit:  aws.String("ms"),
					Value: aws.Int64(200),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_ListenerTimeoutTcp_To_SDK_ListenerTimeoutTcp(tt.args.crdObj, tt.args.sdkObj, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_ListenerTimeoutHttp_To_SDK_ListenerTimeoutHttp(t *testing.T) {
	type args struct {
		crdObj *appmesh.HTTPTimeout
		sdkObj *appmeshsdk.HttpTimeout
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.HttpTimeout
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.HTTPTimeout{
					PerRequest: &appmesh.Duration{
						Unit:  "ms",
						Value: int64(100),
					},
					Idle: &appmesh.Duration{
						Unit:  "ms",
						Value: int64(200),
					},
				},
				sdkObj: &appmeshsdk.HttpTimeout{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.HttpTimeout{
				PerRequest: &appmeshsdk.Duration{
					Unit:  aws.String("ms"),
					Value: aws.Int64(100),
				},
				Idle: &appmeshsdk.Duration{
					Unit:  aws.String("ms"),
					Value: aws.Int64(200),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_ListenerTimeoutHttp_To_SDK_ListenerTimeoutHttp(tt.args.crdObj, tt.args.sdkObj, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_ListenerTimeoutGrpc_To_SDK_ListenerTimeoutGrpc(t *testing.T) {
	type args struct {
		crdObj *appmesh.GRPCTimeout
		sdkObj *appmeshsdk.GrpcTimeout
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.GrpcTimeout
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.GRPCTimeout{
					PerRequest: &appmesh.Duration{
						Unit:  "ms",
						Value: int64(100),
					},
					Idle: &appmesh.Duration{
						Unit:  "ms",
						Value: int64(200),
					},
				},
				sdkObj: &appmeshsdk.GrpcTimeout{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.GrpcTimeout{
				PerRequest: &appmeshsdk.Duration{
					Unit:  aws.String("ms"),
					Value: aws.Int64(100),
				},
				Idle: &appmeshsdk.Duration{
					Unit:  aws.String("ms"),
					Value: aws.Int64(200),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_ListenerTimeoutGrpc_To_SDK_ListenerTimeoutGrpc(tt.args.crdObj, tt.args.sdkObj, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_ListenerTimeout_To_SDK_ListenerTimeout(t *testing.T) {
	type args struct {
		crdObj *appmesh.ListenerTimeout
		sdkObj *appmeshsdk.ListenerTimeout
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.ListenerTimeout
		wantErr    error
	}{
		{
			name: "tcp timeout case",
			args: args{
				crdObj: &appmesh.ListenerTimeout{
					TCP: &appmesh.TCPTimeout{
						Idle: &appmesh.Duration{
							Unit:  "ms",
							Value: int64(200),
						},
					},
				},
				sdkObj: &appmeshsdk.ListenerTimeout{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.ListenerTimeout{
				Tcp: &appmeshsdk.TcpTimeout{
					Idle: &appmeshsdk.Duration{
						Unit:  aws.String("ms"),
						Value: aws.Int64(200),
					},
				},
			},
		},
		{
			name: "http timeout case",
			args: args{
				crdObj: &appmesh.ListenerTimeout{
					HTTP: &appmesh.HTTPTimeout{
						PerRequest: &appmesh.Duration{
							Unit:  "ms",
							Value: int64(100),
						},
						Idle: &appmesh.Duration{
							Unit:  "ms",
							Value: int64(200),
						},
					},
				},
				sdkObj: &appmeshsdk.ListenerTimeout{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.ListenerTimeout{
				Http: &appmeshsdk.HttpTimeout{
					PerRequest: &appmeshsdk.Duration{
						Unit:  aws.String("ms"),
						Value: aws.Int64(100),
					},
					Idle: &appmeshsdk.Duration{
						Unit:  aws.String("ms"),
						Value: aws.Int64(200),
					},
				},
			},
		},
		{
			name: "http2 timeout case",
			args: args{
				crdObj: &appmesh.ListenerTimeout{
					HTTP2: &appmesh.HTTPTimeout{
						PerRequest: &appmesh.Duration{
							Unit:  "ms",
							Value: int64(100),
						},
						Idle: &appmesh.Duration{
							Unit:  "ms",
							Value: int64(200),
						},
					},
				},
				sdkObj: &appmeshsdk.ListenerTimeout{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.ListenerTimeout{
				Http2: &appmeshsdk.HttpTimeout{
					PerRequest: &appmeshsdk.Duration{
						Unit:  aws.String("ms"),
						Value: aws.Int64(100),
					},
					Idle: &appmeshsdk.Duration{
						Unit:  aws.String("ms"),
						Value: aws.Int64(200),
					},
				},
			},
		},
		{
			name: "grpc timeout case",
			args: args{
				crdObj: &appmesh.ListenerTimeout{
					GRPC: &appmesh.GRPCTimeout{
						PerRequest: &appmesh.Duration{
							Unit:  "ms",
							Value: int64(100),
						},
						Idle: &appmesh.Duration{
							Unit:  "ms",
							Value: int64(200),
						},
					},
				},
				sdkObj: &appmeshsdk.ListenerTimeout{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.ListenerTimeout{
				Grpc: &appmeshsdk.GrpcTimeout{
					PerRequest: &appmeshsdk.Duration{
						Unit:  aws.String("ms"),
						Value: aws.Int64(100),
					},
					Idle: &appmeshsdk.Duration{
						Unit:  aws.String("ms"),
						Value: aws.Int64(200),
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_ListenerTimeout_To_SDK_ListenerTimeout(tt.args.crdObj, tt.args.sdkObj, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_Listener_To_SDK_Listener(t *testing.T) {
	port80 := appmesh.PortNumber(80)
	protocolHTTP := appmesh.PortProtocolHTTP
	type args struct {
		crdObj *appmesh.Listener
		sdkObj *appmeshsdk.Listener
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.Listener
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.Listener{
					PortMapping: appmesh.PortMapping{
						Port:     port80,
						Protocol: protocolHTTP,
					},
					HealthCheck: &appmesh.HealthCheckPolicy{
						HealthyThreshold:   3,
						IntervalMillis:     60,
						Path:               aws.String("/"),
						Port:               &port80,
						Protocol:           protocolHTTP,
						TimeoutMillis:      30,
						UnhealthyThreshold: 2,
					},
					TLS: &appmesh.ListenerTLS{
						Certificate: appmesh.ListenerTLSCertificate{
							ACM: &appmesh.ListenerTLSACMCertificate{
								CertificateARN: "arn-1",
							},
						},
						Mode: appmesh.ListenerTLSModeStrict,
					},
				},
				sdkObj: &appmeshsdk.Listener{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.Listener{
				PortMapping: &appmeshsdk.PortMapping{
					Port:     aws.Int64(80),
					Protocol: aws.String("http"),
				},
				HealthCheck: &appmeshsdk.HealthCheckPolicy{
					HealthyThreshold:   aws.Int64(3),
					IntervalMillis:     aws.Int64(60),
					Path:               aws.String("/"),
					Port:               aws.Int64(80),
					Protocol:           aws.String("http"),
					TimeoutMillis:      aws.Int64(30),
					UnhealthyThreshold: aws.Int64(2),
				},
				Tls: &appmeshsdk.ListenerTls{
					Certificate: &appmeshsdk.ListenerTlsCertificate{
						Acm: &appmeshsdk.ListenerTlsAcmCertificate{
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
				crdObj: &appmesh.Listener{
					PortMapping: appmesh.PortMapping{
						Port:     port80,
						Protocol: protocolHTTP,
					},
					HealthCheck: nil,
					TLS: &appmesh.ListenerTLS{
						Certificate: appmesh.ListenerTLSCertificate{
							ACM: &appmesh.ListenerTLSACMCertificate{
								CertificateARN: "arn-1",
							},
						},
						Mode: appmesh.ListenerTLSModeStrict,
					},
				},
				sdkObj: &appmeshsdk.Listener{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.Listener{
				PortMapping: &appmeshsdk.PortMapping{
					Port:     aws.Int64(80),
					Protocol: aws.String("http"),
				},
				HealthCheck: nil,
				Tls: &appmeshsdk.ListenerTls{
					Certificate: &appmeshsdk.ListenerTlsCertificate{
						Acm: &appmeshsdk.ListenerTlsAcmCertificate{
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
				crdObj: &appmesh.Listener{
					PortMapping: appmesh.PortMapping{
						Port:     port80,
						Protocol: protocolHTTP,
					},
					HealthCheck: &appmesh.HealthCheckPolicy{
						HealthyThreshold:   3,
						IntervalMillis:     60,
						Path:               aws.String("/"),
						Port:               &port80,
						Protocol:           protocolHTTP,
						TimeoutMillis:      30,
						UnhealthyThreshold: 2,
					},
					TLS: nil,
				},
				sdkObj: &appmeshsdk.Listener{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.Listener{
				PortMapping: &appmeshsdk.PortMapping{
					Port:     aws.Int64(80),
					Protocol: aws.String("http"),
				},
				HealthCheck: &appmeshsdk.HealthCheckPolicy{
					HealthyThreshold:   aws.Int64(3),
					IntervalMillis:     aws.Int64(60),
					Path:               aws.String("/"),
					Port:               aws.Int64(80),
					Protocol:           aws.String("http"),
					TimeoutMillis:      aws.Int64(30),
					UnhealthyThreshold: aws.Int64(2),
				},
				Tls: nil,
			},
		},
		{
			name: "normal case + nil HealthCheck and nil TLS",
			args: args{
				crdObj: &appmesh.Listener{
					PortMapping: appmesh.PortMapping{
						Port:     port80,
						Protocol: protocolHTTP,
					},
					HealthCheck: nil,
					TLS:         nil,
				},
				sdkObj: &appmeshsdk.Listener{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.Listener{
				PortMapping: &appmeshsdk.PortMapping{
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
			err := Convert_CRD_Listener_To_SDK_Listener(tt.args.crdObj, tt.args.sdkObj, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_AWSCloudMapInstanceAttribute_To_SDK_AWSCloudMapInstanceAttribute(t *testing.T) {
	type args struct {
		crdObj *appmesh.AWSCloudMapInstanceAttribute
		sdkObj *appmeshsdk.AwsCloudMapInstanceAttribute
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.AwsCloudMapInstanceAttribute
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.AWSCloudMapInstanceAttribute{
					Key:   "key",
					Value: "value",
				},
				sdkObj: &appmeshsdk.AwsCloudMapInstanceAttribute{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.AwsCloudMapInstanceAttribute{
				Key:   aws.String("key"),
				Value: aws.String("value"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_AWSCloudMapInstanceAttribute_To_SDK_AWSCloudMapInstanceAttribute(tt.args.crdObj, tt.args.sdkObj, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_AWSCloudMapServiceDiscovery_To_SDK_AWSCloudMapServiceDiscovery(t *testing.T) {
	type args struct {
		crdObj *appmesh.AWSCloudMapServiceDiscovery
		sdkObj *appmeshsdk.AwsCloudMapServiceDiscovery
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.AwsCloudMapServiceDiscovery
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.AWSCloudMapServiceDiscovery{
					NamespaceName: "my-ns",
					ServiceName:   "my-svc",
					Attributes: []appmesh.AWSCloudMapInstanceAttribute{
						{
							Key:   "key1",
							Value: "value1",
						},
						{
							Key:   "key2",
							Value: "value2",
						},
					},
				},
				sdkObj: &appmeshsdk.AwsCloudMapServiceDiscovery{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.AwsCloudMapServiceDiscovery{
				NamespaceName: aws.String("my-ns"),
				ServiceName:   aws.String("my-svc"),
				Attributes: []*appmeshsdk.AwsCloudMapInstanceAttribute{
					{
						Key:   aws.String("key1"),
						Value: aws.String("value1"),
					},
					{
						Key:   aws.String("key2"),
						Value: aws.String("value2"),
					},
				},
			},
		},
		{
			name: "normal case + nil attributes",
			args: args{
				crdObj: &appmesh.AWSCloudMapServiceDiscovery{
					NamespaceName: "my-ns",
					ServiceName:   "my-svc",
					Attributes:    nil,
				},
				sdkObj: &appmeshsdk.AwsCloudMapServiceDiscovery{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.AwsCloudMapServiceDiscovery{
				NamespaceName: aws.String("my-ns"),
				ServiceName:   aws.String("my-svc"),
				Attributes:    nil,
			},
		},
		{
			name: "normal case + empty attributes",
			args: args{
				crdObj: &appmesh.AWSCloudMapServiceDiscovery{
					NamespaceName: "my-ns",
					ServiceName:   "my-svc",
					Attributes:    []appmesh.AWSCloudMapInstanceAttribute{},
				},
				sdkObj: &appmeshsdk.AwsCloudMapServiceDiscovery{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.AwsCloudMapServiceDiscovery{
				NamespaceName: aws.String("my-ns"),
				ServiceName:   aws.String("my-svc"),
				Attributes:    nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_AWSCloudMapServiceDiscovery_To_SDK_AWSCloudMapServiceDiscovery(tt.args.crdObj, tt.args.sdkObj, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_DNSServiceDiscovery_To_SDK_DNSServiceDiscovery(t *testing.T) {
	type args struct {
		crdObj *appmesh.DNSServiceDiscovery
		sdkObj *appmeshsdk.DnsServiceDiscovery
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.DnsServiceDiscovery
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.DNSServiceDiscovery{
					Hostname: "www.example.com",
				},
				sdkObj: &appmeshsdk.DnsServiceDiscovery{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.DnsServiceDiscovery{
				Hostname: aws.String("www.example.com"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_DNSServiceDiscovery_To_SDK_DNSServiceDiscovery(tt.args.crdObj, tt.args.sdkObj, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_ServiceDiscovery_To_SDK_ServiceDiscovery(t *testing.T) {
	type args struct {
		crdObj *appmesh.ServiceDiscovery
		sdkObj *appmeshsdk.ServiceDiscovery
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.ServiceDiscovery
		wantErr    error
	}{
		{
			name: "AWSCloudMap discovery",
			args: args{
				crdObj: &appmesh.ServiceDiscovery{
					AWSCloudMap: &appmesh.AWSCloudMapServiceDiscovery{
						NamespaceName: "my-ns",
						ServiceName:   "my-svc",
						Attributes: []appmesh.AWSCloudMapInstanceAttribute{
							{
								Key:   "key1",
								Value: "value1",
							},
							{
								Key:   "key2",
								Value: "value2",
							},
						},
					},
				},
				sdkObj: &appmeshsdk.ServiceDiscovery{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.ServiceDiscovery{
				AwsCloudMap: &appmeshsdk.AwsCloudMapServiceDiscovery{
					NamespaceName: aws.String("my-ns"),
					ServiceName:   aws.String("my-svc"),
					Attributes: []*appmeshsdk.AwsCloudMapInstanceAttribute{
						{
							Key:   aws.String("key1"),
							Value: aws.String("value1"),
						},
						{
							Key:   aws.String("key2"),
							Value: aws.String("value2"),
						},
					},
				},
			},
		},
		{
			name: "DNS discovery",
			args: args{
				crdObj: &appmesh.ServiceDiscovery{
					DNS: &appmesh.DNSServiceDiscovery{
						Hostname: "www.example.com",
					},
				},
				sdkObj: &appmeshsdk.ServiceDiscovery{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.ServiceDiscovery{
				Dns: &appmeshsdk.DnsServiceDiscovery{
					Hostname: aws.String("www.example.com"),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_ServiceDiscovery_To_SDK_ServiceDiscovery(tt.args.crdObj, tt.args.sdkObj, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_FileAccessLog_To_SDK_FileAccessLog(t *testing.T) {
	type args struct {
		crdObj *appmesh.FileAccessLog
		sdkObj *appmeshsdk.FileAccessLog
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.FileAccessLog
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.FileAccessLog{
					Path: "/",
				},
				sdkObj: &appmeshsdk.FileAccessLog{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.FileAccessLog{
				Path: aws.String("/"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_FileAccessLog_To_SDK_FileAccessLog(tt.args.crdObj, tt.args.sdkObj, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_AccessLog_To_SDK_AccessLog(t *testing.T) {
	type args struct {
		crdObj *appmesh.AccessLog
		sdkObj *appmeshsdk.AccessLog
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.AccessLog
		wantErr    error
	}{
		{
			name: "non-nil file access log",
			args: args{
				crdObj: &appmesh.AccessLog{
					File: &appmesh.FileAccessLog{
						Path: "/",
					},
				},
				sdkObj: &appmeshsdk.AccessLog{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.AccessLog{
				File: &appmeshsdk.FileAccessLog{
					Path: aws.String("/"),
				},
			},
		},
		{
			name: "nil file access log",
			args: args{
				crdObj: &appmesh.AccessLog{
					File: nil,
				},
				sdkObj: &appmeshsdk.AccessLog{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.AccessLog{
				File: nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_AccessLog_To_SDK_AccessLog(tt.args.crdObj, tt.args.sdkObj, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_Logging_To_SDK_Logging(t *testing.T) {
	type args struct {
		crdObj *appmesh.Logging
		sdkObj *appmeshsdk.Logging
		scope  conversion.Scope
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.Logging
		wantErr    error
	}{
		{
			name: "non-nil AccessLog",
			args: args{
				crdObj: &appmesh.Logging{
					AccessLog: &appmesh.AccessLog{
						File: &appmesh.FileAccessLog{
							Path: "/",
						},
					},
				},
				sdkObj: &appmeshsdk.Logging{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.Logging{
				AccessLog: &appmeshsdk.AccessLog{
					File: &appmeshsdk.FileAccessLog{
						Path: aws.String("/"),
					},
				},
			},
		},
		{
			name: "nil AccessLog",
			args: args{
				crdObj: &appmesh.Logging{
					AccessLog: nil,
				},
				sdkObj: &appmeshsdk.Logging{},
				scope:  nil,
			},
			wantSDKObj: &appmeshsdk.Logging{
				AccessLog: nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert_CRD_Logging_To_SDK_Logging(tt.args.crdObj, tt.args.sdkObj, tt.args.scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}

func TestConvert_CRD_VirtualNodeSpec_To_SDK_VirtualNodeSpec(t *testing.T) {
	port80 := appmesh.PortNumber(80)
	port443 := appmesh.PortNumber(443)
	protocolHTTP := appmesh.PortProtocolHTTP
	protocolHTTP2 := appmesh.PortProtocolHTTP2
	type args struct {
		crdObj           *appmesh.VirtualNodeSpec
		sdkObj           *appmeshsdk.VirtualNodeSpec
		scopeConvertFunc func(src, dest interface{}, flags conversion.FieldMatchingFlags) error
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.VirtualNodeSpec
		wantErr    error
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &appmesh.VirtualNodeSpec{
					Listeners: []appmesh.Listener{
						{
							PortMapping: appmesh.PortMapping{
								Port:     port80,
								Protocol: protocolHTTP,
							},
							HealthCheck: &appmesh.HealthCheckPolicy{
								HealthyThreshold:   3,
								IntervalMillis:     60,
								Path:               aws.String("/"),
								Port:               &port80,
								Protocol:           protocolHTTP,
								TimeoutMillis:      30,
								UnhealthyThreshold: 2,
							},
							TLS: &appmesh.ListenerTLS{
								Certificate: appmesh.ListenerTLSCertificate{
									ACM: &appmesh.ListenerTLSACMCertificate{
										CertificateARN: "arn-1",
									},
								},
								Mode: appmesh.ListenerTLSModeStrict,
							},
						},
						{
							PortMapping: appmesh.PortMapping{
								Port:     port443,
								Protocol: protocolHTTP2,
							},
						},
					},
					ServiceDiscovery: &appmesh.ServiceDiscovery{
						DNS: &appmesh.DNSServiceDiscovery{
							Hostname: "www.example.com",
						},
					},
					Backends: []appmesh.Backend{
						{
							VirtualService: appmesh.VirtualServiceBackend{
								VirtualServiceRef: appmesh.VirtualServiceReference{
									Namespace: aws.String("ns-1"),
									Name:      "vs-1",
								},
								ClientPolicy: &appmesh.ClientPolicy{
									TLS: &appmesh.ClientPolicyTLS{
										Enforce: aws.Bool(true),
										Ports:   []appmesh.PortNumber{80, 443},
										Validation: appmesh.TLSValidationContext{
											Trust: appmesh.TLSValidationContextTrust{
												ACM: &appmesh.TLSValidationContextACMTrust{
													CertificateAuthorityARNs: []string{"arn-1", "arn-2"},
												},
											},
										},
									},
								},
							},
						},
						{
							VirtualService: appmesh.VirtualServiceBackend{
								VirtualServiceRef: appmesh.VirtualServiceReference{
									Namespace: aws.String("ns-2"),
									Name:      "vs-2",
								},
							},
						},
					},
					BackendDefaults: &appmesh.BackendDefaults{
						ClientPolicy: &appmesh.ClientPolicy{
							TLS: &appmesh.ClientPolicyTLS{
								Enforce: aws.Bool(true),
								Ports:   []appmesh.PortNumber{80, 443},
								Validation: appmesh.TLSValidationContext{
									Trust: appmesh.TLSValidationContextTrust{
										ACM: &appmesh.TLSValidationContextACMTrust{
											CertificateAuthorityARNs: []string{"arn-1", "arn-2"},
										},
									},
								},
							},
						},
					},
					Logging: &appmesh.Logging{
						AccessLog: &appmesh.AccessLog{
							File: &appmesh.FileAccessLog{
								Path: "/",
							},
						},
					},
					MeshRef: nil,
				},
				sdkObj: &appmeshsdk.VirtualNodeSpec{},
				scopeConvertFunc: func(src, dest interface{}, flags conversion.FieldMatchingFlags) error {
					vsRef := src.(*appmesh.VirtualServiceReference)
					vsNamePtr := dest.(*string)
					*vsNamePtr = fmt.Sprintf("%s.%s", vsRef.Name, aws.StringValue(vsRef.Namespace))
					return nil
				},
			},
			wantSDKObj: &appmeshsdk.VirtualNodeSpec{
				Listeners: []*appmeshsdk.Listener{
					{
						PortMapping: &appmeshsdk.PortMapping{
							Port:     aws.Int64(80),
							Protocol: aws.String("http"),
						},
						HealthCheck: &appmeshsdk.HealthCheckPolicy{
							HealthyThreshold:   aws.Int64(3),
							IntervalMillis:     aws.Int64(60),
							Path:               aws.String("/"),
							Port:               aws.Int64(80),
							Protocol:           aws.String("http"),
							TimeoutMillis:      aws.Int64(30),
							UnhealthyThreshold: aws.Int64(2),
						},
						Tls: &appmeshsdk.ListenerTls{
							Certificate: &appmeshsdk.ListenerTlsCertificate{
								Acm: &appmeshsdk.ListenerTlsAcmCertificate{
									CertificateArn: aws.String("arn-1"),
								},
							},
							Mode: aws.String("STRICT"),
						},
					},
					{
						PortMapping: &appmeshsdk.PortMapping{
							Port:     aws.Int64(443),
							Protocol: aws.String("http2"),
						},
					},
				},
				ServiceDiscovery: &appmeshsdk.ServiceDiscovery{
					Dns: &appmeshsdk.DnsServiceDiscovery{
						Hostname: aws.String("www.example.com"),
					},
				},
				Backends: []*appmeshsdk.Backend{
					{
						VirtualService: &appmeshsdk.VirtualServiceBackend{
							VirtualServiceName: aws.String("vs-1.ns-1"),
							ClientPolicy: &appmeshsdk.ClientPolicy{
								Tls: &appmeshsdk.ClientPolicyTls{
									Enforce: aws.Bool(true),
									Ports:   []*int64{aws.Int64(80), aws.Int64(443)},
									Validation: &appmeshsdk.TlsValidationContext{
										Trust: &appmeshsdk.TlsValidationContextTrust{
											Acm: &appmeshsdk.TlsValidationContextAcmTrust{
												CertificateAuthorityArns: []*string{aws.String("arn-1"), aws.String("arn-2")},
											},
										},
									},
								},
							},
						},
					},
					{
						VirtualService: &appmeshsdk.VirtualServiceBackend{
							VirtualServiceName: aws.String("vs-2.ns-2"),
						},
					},
				},
				BackendDefaults: &appmeshsdk.BackendDefaults{
					ClientPolicy: &appmeshsdk.ClientPolicy{
						Tls: &appmeshsdk.ClientPolicyTls{
							Enforce: aws.Bool(true),
							Ports:   []*int64{aws.Int64(80), aws.Int64(443)},
							Validation: &appmeshsdk.TlsValidationContext{
								Trust: &appmeshsdk.TlsValidationContextTrust{
									Acm: &appmeshsdk.TlsValidationContextAcmTrust{
										CertificateAuthorityArns: []*string{aws.String("arn-1"), aws.String("arn-2")},
									},
								},
							},
						},
					},
				},
				Logging: &appmeshsdk.Logging{
					AccessLog: &appmeshsdk.AccessLog{
						File: &appmeshsdk.FileAccessLog{
							Path: aws.String("/"),
						},
					},
				},
			},
		},
		{
			name: "normal case + nil listener",
			args: args{
				crdObj: &appmesh.VirtualNodeSpec{
					Listeners: nil,
					ServiceDiscovery: &appmesh.ServiceDiscovery{
						DNS: &appmesh.DNSServiceDiscovery{
							Hostname: "www.example.com",
						},
					},
					Backends: []appmesh.Backend{
						{
							VirtualService: appmesh.VirtualServiceBackend{
								VirtualServiceRef: appmesh.VirtualServiceReference{
									Namespace: aws.String("ns-1"),
									Name:      "vs-1",
								},
								ClientPolicy: &appmesh.ClientPolicy{
									TLS: &appmesh.ClientPolicyTLS{
										Enforce: aws.Bool(true),
										Ports:   []appmesh.PortNumber{80, 443},
										Validation: appmesh.TLSValidationContext{
											Trust: appmesh.TLSValidationContextTrust{
												ACM: &appmesh.TLSValidationContextACMTrust{
													CertificateAuthorityARNs: []string{"arn-1", "arn-2"},
												},
											},
										},
									},
								},
							},
						},
						{
							VirtualService: appmesh.VirtualServiceBackend{
								VirtualServiceRef: appmesh.VirtualServiceReference{
									Namespace: aws.String("ns-2"),
									Name:      "vs-2",
								},
							},
						},
					},
					BackendDefaults: &appmesh.BackendDefaults{
						ClientPolicy: &appmesh.ClientPolicy{
							TLS: &appmesh.ClientPolicyTLS{
								Enforce: aws.Bool(true),
								Ports:   []appmesh.PortNumber{80, 443},
								Validation: appmesh.TLSValidationContext{
									Trust: appmesh.TLSValidationContextTrust{
										ACM: &appmesh.TLSValidationContextACMTrust{
											CertificateAuthorityARNs: []string{"arn-1", "arn-2"},
										},
									},
								},
							},
						},
					},
					Logging: &appmesh.Logging{
						AccessLog: &appmesh.AccessLog{
							File: &appmesh.FileAccessLog{
								Path: "/",
							},
						},
					},
					MeshRef: nil,
				},
				sdkObj: &appmeshsdk.VirtualNodeSpec{},
				scopeConvertFunc: func(src, dest interface{}, flags conversion.FieldMatchingFlags) error {
					vsRef := src.(*appmesh.VirtualServiceReference)
					vsNamePtr := dest.(*string)
					*vsNamePtr = fmt.Sprintf("%s.%s", vsRef.Name, aws.StringValue(vsRef.Namespace))
					return nil
				},
			},
			wantSDKObj: &appmeshsdk.VirtualNodeSpec{
				Listeners: nil,
				ServiceDiscovery: &appmeshsdk.ServiceDiscovery{
					Dns: &appmeshsdk.DnsServiceDiscovery{
						Hostname: aws.String("www.example.com"),
					},
				},
				Backends: []*appmeshsdk.Backend{
					{
						VirtualService: &appmeshsdk.VirtualServiceBackend{
							VirtualServiceName: aws.String("vs-1.ns-1"),
							ClientPolicy: &appmeshsdk.ClientPolicy{
								Tls: &appmeshsdk.ClientPolicyTls{
									Enforce: aws.Bool(true),
									Ports:   []*int64{aws.Int64(80), aws.Int64(443)},
									Validation: &appmeshsdk.TlsValidationContext{
										Trust: &appmeshsdk.TlsValidationContextTrust{
											Acm: &appmeshsdk.TlsValidationContextAcmTrust{
												CertificateAuthorityArns: []*string{aws.String("arn-1"), aws.String("arn-2")},
											},
										},
									},
								},
							},
						},
					},
					{
						VirtualService: &appmeshsdk.VirtualServiceBackend{
							VirtualServiceName: aws.String("vs-2.ns-2"),
						},
					},
				},
				BackendDefaults: &appmeshsdk.BackendDefaults{
					ClientPolicy: &appmeshsdk.ClientPolicy{
						Tls: &appmeshsdk.ClientPolicyTls{
							Enforce: aws.Bool(true),
							Ports:   []*int64{aws.Int64(80), aws.Int64(443)},
							Validation: &appmeshsdk.TlsValidationContext{
								Trust: &appmeshsdk.TlsValidationContextTrust{
									Acm: &appmeshsdk.TlsValidationContextAcmTrust{
										CertificateAuthorityArns: []*string{aws.String("arn-1"), aws.String("arn-2")},
									},
								},
							},
						},
					},
				},
				Logging: &appmeshsdk.Logging{
					AccessLog: &appmeshsdk.AccessLog{
						File: &appmeshsdk.FileAccessLog{
							Path: aws.String("/"),
						},
					},
				},
			},
		},
		{
			name: "normal case + nil ServiceDiscovery",
			args: args{
				crdObj: &appmesh.VirtualNodeSpec{
					Listeners: []appmesh.Listener{
						{
							PortMapping: appmesh.PortMapping{
								Port:     port80,
								Protocol: protocolHTTP,
							},
							HealthCheck: &appmesh.HealthCheckPolicy{
								HealthyThreshold:   3,
								IntervalMillis:     60,
								Path:               aws.String("/"),
								Port:               &port80,
								Protocol:           protocolHTTP,
								TimeoutMillis:      30,
								UnhealthyThreshold: 2,
							},
							TLS: &appmesh.ListenerTLS{
								Certificate: appmesh.ListenerTLSCertificate{
									ACM: &appmesh.ListenerTLSACMCertificate{
										CertificateARN: "arn-1",
									},
								},
								Mode: appmesh.ListenerTLSModeStrict,
							},
						},
						{
							PortMapping: appmesh.PortMapping{
								Port:     port443,
								Protocol: protocolHTTP2,
							},
						},
					},
					ServiceDiscovery: nil,
					Backends: []appmesh.Backend{
						{
							VirtualService: appmesh.VirtualServiceBackend{
								VirtualServiceRef: appmesh.VirtualServiceReference{
									Namespace: aws.String("ns-1"),
									Name:      "vs-1",
								},
								ClientPolicy: &appmesh.ClientPolicy{
									TLS: &appmesh.ClientPolicyTLS{
										Enforce: aws.Bool(true),
										Ports:   []appmesh.PortNumber{80, 443},
										Validation: appmesh.TLSValidationContext{
											Trust: appmesh.TLSValidationContextTrust{
												ACM: &appmesh.TLSValidationContextACMTrust{
													CertificateAuthorityARNs: []string{"arn-1", "arn-2"},
												},
											},
										},
									},
								},
							},
						},
						{
							VirtualService: appmesh.VirtualServiceBackend{
								VirtualServiceRef: appmesh.VirtualServiceReference{
									Namespace: aws.String("ns-2"),
									Name:      "vs-2",
								},
							},
						},
					},
					BackendDefaults: &appmesh.BackendDefaults{
						ClientPolicy: &appmesh.ClientPolicy{
							TLS: &appmesh.ClientPolicyTLS{
								Enforce: aws.Bool(true),
								Ports:   []appmesh.PortNumber{80, 443},
								Validation: appmesh.TLSValidationContext{
									Trust: appmesh.TLSValidationContextTrust{
										ACM: &appmesh.TLSValidationContextACMTrust{
											CertificateAuthorityARNs: []string{"arn-1", "arn-2"},
										},
									},
								},
							},
						},
					},
					Logging: &appmesh.Logging{
						AccessLog: &appmesh.AccessLog{
							File: &appmesh.FileAccessLog{
								Path: "/",
							},
						},
					},
					MeshRef: nil,
				},
				sdkObj: &appmeshsdk.VirtualNodeSpec{},
				scopeConvertFunc: func(src, dest interface{}, flags conversion.FieldMatchingFlags) error {
					vsRef := src.(*appmesh.VirtualServiceReference)
					vsNamePtr := dest.(*string)
					*vsNamePtr = fmt.Sprintf("%s.%s", vsRef.Name, aws.StringValue(vsRef.Namespace))
					return nil
				},
			},
			wantSDKObj: &appmeshsdk.VirtualNodeSpec{
				Listeners: []*appmeshsdk.Listener{
					{
						PortMapping: &appmeshsdk.PortMapping{
							Port:     aws.Int64(80),
							Protocol: aws.String("http"),
						},
						HealthCheck: &appmeshsdk.HealthCheckPolicy{
							HealthyThreshold:   aws.Int64(3),
							IntervalMillis:     aws.Int64(60),
							Path:               aws.String("/"),
							Port:               aws.Int64(80),
							Protocol:           aws.String("http"),
							TimeoutMillis:      aws.Int64(30),
							UnhealthyThreshold: aws.Int64(2),
						},
						Tls: &appmeshsdk.ListenerTls{
							Certificate: &appmeshsdk.ListenerTlsCertificate{
								Acm: &appmeshsdk.ListenerTlsAcmCertificate{
									CertificateArn: aws.String("arn-1"),
								},
							},
							Mode: aws.String("STRICT"),
						},
					},
					{
						PortMapping: &appmeshsdk.PortMapping{
							Port:     aws.Int64(443),
							Protocol: aws.String("http2"),
						},
					},
				},
				ServiceDiscovery: nil,
				Backends: []*appmeshsdk.Backend{
					{
						VirtualService: &appmeshsdk.VirtualServiceBackend{
							VirtualServiceName: aws.String("vs-1.ns-1"),
							ClientPolicy: &appmeshsdk.ClientPolicy{
								Tls: &appmeshsdk.ClientPolicyTls{
									Enforce: aws.Bool(true),
									Ports:   []*int64{aws.Int64(80), aws.Int64(443)},
									Validation: &appmeshsdk.TlsValidationContext{
										Trust: &appmeshsdk.TlsValidationContextTrust{
											Acm: &appmeshsdk.TlsValidationContextAcmTrust{
												CertificateAuthorityArns: []*string{aws.String("arn-1"), aws.String("arn-2")},
											},
										},
									},
								},
							},
						},
					},
					{
						VirtualService: &appmeshsdk.VirtualServiceBackend{
							VirtualServiceName: aws.String("vs-2.ns-2"),
						},
					},
				},
				BackendDefaults: &appmeshsdk.BackendDefaults{
					ClientPolicy: &appmeshsdk.ClientPolicy{
						Tls: &appmeshsdk.ClientPolicyTls{
							Enforce: aws.Bool(true),
							Ports:   []*int64{aws.Int64(80), aws.Int64(443)},
							Validation: &appmeshsdk.TlsValidationContext{
								Trust: &appmeshsdk.TlsValidationContextTrust{
									Acm: &appmeshsdk.TlsValidationContextAcmTrust{
										CertificateAuthorityArns: []*string{aws.String("arn-1"), aws.String("arn-2")},
									},
								},
							},
						},
					},
				},
				Logging: &appmeshsdk.Logging{
					AccessLog: &appmeshsdk.AccessLog{
						File: &appmeshsdk.FileAccessLog{
							Path: aws.String("/"),
						},
					},
				},
			},
		},
		{
			name: "normal case + nil backends",
			args: args{
				crdObj: &appmesh.VirtualNodeSpec{
					Listeners: []appmesh.Listener{
						{
							PortMapping: appmesh.PortMapping{
								Port:     port80,
								Protocol: protocolHTTP,
							},
							HealthCheck: &appmesh.HealthCheckPolicy{
								HealthyThreshold:   3,
								IntervalMillis:     60,
								Path:               aws.String("/"),
								Port:               &port80,
								Protocol:           protocolHTTP,
								TimeoutMillis:      30,
								UnhealthyThreshold: 2,
							},
							TLS: &appmesh.ListenerTLS{
								Certificate: appmesh.ListenerTLSCertificate{
									ACM: &appmesh.ListenerTLSACMCertificate{
										CertificateARN: "arn-1",
									},
								},
								Mode: appmesh.ListenerTLSModeStrict,
							},
						},
						{
							PortMapping: appmesh.PortMapping{
								Port:     port443,
								Protocol: protocolHTTP2,
							},
						},
					},
					ServiceDiscovery: &appmesh.ServiceDiscovery{
						DNS: &appmesh.DNSServiceDiscovery{
							Hostname: "www.example.com",
						},
					},
					Backends: nil,
					BackendDefaults: &appmesh.BackendDefaults{
						ClientPolicy: &appmesh.ClientPolicy{
							TLS: &appmesh.ClientPolicyTLS{
								Enforce: aws.Bool(true),
								Ports:   []appmesh.PortNumber{80, 443},
								Validation: appmesh.TLSValidationContext{
									Trust: appmesh.TLSValidationContextTrust{
										ACM: &appmesh.TLSValidationContextACMTrust{
											CertificateAuthorityARNs: []string{"arn-1", "arn-2"},
										},
									},
								},
							},
						},
					},
					Logging: &appmesh.Logging{
						AccessLog: &appmesh.AccessLog{
							File: &appmesh.FileAccessLog{
								Path: "/",
							},
						},
					},
					MeshRef: nil,
				},
				sdkObj: &appmeshsdk.VirtualNodeSpec{},
				scopeConvertFunc: func(src, dest interface{}, flags conversion.FieldMatchingFlags) error {
					vsRef := src.(*appmesh.VirtualServiceReference)
					vsNamePtr := dest.(*string)
					*vsNamePtr = fmt.Sprintf("%s.%s", vsRef.Name, aws.StringValue(vsRef.Namespace))
					return nil
				},
			},
			wantSDKObj: &appmeshsdk.VirtualNodeSpec{
				Listeners: []*appmeshsdk.Listener{
					{
						PortMapping: &appmeshsdk.PortMapping{
							Port:     aws.Int64(80),
							Protocol: aws.String("http"),
						},
						HealthCheck: &appmeshsdk.HealthCheckPolicy{
							HealthyThreshold:   aws.Int64(3),
							IntervalMillis:     aws.Int64(60),
							Path:               aws.String("/"),
							Port:               aws.Int64(80),
							Protocol:           aws.String("http"),
							TimeoutMillis:      aws.Int64(30),
							UnhealthyThreshold: aws.Int64(2),
						},
						Tls: &appmeshsdk.ListenerTls{
							Certificate: &appmeshsdk.ListenerTlsCertificate{
								Acm: &appmeshsdk.ListenerTlsAcmCertificate{
									CertificateArn: aws.String("arn-1"),
								},
							},
							Mode: aws.String("STRICT"),
						},
					},
					{
						PortMapping: &appmeshsdk.PortMapping{
							Port:     aws.Int64(443),
							Protocol: aws.String("http2"),
						},
					},
				},
				ServiceDiscovery: &appmeshsdk.ServiceDiscovery{
					Dns: &appmeshsdk.DnsServiceDiscovery{
						Hostname: aws.String("www.example.com"),
					},
				},
				Backends: nil,
				BackendDefaults: &appmeshsdk.BackendDefaults{
					ClientPolicy: &appmeshsdk.ClientPolicy{
						Tls: &appmeshsdk.ClientPolicyTls{
							Enforce: aws.Bool(true),
							Ports:   []*int64{aws.Int64(80), aws.Int64(443)},
							Validation: &appmeshsdk.TlsValidationContext{
								Trust: &appmeshsdk.TlsValidationContextTrust{
									Acm: &appmeshsdk.TlsValidationContextAcmTrust{
										CertificateAuthorityArns: []*string{aws.String("arn-1"), aws.String("arn-2")},
									},
								},
							},
						},
					},
				},
				Logging: &appmeshsdk.Logging{
					AccessLog: &appmeshsdk.AccessLog{
						File: &appmeshsdk.FileAccessLog{
							Path: aws.String("/"),
						},
					},
				},
			},
		},
		{
			name: "normal case + nil BackendDefaults",
			args: args{
				crdObj: &appmesh.VirtualNodeSpec{
					Listeners: []appmesh.Listener{
						{
							PortMapping: appmesh.PortMapping{
								Port:     port80,
								Protocol: protocolHTTP,
							},
							HealthCheck: &appmesh.HealthCheckPolicy{
								HealthyThreshold:   3,
								IntervalMillis:     60,
								Path:               aws.String("/"),
								Port:               &port80,
								Protocol:           protocolHTTP,
								TimeoutMillis:      30,
								UnhealthyThreshold: 2,
							},
							TLS: &appmesh.ListenerTLS{
								Certificate: appmesh.ListenerTLSCertificate{
									ACM: &appmesh.ListenerTLSACMCertificate{
										CertificateARN: "arn-1",
									},
								},
								Mode: appmesh.ListenerTLSModeStrict,
							},
						},
						{
							PortMapping: appmesh.PortMapping{
								Port:     port443,
								Protocol: protocolHTTP2,
							},
						},
					},
					ServiceDiscovery: &appmesh.ServiceDiscovery{
						DNS: &appmesh.DNSServiceDiscovery{
							Hostname: "www.example.com",
						},
					},
					Backends: []appmesh.Backend{
						{
							VirtualService: appmesh.VirtualServiceBackend{
								VirtualServiceRef: appmesh.VirtualServiceReference{
									Namespace: aws.String("ns-1"),
									Name:      "vs-1",
								},
								ClientPolicy: &appmesh.ClientPolicy{
									TLS: &appmesh.ClientPolicyTLS{
										Enforce: aws.Bool(true),
										Ports:   []appmesh.PortNumber{80, 443},
										Validation: appmesh.TLSValidationContext{
											Trust: appmesh.TLSValidationContextTrust{
												ACM: &appmesh.TLSValidationContextACMTrust{
													CertificateAuthorityARNs: []string{"arn-1", "arn-2"},
												},
											},
										},
									},
								},
							},
						},
						{
							VirtualService: appmesh.VirtualServiceBackend{
								VirtualServiceRef: appmesh.VirtualServiceReference{
									Namespace: aws.String("ns-2"),
									Name:      "vs-2",
								},
							},
						},
					},
					BackendDefaults: nil,
					Logging: &appmesh.Logging{
						AccessLog: &appmesh.AccessLog{
							File: &appmesh.FileAccessLog{
								Path: "/",
							},
						},
					},
					MeshRef: nil,
				},
				sdkObj: &appmeshsdk.VirtualNodeSpec{},
				scopeConvertFunc: func(src, dest interface{}, flags conversion.FieldMatchingFlags) error {
					vsRef := src.(*appmesh.VirtualServiceReference)
					vsNamePtr := dest.(*string)
					*vsNamePtr = fmt.Sprintf("%s.%s", vsRef.Name, aws.StringValue(vsRef.Namespace))
					return nil
				},
			},
			wantSDKObj: &appmeshsdk.VirtualNodeSpec{
				Listeners: []*appmeshsdk.Listener{
					{
						PortMapping: &appmeshsdk.PortMapping{
							Port:     aws.Int64(80),
							Protocol: aws.String("http"),
						},
						HealthCheck: &appmeshsdk.HealthCheckPolicy{
							HealthyThreshold:   aws.Int64(3),
							IntervalMillis:     aws.Int64(60),
							Path:               aws.String("/"),
							Port:               aws.Int64(80),
							Protocol:           aws.String("http"),
							TimeoutMillis:      aws.Int64(30),
							UnhealthyThreshold: aws.Int64(2),
						},
						Tls: &appmeshsdk.ListenerTls{
							Certificate: &appmeshsdk.ListenerTlsCertificate{
								Acm: &appmeshsdk.ListenerTlsAcmCertificate{
									CertificateArn: aws.String("arn-1"),
								},
							},
							Mode: aws.String("STRICT"),
						},
					},
					{
						PortMapping: &appmeshsdk.PortMapping{
							Port:     aws.Int64(443),
							Protocol: aws.String("http2"),
						},
					},
				},
				ServiceDiscovery: &appmeshsdk.ServiceDiscovery{
					Dns: &appmeshsdk.DnsServiceDiscovery{
						Hostname: aws.String("www.example.com"),
					},
				},
				Backends: []*appmeshsdk.Backend{
					{
						VirtualService: &appmeshsdk.VirtualServiceBackend{
							VirtualServiceName: aws.String("vs-1.ns-1"),
							ClientPolicy: &appmeshsdk.ClientPolicy{
								Tls: &appmeshsdk.ClientPolicyTls{
									Enforce: aws.Bool(true),
									Ports:   []*int64{aws.Int64(80), aws.Int64(443)},
									Validation: &appmeshsdk.TlsValidationContext{
										Trust: &appmeshsdk.TlsValidationContextTrust{
											Acm: &appmeshsdk.TlsValidationContextAcmTrust{
												CertificateAuthorityArns: []*string{aws.String("arn-1"), aws.String("arn-2")},
											},
										},
									},
								},
							},
						},
					},
					{
						VirtualService: &appmeshsdk.VirtualServiceBackend{
							VirtualServiceName: aws.String("vs-2.ns-2"),
						},
					},
				},
				BackendDefaults: nil,
				Logging: &appmeshsdk.Logging{
					AccessLog: &appmeshsdk.AccessLog{
						File: &appmeshsdk.FileAccessLog{
							Path: aws.String("/"),
						},
					},
				},
			},
		},
		{
			name: "normal case + nil logging",
			args: args{
				crdObj: &appmesh.VirtualNodeSpec{
					Listeners: []appmesh.Listener{
						{
							PortMapping: appmesh.PortMapping{
								Port:     port80,
								Protocol: protocolHTTP,
							},
							HealthCheck: &appmesh.HealthCheckPolicy{
								HealthyThreshold:   3,
								IntervalMillis:     60,
								Path:               aws.String("/"),
								Port:               &port80,
								Protocol:           protocolHTTP,
								TimeoutMillis:      30,
								UnhealthyThreshold: 2,
							},
							TLS: &appmesh.ListenerTLS{
								Certificate: appmesh.ListenerTLSCertificate{
									ACM: &appmesh.ListenerTLSACMCertificate{
										CertificateARN: "arn-1",
									},
								},
								Mode: appmesh.ListenerTLSModeStrict,
							},
						},
						{
							PortMapping: appmesh.PortMapping{
								Port:     port443,
								Protocol: protocolHTTP2,
							},
						},
					},
					ServiceDiscovery: &appmesh.ServiceDiscovery{
						DNS: &appmesh.DNSServiceDiscovery{
							Hostname: "www.example.com",
						},
					},
					Backends: []appmesh.Backend{
						{
							VirtualService: appmesh.VirtualServiceBackend{
								VirtualServiceRef: appmesh.VirtualServiceReference{
									Namespace: aws.String("ns-1"),
									Name:      "vs-1",
								},
								ClientPolicy: &appmesh.ClientPolicy{
									TLS: &appmesh.ClientPolicyTLS{
										Enforce: aws.Bool(true),
										Ports:   []appmesh.PortNumber{80, 443},
										Validation: appmesh.TLSValidationContext{
											Trust: appmesh.TLSValidationContextTrust{
												ACM: &appmesh.TLSValidationContextACMTrust{
													CertificateAuthorityARNs: []string{"arn-1", "arn-2"},
												},
											},
										},
									},
								},
							},
						},
						{
							VirtualService: appmesh.VirtualServiceBackend{
								VirtualServiceRef: appmesh.VirtualServiceReference{
									Namespace: aws.String("ns-2"),
									Name:      "vs-2",
								},
							},
						},
					},
					BackendDefaults: &appmesh.BackendDefaults{
						ClientPolicy: &appmesh.ClientPolicy{
							TLS: &appmesh.ClientPolicyTLS{
								Enforce: aws.Bool(true),
								Ports:   []appmesh.PortNumber{80, 443},
								Validation: appmesh.TLSValidationContext{
									Trust: appmesh.TLSValidationContextTrust{
										ACM: &appmesh.TLSValidationContextACMTrust{
											CertificateAuthorityARNs: []string{"arn-1", "arn-2"},
										},
									},
								},
							},
						},
					},
					Logging: nil,
					MeshRef: nil,
				},
				sdkObj: &appmeshsdk.VirtualNodeSpec{},
				scopeConvertFunc: func(src, dest interface{}, flags conversion.FieldMatchingFlags) error {
					vsRef := src.(*appmesh.VirtualServiceReference)
					vsNamePtr := dest.(*string)
					*vsNamePtr = fmt.Sprintf("%s.%s", vsRef.Name, aws.StringValue(vsRef.Namespace))
					return nil
				},
			},
			wantSDKObj: &appmeshsdk.VirtualNodeSpec{
				Listeners: []*appmeshsdk.Listener{
					{
						PortMapping: &appmeshsdk.PortMapping{
							Port:     aws.Int64(80),
							Protocol: aws.String("http"),
						},
						HealthCheck: &appmeshsdk.HealthCheckPolicy{
							HealthyThreshold:   aws.Int64(3),
							IntervalMillis:     aws.Int64(60),
							Path:               aws.String("/"),
							Port:               aws.Int64(80),
							Protocol:           aws.String("http"),
							TimeoutMillis:      aws.Int64(30),
							UnhealthyThreshold: aws.Int64(2),
						},
						Tls: &appmeshsdk.ListenerTls{
							Certificate: &appmeshsdk.ListenerTlsCertificate{
								Acm: &appmeshsdk.ListenerTlsAcmCertificate{
									CertificateArn: aws.String("arn-1"),
								},
							},
							Mode: aws.String("STRICT"),
						},
					},
					{
						PortMapping: &appmeshsdk.PortMapping{
							Port:     aws.Int64(443),
							Protocol: aws.String("http2"),
						},
					},
				},
				ServiceDiscovery: &appmeshsdk.ServiceDiscovery{
					Dns: &appmeshsdk.DnsServiceDiscovery{
						Hostname: aws.String("www.example.com"),
					},
				},
				Backends: []*appmeshsdk.Backend{
					{
						VirtualService: &appmeshsdk.VirtualServiceBackend{
							VirtualServiceName: aws.String("vs-1.ns-1"),
							ClientPolicy: &appmeshsdk.ClientPolicy{
								Tls: &appmeshsdk.ClientPolicyTls{
									Enforce: aws.Bool(true),
									Ports:   []*int64{aws.Int64(80), aws.Int64(443)},
									Validation: &appmeshsdk.TlsValidationContext{
										Trust: &appmeshsdk.TlsValidationContextTrust{
											Acm: &appmeshsdk.TlsValidationContextAcmTrust{
												CertificateAuthorityArns: []*string{aws.String("arn-1"), aws.String("arn-2")},
											},
										},
									},
								},
							},
						},
					},
					{
						VirtualService: &appmeshsdk.VirtualServiceBackend{
							VirtualServiceName: aws.String("vs-2.ns-2"),
						},
					},
				},
				BackendDefaults: &appmeshsdk.BackendDefaults{
					ClientPolicy: &appmeshsdk.ClientPolicy{
						Tls: &appmeshsdk.ClientPolicyTls{
							Enforce: aws.Bool(true),
							Ports:   []*int64{aws.Int64(80), aws.Int64(443)},
							Validation: &appmeshsdk.TlsValidationContext{
								Trust: &appmeshsdk.TlsValidationContextTrust{
									Acm: &appmeshsdk.TlsValidationContextAcmTrust{
										CertificateAuthorityArns: []*string{aws.String("arn-1"), aws.String("arn-2")},
									},
								},
							},
						},
					},
				},
				Logging: nil,
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
			err := Convert_CRD_VirtualNodeSpec_To_SDK_VirtualNodeSpec(tt.args.crdObj, tt.args.sdkObj, scope)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
			}
		})
	}
}
