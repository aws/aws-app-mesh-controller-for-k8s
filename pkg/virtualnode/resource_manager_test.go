package virtualnode

import (
	"context"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/equality"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/k8s"
	"github.com/aws/aws-sdk-go/aws"
	appmeshsdk "github.com/aws/aws-sdk-go/service/appmesh"
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
				log:       log.NullLogger{},
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
				log:       &log.NullLogger{},
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
				log:       &log.NullLogger{},
			}
			got := m.isSDKVirtualNodeOwnedByCRDVirtualNode(ctx, tt.args.sdkVN, tt.args.vn)
			assert.Equal(t, tt.want, got)
		})
	}
}
