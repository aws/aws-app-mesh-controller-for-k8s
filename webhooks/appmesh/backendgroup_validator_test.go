package appmesh

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func Test_backendGroupValidator_enforceFieldsImmutability(t *testing.T) {
	type args struct {
		bg    *appmesh.BackendGroup
		oldbg *appmesh.BackendGroup
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{
			name: "BackendGroup immutable fields didn't change",
			args: args{
				bg: &appmesh.BackendGroup{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-bg",
					},
					Spec: appmesh.BackendGroupSpec{
						VirtualServices: []appmesh.VirtualServiceReference{
							{
								Name: "my-vs",
							},
						},
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
					},
				},
				oldbg: &appmesh.BackendGroup{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-bg",
					},
					Spec: appmesh.BackendGroupSpec{
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "BackendGroup field meshRef changed",
			args: args{
				bg: &appmesh.BackendGroup{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-bg",
					},
					Spec: appmesh.BackendGroupSpec{
						VirtualServices: []appmesh.VirtualServiceReference{
							{
								Name: "my-vs",
							},
						},
						MeshRef: &appmesh.MeshReference{
							Name: "another-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
					},
				},
				oldbg: &appmesh.BackendGroup{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-bg",
					},
					Spec: appmesh.BackendGroupSpec{
						VirtualServices: []appmesh.VirtualServiceReference{
							{
								Name: "my-vs",
							},
						},
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
					},
				},
			},
			wantErr: errors.New("BackendGroup update may not change these fields: spec.meshRef"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &backendGroupValidator{}
			err := v.enforceFieldsImmutability(tt.args.bg, tt.args.oldbg)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
