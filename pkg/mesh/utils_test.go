package mesh

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestIsMeshActive(t *testing.T) {
	type args struct {
		mesh *appmesh.Mesh
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "mesh have true meshActive condition",
			args: args{
				mesh: &appmesh.Mesh{
					Status: appmesh.MeshStatus{
						Conditions: []appmesh.MeshCondition{
							{
								Type:   appmesh.MeshActive,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "mesh have false meshActive condition",
			args: args{
				mesh: &appmesh.Mesh{
					Status: appmesh.MeshStatus{
						Conditions: []appmesh.MeshCondition{
							{
								Type:   appmesh.MeshActive,
								Status: corev1.ConditionFalse,
							},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "mesh have unknown meshActive condition",
			args: args{
				mesh: &appmesh.Mesh{
					Status: appmesh.MeshStatus{
						Conditions: []appmesh.MeshCondition{
							{
								Type:   appmesh.MeshActive,
								Status: corev1.ConditionUnknown,
							},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "mesh doesn't have meshActive condition",
			args: args{
				mesh: &appmesh.Mesh{
					Status: appmesh.MeshStatus{
						Conditions: []appmesh.MeshCondition{},
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsMeshActive(tt.args.mesh)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsMeshReferenced(t *testing.T) {
	type args struct {
		ms        *appmesh.Mesh
		reference appmesh.MeshReference
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "mesh is referenced when both name and UID matches",
			args: args{
				ms: &appmesh.Mesh{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-mesh",
						UID:  "a385048d-aba8-4235-9a11-4173764c8ab7",
					},
				},
				reference: appmesh.MeshReference{
					Name: "my-mesh",
					UID:  "a385048d-aba8-4235-9a11-4173764c8ab7",
				},
			},
			want: true,
		},
		{
			name: "mesh is not referenced when name mismatches",
			args: args{
				ms: &appmesh.Mesh{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-mesh",
						UID:  "a385048d-aba8-4235-9a11-4173764c8ab7",
					},
				},
				reference: appmesh.MeshReference{
					Name: "another-mesh",
					UID:  "a385048d-aba8-4235-9a11-4173764c8ab7",
				},
			},
			want: false,
		},
		{
			name: "mesh is not referenced when UID mismatches",
			args: args{
				ms: &appmesh.Mesh{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-mesh",
						UID:  "a385048d-aba8-4235-9a11-4173764c8ab7",
					},
				},
				reference: appmesh.MeshReference{
					Name: "my-mesh",
					UID:  "f7d10a22-e8d5-4626-b780-261374fc68d4",
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsMeshReferenced(tt.args.ms, tt.args.reference); got != tt.want {
				t.Errorf("IsMeshReferenced() = %v, want %v", got, tt.want)
			}
		})
	}
}
