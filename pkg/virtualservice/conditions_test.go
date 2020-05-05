package virtualservice

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
		vs            *appmesh.VirtualService
		conditionType appmesh.VirtualServiceConditionType
	}
	tests := []struct {
		name string
		args args
		want *appmesh.VirtualServiceCondition
	}{
		{
			name: "condition found",
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
				conditionType: appmesh.VirtualServiceActive,
			},
			want: &appmesh.VirtualServiceCondition{
				Type:   appmesh.VirtualServiceActive,
				Status: corev1.ConditionFalse,
			},
		},
		{
			name: "condition not found",
			args: args{
				vs: &appmesh.VirtualService{
					Status: appmesh.VirtualServiceStatus{
						Conditions: []appmesh.VirtualServiceCondition{},
					},
				},
				conditionType: appmesh.VirtualServiceActive,
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getCondition(tt.args.vs, tt.args.conditionType)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_updateCondition(t *testing.T) {
	type args struct {
		vs            *appmesh.VirtualService
		conditionType appmesh.VirtualServiceConditionType
		status        corev1.ConditionStatus
		reason        *string
		message       *string
	}
	tests := []struct {
		name        string
		args        args
		wantVS      *appmesh.VirtualService
		wantChanged bool
	}{
		{
			name: "condition updated by modify condition status",
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
				conditionType: appmesh.VirtualServiceActive,
				status:        corev1.ConditionTrue,
			},
			wantVS: &appmesh.VirtualService{
				Status: appmesh.VirtualServiceStatus{
					Conditions: []appmesh.VirtualServiceCondition{
						{
							Type:   appmesh.VirtualServiceActive,
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
				conditionType: appmesh.VirtualServiceActive,
				status:        corev1.ConditionTrue,
				reason:        aws.String("reason"),
			},
			wantVS: &appmesh.VirtualService{
				Status: appmesh.VirtualServiceStatus{
					Conditions: []appmesh.VirtualServiceCondition{
						{
							Type:   appmesh.VirtualServiceActive,
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
				conditionType: appmesh.VirtualServiceActive,
				status:        corev1.ConditionTrue,
				message:       aws.String("message"),
			},
			wantVS: &appmesh.VirtualService{
				Status: appmesh.VirtualServiceStatus{
					Conditions: []appmesh.VirtualServiceCondition{
						{
							Type:    appmesh.VirtualServiceActive,
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
				conditionType: appmesh.VirtualServiceActive,
				status:        corev1.ConditionTrue,
				reason:        aws.String("reason"),
				message:       aws.String("message"),
			},
			wantVS: &appmesh.VirtualService{
				Status: appmesh.VirtualServiceStatus{
					Conditions: []appmesh.VirtualServiceCondition{
						{
							Type:    appmesh.VirtualServiceActive,
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
				vs: &appmesh.VirtualService{
					Status: appmesh.VirtualServiceStatus{
						Conditions: nil,
					},
				},
				conditionType: appmesh.VirtualServiceActive,
				status:        corev1.ConditionTrue,
				reason:        aws.String("reason"),
				message:       aws.String("message"),
			},
			wantVS: &appmesh.VirtualService{
				Status: appmesh.VirtualServiceStatus{
					Conditions: []appmesh.VirtualServiceCondition{
						{
							Type:    appmesh.VirtualServiceActive,
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
				vs: &appmesh.VirtualService{
					Status: appmesh.VirtualServiceStatus{
						Conditions: []appmesh.VirtualServiceCondition{
							{
								Type:    appmesh.VirtualServiceActive,
								Status:  corev1.ConditionTrue,
								Reason:  aws.String("reason"),
								Message: aws.String("message"),
							},
						},
					},
				},
				conditionType: appmesh.VirtualServiceActive,
				status:        corev1.ConditionTrue,
				reason:        aws.String("reason"),
				message:       aws.String("message"),
			},
			wantVS: &appmesh.VirtualService{
				Status: appmesh.VirtualServiceStatus{
					Conditions: []appmesh.VirtualServiceCondition{
						{
							Type:    appmesh.VirtualServiceActive,
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
			gotChanged := updateCondition(tt.args.vs, tt.args.conditionType, tt.args.status, tt.args.reason, tt.args.message)
			opts := cmpopts.IgnoreTypes((*metav1.Time)(nil))
			assert.True(t, cmp.Equal(tt.wantVS, tt.args.vs, opts), "diff", cmp.Diff(tt.wantVS, tt.args.vs, opts))
			assert.Equal(t, tt.wantChanged, gotChanged)
		})
	}
}
