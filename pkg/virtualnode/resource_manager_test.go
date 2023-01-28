package virtualnode

import (
	"context"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	mock_resolver "github.com/aws/aws-app-mesh-controller-for-k8s/mocks/aws-app-mesh-controller-for-k8s/pkg/references"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/equality"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/k8s"
	"github.com/aws/aws-sdk-go/aws"
	appmeshsdk "github.com/aws/aws-sdk-go/service/appmesh"
	"github.com/go-logr/logr"
	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	testclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"testing"
)

func Test_defaultResourceManager_updateCRDVirtualNode(t *testing.T) {
	type args struct {
		vn    *appmesh.VirtualNode
		sdkVN *appmeshsdk.VirtualNodeData
	}
	tests := []struct {
		name    string
		args    args
		wantVN  *appmesh.VirtualNode
		wantErr error
	}{
		{
			name: "virtualNode needs patch both arn and condition",
			args: args{
				vn: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Name: "vn-1",
					},
					Status: appmesh.VirtualNodeStatus{},
				},
				sdkVN: &appmeshsdk.VirtualNodeData{
					Metadata: &appmeshsdk.ResourceMetadata{
						Arn: aws.String("arn-1"),
					},
					Status: &appmeshsdk.VirtualNodeStatus{
						Status: aws.String(appmeshsdk.VirtualNodeStatusCodeActive),
					},
				},
			},
			wantVN: &appmesh.VirtualNode{
				ObjectMeta: metav1.ObjectMeta{
					Name: "vn-1",
				},
				Status: appmesh.VirtualNodeStatus{
					VirtualNodeARN: aws.String("arn-1"),
					Conditions: []appmesh.VirtualNodeCondition{
						{
							Type:   appmesh.VirtualNodeActive,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
		},
		{
			name: "virtualNode needs patch condition only",
			args: args{
				vn: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Name: "vn-1",
					},
					Status: appmesh.VirtualNodeStatus{
						VirtualNodeARN: aws.String("arn-1"),
						Conditions: []appmesh.VirtualNodeCondition{
							{
								Type:   appmesh.VirtualNodeActive,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
				sdkVN: &appmeshsdk.VirtualNodeData{
					Metadata: &appmeshsdk.ResourceMetadata{
						Arn: aws.String("arn-1"),
					},
					Status: &appmeshsdk.VirtualNodeStatus{
						Status: aws.String(appmeshsdk.VirtualNodeStatusCodeInactive),
					},
				},
			},
			wantVN: &appmesh.VirtualNode{
				ObjectMeta: metav1.ObjectMeta{
					Name: "vn-1",
				},
				Status: appmesh.VirtualNodeStatus{
					VirtualNodeARN: aws.String("arn-1"),
					Conditions: []appmesh.VirtualNodeCondition{
						{
							Type:   appmesh.VirtualNodeActive,
							Status: corev1.ConditionFalse,
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			k8sSchema := runtime.NewScheme()
			clientgoscheme.AddToScheme(k8sSchema)
			appmesh.AddToScheme(k8sSchema)
			k8sClient := testclient.NewFakeClientWithScheme(k8sSchema)
			m := &defaultResourceManager{
				k8sClient: k8sClient,
				log:       logr.New(&log.NullLogSink{}),
			}

			err := k8sClient.Create(ctx, tt.args.vn.DeepCopy())
			assert.NoError(t, err)
			err = m.updateCRDVirtualNode(ctx, tt.args.vn, tt.args.sdkVN)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				gotVN := &appmesh.VirtualNode{}
				err = k8sClient.Get(ctx, k8s.NamespacedName(tt.args.vn), gotVN)
				assert.NoError(t, err)
				opts := cmp.Options{
					equality.IgnoreFakeClientPopulatedFields(),
					cmpopts.IgnoreTypes((*metav1.Time)(nil)),
				}
				assert.True(t, cmp.Equal(tt.wantVN, gotVN, opts), "diff", cmp.Diff(tt.wantVN, gotVN, opts))
			}
		})
	}
}

func Test_defaultResourceManager_isSDKVirtualNodeControlledByCRDVirtualNode(t *testing.T) {
	type fields struct {
		accountID string
	}
	type args struct {
		sdkVN *appmeshsdk.VirtualNodeData
		vn    *appmesh.VirtualNode
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name:   "sdkVN is controlled by crdVN",
			fields: fields{accountID: "222222222"},
			args: args{
				sdkVN: &appmeshsdk.VirtualNodeData{
					Metadata: &appmeshsdk.ResourceMetadata{
						ResourceOwner: aws.String("222222222"),
					},
				},
				vn: &appmesh.VirtualNode{},
			},
			want: true,
		},
		{
			name:   "sdkVN isn't controlled by crdVN",
			fields: fields{accountID: "222222222"},
			args: args{
				sdkVN: &appmeshsdk.VirtualNodeData{
					Metadata: &appmeshsdk.ResourceMetadata{
						ResourceOwner: aws.String("33333333"),
					},
				},
				vn: &appmesh.VirtualNode{},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			m := &defaultResourceManager{
				accountID: tt.fields.accountID,
				log:       logr.New(&log.NullLogSink{}),
			}
			got := m.isSDKVirtualNodeControlledByCRDVirtualNode(ctx, tt.args.sdkVN, tt.args.vn)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_defaultResourceManager_isSDKVirtualNodeOwnedByCRDVirtualNode(t *testing.T) {
	type fields struct {
		accountID string
	}
	type args struct {
		sdkVN *appmeshsdk.VirtualNodeData
		vn    *appmesh.VirtualNode
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name:   "sdkVN is owned by crdVN",
			fields: fields{accountID: "222222222"},
			args: args{
				sdkVN: &appmeshsdk.VirtualNodeData{
					Metadata: &appmeshsdk.ResourceMetadata{
						ResourceOwner: aws.String("222222222"),
					},
				},
				vn: &appmesh.VirtualNode{},
			},
			want: true,
		},
		{
			name:   "sdkVN isn't owned by crdVN",
			fields: fields{accountID: "222222222"},
			args: args{
				sdkVN: &appmeshsdk.VirtualNodeData{
					Metadata: &appmeshsdk.ResourceMetadata{
						ResourceOwner: aws.String("33333333"),
					},
				},
				vn: &appmesh.VirtualNode{},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			m := &defaultResourceManager{
				accountID: tt.fields.accountID,
				log:       logr.New(&log.NullLogSink{}),
			}
			got := m.isSDKVirtualNodeOwnedByCRDVirtualNode(ctx, tt.args.sdkVN, tt.args.vn)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_defaultResourceManager_findMeshDependency(t *testing.T) {
	type fields struct {
		ResolveMeshReference func(ctx context.Context, ref appmesh.MeshReference) (*appmesh.Mesh, error)
	}
	type args struct {
		vn *appmesh.VirtualNode
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *appmesh.Mesh
		wantErr error
	}{
		{
			name: "virtualNode with mesh",
			args: args{
				vn: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Name: "vn-1",
					},
					Spec: appmesh.VirtualNodeSpec{
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "a385048d-aba8-4235-9a11-4173764c8ab7",
						},
					},
					Status: appmesh.VirtualNodeStatus{},
				},
			},
			fields: fields{
				ResolveMeshReference: func(ctx context.Context, ref appmesh.MeshReference) (*appmesh.Mesh, error) {
					return &appmesh.Mesh{
						ObjectMeta: metav1.ObjectMeta{
							Name: "my-mesh",
							UID:  "a385048d-aba8-4235-9a11-4173764c8ab7",
						},
						Spec: appmesh.MeshSpec{
							AWSName: aws.String("my-mesh"),
						},
					}, nil
				},
			},
			want: &appmesh.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-mesh",
					UID:  "a385048d-aba8-4235-9a11-4173764c8ab7",
				},
				Spec: appmesh.MeshSpec{
					AWSName: aws.String("my-mesh"),
				},
			},
			wantErr: nil,
		},
		{
			name: "virtualNode with missing MeshRef",
			args: args{
				vn: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Name: "vn-1",
					},
					Status: appmesh.VirtualNodeStatus{},
				},
			},
			want: &appmesh.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-mesh",
					UID:  "a385048d-aba8-4235-9a11-4173764c8ab7",
				},
				Spec: appmesh.MeshSpec{
					AWSName: aws.String("my-mesh"),
				},
			},
			wantErr: errors.New("meshRef shouldn't be nil, please check webhook setup"),
		},
		{
			name: "virtualNode failed to resolve mesh",
			args: args{
				vn: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Name: "vn-1",
					},
					Spec: appmesh.VirtualNodeSpec{
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "a385048d-aba8-4235-9a11-4173764c8ab7",
						},
					},
					Status: appmesh.VirtualNodeStatus{},
				},
			},
			fields: fields{
				ResolveMeshReference: func(ctx context.Context, ref appmesh.MeshReference) (*appmesh.Mesh, error) {
					return nil, errors.New("mesh not found")
				},
			},
			want: &appmesh.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-mesh",
					UID:  "a385048d-aba8-4235-9a11-4173764c8ab7",
				},
				Spec: appmesh.MeshSpec{
					AWSName: aws.String("my-mesh"),
				},
			},
			wantErr: errors.New("failed to resolve meshRef: mesh not found"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			resolver := mock_resolver.NewMockResolver(ctrl)

			m := &defaultResourceManager{
				referencesResolver: resolver,
				log:                logr.New(&log.NullLogSink{}),
			}

			if tt.fields.ResolveMeshReference != nil {
				resolver.EXPECT().ResolveMeshReference(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveMeshReference)
			}

			got_mesh, err := m.findMeshDependency(ctx, tt.args.vn)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got_mesh)
			}
		})
	}
}

