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

func Test_meshMutator_defaultingIpPreference_IPv6(t *testing.T) {
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
			name: "Mesh didn't specify ipPreference",
			args: args{
				mesh: &appmesh.Mesh{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-mesh",
					},
					Spec: appmesh.MeshSpec{
						AWSName: aws.String("my-mesh"),
					},
				},
			},
			want: &appmesh.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-mesh",
				},
				Spec: appmesh.MeshSpec{
					AWSName: aws.String("my-mesh"),
					ServiceDiscovery: &appmesh.MeshServiceDiscovery{
						IpPreference: aws.String(appmesh.IpPreferenceIPv6),
					},
				},
			},
		},
		{
			name: "Mesh specified non-empty ipPreference",
			args: args{
				mesh: &appmesh.Mesh{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-mesh",
					},
					Spec: appmesh.MeshSpec{
						AWSName: aws.String("my-mesh-for-my-cluster"),
						ServiceDiscovery: &appmesh.MeshServiceDiscovery{
							IpPreference: aws.String(appmesh.IpPreferenceIPv4),
						},
					},
				},
			},
			want: &appmesh.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-mesh",
				},
				Spec: appmesh.MeshSpec{
					AWSName: aws.String("my-mesh-for-my-cluster"),
					ServiceDiscovery: &appmesh.MeshServiceDiscovery{
						IpPreference: aws.String(appmesh.IpPreferenceIPv6),
					},
				},
			},
		},
		{
			name: "Mesh specified non-empty ipPreference",
			args: args{
				mesh: &appmesh.Mesh{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-mesh",
					},
					Spec: appmesh.MeshSpec{
						AWSName: aws.String("my-mesh-for-my-cluster"),
						ServiceDiscovery: &appmesh.MeshServiceDiscovery{
							IpPreference: aws.String(appmesh.IpPreferenceIPv6),
						},
					},
				},
			},
			want: &appmesh.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-mesh",
				},
				Spec: appmesh.MeshSpec{
					AWSName: aws.String("my-mesh-for-my-cluster"),
					ServiceDiscovery: &appmesh.MeshServiceDiscovery{
						IpPreference: aws.String(appmesh.IpPreferenceIPv6),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &meshMutator{
				ipFamily: IPv6,
			}
			err := m.defaultingIpPreference(tt.args.mesh)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, tt.args.mesh)
			}
		})
	}
}

func Test_meshMutator_defaultingIpPreference_IPv4(t *testing.T) {
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
			name: "Mesh didn't specify ipPreference",
			args: args{
				mesh: &appmesh.Mesh{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-mesh",
					},
					Spec: appmesh.MeshSpec{
						AWSName: aws.String("my-mesh"),
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
			name: "Mesh specified non-empty ipPreference",
			args: args{
				mesh: &appmesh.Mesh{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-mesh",
					},
					Spec: appmesh.MeshSpec{
						AWSName: aws.String("my-mesh-for-my-cluster"),
						ServiceDiscovery: &appmesh.MeshServiceDiscovery{
							IpPreference: aws.String(appmesh.IpPreferenceIPv4),
						},
					},
				},
			},
			want: &appmesh.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-mesh",
				},
				Spec: appmesh.MeshSpec{
					AWSName: aws.String("my-mesh-for-my-cluster"),
					ServiceDiscovery: &appmesh.MeshServiceDiscovery{
						IpPreference: aws.String(appmesh.IpPreferenceIPv4),
					},
				},
			},
		},
		{
			name: "Mesh specified non-empty ipPreference",
			args: args{
				mesh: &appmesh.Mesh{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-mesh",
					},
					Spec: appmesh.MeshSpec{
						AWSName: aws.String("my-mesh-for-my-cluster"),
						ServiceDiscovery: &appmesh.MeshServiceDiscovery{
							IpPreference: aws.String(appmesh.IpPreferenceIPv6),
						},
					},
				},
			},
			want: &appmesh.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-mesh",
				},
				Spec: appmesh.MeshSpec{
					AWSName: aws.String("my-mesh-for-my-cluster"),
					ServiceDiscovery: &appmesh.MeshServiceDiscovery{
						IpPreference: aws.String(appmesh.IpPreferenceIPv4),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &meshMutator{
				ipFamily: IPv4,
			}
			err := m.defaultingIpPreference(tt.args.mesh)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, tt.args.mesh)
			}
		})
	}
}
