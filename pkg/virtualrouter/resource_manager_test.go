package virtualrouter

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

func Test_defaultResourceManager_updateCRDVirtualRouter(t *testing.T) {
	type args struct {
		vr             *appmesh.VirtualRouter
		sdkVR          *appmeshsdk.VirtualRouterData
		sdkRouteByName map[string]*appmeshsdk.RouteData
	}
	tests := []struct {
		name    string
		args    args
		wantVR  *appmesh.VirtualRouter
		wantErr error
	}{
		{
			name: "virtualRouter needs patch arn, routeARNs and condition",
			args: args{
				vr: &appmesh.VirtualRouter{
					ObjectMeta: metav1.ObjectMeta{
						Name: "vr-1",
					},
					Status: appmesh.VirtualRouterStatus{},
				},
				sdkVR: &appmeshsdk.VirtualRouterData{
					Metadata: &appmeshsdk.ResourceMetadata{
						Arn: aws.String("arn-1"),
					},
					Status: &appmeshsdk.VirtualRouterStatus{
						Status: aws.String(appmeshsdk.VirtualRouterStatusCodeActive),
					},
				},
				sdkRouteByName: map[string]*appmeshsdk.RouteData{
					"route-1": {
						Metadata: &appmeshsdk.ResourceMetadata{
							Arn: aws.String("route-arn-1"),
						},
					},
					"route-2": {
						Metadata: &appmeshsdk.ResourceMetadata{
							Arn: aws.String("route-arn-2"),
						},
					},
				},
			},
			wantVR: &appmesh.VirtualRouter{
				ObjectMeta: metav1.ObjectMeta{
					Name: "vr-1",
				},
				Status: appmesh.VirtualRouterStatus{
					VirtualRouterARN: aws.String("arn-1"),
					RouteARNs: map[string]string{
						"route-1": "route-arn-1",
						"route-2": "route-arn-2",
					},
					Conditions: []appmesh.VirtualRouterCondition{
						{
							Type:   appmesh.VirtualRouterActive,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
		},
		{
			name: "virtualRouter needs patch condition only",
			args: args{
				vr: &appmesh.VirtualRouter{
					ObjectMeta: metav1.ObjectMeta{
						Name: "vr-1",
					},
					Status: appmesh.VirtualRouterStatus{
						VirtualRouterARN: aws.String("arn-1"),
						Conditions: []appmesh.VirtualRouterCondition{
							{
								Type:   appmesh.VirtualRouterActive,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
				sdkVR: &appmeshsdk.VirtualRouterData{
					Metadata: &appmeshsdk.ResourceMetadata{
						Arn: aws.String("arn-1"),
					},
					Status: &appmeshsdk.VirtualRouterStatus{
						Status: aws.String(appmeshsdk.VirtualRouterStatusCodeInactive),
					},
				},
			},
			wantVR: &appmesh.VirtualRouter{
				ObjectMeta: metav1.ObjectMeta{
					Name: "vr-1",
				},
				Status: appmesh.VirtualRouterStatus{
					VirtualRouterARN: aws.String("arn-1"),
					Conditions: []appmesh.VirtualRouterCondition{
						{
							Type:   appmesh.VirtualRouterActive,
							Status: corev1.ConditionFalse,
						},
					},
				},
			},
		},
		{
			name: "virtualRouter needs patch routeARNs only",
			args: args{
				vr: &appmesh.VirtualRouter{
					ObjectMeta: metav1.ObjectMeta{
						Name: "vr-1",
					},
					Status: appmesh.VirtualRouterStatus{
						VirtualRouterARN: aws.String("arn-1"),
						RouteARNs: map[string]string{
							"route-1": "route-arn-1",
							"route-2": "route-arn-2",
						},
						Conditions: []appmesh.VirtualRouterCondition{
							{
								Type:   appmesh.VirtualRouterActive,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
				sdkVR: &appmeshsdk.VirtualRouterData{
					Metadata: &appmeshsdk.ResourceMetadata{
						Arn: aws.String("arn-1"),
					},
					Status: &appmeshsdk.VirtualRouterStatus{
						Status: aws.String(appmeshsdk.VirtualRouterStatusCodeActive),
					},
				},
				sdkRouteByName: map[string]*appmeshsdk.RouteData{
					"route-1": {
						Metadata: &appmeshsdk.ResourceMetadata{
							Arn: aws.String("route-arn-1"),
						},
					},
					"route-2": {
						Metadata: &appmeshsdk.ResourceMetadata{
							Arn: aws.String("route-arn-2"),
						},
					},
					"route-3": {
						Metadata: &appmeshsdk.ResourceMetadata{
							Arn: aws.String("route-arn-3"),
						},
					},
				},
			},
			wantVR: &appmesh.VirtualRouter{
				ObjectMeta: metav1.ObjectMeta{
					Name: "vr-1",
				},
				Status: appmesh.VirtualRouterStatus{
					VirtualRouterARN: aws.String("arn-1"),
					RouteARNs: map[string]string{
						"route-1": "route-arn-1",
						"route-2": "route-arn-2",
						"route-3": "route-arn-3",
					},
					Conditions: []appmesh.VirtualRouterCondition{
						{
							Type:   appmesh.VirtualRouterActive,
							Status: corev1.ConditionTrue,
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

			err := k8sClient.Create(ctx, tt.args.vr.DeepCopy())
			assert.NoError(t, err)
			err = m.updateCRDVirtualRouter(ctx, tt.args.vr, tt.args.sdkVR, tt.args.sdkRouteByName)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				gotVR := &appmesh.VirtualRouter{}
				err = k8sClient.Get(ctx, k8s.NamespacedName(tt.args.vr), gotVR)
				assert.NoError(t, err)
				opts := cmp.Options{
					equality.IgnoreFakeClientPopulatedFields(),
					cmpopts.IgnoreTypes((*metav1.Time)(nil)),
				}
				assert.True(t, cmp.Equal(tt.wantVR, gotVR, opts), "diff", cmp.Diff(tt.wantVR, gotVR, opts))
			}
		})
	}
}

func Test_defaultResourceManager_isSDKVirtualRouterControlledByCRDVirtualRouter(t *testing.T) {
	type fields struct {
		accountID string
	}
	type args struct {
		sdkVR *appmeshsdk.VirtualRouterData
		vr    *appmesh.VirtualRouter
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name:   "sdkVR is controlled by vr",
			fields: fields{accountID: "222222222"},
			args: args{
				sdkVR: &appmeshsdk.VirtualRouterData{
					Metadata: &appmeshsdk.ResourceMetadata{
						ResourceOwner: aws.String("222222222"),
					},
				},
				vr: &appmesh.VirtualRouter{},
			},
			want: true,
		},
		{
			name:   "sdkVR isn't controlled by vr",
			fields: fields{accountID: "222222222"},
			args: args{
				sdkVR: &appmeshsdk.VirtualRouterData{
					Metadata: &appmeshsdk.ResourceMetadata{
						ResourceOwner: aws.String("33333333"),
					},
				},
				vr: &appmesh.VirtualRouter{},
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
			got := m.isSDKVirtualRouterControlledByCRDVirtualRouter(ctx, tt.args.sdkVR, tt.args.vr)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_defaultResourceManager_isSDKVirtualRouterOwnedByCRDVirtualRouter(t *testing.T) {
	type fields struct {
		accountID string
	}
	type args struct {
		sdkVR *appmeshsdk.VirtualRouterData
		vr    *appmesh.VirtualRouter
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name:   "sdkVR is owned by vr",
			fields: fields{accountID: "222222222"},
			args: args{
				sdkVR: &appmeshsdk.VirtualRouterData{
					Metadata: &appmeshsdk.ResourceMetadata{
						ResourceOwner: aws.String("222222222"),
					},
				},
				vr: &appmesh.VirtualRouter{},
			},
			want: true,
		},
		{
			name:   "sdkVR isn't owned by vr",
			fields: fields{accountID: "222222222"},
			args: args{
				sdkVR: &appmeshsdk.VirtualRouterData{
					Metadata: &appmeshsdk.ResourceMetadata{
						ResourceOwner: aws.String("33333333"),
					},
				},
				vr: &appmesh.VirtualRouter{},
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
			got := m.isSDKVirtualRouterOwnedByCRDVirtualRouter(ctx, tt.args.sdkVR, tt.args.vr)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_BuildSDKVirtualRouterSpec(t *testing.T) {
	type args struct {
		vr *appmesh.VirtualRouter
	}
	tests := []struct {
		name    string
		args    args
		want    *appmeshsdk.VirtualRouterSpec
		wantErr error
	}{
		{
			name: "normal case",
			args: args{
				vr: &appmesh.VirtualRouter{
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
						},
					},
				},
			},
			want: &appmeshsdk.VirtualRouterSpec{
				Listeners: []*appmeshsdk.VirtualRouterListener{
					{
						PortMapping: &appmeshsdk.PortMapping{
							Port:     aws.Int64(80),
							Protocol: aws.String("http"),
						},
					},
					{
						PortMapping: &appmeshsdk.PortMapping{
							Port:     aws.Int64(443),
							Protocol: aws.String("http"),
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildSDKVirtualRouterSpec(tt.args.vr)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
