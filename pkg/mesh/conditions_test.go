package mesh

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
		ms            *appmesh.Mesh
		conditionType appmesh.MeshConditionType
	}
	tests := []struct {
		name string
		args args
		want *appmesh.MeshCondition
	}{
		{
			name: "condition found",
			args: args{
				ms: &appmesh.Mesh{
					Status: appmesh.MeshStatus{
						Conditions: []appmesh.MeshCondition{
							{
								Type:   appmesh.MeshActive,
								Status: corev1.ConditionFalse,
							},
						},
					},
				},
				conditionType: appmesh.MeshActive,
			},
			want: &appmesh.MeshCondition{
				Type:   appmesh.MeshActive,
				Status: corev1.ConditionFalse,
			},
		},
		{
			name: "condition not found",
			args: args{
				ms: &appmesh.Mesh{
					Status: appmesh.MeshStatus{
						Conditions: []appmesh.MeshCondition{},
					},
				},
				conditionType: appmesh.MeshActive,
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getCondition(tt.args.ms, tt.args.conditionType)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_updateCondition(t *testing.T) {
	type args struct {
		ms            *appmesh.Mesh
		conditionType appmesh.MeshConditionType
		status        corev1.ConditionStatus
		reason        *string
		message       *string
	}
	tests := []struct {
		name        string
		args        args
		wantMS      *appmesh.Mesh
		wantChanged bool
	}{
		{
			name: "condition updated by modify condition status",
			args: args{
				ms: &appmesh.Mesh{
					Status: appmesh.MeshStatus{
						Conditions: []appmesh.MeshCondition{
							{
								Type:   appmesh.MeshActive,
								Status: corev1.ConditionFalse,
							},
						},
					},
				},
				conditionType: appmesh.MeshActive,
				status:        corev1.ConditionTrue,
			},
			wantMS: &appmesh.Mesh{
				Status: appmesh.MeshStatus{
					Conditions: []appmesh.MeshCondition{
						{
							Type:   appmesh.MeshActive,
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
				ms: &appmesh.Mesh{
					Status: appmesh.MeshStatus{
						Conditions: []appmesh.MeshCondition{
							{
								Type:   appmesh.MeshActive,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
				conditionType: appmesh.MeshActive,
				status:        corev1.ConditionTrue,
				reason:        aws.String("reason"),
			},
			wantMS: &appmesh.Mesh{
				Status: appmesh.MeshStatus{
					Conditions: []appmesh.MeshCondition{
						{
							Type:   appmesh.MeshActive,
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
				ms: &appmesh.Mesh{
					Status: appmesh.MeshStatus{
						Conditions: []appmesh.MeshCondition{
							{
								Type:   appmesh.MeshActive,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
				conditionType: appmesh.MeshActive,
				status:        corev1.ConditionTrue,
				message:       aws.String("message"),
			},
			wantMS: &appmesh.Mesh{
				Status: appmesh.MeshStatus{
					Conditions: []appmesh.MeshCondition{
						{
							Type:    appmesh.MeshActive,
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
				ms: &appmesh.Mesh{
					Status: appmesh.MeshStatus{
						Conditions: []appmesh.MeshCondition{
							{
								Type:   appmesh.MeshActive,
								Status: corev1.ConditionFalse,
							},
						},
					},
				},
				conditionType: appmesh.MeshActive,
				status:        corev1.ConditionTrue,
				reason:        aws.String("reason"),
				message:       aws.String("message"),
			},
			wantMS: &appmesh.Mesh{
				Status: appmesh.MeshStatus{
					Conditions: []appmesh.MeshCondition{
						{
							Type:    appmesh.MeshActive,
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
				ms: &appmesh.Mesh{
					Status: appmesh.MeshStatus{
						Conditions: nil,
					},
				},
				conditionType: appmesh.MeshActive,
				status:        corev1.ConditionTrue,
				reason:        aws.String("reason"),
				message:       aws.String("message"),
			},
			wantMS: &appmesh.Mesh{
				Status: appmesh.MeshStatus{
					Conditions: []appmesh.MeshCondition{
						{
							Type:    appmesh.MeshActive,
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
				ms: &appmesh.Mesh{
					Status: appmesh.MeshStatus{
						Conditions: []appmesh.MeshCondition{
							{
								Type:    appmesh.MeshActive,
								Status:  corev1.ConditionTrue,
								Reason:  aws.String("reason"),
								Message: aws.String("message"),
							},
						},
					},
				},
				conditionType: appmesh.MeshActive,
				status:        corev1.ConditionTrue,
				reason:        aws.String("reason"),
				message:       aws.String("message"),
			},
			wantMS: &appmesh.Mesh{
				Status: appmesh.MeshStatus{
					Conditions: []appmesh.MeshCondition{
						{
							Type:    appmesh.MeshActive,
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
			gotChanged := updateCondition(tt.args.ms, tt.args.conditionType, tt.args.status, tt.args.reason, tt.args.message)
			opts := cmpopts.IgnoreTypes((*metav1.Time)(nil))
			assert.True(t, cmp.Equal(tt.wantMS, tt.args.ms, opts), "diff", cmp.Diff(tt.wantMS, tt.args.ms, opts))
			assert.Equal(t, tt.wantChanged, gotChanged)
		})
	}
}
