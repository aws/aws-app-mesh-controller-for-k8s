// +build preview

package gatewayroute

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"testing"
)

func TestIsGatewayRouteActive(t *testing.T) {
	type args struct {
		gr *appmesh.GatewayRoute
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "gatewayRoute has true GatewayRouteActive condition",
			args: args{
				gr: &appmesh.GatewayRoute{
					Status: appmesh.GatewayRouteStatus{
						Conditions: []appmesh.GatewayRouteCondition{
							{
								Type:   appmesh.GatewayRouteActive,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "gatewayRoute has false GatewayRouteActive condition",
			args: args{
				gr: &appmesh.GatewayRoute{
					Status: appmesh.GatewayRouteStatus{
						Conditions: []appmesh.GatewayRouteCondition{
							{
								Type:   appmesh.GatewayRouteActive,
								Status: corev1.ConditionFalse,
							},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "gatewayRoute have unknown GatewayRouteActive condition",
			args: args{
				gr: &appmesh.GatewayRoute{
					Status: appmesh.GatewayRouteStatus{
						Conditions: []appmesh.GatewayRouteCondition{
							{
								Type:   appmesh.GatewayRouteActive,
								Status: corev1.ConditionUnknown,
							},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "gatewayRoute doesn't have GatewayRouteActive condition",
			args: args{
				gr: &appmesh.GatewayRoute{
					Status: appmesh.GatewayRouteStatus{
						Conditions: []appmesh.GatewayRouteCondition{},
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsGatewayRouteActive(tt.args.gr)
			assert.Equal(t, tt.want, got)
		})
	}
}
