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

func Test_virtualNodeMutator_defaultingAWSName(t *testing.T) {
	type args struct {
		vn *appmesh.VirtualNode
	}
	tests := []struct {
		name    string
		args    args
		want    *appmesh.VirtualNode
		wantErr error
	}{
		{
			name: "VirtualNode didn't specify awsName",
			args: args{
				vn: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vn",
					},
					Spec: appmesh.VirtualNodeSpec{},
				},
			},
			want: &appmesh.VirtualNode{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "awesome-ns",
					Name:      "my-vn",
				},
				Spec: appmesh.VirtualNodeSpec{
					AWSName: aws.String("my-vn_awesome-ns"),
				},
			},
		},
		{
			name: "VirtualNode specified empty awsName",
			args: args{
				vn: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vn",
					},
					Spec: appmesh.VirtualNodeSpec{
						AWSName: aws.String(""),
					},
				},
			},
			want: &appmesh.VirtualNode{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "awesome-ns",
					Name:      "my-vn",
				},
				Spec: appmesh.VirtualNodeSpec{
					AWSName: aws.String("my-vn_awesome-ns"),
				},
			},
		},
		{
			name: "VirtualNode specified non-empty awsName",
			args: args{
				vn: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vn",
					},
					Spec: appmesh.VirtualNodeSpec{
						AWSName: aws.String("my-vn_awesome-ns_my-cluster"),
					},
				},
			},
			want: &appmesh.VirtualNode{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "awesome-ns",
					Name:      "my-vn",
				},
				Spec: appmesh.VirtualNodeSpec{
					AWSName: aws.String("my-vn_awesome-ns_my-cluster"),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &virtualNodeMutator{}
			err := m.defaultingAWSName(tt.args.vn)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, tt.args.vn)
			}
		})
	}
}

func Test_virtualNodeMutator_designateMeshMembership(t *testing.T) {
	type fields struct {
		meshMembershipDesignatorDesignate func(ctx context.Context, obj metav1.Object) (*appmesh.Mesh, error)
	}
	type args struct {
		vn *appmesh.VirtualNode
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *appmesh.VirtualNode
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
				vn: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vn",
					},
					Spec: appmesh.VirtualNodeSpec{},
				},
			},
			want: &appmesh.VirtualNode{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "awesome-ns",
					Name:      "my-vn",
				},
				Spec: appmesh.VirtualNodeSpec{
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
				vn: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vn",
					},
					Spec: appmesh.VirtualNodeSpec{},
				},
			},
			wantErr: errors.New("oops, some error happened"),
		},
		{
			name:   "meshRef already specified",
			fields: fields{},
			args: args{
				vn: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vn",
					},
					Spec: appmesh.VirtualNodeSpec{
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
					},
				},
			},
			wantErr: errors.New("VirtualNode create may not specify read-only field: spec.meshRef"),
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

			m := &virtualNodeMutator{
				meshMembershipDesignator: designator,
			}
			err := m.designateMeshMembership(ctx, tt.args.vn)

			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, tt.args.vn)
			}
		})
	}
}

