package virtualgateway

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestIsVirtualGatewayActive(t *testing.T) {
	type args struct {
		vg *appmesh.VirtualGateway
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "virtualGateway has true VirtualGatewayActive condition",
			args: args{
				vg: &appmesh.VirtualGateway{
					Status: appmesh.VirtualGatewayStatus{
						Conditions: []appmesh.VirtualGatewayCondition{
							{
								Type:   appmesh.VirtualGatewayActive,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "virtualGateway has false VirtualGatewayActive condition",
			args: args{
				vg: &appmesh.VirtualGateway{
					Status: appmesh.VirtualGatewayStatus{
						Conditions: []appmesh.VirtualGatewayCondition{
							{
								Type:   appmesh.VirtualGatewayActive,
								Status: corev1.ConditionFalse,
							},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "virtualGateway have unknown VirtualGatewayActive condition",
			args: args{
				vg: &appmesh.VirtualGateway{
					Status: appmesh.VirtualGatewayStatus{
						Conditions: []appmesh.VirtualGatewayCondition{
							{
								Type:   appmesh.VirtualGatewayActive,
								Status: corev1.ConditionUnknown,
							},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "virtualGateway doesn't have VirtualGatewayActive condition",
			args: args{
				vg: &appmesh.VirtualGateway{
					Status: appmesh.VirtualGatewayStatus{
						Conditions: []appmesh.VirtualGatewayCondition{},
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsVirtualGatewayActive(tt.args.vg)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsVirtualGatewayReferenced(t *testing.T) {
	type args struct {
		vg        *appmesh.VirtualGateway
		reference appmesh.VirtualGatewayReference
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "virtualGateway is referenced when name, namespace and UID matches",
			args: args{
				vg: &appmesh.VirtualGateway{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-vg",
						Namespace: "my-ns",
						UID:       "a385048d-aba8-4235-9a11-4173764c8ab7",
					},
				},
				reference: appmesh.VirtualGatewayReference{
					Name:      "my-vg",
					Namespace: aws.String("my-ns"),
					UID:       "a385048d-aba8-4235-9a11-4173764c8ab7",
				},
			},
			want: true,
		},
		{
			name: "virtualGateway is not referenced when name mismatches",
			args: args{
				vg: &appmesh.VirtualGateway{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-vg",
						Namespace: "my-ns",
						UID:       "a385048d-aba8-4235-9a11-4173764c8ab7",
					},
				},
				reference: appmesh.VirtualGatewayReference{
					Name:      "another-vg",
					Namespace: aws.String("my-ns"),
					UID:       "a385048d-aba8-4235-9a11-4173764c8ab7",
				},
			},
			want: false,
		},
		{
			name: "virtualGateway is not referenced when UID mismatches",
			args: args{
				vg: &appmesh.VirtualGateway{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-vg",
						Namespace: "my-ns",
						UID:       "a385048d-aba8-4235-9a11-4173764c8ab7",
					},
				},
				reference: appmesh.VirtualGatewayReference{
					Name:      "my-vg",
					Namespace: aws.String("my-ns"),
					UID:       "f7d10a22-e8d5-4626-b780-261374fc68d4",
				},
			},
			want: false,
		},
		{
			name: "virtualGateway is not referenced when Namespace mismatches",
			args: args{
				vg: &appmesh.VirtualGateway{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-vg",
						Namespace: "my-ns",
						UID:       "a385048d-aba8-4235-9a11-4173764c8ab7",
					},
				},
				reference: appmesh.VirtualGatewayReference{
					Name:      "my-vg",
					Namespace: aws.String("other-ns"),
					UID:       "f7d10a22-e8d5-4626-b780-261374fc68d4",
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsVirtualGatewayReferenced(tt.args.vg, tt.args.reference); got != tt.want {
				t.Errorf("IsVirtualGatewayReferenced() = %v, want %v", got, tt.want)
			}
		})
	}
}
