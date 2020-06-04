package gatewayroute

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
				log:       log.NullLogger{},
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
				log:       &log.NullLogger{},
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
				log:       &log.NullLogger{},
			}
			got := m.isSDKGatewayRouteOwnedByCRDGatewayRoute(ctx, tt.args.sdkGR, tt.args.gr)
			assert.Equal(t, tt.want, got)
		})
	}
}
