package virtualservice

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"testing"
)

func TestIsVirtualServiceActive(t *testing.T) {
	type args struct {
		vs *appmesh.VirtualService
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "virtualService have true virtualServiceActive condition",
			args: args{
				vs: &appmesh.VirtualService{
					Status: appmesh.VirtualServiceStatus{
						Conditions: []appmesh.VirtualServiceCondition{
							{
								Type:   appmesh.VirtualServiceActive,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "virtualService have false virtualServiceActive condition",
			args: args{
				vs: &appmesh.VirtualService{
					Status: appmesh.VirtualServiceStatus{
						Conditions: []appmesh.VirtualServiceCondition{
							{
								Type:   appmesh.VirtualServiceActive,
								Status: corev1.ConditionFalse,
							},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "virtualService have unknown virtualServiceActive condition",
			args: args{
				vs: &appmesh.VirtualService{
					Status: appmesh.VirtualServiceStatus{
						Conditions: []appmesh.VirtualServiceCondition{
							{
								Type:   appmesh.VirtualServiceActive,
								Status: corev1.ConditionUnknown,
							},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "virtualService doesn't have virtualServiceActive condition",
			args: args{
				vs: &appmesh.VirtualService{
					Status: appmesh.VirtualServiceStatus{
						Conditions: []appmesh.VirtualServiceCondition{},
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsVirtualServiceActive(tt.args.vs)
			assert.Equal(t, tt.want, got)
		})
	}
}
