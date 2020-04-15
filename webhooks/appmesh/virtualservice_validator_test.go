package appmesh

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func Test_virtualServiceValidator_enforceFieldsImmutability(t *testing.T) {
	type args struct {
		vs    *appmesh.VirtualService
		oldvs *appmesh.VirtualService
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{
			name: "VirtualService immutable fields didn't change",
			args: args{
				vs: &appmesh.VirtualService{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vs",
					},
					Spec: appmesh.VirtualServiceSpec{
						AWSName: aws.String("my-vs.awesome-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
					},
				},
				oldvs: &appmesh.VirtualService{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vs",
					},
					Spec: appmesh.VirtualServiceSpec{
						AWSName: aws.String("my-vs.awesome-ns"),
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
			name: "VirtualService field awsName changed",
			args: args{
				vs: &appmesh.VirtualService{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vs",
					},
					Spec: appmesh.VirtualServiceSpec{
						AWSName: aws.String("my-vs.awesome-ns.svc.cluster.local"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
					},
				},
				oldvs: &appmesh.VirtualService{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vs",
					},
					Spec: appmesh.VirtualServiceSpec{
						AWSName: aws.String("my-vs.awesome-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
					},
				},
			},
			wantErr: errors.New("VirtualService update may not change these fields: spec.awsName"),
		},
		{
			name: "VirtualService field meshRef changed",
			args: args{
				vs: &appmesh.VirtualService{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vs",
					},
					Spec: appmesh.VirtualServiceSpec{
						AWSName: aws.String("my-vs.awesome-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "another-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
					},
				},
				oldvs: &appmesh.VirtualService{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vs",
					},
					Spec: appmesh.VirtualServiceSpec{
						AWSName: aws.String("my-vs.awesome-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
					},
				},
			},
			wantErr: errors.New("VirtualService update may not change these fields: spec.meshRef"),
		},
		{
			name: "VirtualService fields awsName and meshRef changed",
			args: args{
				vs: &appmesh.VirtualService{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vs",
					},
					Spec: appmesh.VirtualServiceSpec{
						AWSName: aws.String("my-vs.awesome-ns.svc.cluster.local"),
						MeshRef: &appmesh.MeshReference{
							Name: "another-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
					},
				},
				oldvs: &appmesh.VirtualService{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vs",
					},
					Spec: appmesh.VirtualServiceSpec{
						AWSName: aws.String("my-vs.awesome-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
					},
				},
			},
			wantErr: errors.New("VirtualService update may not change these fields: spec.awsName,spec.meshRef"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &virtualServiceValidator{}
			err := v.enforceFieldsImmutability(tt.args.vs, tt.args.oldvs)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
