package virtualrouter

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
		vr            *appmesh.VirtualRouter
		conditionType appmesh.VirtualRouterConditionType
	}
	tests := []struct {
		name string
		args args
		want *appmesh.VirtualRouterCondition
	}{
		{
			name: "condition found",
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
				conditionType: appmesh.VirtualRouterActive,
			},
			want: &appmesh.VirtualRouterCondition{
				Type:   appmesh.VirtualRouterActive,
				Status: corev1.ConditionFalse,
			},
		},
		{
			name: "condition not found",
			args: args{
				vr: &appmesh.VirtualRouter{
					Status: appmesh.VirtualRouterStatus{
						Conditions: []appmesh.VirtualRouterCondition{},
					},
				},
				conditionType: appmesh.VirtualRouterActive,
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getCondition(tt.args.vr, tt.args.conditionType)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_updateCondition(t *testing.T) {
	type args struct {
		vr            *appmesh.VirtualRouter
		conditionType appmesh.VirtualRouterConditionType
		status        corev1.ConditionStatus
		reason        *string
		message       *string
	}
	tests := []struct {
		name        string
		args        args
		wantVR      *appmesh.VirtualRouter
		wantChanged bool
	}{
		{
			name: "condition updated by modify condition status",
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
				conditionType: appmesh.VirtualRouterActive,
				status:        corev1.ConditionTrue,
			},
			wantVR: &appmesh.VirtualRouter{
				Status: appmesh.VirtualRouterStatus{
					Conditions: []appmesh.VirtualRouterCondition{
						{
							Type:   appmesh.VirtualRouterActive,
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
				conditionType: appmesh.VirtualRouterActive,
				status:        corev1.ConditionTrue,
				reason:        aws.String("reason"),
			},
			wantVR: &appmesh.VirtualRouter{
				Status: appmesh.VirtualRouterStatus{
					Conditions: []appmesh.VirtualRouterCondition{
						{
							Type:   appmesh.VirtualRouterActive,
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
				conditionType: appmesh.VirtualRouterActive,
				status:        corev1.ConditionTrue,
				message:       aws.String("message"),
			},
			wantVR: &appmesh.VirtualRouter{
				Status: appmesh.VirtualRouterStatus{
					Conditions: []appmesh.VirtualRouterCondition{
						{
							Type:    appmesh.VirtualRouterActive,
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
				conditionType: appmesh.VirtualRouterActive,
				status:        corev1.ConditionTrue,
				reason:        aws.String("reason"),
				message:       aws.String("message"),
			},
			wantVR: &appmesh.VirtualRouter{
				Status: appmesh.VirtualRouterStatus{
					Conditions: []appmesh.VirtualRouterCondition{
						{
							Type:    appmesh.VirtualRouterActive,
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
				vr: &appmesh.VirtualRouter{
					Status: appmesh.VirtualRouterStatus{
						Conditions: nil,
					},
				},
				conditionType: appmesh.VirtualRouterActive,
				status:        corev1.ConditionTrue,
				reason:        aws.String("reason"),
				message:       aws.String("message"),
			},
			wantVR: &appmesh.VirtualRouter{
				Status: appmesh.VirtualRouterStatus{
					Conditions: []appmesh.VirtualRouterCondition{
						{
							Type:    appmesh.VirtualRouterActive,
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
				vr: &appmesh.VirtualRouter{
					Status: appmesh.VirtualRouterStatus{
						Conditions: []appmesh.VirtualRouterCondition{
							{
								Type:    appmesh.VirtualRouterActive,
								Status:  corev1.ConditionTrue,
								Reason:  aws.String("reason"),
								Message: aws.String("message"),
							},
						},
					},
				},
				conditionType: appmesh.VirtualRouterActive,
				status:        corev1.ConditionTrue,
				reason:        aws.String("reason"),
				message:       aws.String("message"),
			},
			wantVR: &appmesh.VirtualRouter{
				Status: appmesh.VirtualRouterStatus{
					Conditions: []appmesh.VirtualRouterCondition{
						{
							Type:    appmesh.VirtualRouterActive,
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
			gotChanged := updateCondition(tt.args.vr, tt.args.conditionType, tt.args.status, tt.args.reason, tt.args.message)
			opts := cmpopts.IgnoreTypes((*metav1.Time)(nil))
			assert.True(t, cmp.Equal(tt.wantVR, tt.args.vr, opts), "diff", cmp.Diff(tt.wantVR, tt.args.vr, opts))
			assert.Equal(t, tt.wantChanged, gotChanged)
		})
	}
}
