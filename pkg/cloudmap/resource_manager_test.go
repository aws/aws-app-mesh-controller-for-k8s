package cloudmap

import (
	"context"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/equality"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/k8s"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	testclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"testing"
)

func Test_defaultResourceManager_updateCRDVirtualNode(t *testing.T) {
	type args struct {
		vn         *appmesh.VirtualNode
		svcSummary *serviceSummary
	}
	testARN := "cloudMapARN"
	tests := []struct {
		name    string
		args    args
		wantVN  *appmesh.VirtualNode
		wantErr error
	}{
		{
			name: "virtualNode needs cloudMap service ARN patch",
			args: args{
				vn: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "vn-1",
						Annotations: map[string]string{},
					},
					Status: appmesh.VirtualNodeStatus{},
				},
				svcSummary: &serviceSummary{
					serviceID:  "cloudMapService",
					serviceARN: &testARN,
				},
			},
			wantVN: &appmesh.VirtualNode{
				ObjectMeta: metav1.ObjectMeta{
					Name: "vn-1",
					Annotations: map[string]string{
						"cloudMapServiceARN": "cloudMapARN",
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
			err = m.updateCRDVirtualNode(ctx, tt.args.vn, tt.args.svcSummary)
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

func Test_defaultResourceManager_isCloudMapServiceCreated(t *testing.T) {
	type args struct {
		vn *appmesh.VirtualNode
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "virtualNode has cloudMap service annotation",
			args: args{
				vn: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Name: "vn-1",
						Annotations: map[string]string{
							"cloudMapServiceARN": "cloudMapARN",
						},
					},
					Status: appmesh.VirtualNodeStatus{},
				},
			},
			want: true,
		},
		{
			name: "virtualNode doesn't have cloudMap service annotation",
			args: args{
				vn: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Name: "vn-1",
					},
					Status: appmesh.VirtualNodeStatus{},
				},
			},
			want: false,
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
			response := m.isCloudMapServiceCreated(ctx, tt.args.vn)
			assert.True(t, cmp.Equal(tt.want, response), "diff", cmp.Diff(tt.want, response))
		})
	}
}
