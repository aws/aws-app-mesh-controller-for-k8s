package virtualservice

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

func Test_defaultResourceManager_updateCRDVirtualService(t *testing.T) {
	type args struct {
		vs    *appmesh.VirtualService
		sdkVS *appmeshsdk.VirtualServiceData
	}
	tests := []struct {
		name    string
		args    args
		wantVS  *appmesh.VirtualService
		wantErr error
	}{
		{
			name: "VirtualService needs patch both arn and condition",
			args: args{
				vs: &appmesh.VirtualService{
					ObjectMeta: metav1.ObjectMeta{
						Name: "vs-1",
					},
					Status: appmesh.VirtualServiceStatus{},
				},
				sdkVS: &appmeshsdk.VirtualServiceData{
					Metadata: &appmeshsdk.ResourceMetadata{
						Arn: aws.String("arn-1"),
					},
					Status: &appmeshsdk.VirtualServiceStatus{
						Status: aws.String(appmeshsdk.VirtualServiceStatusCodeActive),
					},
				},
			},
			wantVS: &appmesh.VirtualService{
				ObjectMeta: metav1.ObjectMeta{
					Name: "vs-1",
				},
				Status: appmesh.VirtualServiceStatus{
					VirtualServiceARN: aws.String("arn-1"),
					Conditions: []appmesh.VirtualServiceCondition{
						{
							Type:   appmesh.VirtualServiceActive,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
		},
		{
			name: "VirtualService needs patch condition only",
			args: args{
				vs: &appmesh.VirtualService{
					ObjectMeta: metav1.ObjectMeta{
						Name: "vs-1",
					},
					Status: appmesh.VirtualServiceStatus{
						VirtualServiceARN: aws.String("arn-1"),
						Conditions: []appmesh.VirtualServiceCondition{
							{
								Type:   appmesh.VirtualServiceActive,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
				sdkVS: &appmeshsdk.VirtualServiceData{
					Metadata: &appmeshsdk.ResourceMetadata{
						Arn: aws.String("arn-1"),
					},
					Status: &appmeshsdk.VirtualServiceStatus{
						Status: aws.String(appmeshsdk.VirtualNodeStatusCodeInactive),
					},
				},
			},
			wantVS: &appmesh.VirtualService{
				ObjectMeta: metav1.ObjectMeta{
					Name: "vs-1",
				},
				Status: appmesh.VirtualServiceStatus{
					VirtualServiceARN: aws.String("arn-1"),
					Conditions: []appmesh.VirtualServiceCondition{
						{
							Type:   appmesh.VirtualServiceActive,
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

			err := k8sClient.Create(ctx, tt.args.vs.DeepCopy())
			assert.NoError(t, err)
			err = m.updateCRDVirtualService(ctx, tt.args.vs, tt.args.sdkVS)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				gotVS := &appmesh.VirtualService{}
				err = k8sClient.Get(ctx, k8s.NamespacedName(tt.args.vs), gotVS)
				assert.NoError(t, err)
				opts := cmp.Options{
					equality.IgnoreFakeClientPopulatedFields(),
					cmpopts.IgnoreTypes((*metav1.Time)(nil)),
				}
				assert.True(t, cmp.Equal(tt.wantVS, gotVS, opts), "diff", cmp.Diff(tt.wantVS, gotVS, opts))
			}
		})
	}
}

func Test_defaultResourceManager_isSDKVirtualServiceControlledByCRDVirtualService(t *testing.T) {
	type fields struct {
		accountID string
	}
	type args struct {
		sdkVS *appmeshsdk.VirtualServiceData
		vs    *appmesh.VirtualService
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name:   "sdkVS is controlled by vs",
			fields: fields{accountID: "222222222"},
			args: args{
				sdkVS: &appmeshsdk.VirtualServiceData{
					Metadata: &appmeshsdk.ResourceMetadata{
						ResourceOwner: aws.String("222222222"),
					},
				},
				vs: &appmesh.VirtualService{},
			},
			want: true,
		},
		{
			name:   "sdkVS isn't controlled by vs",
			fields: fields{accountID: "222222222"},
			args: args{
				sdkVS: &appmeshsdk.VirtualServiceData{
					Metadata: &appmeshsdk.ResourceMetadata{
						ResourceOwner: aws.String("33333333"),
					},
				},
				vs: &appmesh.VirtualService{},
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
			got := m.isSDKVirtualServiceControlledByCRDVirtualService(ctx, tt.args.sdkVS, tt.args.vs)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_defaultResourceManager_isSDKVirtualServiceOwnedByCRDVirtualService(t *testing.T) {
	type fields struct {
		accountID string
	}
	type args struct {
		sdkVS *appmeshsdk.VirtualServiceData
		vs    *appmesh.VirtualService
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name:   "sdkVS is controlled by vs",
			fields: fields{accountID: "222222222"},
			args: args{
				sdkVS: &appmeshsdk.VirtualServiceData{
					Metadata: &appmeshsdk.ResourceMetadata{
						ResourceOwner: aws.String("222222222"),
					},
				},
				vs: &appmesh.VirtualService{},
			},
			want: true,
		},
		{
			name:   "sdkVS isn't controlled by vs",
			fields: fields{accountID: "222222222"},
			args: args{
				sdkVS: &appmeshsdk.VirtualServiceData{
					Metadata: &appmeshsdk.ResourceMetadata{
						ResourceOwner: aws.String("33333333"),
					},
				},
				vs: &appmesh.VirtualService{},
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
			got := m.isSDKVirtualServiceControlledByCRDVirtualService(ctx, tt.args.sdkVS, tt.args.vs)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_BuildSDKVirtualServiceSpec(t *testing.T) {
	type args struct {
		vs      *appmesh.VirtualService
		vnByKey map[types.NamespacedName]*appmesh.VirtualNode
		vrByKey map[types.NamespacedName]*appmesh.VirtualRouter
	}
	tests := []struct {
		name    string
		args    args
		want    *appmeshsdk.VirtualServiceSpec
		wantErr error
	}{
		{
			name: "VirtualNode provider - same namespace",
			args: args{
				vs: &appmesh.VirtualService{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "my-ns",
						Name:      "vs",
					},
					Spec: appmesh.VirtualServiceSpec{
						Provider: &appmesh.VirtualServiceProvider{
							VirtualNode: &appmesh.VirtualNodeServiceProvider{
								VirtualNodeRef: &appmesh.VirtualNodeReference{
									Name: "vn",
								},
							},
						},
					},
				},
				vnByKey: map[types.NamespacedName]*appmesh.VirtualNode{
					types.NamespacedName{Namespace: "my-ns", Name: "vn"}: &appmesh.VirtualNode{
						Spec: appmesh.VirtualNodeSpec{
							AWSName: aws.String("vn_my-ns"),
						},
					},
				},
			},
			want: &appmeshsdk.VirtualServiceSpec{
				Provider: &appmeshsdk.VirtualServiceProvider{
					VirtualNode: &appmeshsdk.VirtualNodeServiceProvider{
						VirtualNodeName: aws.String("vn_my-ns"),
					},
				},
			},
		},
		{
			name: "VirtualNode provider - different namespace",
			args: args{
				vs: &appmesh.VirtualService{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "my-ns",
						Name:      "vs",
					},
					Spec: appmesh.VirtualServiceSpec{
						Provider: &appmesh.VirtualServiceProvider{
							VirtualNode: &appmesh.VirtualNodeServiceProvider{
								VirtualNodeRef: &appmesh.VirtualNodeReference{
									Namespace: aws.String("my-other-ns"),
									Name:      "vn",
								},
							},
						},
					},
				},
				vnByKey: map[types.NamespacedName]*appmesh.VirtualNode{
					types.NamespacedName{Namespace: "my-other-ns", Name: "vn"}: &appmesh.VirtualNode{
						Spec: appmesh.VirtualNodeSpec{
							AWSName: aws.String("vn_my-other-ns"),
						},
					},
				},
			},
			want: &appmeshsdk.VirtualServiceSpec{
				Provider: &appmeshsdk.VirtualServiceProvider{
					VirtualNode: &appmeshsdk.VirtualNodeServiceProvider{
						VirtualNodeName: aws.String("vn_my-other-ns"),
					},
				},
			},
		},
		{
			name: "VirtualRouter provider - same namespace",
			args: args{
				vs: &appmesh.VirtualService{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "my-ns",
						Name:      "vs",
					},
					Spec: appmesh.VirtualServiceSpec{
						Provider: &appmesh.VirtualServiceProvider{
							VirtualRouter: &appmesh.VirtualRouterServiceProvider{
								VirtualRouterRef: &appmesh.VirtualRouterReference{
									Name: "vr",
								},
							},
						},
					},
				},
				vrByKey: map[types.NamespacedName]*appmesh.VirtualRouter{
					types.NamespacedName{Namespace: "my-ns", Name: "vr"}: &appmesh.VirtualRouter{
						Spec: appmesh.VirtualRouterSpec{
							AWSName: aws.String("vr_my-ns"),
						},
					},
				},
			},
			want: &appmeshsdk.VirtualServiceSpec{
				Provider: &appmeshsdk.VirtualServiceProvider{
					VirtualRouter: &appmeshsdk.VirtualRouterServiceProvider{
						VirtualRouterName: aws.String("vr_my-ns"),
					},
				},
			},
		},
		{
			name: "VirtualRouter provider - different namespace",
			args: args{
				vs: &appmesh.VirtualService{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "my-ns",
						Name:      "vs",
					},
					Spec: appmesh.VirtualServiceSpec{
						Provider: &appmesh.VirtualServiceProvider{
							VirtualRouter: &appmesh.VirtualRouterServiceProvider{
								VirtualRouterRef: &appmesh.VirtualRouterReference{
									Namespace: aws.String("my-other-ns"),
									Name:      "vr",
								},
							},
						},
					},
				},
				vrByKey: map[types.NamespacedName]*appmesh.VirtualRouter{
					types.NamespacedName{Namespace: "my-other-ns", Name: "vr"}: &appmesh.VirtualRouter{
						Spec: appmesh.VirtualRouterSpec{
							AWSName: aws.String("vr_my-other-ns"),
						},
					},
				},
			},
			want: &appmeshsdk.VirtualServiceSpec{
				Provider: &appmeshsdk.VirtualServiceProvider{
					VirtualRouter: &appmeshsdk.VirtualRouterServiceProvider{
						VirtualRouterName: aws.String("vr_my-other-ns"),
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildSDKVirtualServiceSpec(tt.args.vs, tt.args.vnByKey, tt.args.vrByKey)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func Test_defaultResourceManager_findMeshDependency(t *testing.T) {
	type fields struct {
		ResolveMeshReference func(ctx context.Context, ref appmesh.MeshReference) (*appmesh.Mesh, error)
	}
	type args struct {
		vs *appmesh.VirtualService
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *appmesh.Mesh
		wantErr error
	}{
		{
			name: "virtualService with mesh",
			args: args{
				vs: &appmesh.VirtualService{
					ObjectMeta: metav1.ObjectMeta{
						Name: "vs-1",
					},
					Spec: appmesh.VirtualServiceSpec{
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "a385048d-aba8-4235-9a11-4173764c8ab7",
						},
					},
					Status: appmesh.VirtualServiceStatus{},
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
			name: "virtualService with missing MeshRef",
			args: args{
				vs: &appmesh.VirtualService{
					ObjectMeta: metav1.ObjectMeta{
						Name: "vs-1",
					},
					Status: appmesh.VirtualServiceStatus{},
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
			name: "virtualService failed to resolve mesh",
			args: args{
				vs: &appmesh.VirtualService{
					ObjectMeta: metav1.ObjectMeta{
						Name: "vs-1",
					},
					Spec: appmesh.VirtualServiceSpec{
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "a385048d-aba8-4235-9a11-4173764c8ab7",
						},
					},
					Status: appmesh.VirtualServiceStatus{},
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

			got_mesh, err := m.findMeshDependency(ctx, tt.args.vs)
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

func Test_defaultResourceManager_findVirtualNodeDependencies(t *testing.T) {
	type fields struct {
		ResolveVirtualNodeReference func(ctx context.Context, obj metav1.Object, ref appmesh.VirtualNodeReference) (*appmesh.VirtualNode, error)
	}
	type args struct {
		vs *appmesh.VirtualService
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    map[types.NamespacedName]*appmesh.VirtualNode
		wantErr error
	}{
		{
			name: "virtualService with a virtualnode backend",
			args: args{
				vs: &appmesh.VirtualService{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns-1",
						Name:      "vs-1",
					},
					Spec: appmesh.VirtualServiceSpec{
						AWSName: aws.String("app1"),
						Provider: &appmesh.VirtualServiceProvider{
							VirtualNode: &appmesh.VirtualNodeServiceProvider{
								VirtualNodeRef: &appmesh.VirtualNodeReference{
									Namespace: aws.String("ns-1"),
									Name:      "vn-1",
								},
							},
						}}}},
			fields: fields{
				ResolveVirtualNodeReference: func(ctx context.Context, obj metav1.Object, ref appmesh.VirtualNodeReference) (*appmesh.VirtualNode, error) {
					return &appmesh.VirtualNode{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "vn-1",
							Namespace: "ns-1",
						},
					}, nil
				},
			},
			want: map[types.NamespacedName]*appmesh.VirtualNode{types.NamespacedName{
				Namespace: "ns-1", Name: "vn-1"}: &appmesh.VirtualNode{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vn-1",
					Namespace: "ns-1",
				},
			}},
			wantErr: nil,
		},
		{
			name: "virtualService failed virtualnode reference",
			args: args{
				vs: &appmesh.VirtualService{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns-1",
						Name:      "vs-1",
					},
					Spec: appmesh.VirtualServiceSpec{
						AWSName: aws.String("app1"),
						Provider: &appmesh.VirtualServiceProvider{
							VirtualNode: &appmesh.VirtualNodeServiceProvider{
								VirtualNodeRef: &appmesh.VirtualNodeReference{
									Namespace: aws.String("ns-1"),
									Name:      "vn-1",
								},
							},
						}}}},
			fields: fields{
				ResolveVirtualNodeReference: func(ctx context.Context, obj metav1.Object, ref appmesh.VirtualNodeReference) (*appmesh.VirtualNode, error) {
					return nil, errors.New("virtual node not found")
				},
			},
			want: map[types.NamespacedName]*appmesh.VirtualNode{types.NamespacedName{
				Namespace: "ns-1", Name: "vn-1"}: &appmesh.VirtualNode{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vn-1",
					Namespace: "ns-1",
				},
			}},
			wantErr: errors.New("failed to resolve virtualNodeRef: virtual node not found"),
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

			if tt.fields.ResolveVirtualNodeReference != nil {
				resolver.EXPECT().ResolveVirtualNodeReference(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveVirtualNodeReference)
			}

			vnmap, err := m.findVirtualNodeDependencies(ctx, tt.args.vs)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, vnmap)
			}
		})
	}
}

func Test_defaultResourceManager_findVirtualRouterDependencies(t *testing.T) {
	type fields struct {
		ResolveVirtualRouterReference func(ctx context.Context, obj metav1.Object, ref appmesh.VirtualRouterReference) (*appmesh.VirtualRouter, error)
	}
	type args struct {
		vs *appmesh.VirtualService
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    map[types.NamespacedName]*appmesh.VirtualRouter
		wantErr error
	}{
		{
			name: "virtualService with a virtualrouter backend",
			args: args{
				vs: &appmesh.VirtualService{
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
						}}}},
			fields: fields{
				ResolveVirtualRouterReference: func(ctx context.Context, obj metav1.Object, ref appmesh.VirtualRouterReference) (*appmesh.VirtualRouter, error) {
					return &appmesh.VirtualRouter{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "vr-1",
							Namespace: "ns-1",
						}, Spec: appmesh.VirtualRouterSpec{
							Listeners: []appmesh.VirtualRouterListener{
								{
									PortMapping: appmesh.PortMapping{
										Port:     80,
										Protocol: "http",
									},
								},
								{
									PortMapping: appmesh.PortMapping{
										Port:     443,
										Protocol: "http",
									},
								},
							},
						},
					}, nil
				},
			},
			want: map[types.NamespacedName]*appmesh.VirtualRouter{types.NamespacedName{
				Namespace: "ns-1", Name: "vr-1"}: &appmesh.VirtualRouter{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vr-1",
					Namespace: "ns-1",
				},
				Spec: appmesh.VirtualRouterSpec{
					Listeners: []appmesh.VirtualRouterListener{
						{
							PortMapping: appmesh.PortMapping{
								Port:     80,
								Protocol: "http",
							},
						},
						{
							PortMapping: appmesh.PortMapping{
								Port:     443,
								Protocol: "http",
							},
						},
					}},
			}},
			wantErr: nil,
		},
		{
			name: "virtualService with a virtualrouter backend",
			args: args{
				vs: &appmesh.VirtualService{
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
						}}}},
			fields: fields{
				ResolveVirtualRouterReference: func(ctx context.Context, obj metav1.Object, ref appmesh.VirtualRouterReference) (*appmesh.VirtualRouter, error) {
					return nil, errors.New("virtual router not found")
				}},
			want: map[types.NamespacedName]*appmesh.VirtualRouter{types.NamespacedName{
				Namespace: "ns-1", Name: "vr-1"}: &appmesh.VirtualRouter{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vr-1",
					Namespace: "ns-1",
				},
				Spec: appmesh.VirtualRouterSpec{
					Listeners: []appmesh.VirtualRouterListener{
						{
							PortMapping: appmesh.PortMapping{
								Port:     80,
								Protocol: "http",
							},
						},
						{
							PortMapping: appmesh.PortMapping{
								Port:     443,
								Protocol: "http",
							},
						},
					}},
			}},
			wantErr: errors.New("failed to resolve virtualRouterRef: virtual router not found"),
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

			if tt.fields.ResolveVirtualRouterReference != nil {
				resolver.EXPECT().ResolveVirtualRouterReference(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveVirtualRouterReference)
			}

			vrmap, err := m.findVirtualRouterDependencies(ctx, tt.args.vs)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, vrmap)
			}
		})
	}
}
