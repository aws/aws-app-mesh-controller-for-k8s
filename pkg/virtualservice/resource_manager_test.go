package virtualservice

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
				log:       log.NullLogger{},
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
				log:       &log.NullLogger{},
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
				log:       &log.NullLogger{},
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
								VirtualNodeRef: appmesh.VirtualNodeReference{
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
								VirtualNodeRef: appmesh.VirtualNodeReference{
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
								VirtualRouterRef: appmesh.VirtualRouterReference{
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
								VirtualRouterRef: appmesh.VirtualRouterReference{
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
