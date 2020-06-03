package virtualgateway

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
				log:       log.NullLogger{},
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
				log:       &log.NullLogger{},
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
				log:       &log.NullLogger{},
			}
			got := m.isSDKVirtualGatewayOwnedByCRDVirtualGateway(ctx, tt.args.sdkVG, tt.args.vg)
			assert.Equal(t, tt.want, got)
		})
	}
}
