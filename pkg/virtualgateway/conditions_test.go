// +build preview

package virtualgateway

import (
	"github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func Test_getCondition(t *testing.T) {
	type args struct {
		vg            *appmesh.VirtualGateway
		conditionType v1beta2.VirtualGatewayConditionType
	}
	tests := []struct {
		name string
		args args
		want *appmesh.VirtualGatewayCondition
	}{
		{
			name: "condition found",
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
				conditionType: appmesh.VirtualGatewayActive,
			},
			want: &appmesh.VirtualGatewayCondition{
				Type:   appmesh.VirtualGatewayActive,
				Status: corev1.ConditionFalse,
			},
		},
		{
			name: "condition not found",
			args: args{
				vg: &appmesh.VirtualGateway{
					Status: appmesh.VirtualGatewayStatus{
						Conditions: []appmesh.VirtualGatewayCondition{},
					},
				},
				conditionType: appmesh.VirtualGatewayActive,
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getCondition(tt.args.vg, tt.args.conditionType)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_updateCondition(t *testing.T) {
	type args struct {
		vg            *appmesh.VirtualGateway
		conditionType appmesh.VirtualGatewayConditionType
		status        corev1.ConditionStatus
		reason        *string
		message       *string
	}
	tests := []struct {
		name        string
		args        args
		wantVG      *appmesh.VirtualGateway
		wantChanged bool
	}{
		{
			name: "condition updated by modify condition status",
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
				conditionType: appmesh.VirtualGatewayActive,
				status:        corev1.ConditionTrue,
			},
			wantVG: &appmesh.VirtualGateway{
				Status: appmesh.VirtualGatewayStatus{
					Conditions: []appmesh.VirtualGatewayCondition{
						{
							Type:   appmesh.VirtualGatewayActive,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
			wantChanged: true,
		},
		{
			name: "condition updated by modify condition reason",
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
				conditionType: appmesh.VirtualGatewayActive,
				status:        corev1.ConditionTrue,
				reason:        aws.String("reason"),
			},
			wantVG: &appmesh.VirtualGateway{
				Status: appmesh.VirtualGatewayStatus{
					Conditions: []appmesh.VirtualGatewayCondition{
						{
							Type:   appmesh.VirtualGatewayActive,
							Status: corev1.ConditionTrue,
							Reason: aws.String("reason"),
						},
					},
				},
			},
			wantChanged: true,
		},
		{
			name: "condition updated by modify condition message",
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
				conditionType: appmesh.VirtualGatewayActive,
				status:        corev1.ConditionTrue,
				message:       aws.String("message"),
			},
			wantVG: &appmesh.VirtualGateway{
				Status: appmesh.VirtualGatewayStatus{
					Conditions: []appmesh.VirtualGatewayCondition{
						{
							Type:    appmesh.VirtualGatewayActive,
							Status:  corev1.ConditionTrue,
							Message: aws.String("message"),
						},
					},
				},
			},
			wantChanged: true,
		},
		{
			name: "condition updated by modify condition status/reason/message",
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
				conditionType: appmesh.VirtualGatewayActive,
				status:        corev1.ConditionTrue,
				reason:        aws.String("reason"),
				message:       aws.String("message"),
			},
			wantVG: &appmesh.VirtualGateway{
				Status: appmesh.VirtualGatewayStatus{
					Conditions: []appmesh.VirtualGatewayCondition{
						{
							Type:    appmesh.VirtualGatewayActive,
							Status:  corev1.ConditionTrue,
							Reason:  aws.String("reason"),
							Message: aws.String("message"),
						},
					},
				},
			},
			wantChanged: true,
		},
		{
			name: "condition updated by new condition",
			args: args{
				vg: &appmesh.VirtualGateway{
					Status: appmesh.VirtualGatewayStatus{
						Conditions: nil,
					},
				},
				conditionType: appmesh.VirtualGatewayActive,
				status:        corev1.ConditionTrue,
				reason:        aws.String("reason"),
				message:       aws.String("message"),
			},
			wantVG: &appmesh.VirtualGateway{
				Status: appmesh.VirtualGatewayStatus{
					Conditions: []appmesh.VirtualGatewayCondition{
						{
							Type:    appmesh.VirtualGatewayActive,
							Status:  corev1.ConditionTrue,
							Reason:  aws.String("reason"),
							Message: aws.String("message"),
						},
					},
				},
			},
			wantChanged: true,
		},
		{
			name: "condition unmodified",
			args: args{
				vg: &appmesh.VirtualGateway{
					Status: appmesh.VirtualGatewayStatus{
						Conditions: []appmesh.VirtualGatewayCondition{
							{
								Type:    appmesh.VirtualGatewayActive,
								Status:  corev1.ConditionTrue,
								Reason:  aws.String("reason"),
								Message: aws.String("message"),
							},
						},
					},
				},
				conditionType: appmesh.VirtualGatewayActive,
				status:        corev1.ConditionTrue,
				reason:        aws.String("reason"),
				message:       aws.String("message"),
			},
			wantVG: &appmesh.VirtualGateway{
				Status: appmesh.VirtualGatewayStatus{
					Conditions: []appmesh.VirtualGatewayCondition{
						{
							Type:    appmesh.VirtualGatewayActive,
							Status:  corev1.ConditionTrue,
							Reason:  aws.String("reason"),
							Message: aws.String("message"),
						},
					},
				},
			},
			wantChanged: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotChanged := updateCondition(tt.args.vg, tt.args.conditionType, tt.args.status, tt.args.reason, tt.args.message)
			opts := cmpopts.IgnoreTypes((*metav1.Time)(nil))
			assert.True(t, cmp.Equal(tt.wantVG, tt.args.vg, opts), "diff", cmp.Diff(tt.wantVG, tt.args.vg, opts))
			assert.Equal(t, tt.wantChanged, gotChanged)
		})
	}
}
