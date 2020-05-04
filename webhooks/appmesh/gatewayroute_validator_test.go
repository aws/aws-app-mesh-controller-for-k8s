package appmesh

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func Test_gatewayRouteValidator_enforceFieldsImmutability(t *testing.T) {
	type args struct {
		newGR *appmesh.GatewayRoute
		oldGR *appmesh.GatewayRoute
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{
			name: "GatewayRoute immutable fields didn't change",
			args: args{
				newGR: &appmesh.GatewayRoute{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-gr",
					},
					Spec: appmesh.GatewayRouteSpec{
						AWSName: aws.String("my-gr_awesome-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
						VirtualGatewayRef: &appmesh.VirtualGatewayReference{
							Name:      "my-vg",
							Namespace: aws.String("gateway-ns"),
							UID:       "346d3036-7dec-11ea-b678-0e30aabe1dg2",
						},
					},
				},
				oldGR: &appmesh.GatewayRoute{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-gr",
					},
					Spec: appmesh.GatewayRouteSpec{
						AWSName: aws.String("my-gr_awesome-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
						VirtualGatewayRef: &appmesh.VirtualGatewayReference{
							Name:      "my-vg",
							Namespace: aws.String("gateway-ns"),
							UID:       "346d3036-7dec-11ea-b678-0e30aabe1dg2",
						},
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "GatewayRoute field awsName changed",
			args: args{
				newGR: &appmesh.GatewayRoute{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-gr",
					},
					Spec: appmesh.GatewayRouteSpec{
						AWSName: aws.String("my-gr_awesome-ns_my-cluster"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
						VirtualGatewayRef: &appmesh.VirtualGatewayReference{
							Name:      "my-vg",
							Namespace: aws.String("gateway-ns"),
							UID:       "346d3036-7dec-11ea-b678-0e30aabe1dg2",
						},
					},
				},
				oldGR: &appmesh.GatewayRoute{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-gr",
					},
					Spec: appmesh.GatewayRouteSpec{
						AWSName: aws.String("my-gr_awesome-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
						VirtualGatewayRef: &appmesh.VirtualGatewayReference{
							Name:      "my-vg",
							Namespace: aws.String("gateway-ns"),
							UID:       "346d3036-7dec-11ea-b678-0e30aabe1dg2",
						},
					},
				},
			},
			wantErr: errors.New("GatewayRoute update may not change these fields: spec.awsName"),
		},
		{
			name: "GatewayRoute field meshRef changed",
			args: args{
				newGR: &appmesh.GatewayRoute{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-gr",
					},
					Spec: appmesh.GatewayRouteSpec{
						AWSName: aws.String("my-gr_awesome-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "another-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
						VirtualGatewayRef: &appmesh.VirtualGatewayReference{
							Name:      "my-vg",
							Namespace: aws.String("gateway-ns"),
							UID:       "346d3036-7dec-11ea-b678-0e30aabe1dg2",
						},
					},
				},
				oldGR: &appmesh.GatewayRoute{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-gr",
					},
					Spec: appmesh.GatewayRouteSpec{
						AWSName: aws.String("my-gr_awesome-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
						VirtualGatewayRef: &appmesh.VirtualGatewayReference{
							Name:      "my-vg",
							Namespace: aws.String("gateway-ns"),
							UID:       "346d3036-7dec-11ea-b678-0e30aabe1dg2",
						},
					},
				},
			},
			wantErr: errors.New("GatewayRoute update may not change these fields: spec.meshRef"),
		},
		{
			name: "GatewayRoute field virtualGatewayRef changed",
			args: args{
				newGR: &appmesh.GatewayRoute{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-gr",
					},
					Spec: appmesh.GatewayRouteSpec{
						AWSName: aws.String("my-gr_awesome-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
						VirtualGatewayRef: &appmesh.VirtualGatewayReference{
							Name:      "another-vg",
							Namespace: aws.String("gateway-ns"),
							UID:       "346d3036-7dec-11ea-b678-0e30aabe1dg2",
						},
					},
				},
				oldGR: &appmesh.GatewayRoute{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-gr",
					},
					Spec: appmesh.GatewayRouteSpec{
						AWSName: aws.String("my-gr_awesome-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
						VirtualGatewayRef: &appmesh.VirtualGatewayReference{
							Name:      "my-vg",
							Namespace: aws.String("gateway-ns"),
							UID:       "346d3036-7dec-11ea-b678-0e30aabe1dg2",
						},
					},
				},
			},
			wantErr: errors.New("GatewayRoute update may not change these fields: spec.virtualGatewayRef"),
		},
		{
			name: "GatewayRoute fields awsName, meshRef and virtualGatewayRef changed",
			args: args{
				newGR: &appmesh.GatewayRoute{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-gr",
					},
					Spec: appmesh.GatewayRouteSpec{
						AWSName: aws.String("my-gr_awesome-ns-my-cluster"),
						MeshRef: &appmesh.MeshReference{
							Name: "another-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
						VirtualGatewayRef: &appmesh.VirtualGatewayReference{
							Name:      "another-vg",
							Namespace: aws.String("gateway-ns"),
							UID:       "346d3036-7dec-11ea-b678-0e30aabe1dg2",
						},
					},
				},
				oldGR: &appmesh.GatewayRoute{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-gr",
					},
					Spec: appmesh.GatewayRouteSpec{
						AWSName: aws.String("my-gr_awesome-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
						VirtualGatewayRef: &appmesh.VirtualGatewayReference{
							Name:      "my-vg",
							Namespace: aws.String("gateway-ns"),
							UID:       "346d3036-7dec-11ea-b678-0e30aabe1dg2",
						},
					},
				},
			},
			wantErr: errors.New("GatewayRoute update may not change these fields: spec.awsName,spec.meshRef,spec.virtualGatewayRef"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &gatewayRouteValidator{}
			err := v.enforceFieldsImmutability(tt.args.newGR, tt.args.oldGR)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
