package virtualnode

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	"testing"
)

func TestIsVirtualNodeActive(t *testing.T) {
	type args struct {
		vn *appmesh.VirtualNode
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "virtualNode have true VirtualNodeActive condition",
			args: args{
				vn: &appmesh.VirtualNode{
					Status: appmesh.VirtualNodeStatus{
						Conditions: []appmesh.VirtualNodeCondition{
							{
								Type:   appmesh.VirtualNodeActive,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "virtualNode have false VirtualNodeActive condition",
			args: args{
				vn: &appmesh.VirtualNode{
					Status: appmesh.VirtualNodeStatus{
						Conditions: []appmesh.VirtualNodeCondition{
							{
								Type:   appmesh.VirtualNodeActive,
								Status: corev1.ConditionFalse,
							},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "virtualNode have unknown VirtualNodeActive condition",
			args: args{
				vn: &appmesh.VirtualNode{
					Status: appmesh.VirtualNodeStatus{
						Conditions: []appmesh.VirtualNodeCondition{
							{
								Type:   appmesh.VirtualNodeActive,
								Status: corev1.ConditionUnknown,
							},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "virtualNode doesn't have VirtualNodeActive condition",
			args: args{
				vn: &appmesh.VirtualNode{
					Status: appmesh.VirtualNodeStatus{
						Conditions: []appmesh.VirtualNodeCondition{},
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsVirtualNodeActive(tt.args.vn)
			assert.Equal(t, tt.want, got)
		})
	}
}