func Test_virtualNodeMutator_defaultingIpPreference_DNS(t *testing.T) {
	type args struct {
		vn *appmesh.VirtualNode
	}
	tests := []struct {
		name    string
		args    args
		want    *appmesh.VirtualNode
		wantErr error
	}{
		{
			name: "VirtualNode DNS ServiceDiscovery didn't specify ipPreference",
			args: args{
				vn: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vn",
					},
					Spec: appmesh.VirtualNodeSpec{
						AWSName: aws.String("my-vn_awesome-ns"),
						ServiceDiscovery: &appmesh.ServiceDiscovery{
							DNS: &appmesh.DNSServiceDiscovery{
								Hostname: "hostname.internal",
							},
						},
					},
				},
			},
			want: &appmesh.VirtualNode{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "awesome-ns",
					Name:      "my-vn",
				},
				Spec: appmesh.VirtualNodeSpec{
					AWSName: aws.String("my-vn_awesome-ns"),
					ServiceDiscovery: &appmesh.ServiceDiscovery{
						DNS: &appmesh.DNSServiceDiscovery{
							Hostname:     "hostname.internal",
							IpPreference: aws.String(appmesh.IpPreferenceIPv4),
						},
					},
				},
			},
		},
		{
			name: "VirtualNode specified non-empty ipPreference",
			args: args{
				vn: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vn",
					},
					Spec: appmesh.VirtualNodeSpec{
						AWSName: aws.String("my-vn_awesome-ns_my-cluster"),
						ServiceDiscovery: &appmesh.ServiceDiscovery{
							DNS: &appmesh.DNSServiceDiscovery{
								Hostname:     "hostname.internal",
								IpPreference: aws.String(appmesh.IpPreferenceIPv6),
							},
						},
					},
				},
			},
			want: &appmesh.VirtualNode{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "awesome-ns",
					Name:      "my-vn",
				},
				Spec: appmesh.VirtualNodeSpec{
					AWSName: aws.String("my-vn_awesome-ns_my-cluster"),
					ServiceDiscovery: &appmesh.ServiceDiscovery{
						DNS: &appmesh.DNSServiceDiscovery{
							Hostname:     "hostname.internal",
							IpPreference: aws.String(appmesh.IpPreferenceIPv6),
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &virtualNodeMutator{
				ipFamily: appmesh.IpPreferenceIPv4,
			}
			err := m.defaultingIpPreference(tt.args.vn)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, tt.args.vn)
			}
		})
	}
}

func Test_virtualNodeMutator_defaultingIpPreference_AWSCloudMap(t *testing.T) {
	type args struct {
		vn *appmesh.VirtualNode
	}
	tests := []struct {
		name    string
		args    args
		want    *appmesh.VirtualNode
		wantErr error
	}{
		{
			name: "VirtualNode DNS ServiceDiscovery didn't specify ipPreference",
			args: args{
				vn: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vn",
					},
					Spec: appmesh.VirtualNodeSpec{
						AWSName: aws.String("my-vn_awesome-ns"),
						ServiceDiscovery: &appmesh.ServiceDiscovery{
							AWSCloudMap: &appmesh.AWSCloudMapServiceDiscovery{
								NamespaceName: "namespace",
							},
						},
					},
				},
			},
			want: &appmesh.VirtualNode{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "awesome-ns",
					Name:      "my-vn",
				},
				Spec: appmesh.VirtualNodeSpec{
					AWSName: aws.String("my-vn_awesome-ns"),
					ServiceDiscovery: &appmesh.ServiceDiscovery{
						AWSCloudMap: &appmesh.AWSCloudMapServiceDiscovery{
							NamespaceName: "namespace",
							IpPreference:  aws.String(appmesh.IpPreferenceIPv4),
						},
					},
				},
			},
		},
		{
			name: "VirtualNode specified non-empty ipPreference",
			args: args{
				vn: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vn",
					},
					Spec: appmesh.VirtualNodeSpec{
						AWSName: aws.String("my-vn_awesome-ns_my-cluster"),
						ServiceDiscovery: &appmesh.ServiceDiscovery{
							AWSCloudMap: &appmesh.AWSCloudMapServiceDiscovery{
								NamespaceName: "namespace",
								IpPreference:  aws.String(appmesh.IpPreferenceIPv6),
							},
						},
					},
				},
			},
			want: &appmesh.VirtualNode{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "awesome-ns",
					Name:      "my-vn",
				},
				Spec: appmesh.VirtualNodeSpec{
					AWSName: aws.String("my-vn_awesome-ns_my-cluster"),
					ServiceDiscovery: &appmesh.ServiceDiscovery{
						AWSCloudMap: &appmesh.AWSCloudMapServiceDiscovery{
							NamespaceName: "namespace",
							IpPreference:  aws.String(appmesh.IpPreferenceIPv6),
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &virtualNodeMutator{
				ipFamily: appmesh.IpPreferenceIPv4,
			}
			err := m.defaultingIpPreference(tt.args.vn)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, tt.args.vn)
			}
		})
	}
}
