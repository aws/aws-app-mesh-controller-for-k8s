package appmesh

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func Test_virtualGatewayValidator_enforceFieldsImmutability(t *testing.T) {
	type args struct {
		newVGateway *appmesh.VirtualGateway
		oldVGateway *appmesh.VirtualGateway
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{
			name: "VirtualGateway immutable fields didn't change",
			args: args{
				newVGateway: &appmesh.VirtualGateway{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "app-ns",
						Name:      "my-vg",
					},
					Spec: appmesh.VirtualGatewaySpec{
						AWSName: aws.String("my-vg_app-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
					},
				},
				oldVGateway: &appmesh.VirtualGateway{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "app-ns",
						Name:      "my-vg",
					},
					Spec: appmesh.VirtualGatewaySpec{
						AWSName: aws.String("my-vg_app-ns"),
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
			name: "VirtualGateway field awsName changed",
			args: args{
				newVGateway: &appmesh.VirtualGateway{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "app-ns",
						Name:      "my-vg",
					},
					Spec: appmesh.VirtualGatewaySpec{
						AWSName: aws.String("my-vg_app-ns_my-cluster"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
					},
				},
				oldVGateway: &appmesh.VirtualGateway{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "app-ns",
						Name:      "my-vg",
					},
					Spec: appmesh.VirtualGatewaySpec{
						AWSName: aws.String("my-vg_app-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
					},
				},
			},
			wantErr: errors.New("VirtualGateway update may not change these fields: spec.awsName"),
		},
		{
			name: "VirtualGateway field meshRef changed",
			args: args{
				newVGateway: &appmesh.VirtualGateway{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "app-ns",
						Name:      "my-vg",
					},
					Spec: appmesh.VirtualGatewaySpec{
						AWSName: aws.String("my-vg_app-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "another-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
					},
				},
				oldVGateway: &appmesh.VirtualGateway{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "app-ns",
						Name:      "my-vg",
					},
					Spec: appmesh.VirtualGatewaySpec{
						AWSName: aws.String("my-vg_app-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
					},
				},
			},
			wantErr: errors.New("VirtualGateway update may not change these fields: spec.meshRef"),
		},
		{
			name: "VirtualGateway fields awsName and meshRef changed",
			args: args{
				newVGateway: &appmesh.VirtualGateway{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "app-ns",
						Name:      "my-vg",
					},
					Spec: appmesh.VirtualGatewaySpec{
						AWSName: aws.String("my-vg_app-ns_my-cluster"),
						MeshRef: &appmesh.MeshReference{
							Name: "another-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
					},
				},
				oldVGateway: &appmesh.VirtualGateway{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "app-ns",
						Name:      "my-vg",
					},
					Spec: appmesh.VirtualGatewaySpec{
						AWSName: aws.String("my-vg_app-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
					},
				},
			},
			wantErr: errors.New("VirtualGateway update may not change these fields: spec.awsName,spec.meshRef"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &virtualGatewayValidator{}
			err := v.enforceFieldsImmutability(tt.args.newVGateway, tt.args.oldVGateway)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_virtualGatewayValidator_checkForConnectionPoolProtocols(t *testing.T) {
	type args struct {
		vg *appmesh.VirtualGateway
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{
			name: "Virtual gateway listener with one connection pool type",
			args: args{
				vg: &appmesh.VirtualGateway{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "app-ns",
						Name:      "my-vg",
					},
					Spec: appmesh.VirtualGatewaySpec{
						AWSName: aws.String("my-vg_app-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
						Listeners: []appmesh.VirtualGatewayListener{
							{
								PortMapping: appmesh.VirtualGatewayPortMapping{
									Port:     8080,
									Protocol: "http",
								},
								ConnectionPool: &appmesh.VirtualGatewayConnectionPool{
									HTTP: &appmesh.HTTPConnectionPool{
										MaxConnections:     100,
										MaxPendingRequests: aws.Int64(30),
									},
								},
							},
						},
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "Virtual gateway listener with HTTP and HTTP2 connection pool",
			args: args{
				vg: &appmesh.VirtualGateway{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "app-ns",
						Name:      "my-vg",
					},
					Spec: appmesh.VirtualGatewaySpec{
						AWSName: aws.String("my-vg_app-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
						Listeners: []appmesh.VirtualGatewayListener{
							{
								PortMapping: appmesh.VirtualGatewayPortMapping{
									Port:     8080,
									Protocol: "http",
								},
								ConnectionPool: &appmesh.VirtualGatewayConnectionPool{
									HTTP: &appmesh.HTTPConnectionPool{
										MaxConnections:     100,
										MaxPendingRequests: aws.Int64(30),
									},
									HTTP2: &appmesh.HTTP2ConnectionPool{
										MaxRequests: 30,
									},
								},
							},
						},
					},
				},
			},
			wantErr: errors.New("Only one type of Virtual Gateway Connection Pool is allowed"),
		},
		{
			name: "Virtual gateway listener with HTTP, GRPC and HTTP2 connection pool",
			args: args{
				vg: &appmesh.VirtualGateway{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "app-ns",
						Name:      "my-vg",
					},
					Spec: appmesh.VirtualGatewaySpec{
						AWSName: aws.String("my-vg_app-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
						Listeners: []appmesh.VirtualGatewayListener{
							{
								PortMapping: appmesh.VirtualGatewayPortMapping{
									Port:     8080,
									Protocol: "http",
								},
								ConnectionPool: &appmesh.VirtualGatewayConnectionPool{
									HTTP: &appmesh.HTTPConnectionPool{
										MaxConnections:     100,
										MaxPendingRequests: aws.Int64(30),
									},
									HTTP2: &appmesh.HTTP2ConnectionPool{
										MaxRequests: 30,
									},
									GRPC: &appmesh.GRPCConnectionPool{
										MaxRequests: 30,
									},
								},
							},
						},
					},
				},
			},
			wantErr: errors.New("Only one type of Virtual Gateway Connection Pool is allowed"),
		},
		{
			name: "Virtual gateway listener with no connection pools",
			args: args{
				vg: &appmesh.VirtualGateway{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "app-ns",
						Name:      "my-vg",
					},
					Spec: appmesh.VirtualGatewaySpec{
						AWSName: aws.String("my-vg_app-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
						Listeners: []appmesh.VirtualGatewayListener{
							{
								PortMapping: appmesh.VirtualGatewayPortMapping{
									Port:     8080,
									Protocol: "http",
								},
							},
						},
					},
				},
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &virtualGatewayValidator{}
			err := v.checkForConnectionPoolProtocols(tt.args.vg)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}

}
