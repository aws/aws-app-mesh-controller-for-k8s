package mesh

import (
	"context"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/equality"
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
	meshUIDMatches := &appmesh.Mesh{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-mesh",
			UID:  "a385048d-aba8-4235-9a11-4173764c8ab7",
		},
	}
	meshUIDMismatches := &appmesh.Mesh{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-mesh",
			UID:  "f7d10a22-e8d5-4626-b780-261374fc68d4",
		},
	}

	type env struct {
		meshes []*appmesh.Mesh
	}
	type args struct {
		meshRef appmesh.MeshReference
	}
	tests := []struct {
		name    string
		env     env
		args    args
		want    *appmesh.Mesh
		wantErr error
	}{
		{
			name: "mesh can be resolved when name and UID matches",
			env: env{
				meshes: []*appmesh.Mesh{meshUIDMatches},
			},
			args: args{
				meshRef: appmesh.MeshReference{
					Name: "my-mesh",
					UID:  "a385048d-aba8-4235-9a11-4173764c8ab7",
				},
			},
			want: meshUIDMatches,
		},
		{
			name: "mesh cannot be resolved when UID mismatches",
			env: env{
				meshes: []*appmesh.Mesh{meshUIDMismatches},
			},
			args: args{
				meshRef: appmesh.MeshReference{
					Name: "my-mesh",
					UID:  "a385048d-aba8-4235-9a11-4173764c8ab7",
				},
			},
			wantErr: errors.New("mesh UID mismatch: my-mesh"),
		},
		{
			name: "mesh cannot be resolved if not found",
			env: env{
				meshes: []*appmesh.Mesh{meshUIDMatches},
			},
			args: args{
				meshRef: appmesh.MeshReference{
					Name: "another-mesh",
					UID:  "a385048d-aba8-4235-9a11-4173764c8ab7",
				},
			},
			wantErr: errors.New("unable to fetch mesh: another-mesh: meshs.appmesh.k8s.aws \"another-mesh\" not found"),
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

			for _, ms := range tt.env.meshes {
				err := k8sClient.Create(ctx, ms.DeepCopy())
				assert.NoError(t, err)
			}

			got, err := r.Resolve(ctx, tt.args.meshRef)
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
