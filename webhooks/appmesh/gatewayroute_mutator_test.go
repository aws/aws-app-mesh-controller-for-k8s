// +build preview

package appmesh

import (
	"context"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	mock_mesh "github.com/aws/aws-app-mesh-controller-for-k8s/mocks/aws-app-mesh-controller-for-k8s/pkg/mesh"
	mock_virtualgateway "github.com/aws/aws-app-mesh-controller-for-k8s/mocks/aws-app-mesh-controller-for-k8s/pkg/virtualgateway"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func Test_gatewayRouteMutator_defaultingAWSName(t *testing.T) {
	type args struct {
		gr *appmesh.GatewayRoute
	}
	tests := []struct {
		name    string
		args    args
		want    *appmesh.GatewayRoute
		wantErr error
	}{
		{
			name: "GatewayRoute didn't specify awsName",
			args: args{
				gr: &appmesh.GatewayRoute{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-gr",
					},
					Spec: appmesh.GatewayRouteSpec{},
				},
			},
			want: &appmesh.GatewayRoute{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "awesome-ns",
					Name:      "my-gr",
				},
				Spec: appmesh.GatewayRouteSpec{
					AWSName: aws.String("my-gr_awesome-ns"),
				},
			},
		},
		{
			name: "GatewayRoute specified empty awsName",
			args: args{
				gr: &appmesh.GatewayRoute{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-gr",
					},
					Spec: appmesh.GatewayRouteSpec{
						AWSName: aws.String(""),
					},
				},
			},
			want: &appmesh.GatewayRoute{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "awesome-ns",
					Name:      "my-gr",
				},
				Spec: appmesh.GatewayRouteSpec{
					AWSName: aws.String("my-gr_awesome-ns"),
				},
			},
		},
		{
			name: "GatewayRoute specified non-empty awsName",
			args: args{
				gr: &appmesh.GatewayRoute{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-gr",
					},
					Spec: appmesh.GatewayRouteSpec{
						AWSName: aws.String("my-gr_awesome-ns_my-cluster"),
					},
				},
			},
			want: &appmesh.GatewayRoute{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "awesome-ns",
					Name:      "my-gr",
				},
				Spec: appmesh.GatewayRouteSpec{
					AWSName: aws.String("my-gr_awesome-ns_my-cluster"),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &gatewayRouteMutator{}
			err := m.defaultingAWSName(tt.args.gr)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, tt.args.gr)
			}
		})
	}
}

func Test_gatewayRouteMutator_designateMeshMembership(t *testing.T) {
	type fields struct {
		meshMembershipDesignatorDesignate func(ctx context.Context, obj metav1.Object) (*appmesh.Mesh, error)
	}
	type args struct {
		gr *appmesh.GatewayRoute
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *appmesh.GatewayRoute
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
				gr: &appmesh.GatewayRoute{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-gr",
					},
					Spec: appmesh.GatewayRouteSpec{},
				},
			},
			want: &appmesh.GatewayRoute{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "awesome-ns",
					Name:      "my-gr",
				},
				Spec: appmesh.GatewayRouteSpec{
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
				gr: &appmesh.GatewayRoute{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-gr",
					},
					Spec: appmesh.GatewayRouteSpec{},
				},
			},
			wantErr: errors.New("oops, some error happened"),
		},
		{
			name:   "meshRef already specified",
			fields: fields{},
			args: args{
				gr: &appmesh.GatewayRoute{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-gr",
					},
					Spec: appmesh.GatewayRouteSpec{
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
					},
				},
			},
			wantErr: errors.New("GatewayRoute create may not specify read-only field: spec.meshRef"),
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

			m := &gatewayRouteMutator{
				meshMembershipDesignator:           designator,
				virtualGatewayMembershipDesignator: nil,
			}
			_, err := m.designateMeshMembership(ctx, tt.args.gr)

			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, tt.args.gr)
			}
		})
	}
}

func Test_gatewayRouteMutator_designateVirtualGatewayMembership(t *testing.T) {
	type fields struct {
		vgMembershipDesignatorDesignate func(ctx context.Context, obj metav1.Object) (*appmesh.VirtualGateway, error)
	}
	type args struct {
		gr *appmesh.GatewayRoute
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *appmesh.GatewayRoute
		wantErr error
	}{
		{
			name: "successfully designate virtualGateway membership",
			fields: fields{
				vgMembershipDesignatorDesignate: func(ctx context.Context, obj metav1.Object) (*appmesh.VirtualGateway, error) {
					return &appmesh.VirtualGateway{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "my-vg",
							Namespace: "gateway-ns",
							UID:       "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
					}, nil
				},
			},
			args: args{
				gr: &appmesh.GatewayRoute{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-gr",
					},
					Spec: appmesh.GatewayRouteSpec{},
				},
			},
			want: &appmesh.GatewayRoute{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "awesome-ns",
					Name:      "my-gr",
				},
				Spec: appmesh.GatewayRouteSpec{
					VirtualGatewayRef: &appmesh.VirtualGatewayReference{
						Name:      "my-vg",
						Namespace: aws.String("gateway-ns"),
						UID:       "408d3036-7dec-11ea-b156-0e30aabe1ca8",
					},
				},
			},
		},
		{
			name: "failed to designate virtualGateway membership",
			fields: fields{
				vgMembershipDesignatorDesignate: func(ctx context.Context, obj metav1.Object) (*appmesh.VirtualGateway, error) {
					return nil, errors.New("oops, some error happened")
				},
			},
			args: args{
				gr: &appmesh.GatewayRoute{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-gr",
					},
					Spec: appmesh.GatewayRouteSpec{},
				},
			},
			wantErr: errors.New("oops, some error happened"),
		},
		{
			name:   "virtualGatewayRef already specified",
			fields: fields{},
			args: args{
				gr: &appmesh.GatewayRoute{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-gr",
					},
					Spec: appmesh.GatewayRouteSpec{
						VirtualGatewayRef: &appmesh.VirtualGatewayReference{
							Name:      "my-vg",
							Namespace: aws.String("gateway-ns"),
							UID:       "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
					},
				},
			},
			wantErr: errors.New("GatewayRoute create may not specify read-only field: spec.virtualGatewayRef"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			designator := mock_virtualgateway.NewMockMembershipDesignator(ctrl)
			if tt.fields.vgMembershipDesignatorDesignate != nil {
				designator.EXPECT().DesignateForGatewayRoute(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.vgMembershipDesignatorDesignate)
			}

			m := &gatewayRouteMutator{
				meshMembershipDesignator:           nil,
				virtualGatewayMembershipDesignator: designator,
			}
			_, err := m.designateVirtualGatewayMembership(ctx, tt.args.gr)

			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, tt.args.gr)
			}
		})
	}
}
