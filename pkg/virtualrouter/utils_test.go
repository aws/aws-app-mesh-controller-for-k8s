package virtualrouter

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"testing"
)

func TestIsVirtualRouterActive(t *testing.T) {
	type args struct {
		vr *appmesh.VirtualRouter
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "virtualRouter have true VirtualRouterActive condition",
			args: args{
				vr: &appmesh.VirtualRouter{
					Status: appmesh.VirtualRouterStatus{
						Conditions: []appmesh.VirtualRouterCondition{
							{
								Type:   appmesh.VirtualRouterActive,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "virtualRouter have false VirtualRouterActive condition",
			args: args{
				vr: &appmesh.VirtualRouter{
					Status: appmesh.VirtualRouterStatus{
						Conditions: []appmesh.VirtualRouterCondition{
							{
								Type:   appmesh.VirtualRouterActive,
								Status: corev1.ConditionFalse,
							},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "virtualRouter have unknown VirtualRouterActive condition",
			args: args{
				vr: &appmesh.VirtualRouter{
					Status: appmesh.VirtualRouterStatus{
						Conditions: []appmesh.VirtualRouterCondition{
							{
								Type:   appmesh.VirtualRouterActive,
								Status: corev1.ConditionUnknown,
							},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "virtualRouter doesn't have VirtualRouterActive condition",
			args: args{
				vr: &appmesh.VirtualRouter{
					Status: appmesh.VirtualRouterStatus{
						Conditions: []appmesh.VirtualRouterCondition{},
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsVirtualRouterActive(tt.args.vr)
			assert.Equal(t, tt.want, got)
		})
	}
}
