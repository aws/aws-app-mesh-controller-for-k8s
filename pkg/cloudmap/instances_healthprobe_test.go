package cloudmap

import (
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func Test_defaultInstancesHealthProber_filterUnhealthyInstances(t *testing.T) {
	type args struct {
		instances []InstanceProbe
	}
	tests := []struct {
		name string
		args args
		want []InstanceProbe
	}{
		{
			name: "should filter in instance without condition",
			args: args{
				instances: []InstanceProbe{
					{
						instanceID: "192.168.1.1",
						pod: &corev1.Pod{
							ObjectMeta: metav1.ObjectMeta{
								Name: "pod-1",
							},
							Spec: corev1.PodSpec{},
							Status: corev1.PodStatus{
								Conditions: []corev1.PodCondition{},
							},
						},
					},
				},
			},
			want: []InstanceProbe{
				{
					instanceID: "192.168.1.1",
					pod: &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Name: "pod-1",
						},
						Spec: corev1.PodSpec{},
						Status: corev1.PodStatus{
							Conditions: []corev1.PodCondition{},
						},
					},
				},
			},
		},
		{
			name: "should filter in instance with false condition",
			args: args{
				instances: []InstanceProbe{
					{
						instanceID: "192.168.1.1",
						pod: &corev1.Pod{
							ObjectMeta: metav1.ObjectMeta{
								Name: "pod-1",
							},
							Spec: corev1.PodSpec{},
							Status: corev1.PodStatus{
								Conditions: []corev1.PodCondition{
									{
										Type:   "conditions.appmesh.k8s.aws/aws-cloudmap-healthy",
										Status: corev1.ConditionFalse,
									},
								},
							},
						},
					},
				},
			},
			want: []InstanceProbe{
				{
					instanceID: "192.168.1.1",
					pod: &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Name: "pod-1",
						},
						Spec: corev1.PodSpec{},
						Status: corev1.PodStatus{
							Conditions: []corev1.PodCondition{
								{
									Type:   "conditions.appmesh.k8s.aws/aws-cloudmap-healthy",
									Status: corev1.ConditionFalse,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "should filter out instance with false condition",
			args: args{
				instances: []InstanceProbe{
					{
						instanceID: "192.168.1.1",
						pod: &corev1.Pod{
							ObjectMeta: metav1.ObjectMeta{
								Name: "pod-1",
							},
							Spec: corev1.PodSpec{},
							Status: corev1.PodStatus{
								Conditions: []corev1.PodCondition{
									{
										Type:   "conditions.appmesh.k8s.aws/aws-cloudmap-healthy",
										Status: corev1.ConditionTrue,
									},
								},
							},
						},
					},
				},
			},
			want: nil,
		},
		{
			name: "should work with multiple entries",
			args: args{
				instances: []InstanceProbe{
					{
						instanceID: "192.168.1.1",
						pod: &corev1.Pod{
							ObjectMeta: metav1.ObjectMeta{
								Name: "pod-1",
							},
							Spec: corev1.PodSpec{},
							Status: corev1.PodStatus{
								Conditions: []corev1.PodCondition{
									{
										Type:   "conditions.appmesh.k8s.aws/aws-cloudmap-healthy",
										Status: corev1.ConditionFalse,
									},
								},
							},
						},
					},
					{
						instanceID: "192.168.1.2",
						pod: &corev1.Pod{
							ObjectMeta: metav1.ObjectMeta{
								Name: "pod-2",
							},
							Spec: corev1.PodSpec{},
							Status: corev1.PodStatus{
								Conditions: []corev1.PodCondition{
									{
										Type:   "conditions.appmesh.k8s.aws/aws-cloudmap-healthy",
										Status: corev1.ConditionTrue,
									},
								},
							},
						},
					},
				},
			},
			want: []InstanceProbe{
				{
					instanceID: "192.168.1.1",
					pod: &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Name: "pod-1",
						},
						Spec: corev1.PodSpec{},
						Status: corev1.PodStatus{
							Conditions: []corev1.PodCondition{
								{
									Type:   "conditions.appmesh.k8s.aws/aws-cloudmap-healthy",
									Status: corev1.ConditionFalse,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "when instances to probe is nil",
			args: args{
				instances: nil,
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &defaultInstancesHealthProber{}
			got := p.filterUnhealthyInstances(tt.args.instances)
			assert.Equal(t, tt.want, got)
		})
	}
}
