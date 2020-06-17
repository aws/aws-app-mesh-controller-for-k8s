// +build preview

package gatewayroute

import (
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
		gr            *appmesh.GatewayRoute
		conditionType appmesh.GatewayRouteConditionType
	}
	tests := []struct {
		name string
		args args
		want *appmesh.GatewayRouteCondition
	}{
		{
			name: "condition found",
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
				conditionType: appmesh.GatewayRouteActive,
			},
			want: &appmesh.GatewayRouteCondition{
				Type:   appmesh.GatewayRouteActive,
				Status: corev1.ConditionFalse,
			},
		},
		{
			name: "condition not found",
			args: args{
				gr: &appmesh.GatewayRoute{
					Status: appmesh.GatewayRouteStatus{
						Conditions: []appmesh.GatewayRouteCondition{},
					},
				},
				conditionType: appmesh.GatewayRouteActive,
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getCondition(tt.args.gr, tt.args.conditionType)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_updateCondition(t *testing.T) {
	type args struct {
		gr            *appmesh.GatewayRoute
		conditionType appmesh.GatewayRouteConditionType
		status        corev1.ConditionStatus
		reason        *string
		message       *string
	}
	tests := []struct {
		name        string
		args        args
		wantGR      *appmesh.GatewayRoute
		wantChanged bool
	}{
		{
			name: "condition updated by modify condition status",
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
				conditionType: appmesh.GatewayRouteActive,
				status:        corev1.ConditionTrue,
			},
			wantGR: &appmesh.GatewayRoute{
				Status: appmesh.GatewayRouteStatus{
					Conditions: []appmesh.GatewayRouteCondition{
						{
							Type:   appmesh.GatewayRouteActive,
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
				conditionType: appmesh.GatewayRouteActive,
				status:        corev1.ConditionTrue,
				reason:        aws.String("reason"),
			},
			wantGR: &appmesh.GatewayRoute{
				Status: appmesh.GatewayRouteStatus{
					Conditions: []appmesh.GatewayRouteCondition{
						{
							Type:   appmesh.GatewayRouteActive,
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
				conditionType: appmesh.GatewayRouteActive,
				status:        corev1.ConditionTrue,
				message:       aws.String("message"),
			},
			wantGR: &appmesh.GatewayRoute{
				Status: appmesh.GatewayRouteStatus{
					Conditions: []appmesh.GatewayRouteCondition{
						{
							Type:    appmesh.GatewayRouteActive,
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
				conditionType: appmesh.GatewayRouteActive,
				status:        corev1.ConditionTrue,
				reason:        aws.String("reason"),
				message:       aws.String("message"),
			},
			wantGR: &appmesh.GatewayRoute{
				Status: appmesh.GatewayRouteStatus{
					Conditions: []appmesh.GatewayRouteCondition{
						{
							Type:    appmesh.GatewayRouteActive,
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
				gr: &appmesh.GatewayRoute{
					Status: appmesh.GatewayRouteStatus{
						Conditions: nil,
					},
				},
				conditionType: appmesh.GatewayRouteActive,
				status:        corev1.ConditionTrue,
				reason:        aws.String("reason"),
				message:       aws.String("message"),
			},
			wantGR: &appmesh.GatewayRoute{
				Status: appmesh.GatewayRouteStatus{
					Conditions: []appmesh.GatewayRouteCondition{
						{
							Type:    appmesh.GatewayRouteActive,
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
				gr: &appmesh.GatewayRoute{
					Status: appmesh.GatewayRouteStatus{
						Conditions: []appmesh.GatewayRouteCondition{
							{
								Type:    appmesh.GatewayRouteActive,
								Status:  corev1.ConditionTrue,
								Reason:  aws.String("reason"),
								Message: aws.String("message"),
							},
						},
					},
				},
				conditionType: appmesh.GatewayRouteActive,
				status:        corev1.ConditionTrue,
				reason:        aws.String("reason"),
				message:       aws.String("message"),
			},
			wantGR: &appmesh.GatewayRoute{
				Status: appmesh.GatewayRouteStatus{
					Conditions: []appmesh.GatewayRouteCondition{
						{
							Type:    appmesh.GatewayRouteActive,
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
			gotChanged := updateCondition(tt.args.gr, tt.args.conditionType, tt.args.status, tt.args.reason, tt.args.message)
			opts := cmpopts.IgnoreTypes((*metav1.Time)(nil))
			assert.True(t, cmp.Equal(tt.wantGR, tt.args.gr, opts), "diff", cmp.Diff(tt.wantGR, tt.args.gr, opts))
			assert.Equal(t, tt.wantChanged, gotChanged)
		})
	}
}
