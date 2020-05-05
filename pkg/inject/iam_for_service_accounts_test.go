package inject

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"testing"
)

func Test_iamForServiceAccountsMutator_mutate(t *testing.T) {
	type fields struct {
		enabled   bool
		fsGroupID int64
	}
	type args struct {
		pod *corev1.Pod
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantPod *corev1.Pod
		wantErr error
	}{
		{
			name: "no-op when disabled",
			fields: fields{
				enabled:   false,
				fsGroupID: 1337,
			},
			args: args{
				pod: &corev1.Pod{
					Spec: corev1.PodSpec{},
				},
			},
			wantPod: &corev1.Pod{
				Spec: corev1.PodSpec{},
			},
		},
		{
			name: "inject fsGroup when securityContext is nil",
			fields: fields{
				enabled:   true,
				fsGroupID: 1337,
			},
			args: args{
				pod: &corev1.Pod{
					Spec: corev1.PodSpec{},
				},
			},
			wantPod: &corev1.Pod{
				Spec: corev1.PodSpec{
					SecurityContext: &corev1.PodSecurityContext{
						FSGroup: aws.Int64(1337),
					},
				},
			},
		},
		{
			name: "inject fsGroup when securityContext isn't nil but don't have fsGroup",
			fields: fields{
				enabled:   true,
				fsGroupID: 1337,
			},
			args: args{
				pod: &corev1.Pod{
					Spec: corev1.PodSpec{
						SecurityContext: &corev1.PodSecurityContext{
							RunAsUser: aws.Int64(1),
						},
					},
				},
			},
			wantPod: &corev1.Pod{
				Spec: corev1.PodSpec{
					SecurityContext: &corev1.PodSecurityContext{
						RunAsUser: aws.Int64(1),
						FSGroup:   aws.Int64(1337),
					},
				},
			},
		},
		{
			name: "don't inject fsGroup when securityContext isn't nil and have fsGroup",
			fields: fields{
				enabled: true,
			},
			args: args{
				pod: &corev1.Pod{
					Spec: corev1.PodSpec{
						SecurityContext: &corev1.PodSecurityContext{
							FSGroup: aws.Int64(42),
						},
					},
				},
			},
			wantPod: &corev1.Pod{
				Spec: corev1.PodSpec{
					SecurityContext: &corev1.PodSecurityContext{
						FSGroup: aws.Int64(42),
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &iamForServiceAccountsMutator{
				enabled:   tt.fields.enabled,
				fsGroupID: tt.fields.fsGroupID,
			}
			err := m.mutate(tt.args.pod)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.True(t, cmp.Equal(tt.wantPod, tt.args.pod), "diff", cmp.Diff(tt.wantPod, tt.args.pod))
			}
		})
	}
}
