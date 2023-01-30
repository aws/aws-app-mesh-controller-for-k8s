package gatewayroute

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

func Test_defaultResourceManager_updateCRDGatewayRoute(t *testing.T) {
	type args struct {
		gr    *appmesh.GatewayRoute
		sdkGR *appmeshsdk.GatewayRouteData
	}
	tests := []struct {
		name    string
		args    args
		wantGR  *appmesh.GatewayRoute
		wantErr error
	}{
		{
			name: "gatewayRoute needs to patch both arn and condition",
			args: args{
				gr: &appmesh.GatewayRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name: "gr-1",
					},
					Status: appmesh.GatewayRouteStatus{},
				},
				sdkGR: &appmeshsdk.GatewayRouteData{
					Metadata: &appmeshsdk.ResourceMetadata{
						Arn: aws.String("arn-1"),
					},
					Status: &appmeshsdk.GatewayRouteStatus{
						Status: aws.String(appmeshsdk.GatewayRouteStatusCodeActive),
					},
				},
			},
			wantGR: &appmesh.GatewayRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name: "gr-1",
				},
				Status: appmesh.GatewayRouteStatus{
					GatewayRouteARN: aws.String("arn-1"),
					Conditions: []appmesh.GatewayRouteCondition{
						{
							Type:   appmesh.GatewayRouteActive,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
		},
		{
			name: "gatewayRoute needs to patch condition only",
			args: args{
				gr: &appmesh.GatewayRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name: "gr-1",
					},
					Status: appmesh.GatewayRouteStatus{
						GatewayRouteARN: aws.String("arn-1"),
						Conditions: []appmesh.GatewayRouteCondition{
							{
								Type:   appmesh.GatewayRouteActive,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
				sdkGR: &appmeshsdk.GatewayRouteData{
					Metadata: &appmeshsdk.ResourceMetadata{
						Arn: aws.String("arn-1"),
					},
					Status: &appmeshsdk.GatewayRouteStatus{
						Status: aws.String(appmeshsdk.GatewayRouteStatusCodeInactive),
					},
				},
			},
			wantGR: &appmesh.GatewayRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name: "gr-1",
				},
				Status: appmesh.GatewayRouteStatus{
					GatewayRouteARN: aws.String("arn-1"),
					Conditions: []appmesh.GatewayRouteCondition{
						{
							Type:   appmesh.GatewayRouteActive,
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

			err := k8sClient.Create(ctx, tt.args.gr.DeepCopy())
			assert.NoError(t, err)
			err = m.updateCRDGatewayRoute(ctx, tt.args.gr, tt.args.sdkGR)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				gotGR := &appmesh.GatewayRoute{}
				err = k8sClient.Get(ctx, k8s.NamespacedName(tt.args.gr), gotGR)
				assert.NoError(t, err)
				opts := cmp.Options{
					equality.IgnoreFakeClientPopulatedFields(),
					cmpopts.IgnoreTypes((*metav1.Time)(nil)),
				}
				assert.True(t, cmp.Equal(tt.wantGR, gotGR, opts), "diff", cmp.Diff(tt.wantGR, gotGR, opts))
			}
		})
	}
}

func Test_defaultResourceManager_isSDKGatewayRouteControlledByCRDGatewayRoute(t *testing.T) {
	type fields struct {
		accountID string
	}
	type args struct {
		sdkGR *appmeshsdk.GatewayRouteData
		gr    *appmesh.GatewayRoute
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name:   "sdkGR is controlled by crdGR",
			fields: fields{accountID: "222222222"},
			args: args{
				sdkGR: &appmeshsdk.GatewayRouteData{
					Metadata: &appmeshsdk.ResourceMetadata{
						ResourceOwner: aws.String("222222222"),
					},
				},
				gr: &appmesh.GatewayRoute{},
			},
			want: true,
		},
		{
			name:   "sdkGR isn't controlled by crdGR",
			fields: fields{accountID: "222222222"},
			args: args{
				sdkGR: &appmeshsdk.GatewayRouteData{
					Metadata: &appmeshsdk.ResourceMetadata{
						ResourceOwner: aws.String("33333333"),
					},
				},
				gr: &appmesh.GatewayRoute{},
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
			got := m.isSDKGatewayRouteControlledByCRDGatewayRoute(ctx, tt.args.sdkGR, tt.args.gr)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_defaultResourceManager_isSDKGatewayRouteOwnedByCRDGatewayRoute(t *testing.T) {
	type fields struct {
		accountID string
	}
	type args struct {
		sdkGR *appmeshsdk.GatewayRouteData
		gr    *appmesh.GatewayRoute
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name:   "sdkGR is owned by crdGR",
			fields: fields{accountID: "222222222"},
			args: args{
				sdkGR: &appmeshsdk.GatewayRouteData{
					Metadata: &appmeshsdk.ResourceMetadata{
						ResourceOwner: aws.String("222222222"),
					},
				},
				gr: &appmesh.GatewayRoute{},
			},
			want: true,
		},
		{
			name:   "sdkGR isn't owned by crdGR",
			fields: fields{accountID: "222222222"},
			args: args{
				sdkGR: &appmeshsdk.GatewayRouteData{
					Metadata: &appmeshsdk.ResourceMetadata{
						ResourceOwner: aws.String("33333333"),
					},
				},
				gr: &appmesh.GatewayRoute{},
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
			got := m.isSDKGatewayRouteOwnedByCRDGatewayRoute(ctx, tt.args.sdkGR, tt.args.gr)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_defaultResourceManager_findMeshDependency(t *testing.T) {
	type fields struct {
		ResolveMeshReference func(ctx context.Context, ref appmesh.MeshReference) (*appmesh.Mesh, error)
	}
	type args struct {
		gr *appmesh.GatewayRoute
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *appmesh.Mesh
		wantErr error
	}{
		{
			name: "gatewayRoute with mesh",
			args: args{
				gr: &appmesh.GatewayRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name: "gr-1",
					},
					Spec: appmesh.GatewayRouteSpec{
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "a385048d-aba8-4235-9a11-4173764c8ab7",
						},
					},
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
			name: "gatewayRoute with missing MeshRef",
			args: args{
				gr: &appmesh.GatewayRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name: "gr-1",
					},
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
			name: "gatewayRoute failed to resolve mesh",
			args: args{
				gr: &appmesh.GatewayRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name: "gr-1",
					},
					Spec: appmesh.GatewayRouteSpec{
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "a385048d-aba8-4235-9a11-4173764c8ab7",
						},
					},
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

			got_mesh, err := m.findMeshDependency(ctx, tt.args.gr)
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

			err := m.validateMeshDependency(ctx, tt.args.mesh)
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
		gr *appmesh.GatewayRoute
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    map[types.NamespacedName]*appmesh.VirtualService
		wantErr error
	}{
		{
			name: "gatewayRoute with a virtualservice backend",
			args: args{
				gr: &appmesh.GatewayRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name: "gr-1",
					},
					Spec: appmesh.GatewayRouteSpec{
						HTTPRoute: &appmesh.HTTPGatewayRoute{
							Match: appmesh.HTTPGatewayRouteMatch{
								Prefix: aws.String("prefix"),
							},
							Action: appmesh.HTTPGatewayRouteAction{
								Target: appmesh.GatewayRouteTarget{
									VirtualService: appmesh.GatewayRouteVirtualService{
										VirtualServiceRef: &appmesh.VirtualServiceReference{
											Namespace: aws.String("ns-1"),
											Name:      "vs-1",
										},
									},
								},
							},
						},
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
				Namespace: "ns-1", Name: "vs-1"}: &appmesh.VirtualService{
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

			vsmap, err := m.findVirtualServiceDependencies(ctx, tt.args.gr)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, vsmap)
			}
		})
	}
}
