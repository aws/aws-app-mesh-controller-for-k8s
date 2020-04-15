package appmesh

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func Test_meshMutator_defaultingAWSName(t *testing.T) {
	type args struct {
		mesh *appmesh.Mesh
	}
	tests := []struct {
		name    string
		args    args
		want    *appmesh.Mesh
		wantErr error
	}{
		{
			name: "Mesh didn't specify awsName",
			args: args{
				mesh: &appmesh.Mesh{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-mesh",
					},
					Spec: appmesh.MeshSpec{},
				},
			},
			want: &appmesh.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-mesh",
				},
				Spec: appmesh.MeshSpec{
					AWSName: aws.String("my-mesh"),
				},
			},
		},
		{
			name: "Mesh specified empty awsName",
			args: args{
				mesh: &appmesh.Mesh{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-mesh",
					},
					Spec: appmesh.MeshSpec{
						AWSName: aws.String(""),
					},
				},
			},
			want: &appmesh.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-mesh",
				},
				Spec: appmesh.MeshSpec{
					AWSName: aws.String("my-mesh"),
				},
			},
		},
		{
			name: "Mesh specified non-empty awsName",
			args: args{
				mesh: &appmesh.Mesh{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-mesh",
					},
					Spec: appmesh.MeshSpec{
						AWSName: aws.String("my-mesh-for-my-cluster"),
					},
				},
			},
			want: &appmesh.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-mesh",
				},
				Spec: appmesh.MeshSpec{
					AWSName: aws.String("my-mesh-for-my-cluster"),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &meshMutator{}
			err := m.defaultingAWSName(tt.args.mesh)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, tt.args.mesh)
			}
		})
	}
}
