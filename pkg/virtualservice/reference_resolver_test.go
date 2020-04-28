package virtualservice

import (
	"context"
	"errors"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/equality"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	testclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"testing"
)

func Test_defaultReferenceResolver_Resolve(t *testing.T) {
	vsInNS1 := &appmesh.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns-1",
			Name:      "vs",
		},
	}
	vsInNS2 := &appmesh.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns-2",
			Name:      "vs",
		},
	}

	type env struct {
		virtualServices []*appmesh.VirtualService
	}
	type args struct {
		obj   metav1.Object
		vsRef appmesh.VirtualServiceReference
	}
	tests := []struct {
		name    string
		env     env
		args    args
		want    *appmesh.VirtualService
		wantErr error
	}{
		{
			name: "when VirtualServiceReference contains both namespace and name",
			env: env{
				virtualServices: []*appmesh.VirtualService{vsInNS1, vsInNS2},
			},
			args: args{
				obj: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns-1",
						Name:      "vn",
					},
				},
				vsRef: appmesh.VirtualServiceReference{
					Namespace: aws.String("ns-2"),
					Name:      "vs",
				},
			},
			want: vsInNS2,
		},
		{
			name: "when VirtualServiceReference contains name only",
			env: env{
				virtualServices: []*appmesh.VirtualService{vsInNS1, vsInNS2},
			},
			args: args{
				obj: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns-1",
						Name:      "vn",
					},
				},
				vsRef: appmesh.VirtualServiceReference{
					Name: "vs",
				},
			},
			want: vsInNS1,
		},
		{
			name: "when VirtualServiceReference didn't reference existing vs",
			env: env{
				virtualServices: []*appmesh.VirtualService{vsInNS1, vsInNS2},
			},
			args: args{
				obj: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns-1",
						Name:      "vn",
					},
				},
				vsRef: appmesh.VirtualServiceReference{
					Namespace: aws.String("ns-3"),
					Name:      "vs",
				},
			},
			want:    nil,
			wantErr: errors.New("unable to fetch virtualService: ns-3/vs: virtualservices.appmesh.k8s.aws \"vs\" not found"),
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

			for _, vs := range tt.env.virtualServices {
				err := k8sClient.Create(ctx, vs.DeepCopy())
				assert.NoError(t, err)
			}

			got, err := r.Resolve(ctx, tt.args.obj, tt.args.vsRef)
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