func Test_defaultResourceManager_validateMeshDependency(t *testing.T) {
	type args struct {
		mesh *appmesh.Mesh
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{
			name: "valid mesh",
			args: args{&appmesh.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-mesh",
				},
				Spec: appmesh.MeshSpec{
					AWSName: aws.String("my-mesh"),
				},
				Status: appmesh.MeshStatus{
					Conditions: []appmesh.MeshCondition{
						{
							Type:   appmesh.MeshActive,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
			},
			wantErr: nil,
		},
		{
			name: "inactive mesh",
			args: args{&appmesh.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-mesh",
				},
				Spec: appmesh.MeshSpec{
					AWSName: aws.String("my-mesh"),
				},
				Status: appmesh.MeshStatus{
					Conditions: []appmesh.MeshCondition{
						{
							Type:   appmesh.MeshActive,
							Status: corev1.ConditionFalse,
						},
					},
				},
			},
			},
			wantErr: errors.New("mesh is not active yet"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			m := &defaultResourceManager{
				log: logr.New(&log.NullLogSink{}),
			}

			err := m.validateMeshDependencies(ctx, tt.args.mesh)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_defaultResourceManager_findVirtualServiceDependencies(t *testing.T) {
	type fields struct {
		ResolveVirtualServiceReference func(ctx context.Context, obj metav1.Object, ref appmesh.VirtualServiceReference) (*appmesh.VirtualService, error)
	}
	type args struct {
		vn *appmesh.VirtualNode
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    map[types.NamespacedName]*appmesh.VirtualService
		wantErr error
	}{
		{
			name: "virtualNode with a virtualservice backend",
			args: args{
				vn: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Name: "vn-1",
					},
					Spec: appmesh.VirtualNodeSpec{
						Backends: []appmesh.Backend{
							{
								VirtualService: appmesh.VirtualServiceBackend{
									VirtualServiceRef: &appmesh.VirtualServiceReference{
										Namespace: aws.String("ns-1"),
										Name:      "vs-1",
									},
								}}},
					},
				},
			},
			fields: fields{
				ResolveVirtualServiceReference: func(ctx context.Context, obj metav1.Object, ref appmesh.VirtualServiceReference) (*appmesh.VirtualService, error) {
					return &appmesh.VirtualService{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "ns-1",
							Name:      "vs-1",
						},
						Spec: appmesh.VirtualServiceSpec{
							AWSName: aws.String("app1"),
							Provider: &appmesh.VirtualServiceProvider{
								VirtualRouter: &appmesh.VirtualRouterServiceProvider{
									VirtualRouterRef: &appmesh.VirtualRouterReference{
										Namespace: aws.String("ns-1"),
										Name:      "vr-1",
									},
								}}}}, nil
				},
			},
			want: map[types.NamespacedName]*appmesh.VirtualService{types.NamespacedName{
				Namespace: "ns-1", Name: "vs-1"}: {
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns-1",
					Name:      "vs-1",
				},
				Spec: appmesh.VirtualServiceSpec{
					AWSName: aws.String("app1"),
					Provider: &appmesh.VirtualServiceProvider{
						VirtualRouter: &appmesh.VirtualRouterServiceProvider{
							VirtualRouterRef: &appmesh.VirtualRouterReference{
								Namespace: aws.String("ns-1"),
								Name:      "vr-1",
							},
						},
					},
				}}},
			wantErr: nil,
		},
		{
			name: "virtualNode with a virtualservice backend",
			args: args{
				vn: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Name: "vn-1",
					},
					Spec: appmesh.VirtualNodeSpec{
						Backends: []appmesh.Backend{
							{
								VirtualService: appmesh.VirtualServiceBackend{
									VirtualServiceRef: &appmesh.VirtualServiceReference{
										Namespace: aws.String("ns-1"),
										Name:      "vs-1",
									},
								}}},
					},
				},
			},
			fields: fields{
				ResolveVirtualServiceReference: func(ctx context.Context, obj metav1.Object, ref appmesh.VirtualServiceReference) (*appmesh.VirtualService, error) {
					return nil, errors.New("virtual service not found")
				},
			},
			want: map[types.NamespacedName]*appmesh.VirtualService{types.NamespacedName{
				Namespace: "ns-1", Name: "vs-1"}: {
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns-1",
					Name:      "vs-1",
				},
				Spec: appmesh.VirtualServiceSpec{
					AWSName: aws.String("app1"),
					Provider: &appmesh.VirtualServiceProvider{
						VirtualRouter: &appmesh.VirtualRouterServiceProvider{
							VirtualRouterRef: &appmesh.VirtualRouterReference{
								Namespace: aws.String("ns-1"),
								Name:      "vr-1",
							},
						},
					},
				}}},
			wantErr: errors.New("failed to resolve virtualServiceRef: virtual service not found"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			resolver := mock_resolver.NewMockResolver(ctrl)

			m := &defaultResourceManager{
				referencesResolver: resolver,
				log:                logr.New(&log.NullLogSink{}),
			}

			if tt.fields.ResolveVirtualServiceReference != nil {
				resolver.EXPECT().ResolveVirtualServiceReference(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveVirtualServiceReference)
			}

			vsmap, err := m.findVirtualServiceDependencies(ctx, tt.args.vn)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, vsmap)
			}
		})
	}
}

