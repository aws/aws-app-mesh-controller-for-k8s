package virtualnode

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
		vn            *appmesh.VirtualNode
		conditionType v1beta2.VirtualNodeConditionType
	}
	tests := []struct {
		name string
		args args
		want *appmesh.VirtualNodeCondition
	}{
		{
			name: "condition found",
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
				conditionType: appmesh.VirtualNodeActive,
			},
			want: &appmesh.VirtualNodeCondition{
				Type:   appmesh.VirtualNodeActive,
				Status: corev1.ConditionFalse,
			},
		},
		{
			name: "condition not found",
			args: args{
				vn: &appmesh.VirtualNode{
					Status: appmesh.VirtualNodeStatus{
						Conditions: []appmesh.VirtualNodeCondition{},
					},
				},
				conditionType: appmesh.VirtualNodeActive,
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getCondition(tt.args.vn, tt.args.conditionType)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_updateCondition(t *testing.T) {
	type args struct {
		vn            *appmesh.VirtualNode
		conditionType appmesh.VirtualNodeConditionType
		status        corev1.ConditionStatus
		reason        *string
		message       *string
	}
	tests := []struct {
		name        string
		args        args
		wantVN      *appmesh.VirtualNode
		wantChanged bool
	}{
		{
			name: "condition updated by modify condition status",
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
				conditionType: appmesh.VirtualNodeActive,
				status:        corev1.ConditionTrue,
			},
			wantVN: &appmesh.VirtualNode{
				Status: appmesh.VirtualNodeStatus{
					Conditions: []appmesh.VirtualNodeCondition{
						{
							Type:   appmesh.VirtualNodeActive,
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
				conditionType: appmesh.VirtualNodeActive,
				status:        corev1.ConditionTrue,
				reason:        aws.String("reason"),
			},
			wantVN: &appmesh.VirtualNode{
				Status: appmesh.VirtualNodeStatus{
					Conditions: []appmesh.VirtualNodeCondition{
						{
							Type:   appmesh.VirtualNodeActive,
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
				conditionType: appmesh.VirtualNodeActive,
				status:        corev1.ConditionTrue,
				message:       aws.String("message"),
			},
			wantVN: &appmesh.VirtualNode{
				Status: appmesh.VirtualNodeStatus{
					Conditions: []appmesh.VirtualNodeCondition{
						{
							Type:    appmesh.VirtualNodeActive,
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
				conditionType: appmesh.VirtualNodeActive,
				status:        corev1.ConditionTrue,
				reason:        aws.String("reason"),
				message:       aws.String("message"),
			},
			wantVN: &appmesh.VirtualNode{
				Status: appmesh.VirtualNodeStatus{
					Conditions: []appmesh.VirtualNodeCondition{
						{
							Type:    appmesh.VirtualNodeActive,
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
				vn: &appmesh.VirtualNode{
					Status: appmesh.VirtualNodeStatus{
						Conditions: nil,
					},
				},
				conditionType: appmesh.VirtualNodeActive,
				status:        corev1.ConditionTrue,
				reason:        aws.String("reason"),
				message:       aws.String("message"),
			},
			wantVN: &appmesh.VirtualNode{
				Status: appmesh.VirtualNodeStatus{
					Conditions: []appmesh.VirtualNodeCondition{
						{
							Type:    appmesh.VirtualNodeActive,
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
				vn: &appmesh.VirtualNode{
					Status: appmesh.VirtualNodeStatus{
						Conditions: []appmesh.VirtualNodeCondition{
							{
								Type:    appmesh.VirtualNodeActive,
								Status:  corev1.ConditionTrue,
								Reason:  aws.String("reason"),
								Message: aws.String("message"),
							},
						},
					},
				},
				conditionType: appmesh.VirtualNodeActive,
				status:        corev1.ConditionTrue,
				reason:        aws.String("reason"),
				message:       aws.String("message"),
			},
			wantVN: &appmesh.VirtualNode{
				Status: appmesh.VirtualNodeStatus{
					Conditions: []appmesh.VirtualNodeCondition{
						{
							Type:    appmesh.VirtualNodeActive,
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
			gotChanged := updateCondition(tt.args.vn, tt.args.conditionType, tt.args.status, tt.args.reason, tt.args.message)
			opts := cmpopts.IgnoreTypes((*metav1.Time)(nil))
			assert.True(t, cmp.Equal(tt.wantVN, tt.args.vn, opts), "diff", cmp.Diff(tt.wantVN, tt.args.vn, opts))
			assert.Equal(t, tt.wantChanged, gotChanged)
		})
	}
}
