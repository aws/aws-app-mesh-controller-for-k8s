package virtualrouter

import (
	"context"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/equality"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	testclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"testing"
)

func Test_defaultReferenceResolver_Resolve(t *testing.T) {
	vrInNS1 := &appmesh.VirtualRouter{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns-1",
			Name:      "vr",
		},
	}
	vrInNS2 := &appmesh.VirtualRouter{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns-2",
			Name:      "vr",
		},
	}

	type env struct {
		virtualRouters []*appmesh.VirtualRouter
	}
	type args struct {
		obj   metav1.Object
		vrRef appmesh.VirtualRouterReference
	}
	tests := []struct {
		name    string
		env     env
		args    args
		want    *appmesh.VirtualRouter
		wantErr error
	}{
		{
			name: "when VirtualRouterReference contains both namespace and name",
			env: env{
				virtualRouters: []*appmesh.VirtualRouter{vrInNS1, vrInNS2},
			},
			args: args{
				obj: &appmesh.VirtualService{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns-1",
						Name:      "vs",
					},
				},
				vrRef: appmesh.VirtualRouterReference{
					Namespace: aws.String("ns-2"),
					Name:      "vr",
				},
			},
			want: vrInNS2,
		},
		{
			name: "when VirtualRouterReference contains name only",
			env: env{
				virtualRouters: []*appmesh.VirtualRouter{vrInNS1, vrInNS2},
			},
			args: args{
				obj: &appmesh.VirtualService{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns-1",
						Name:      "vs",
					},
				},
				vrRef: appmesh.VirtualRouterReference{
					Name: "vr",
				},
			},
			want: vrInNS1,
		},
		{
			name: "when VirtualRouterReference didn't reference existing vs",
			env: env{
				virtualRouters: []*appmesh.VirtualRouter{vrInNS1, vrInNS2},
			},
			args: args{
				obj: &appmesh.VirtualService{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns-1",
						Name:      "vs",
					},
				},
				vrRef: appmesh.VirtualRouterReference{
					Namespace: aws.String("ns-3"),
					Name:      "vr",
				},
			},
			want:    nil,
			wantErr: errors.New("unable to fetch virtualRouter: ns-3/vr: virtualrouters.appmesh.k8s.aws \"vr\" not found"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			k8sSchema := runtime.NewScheme()
			clientgoscheme.AddToScheme(k8sSchema)
			appmesh.AddToScheme(k8sSchema)
			k8sClient := testclient.NewFakeClientWithScheme(k8sSchema)
			r := NewDefaultReferenceResolver(k8sClient, &log.NullLogger{})

			for _, vr := range tt.env.virtualRouters {
				err := k8sClient.Create(ctx, vr.DeepCopy())
				assert.NoError(t, err)
			}

			got, err := r.Resolve(ctx, tt.args.obj, tt.args.vrRef)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				opt := equality.IgnoreFakeClientPopulatedFields()
				assert.True(t, cmp.Equal(tt.want, got, opt),
					"diff: %v", cmp.Diff(tt.want, got, opt))
			}
		})
	}
}
