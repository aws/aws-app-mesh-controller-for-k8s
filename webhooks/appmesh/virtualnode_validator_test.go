package appmesh

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func Test_virtualNodeValidator_enforceFieldsImmutability(t *testing.T) {
	type args struct {
		vn    *appmesh.VirtualNode
		oldVN *appmesh.VirtualNode
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{
			name: "VirtualNode immutable fields didn't change",
			args: args{
				vn: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vn",
					},
					Spec: appmesh.VirtualNodeSpec{
						AWSName: aws.String("my-vn_awesome-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
					},
				},
				oldVN: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vn",
					},
					Spec: appmesh.VirtualNodeSpec{
						AWSName: aws.String("my-vn_awesome-ns"),
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
			name: "VirtualNode field awsName changed",
			args: args{
				vn: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vn",
					},
					Spec: appmesh.VirtualNodeSpec{
						AWSName: aws.String("my-vn_awesome-ns_my-cluster"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
					},
				},
				oldVN: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vn",
					},
					Spec: appmesh.VirtualNodeSpec{
						AWSName: aws.String("my-vn_awesome-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
					},
				},
			},
			wantErr: errors.New("VirtualNode update may not change these fields: spec.awsName"),
		},
		{
			name: "VirtualNode field meshRef changed",
			args: args{
				vn: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vn",
					},
					Spec: appmesh.VirtualNodeSpec{
						AWSName: aws.String("my-vn_awesome-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "another-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
					},
				},
				oldVN: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vn",
					},
					Spec: appmesh.VirtualNodeSpec{
						AWSName: aws.String("my-vn_awesome-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
					},
				},
			},
			wantErr: errors.New("VirtualNode update may not change these fields: spec.meshRef"),
		},
		{
			name: "VirtualNode fields awsName and meshRef changed",
			args: args{
				vn: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vn",
					},
					Spec: appmesh.VirtualNodeSpec{
						AWSName: aws.String("my-vn_awesome-ns-my-cluster"),
						MeshRef: &appmesh.MeshReference{
							Name: "another-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
					},
				},
				oldVN: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vn",
					},
					Spec: appmesh.VirtualNodeSpec{
						AWSName: aws.String("my-vn_awesome-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
					},
				},
			},
			wantErr: errors.New("VirtualNode update may not change these fields: spec.awsName,spec.meshRef"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &virtualNodeValidator{}
			err := v.enforceFieldsImmutability(tt.args.vn, tt.args.oldVN)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
