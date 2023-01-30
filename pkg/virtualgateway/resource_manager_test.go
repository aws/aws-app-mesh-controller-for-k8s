package virtualgateway

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
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	testclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"testing"
)

func Test_defaultResourceManager_updateCRDVirtualGateway(t *testing.T) {
	type args struct {
		vg    *appmesh.VirtualGateway
		sdkVG *appmeshsdk.VirtualGatewayData
	}
	tests := []struct {
		name    string
		args    args
		wantVG  *appmesh.VirtualGateway
		wantErr error
	}{
		{
			name: "virtualGateway needs to patch both arn and condition",
			args: args{
				vg: &appmesh.VirtualGateway{
					ObjectMeta: metav1.ObjectMeta{
						Name: "vg-1",
					},
					Status: appmesh.VirtualGatewayStatus{},
				},
				sdkVG: &appmeshsdk.VirtualGatewayData{
					Metadata: &appmeshsdk.ResourceMetadata{
						Arn: aws.String("arn-1"),
					},
					Status: &appmeshsdk.VirtualGatewayStatus{
						Status: aws.String(appmeshsdk.VirtualGatewayStatusCodeActive),
					},
				},
			},
			wantVG: &appmesh.VirtualGateway{
				ObjectMeta: metav1.ObjectMeta{
					Name: "vg-1",
				},
				Status: appmesh.VirtualGatewayStatus{
					VirtualGatewayARN: aws.String("arn-1"),
					Conditions: []appmesh.VirtualGatewayCondition{
						{
							Type:   appmesh.VirtualGatewayActive,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
		},
		{
			name: "virtualGateway needs to patch condition only",
			args: args{
				vg: &appmesh.VirtualGateway{
					ObjectMeta: metav1.ObjectMeta{
						Name: "vg-1",
					},
					Status: appmesh.VirtualGatewayStatus{
						VirtualGatewayARN: aws.String("arn-1"),
						Conditions: []appmesh.VirtualGatewayCondition{
							{
								Type:   appmesh.VirtualGatewayActive,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
				sdkVG: &appmeshsdk.VirtualGatewayData{
					Metadata: &appmeshsdk.ResourceMetadata{
						Arn: aws.String("arn-1"),
					},
					Status: &appmeshsdk.VirtualGatewayStatus{
						Status: aws.String(appmeshsdk.VirtualGatewayStatusCodeInactive),
					},
				},
			},
			wantVG: &appmesh.VirtualGateway{
				ObjectMeta: metav1.ObjectMeta{
					Name: "vg-1",
				},
				Status: appmesh.VirtualGatewayStatus{
					VirtualGatewayARN: aws.String("arn-1"),
					Conditions: []appmesh.VirtualGatewayCondition{
						{
							Type:   appmesh.VirtualGatewayActive,
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

			err := k8sClient.Create(ctx, tt.args.vg.DeepCopy())
			assert.NoError(t, err)
			err = m.updateCRDVirtualGateway(ctx, tt.args.vg, tt.args.sdkVG)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				gotVG := &appmesh.VirtualGateway{}
				err = k8sClient.Get(ctx, k8s.NamespacedName(tt.args.vg), gotVG)
				assert.NoError(t, err)
				opts := cmp.Options{
					equality.IgnoreFakeClientPopulatedFields(),
					cmpopts.IgnoreTypes((*metav1.Time)(nil)),
				}
				assert.True(t, cmp.Equal(tt.wantVG, gotVG, opts), "diff", cmp.Diff(tt.wantVG, gotVG, opts))
			}
		})
	}
}

func Test_defaultResourceManager_isSDKVirtualGatewayControlledByCRDVirtualGateway(t *testing.T) {
	type fields struct {
		accountID string
	}
	type args struct {
		sdkVG *appmeshsdk.VirtualGatewayData
		vg    *appmesh.VirtualGateway
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name:   "sdkVG is controlled by crdVG",
			fields: fields{accountID: "222222222"},
			args: args{
				sdkVG: &appmeshsdk.VirtualGatewayData{
					Metadata: &appmeshsdk.ResourceMetadata{
						ResourceOwner: aws.String("222222222"),
					},
				},
				vg: &appmesh.VirtualGateway{},
			},
			want: true,
		},
		{
			name:   "sdkVG isn't controlled by crdVG",
			fields: fields{accountID: "222222222"},
			args: args{
				sdkVG: &appmeshsdk.VirtualGatewayData{
					Metadata: &appmeshsdk.ResourceMetadata{
						ResourceOwner: aws.String("33333333"),
					},
				},
				vg: &appmesh.VirtualGateway{},
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
			got := m.isSDKVirtualGatewayControlledByCRDVirtualGateway(ctx, tt.args.sdkVG, tt.args.vg)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_defaultResourceManager_isSDKVirtualGatewayOwnedByCRDVirtualGateway(t *testing.T) {
	type fields struct {
		accountID string
	}
	type args struct {
		sdkVG *appmeshsdk.VirtualGatewayData
		vg    *appmesh.VirtualGateway
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name:   "sdkVG is owned by crdVG",
			fields: fields{accountID: "222222222"},
			args: args{
				sdkVG: &appmeshsdk.VirtualGatewayData{
					Metadata: &appmeshsdk.ResourceMetadata{
						ResourceOwner: aws.String("222222222"),
					},
				},
				vg: &appmesh.VirtualGateway{},
			},
			want: true,
		},
		{
			name:   "sdkVG isn't owned by crdVG",
			fields: fields{accountID: "222222222"},
			args: args{
				sdkVG: &appmeshsdk.VirtualGatewayData{
					Metadata: &appmeshsdk.ResourceMetadata{
						ResourceOwner: aws.String("33333333"),
					},
				},
				vg: &appmesh.VirtualGateway{},
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
			got := m.isSDKVirtualGatewayOwnedByCRDVirtualGateway(ctx, tt.args.sdkVG, tt.args.vg)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_defaultResourceManager_findMeshDependency(t *testing.T) {
	type fields struct {
		ResolveMeshReference func(ctx context.Context, ref appmesh.MeshReference) (*appmesh.Mesh, error)
	}
	type args struct {
		vg *appmesh.VirtualGateway
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *appmesh.Mesh
		wantErr error
	}{
		{
			name: "virtualGateway with mesh",
			args: args{
				vg: &appmesh.VirtualGateway{
					ObjectMeta: metav1.ObjectMeta{
						Name: "vg-1",
					},
					Spec: appmesh.VirtualGatewaySpec{
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
			name: "virtualGateway with missing MeshRef",
			args: args{
				vg: &appmesh.VirtualGateway{
					ObjectMeta: metav1.ObjectMeta{
						Name: "vg-1",
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
			name: "virtualGateway failed to resolve mesh",
			args: args{
				vg: &appmesh.VirtualGateway{
					ObjectMeta: metav1.ObjectMeta{
						Name: "vg-1",
					},
					Spec: appmesh.VirtualGatewaySpec{
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

			got_mesh, err := m.findMeshDependency(ctx, tt.args.vg)
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
