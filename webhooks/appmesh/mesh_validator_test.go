package appmesh

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func Test_meshValidator_enforceFieldsImmutability(t *testing.T) {
	type args struct {
		mesh    *appmesh.Mesh
		oldMesh *appmesh.Mesh
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{
			name: "Mesh immutable fields didn't change",
			args: args{
				mesh: &appmesh.Mesh{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-mesh",
					},
					Spec: appmesh.MeshSpec{
						AWSName: aws.String("my-mesh"),
					},
				},
				oldMesh: &appmesh.Mesh{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-mesh",
					},
					Spec: appmesh.MeshSpec{
						AWSName: aws.String("my-mesh"),
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "Mesh field awsName changed",
			args: args{
				mesh: &appmesh.Mesh{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-mesh",
					},
					Spec: appmesh.MeshSpec{
						AWSName: aws.String("my-new-mesh"),
					},
				},
				oldMesh: &appmesh.Mesh{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-mesh",
					},
					Spec: appmesh.MeshSpec{
						AWSName: aws.String("my-mesh"),
					},
				},
			},
			wantErr: errors.New("Mesh update may not change these fields: spec.awsName"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &meshValidator{}
			err := v.enforceFieldsImmutability(tt.args.mesh, tt.args.oldMesh)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
func Test_meshValidator_checkIpPreferenceValues(t *testing.T) {
	type args struct {
		mesh *appmesh.Mesh
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{
			name: "IpPreference is either IPv4_ONLY or IPv6_ONLY",
			args: args{
				mesh: &appmesh.Mesh{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-mesh",
					},
					Spec: appmesh.MeshSpec{
						AWSName: aws.String("my-mesh"),
						MeshServiceDiscovery: &appmesh.MeshServiceDiscovery{
							IpPreference: aws.String(appmesh.IpPreferenceIPv6),
						},
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "IpPreference not specified",
			args: args{
				mesh: &appmesh.Mesh{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-mesh",
					},
					Spec: appmesh.MeshSpec{
						AWSName:              aws.String("my-mesh"),
						MeshServiceDiscovery: &appmesh.MeshServiceDiscovery{},
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "IpPreference is either IPv4_ONLY or IPv6_ONLY if field is non-empty",
			args: args{
				mesh: &appmesh.Mesh{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-mesh",
					},
					Spec: appmesh.MeshSpec{
						AWSName: aws.String("my-mesh"),
						MeshServiceDiscovery: &appmesh.MeshServiceDiscovery{
							IpPreference: aws.String("Test"),
						},
					},
				},
			},
			wantErr: errors.Errorf("Only non-empty values allowed are %s or %s", appmesh.IpPreferenceIPv4, appmesh.IpPreferenceIPv6),
		},
		{
			name: "IpPreference field is an empty string",
			args: args{
				mesh: &appmesh.Mesh{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-mesh",
					},
					Spec: appmesh.MeshSpec{
						AWSName: aws.String("my-mesh"),
						MeshServiceDiscovery: &appmesh.MeshServiceDiscovery{
							IpPreference: aws.String(""),
						},
					},
				},
			},
			wantErr: errors.Errorf("Only non-empty values allowed are %s or %s", appmesh.IpPreferenceIPv4, appmesh.IpPreferenceIPv6),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &meshValidator{}
			err := v.checkIpPreference(tt.args.mesh)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
