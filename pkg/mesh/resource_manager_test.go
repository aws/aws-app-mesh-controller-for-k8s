package mesh

import (
	"context"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/equality"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/k8s"
	"github.com/aws/aws-sdk-go/aws"
	appmeshsdk "github.com/aws/aws-sdk-go/service/appmesh"
	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	testclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"testing"
)

func Test_defaultResourceManager_updateCRDMesh(t *testing.T) {
	type args struct {
		ms    *appmesh.Mesh
		sdkMS *appmeshsdk.MeshData
	}
	tests := []struct {
		name    string
		args    args
		wantMS  *appmesh.Mesh
		wantErr error
	}{
		{
			name: "mesh needs patch both arn and condition",
			args: args{
				ms: &appmesh.Mesh{
					ObjectMeta: metav1.ObjectMeta{
						Name: "mesh-1",
					},
					Status: appmesh.MeshStatus{},
				},
				sdkMS: &appmeshsdk.MeshData{
					Metadata: &appmeshsdk.ResourceMetadata{
						Arn: aws.String("arn-1"),
					},
					Status: &appmeshsdk.MeshStatus{
						Status: aws.String(appmeshsdk.MeshStatusCodeActive),
					},
				},
			},
			wantMS: &appmesh.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Name: "mesh-1",
				},
				Status: appmesh.MeshStatus{
					MeshARN: aws.String("arn-1"),
					Conditions: []appmesh.MeshCondition{
						{
							Type:   appmesh.MeshActive,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
		},
		{
			name: "mesh needs patch condition only",
			args: args{
				ms: &appmesh.Mesh{
					ObjectMeta: metav1.ObjectMeta{
						Name: "mesh-1",
					},
					Status: appmesh.MeshStatus{
						MeshARN: aws.String("arn-1"),
						Conditions: []appmesh.MeshCondition{
							{
								Type:   appmesh.MeshActive,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
				sdkMS: &appmeshsdk.MeshData{
					Metadata: &appmeshsdk.ResourceMetadata{
						Arn: aws.String("arn-1"),
					},
					Status: &appmeshsdk.MeshStatus{
						Status: aws.String(appmeshsdk.MeshStatusCodeInactive),
					},
				},
			},
			wantMS: &appmesh.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Name: "mesh-1",
				},
				Status: appmesh.MeshStatus{
					MeshARN: aws.String("arn-1"),
					Conditions: []appmesh.MeshCondition{
						{
							Type:   appmesh.MeshActive,
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

			err := k8sClient.Create(ctx, tt.args.ms.DeepCopy())
			assert.NoError(t, err)
			err = m.updateCRDMesh(ctx, tt.args.ms, tt.args.sdkMS)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				gotMS := &appmesh.Mesh{}
				err = k8sClient.Get(ctx, k8s.NamespacedName(tt.args.ms), gotMS)
				assert.NoError(t, err)
				opts := cmp.Options{
					equality.IgnoreFakeClientPopulatedFields(),
					cmpopts.IgnoreTypes((*metav1.Time)(nil)),
				}
				assert.True(t, cmp.Equal(tt.wantMS, gotMS, opts), "diff", cmp.Diff(tt.wantMS, gotMS, opts))
			}
		})
	}
}

func Test_defaultResourceManager_isSDKMeshControlledByCRDMesh(t *testing.T) {
	type fields struct {
		accountID string
	}
	type args struct {
		sdkMS *appmeshsdk.MeshData
		ms    *appmesh.Mesh
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name:   "sdkMesh is controlled crdMesh",
			fields: fields{accountID: "222222222"},
			args: args{
				sdkMS: &appmeshsdk.MeshData{
					Metadata: &appmeshsdk.ResourceMetadata{
						ResourceOwner: aws.String("222222222"),
					},
				},
				ms: &appmesh.Mesh{},
			},
			want: true,
		},
		{
			name:   "sdkMesh isn't controlled crdMesh",
			fields: fields{accountID: "222222222"},
			args: args{
				sdkMS: &appmeshsdk.MeshData{
					Metadata: &appmeshsdk.ResourceMetadata{
						ResourceOwner: aws.String("33333333"),
					},
				},
				ms: &appmesh.Mesh{},
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
			got := m.isSDKMeshControlledByCRDMesh(ctx, tt.args.sdkMS, tt.args.ms)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_defaultResourceManager_isSDKMeshOwnedByCRDMesh(t *testing.T) {
	type fields struct {
		accountID string
	}
	type args struct {
		sdkMS *appmeshsdk.MeshData
		ms    *appmesh.Mesh
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name:   "sdkMesh is controlled crdMesh",
			fields: fields{accountID: "222222222"},
			args: args{
				sdkMS: &appmeshsdk.MeshData{
					Metadata: &appmeshsdk.ResourceMetadata{
						ResourceOwner: aws.String("222222222"),
					},
				},
				ms: &appmesh.Mesh{},
			},
			want: true,
		},
		{
			name:   "sdkMesh isn't controlled crdMesh",
			fields: fields{accountID: "222222222"},
			args: args{
				sdkMS: &appmeshsdk.MeshData{
					Metadata: &appmeshsdk.ResourceMetadata{
						ResourceOwner: aws.String("33333333"),
					},
				},
				ms: &appmesh.Mesh{},
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
			got := m.isSDKMeshOwnedByCRDMesh(ctx, tt.args.sdkMS, tt.args.ms)
			assert.Equal(t, tt.want, got)
		})
	}
}
