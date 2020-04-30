package k8s

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestHasFinalizer(t *testing.T) {
	tests := []struct {
		name      string
		obj       metav1.Object
		finalizer string
		want      bool
	}{
		{
			name: "finalizer exists and matches",
			obj: &appmesh.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Finalizers: []string{"finalizers.appmesh.k8s.aws/aws-resources"},
				},
			},
			finalizer: "finalizers.appmesh.k8s.aws/aws-resources",
			want:      true,
		},
		{
			name: "finalizer not exists",
			obj: &appmesh.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Finalizers: []string{},
				},
			},
			finalizer: "finalizers.appmesh.k8s.aws/aws-resources",
			want:      false,
		},
		{
			name: "finalizer exists but not matches",
			obj: &appmesh.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Finalizers: []string{"finalizers.appmesh.k8s.aws/mesh-members"},
				},
			},
			finalizer: "finalizers.appmesh.k8s.aws/aws-resources",
			want:      false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasFinalizer(tt.obj, tt.finalizer)
			assert.Equal(t, tt.want, got)
		})
	}
}
