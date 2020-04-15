package appmesh

import (
	"context"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	mock_mesh "github.com/aws/aws-app-mesh-controller-for-k8s/mocks/aws-app-mesh-controller-for-k8s/pkg/mesh"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func Test_virtualRouterMutator_defaultingAWSName(t *testing.T) {
	type args struct {
		vr *appmesh.VirtualRouter
	}
	tests := []struct {
		name    string
		args    args
		want    *appmesh.VirtualRouter
		wantErr error
	}{
		{
			name: "VirtualRouter didn't specify awsName",
			args: args{
				vr: &appmesh.VirtualRouter{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vr",
					},
					Spec: appmesh.VirtualRouterSpec{},
				},
			},
			want: &appmesh.VirtualRouter{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "awesome-ns",
					Name:      "my-vr",
				},
				Spec: appmesh.VirtualRouterSpec{
					AWSName: aws.String("my-vr_awesome-ns"),
				},
			},
		},
		{
			name: "VirtualRouter specified empty awsName",
			args: args{
				vr: &appmesh.VirtualRouter{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vr",
					},
					Spec: appmesh.VirtualRouterSpec{
						AWSName: aws.String(""),
					},
				},
			},
			want: &appmesh.VirtualRouter{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "awesome-ns",
					Name:      "my-vr",
				},
				Spec: appmesh.VirtualRouterSpec{
					AWSName: aws.String("my-vr_awesome-ns"),
				},
			},
		},
		{
			name: "VirtualRouter specified non-empty awsName",
			args: args{
				vr: &appmesh.VirtualRouter{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vr",
					},
					Spec: appmesh.VirtualRouterSpec{
						AWSName: aws.String("my-vr_awesome-ns_my-cluster"),
					},
				},
			},
			want: &appmesh.VirtualRouter{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "awesome-ns",
					Name:      "my-vr",
				},
				Spec: appmesh.VirtualRouterSpec{
					AWSName: aws.String("my-vr_awesome-ns_my-cluster"),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &virtualRouterMutator{}
			err := m.defaultingAWSName(tt.args.vr)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, tt.args.vr)
			}
		})
	}
}

func Test_virtualRouterMutator_designateMeshMembership(t *testing.T) {
	type fields struct {
		meshMembershipDesignatorDesignate func(ctx context.Context, obj metav1.Object) (*appmesh.Mesh, error)
	}
	type args struct {
		vr *appmesh.VirtualRouter
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *appmesh.VirtualRouter
		wantErr error
	}{
		{
			name: "successfully designate mesh membership",
			fields: fields{
				meshMembershipDesignatorDesignate: func(ctx context.Context, obj metav1.Object) (*appmesh.Mesh, error) {
					return &appmesh.Mesh{
						ObjectMeta: metav1.ObjectMeta{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
					}, nil
				},
			},
			args: args{
				vr: &appmesh.VirtualRouter{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vr",
					},
					Spec: appmesh.VirtualRouterSpec{},
				},
			},
			want: &appmesh.VirtualRouter{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "awesome-ns",
					Name:      "my-vr",
				},
				Spec: appmesh.VirtualRouterSpec{
					MeshRef: &appmesh.MeshReference{
						Name: "my-mesh",
						UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
					},
				},
			},
		},
		{
			name: "failed to designate mesh membership",
			fields: fields{
				meshMembershipDesignatorDesignate: func(ctx context.Context, obj metav1.Object) (*appmesh.Mesh, error) {
					return nil, errors.New("oops, some error happened")
				},
			},
			args: args{
				vr: &appmesh.VirtualRouter{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vr",
					},
					Spec: appmesh.VirtualRouterSpec{},
				},
			},
			wantErr: errors.New("oops, some error happened"),
		},
		{
			name:   "meshRef already specified",
			fields: fields{},
			args: args{
				vr: &appmesh.VirtualRouter{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vr",
					},
					Spec: appmesh.VirtualRouterSpec{
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
					},
				},
			},
			wantErr: errors.New("VirtualRouter create may not specify read-only field: spec.meshRef"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			designator := mock_mesh.NewMockMembershipDesignator(ctrl)
			if tt.fields.meshMembershipDesignatorDesignate != nil {
				designator.EXPECT().Designate(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.meshMembershipDesignatorDesignate)
			}

			m := &virtualRouterMutator{
				meshMembershipDesignator: designator,
			}
			err := m.designateMeshMembership(ctx, tt.args.vr)

			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, tt.args.vr)
			}
		})
	}
}