/*
This is a bit tricky unit testing. First we created a vn having VirtualServiceRef and ClientPolicy. The VirtualServiceRef key is (ns-1/vs-1).
Later we created two entries of vsByKey having keys (ns-2/vs-2) with a VirtualRouterServiceProvider and (ns-1/vs-1) with an empty body.
The reason behind that, the BuildSDKVirtualNodeSpec function will not modify the vn spec under key (ns-1/vs-1) as it is part of the
Backends. However, VirtualRouterServiceProvider will get wiped out because it is under key (ns-2/vs-2) and will be treated as flexible backend.
*/
func Test_BuildSDKVirtualNodeSpec(t *testing.T) {
	type args struct {
		vn      *appmesh.VirtualNode
		vsByKey map[types.NamespacedName]*appmesh.VirtualService
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *appmeshsdk.ClientPolicy
		wantErr    error
	}{
		{
			name: "non nil TLS from vn backends spec having VirtualServiceRef",
			args: args{
				vn: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Name: "vn-1",
					},
					Spec: appmesh.VirtualNodeSpec{
						AWSName: aws.String("app1"),
						Backends: []appmesh.Backend{
							{
								VirtualService: appmesh.VirtualServiceBackend{
									VirtualServiceRef: &appmesh.VirtualServiceReference{
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
								}}},
					},
				},
				vsByKey: map[types.NamespacedName]*appmesh.VirtualService{
					types.NamespacedName{Namespace: "ns-2", Name: "vs-2"}: {
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "ns-2",
							Name:      "vs-2",
						},
						Spec: appmesh.VirtualServiceSpec{
							AWSName: aws.String("app2"),
							Provider: &appmesh.VirtualServiceProvider{
								VirtualRouter: &appmesh.VirtualRouterServiceProvider{
									VirtualRouterRef: &appmesh.VirtualRouterReference{
										Namespace: aws.String("ns-2"),
										Name:      "vr-2",
									},
								},
							},
						}},
					types.NamespacedName{Namespace: "ns-1", Name: "vs-1"}: {},
				},
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
			wantErr: nil,
		},
		{
			name: "non nil TLS from vn backends spec having VirtualServiceARN instead of VirtualServiceRef",
			args: args{
				vn: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Name: "vn-1",
					},
					Spec: appmesh.VirtualNodeSpec{
						AWSName: aws.String("app1"),
						Backends: []appmesh.Backend{
							{
								VirtualService: appmesh.VirtualServiceBackend{
									VirtualServiceARN: aws.String("arn:aws:appmesh:us-west-2:233846545377:mesh/howto-k8s-http2/virtualService/color.howto-k8s-http2.svc.cluster.local"),
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
								}}},
					},
				},
				vsByKey: map[types.NamespacedName]*appmesh.VirtualService{
					types.NamespacedName{Namespace: "ns-2", Name: "vs-2"}: {
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "ns-2",
							Name:      "vs-2",
						},
						Spec: appmesh.VirtualServiceSpec{
							AWSName: aws.String("app2"),
							Provider: &appmesh.VirtualServiceProvider{
								VirtualRouter: &appmesh.VirtualRouterServiceProvider{
									VirtualRouterRef: &appmesh.VirtualRouterReference{
										Namespace: aws.String("ns-2"),
										Name:      "vr-2",
									},
								},
							},
						}},
				},
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
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sdkVnSpec, err := BuildSDKVirtualNodeSpec(tt.args.vn, tt.args.vsByKey)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSDKObj, sdkVnSpec.Backends[0].VirtualService.ClientPolicy)
				assert.Nil(t, sdkVnSpec.Backends[1].VirtualService.ClientPolicy)
			}
		})
	}
}
