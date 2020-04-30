package equality

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestIgnoreFakeClientPopulatedFields(t *testing.T) {
	tests := []struct {
		name       string
		meshLeft   *appmesh.Mesh
		meshRight  *appmesh.Mesh
		wantEquals bool
	}{
		{
			name: "objects should be equal if only TypeMeta and ObjectMeta.ResourceVersion diffs",
			meshLeft: &appmesh.Mesh{
				TypeMeta: v1.TypeMeta{
					Kind:       "mesh",
					APIVersion: "appmesh.k8s.aws/v1beta1",
				},
				ObjectMeta: v1.ObjectMeta{
					ResourceVersion: "0",
					Annotations: map[string]string{
						"k": "v1",
					},
				},
			},
			meshRight: &appmesh.Mesh{
				TypeMeta: v1.TypeMeta{
					Kind:       "mesh",
					APIVersion: "appmesh.k8s.aws/v1beta2",
				},
				ObjectMeta: v1.ObjectMeta{
					ResourceVersion: "1",
					Annotations: map[string]string{
						"k": "v1",
					},
				},
			},
			wantEquals: true,
		},
		{
			name: "objects shouldn't be equal if more fields than TypeMeta and ObjectMeta.ResourceVersion diffs",
			meshLeft: &appmesh.Mesh{
				TypeMeta: v1.TypeMeta{
					Kind:       "mesh",
					APIVersion: "appmesh.k8s.aws/v1beta1",
				},
				ObjectMeta: v1.ObjectMeta{
					ResourceVersion: "0",
					Annotations: map[string]string{
						"k": "v1",
					},
				},
			},
			meshRight: &appmesh.Mesh{
				TypeMeta: v1.TypeMeta{
					Kind:       "mesh",
					APIVersion: "appmesh.k8s.aws/v1beta2",
				},
				ObjectMeta: v1.ObjectMeta{
					ResourceVersion: "1",
					Annotations: map[string]string{
						"k": "v2",
					},
				},
			},
			wantEquals: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := IgnoreFakeClientPopulatedFields()
			gotEquals := cmp.Equal(tt.meshLeft, tt.meshRight, opts)
			assert.Equal(t, tt.wantEquals, gotEquals)
		})
	}
}
