package virtualnode

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
	vnInNS1 := &appmesh.VirtualNode{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns-1",
			Name:      "vn",
		},
	}
	vnInNS2 := &appmesh.VirtualNode{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns-2",
			Name:      "vn",
		},
	}

	type env struct {
		virtualNodes []*appmesh.VirtualNode
	}
	type args struct {
		obj   metav1.Object
		vnRef appmesh.VirtualNodeReference
	}
	tests := []struct {
		name    string
		env     env
		args    args
		want    *appmesh.VirtualNode
		wantErr error
	}{
		{
			name: "when VirtualNodeReference contains both namespace and name",
			env: env{
				virtualNodes: []*appmesh.VirtualNode{vnInNS1, vnInNS2},
			},
			args: args{
				obj: &appmesh.VirtualRouter{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns-1",
						Name:      "vr",
					},
				},
				vnRef: appmesh.VirtualNodeReference{
					Namespace: aws.String("ns-2"),
					Name:      "vn",
				},
			},
			want: vnInNS2,
		},
		{
			name: "when VirtualNodeReference contains name only",
			env: env{
				virtualNodes: []*appmesh.VirtualNode{vnInNS1, vnInNS2},
			},
			args: args{
				obj: &appmesh.VirtualRouter{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns-1",
						Name:      "vr",
					},
				},
				vnRef: appmesh.VirtualNodeReference{
					Name: "vn",
				},
			},
			want: vnInNS1,
		},
		{
			name: "when VirtualNodeReference didn't reference existing vs",
			env: env{
				virtualNodes: []*appmesh.VirtualNode{vnInNS1, vnInNS2},
			},
			args: args{
				obj: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns-1",
						Name:      "vn",
					},
				},
				vnRef: appmesh.VirtualNodeReference{
					Namespace: aws.String("ns-3"),
					Name:      "vn",
				},
			},
			want:    nil,
			wantErr: errors.New("unable to fetch virtualNode: ns-3/vn: virtualnodes.appmesh.k8s.aws \"vn\" not found"),
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

			for _, vn := range tt.env.virtualNodes {
				err := k8sClient.Create(ctx, vn.DeepCopy())
				assert.NoError(t, err)
			}

			got, err := r.Resolve(ctx, tt.args.obj, tt.args.vnRef)
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
