package inject

import (
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"testing"
)

func Test_ecrSecretMutator_mutate(t *testing.T) {
	type fields struct {
		enabled    bool
		secretName string
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
				enabled:    false,
				secretName: "appmesh-ecr-secret",
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
			name: "inject image pull secret when it's empty",
			fields: fields{
				enabled:    true,
				secretName: "appmesh-ecr-secret",
			},
			args: args{
				pod: &corev1.Pod{
					Spec: corev1.PodSpec{
						ImagePullSecrets: nil,
					},
				},
			},
			wantPod: &corev1.Pod{
				Spec: corev1.PodSpec{
					ImagePullSecrets: []corev1.LocalObjectReference{
						{
							Name: "appmesh-ecr-secret",
						},
					},
				},
			},
		},
		{
			name: "don't inject image pull secret when it already contain intended secret",
			fields: fields{
				enabled:    true,
				secretName: "appmesh-ecr-secret",
			},
			args: args{
				pod: &corev1.Pod{
					Spec: corev1.PodSpec{
						ImagePullSecrets: []corev1.LocalObjectReference{
							{
								Name: "appmesh-ecr-secret",
							},
						},
					},
				},
			},
			wantPod: &corev1.Pod{
				Spec: corev1.PodSpec{
					ImagePullSecrets: []corev1.LocalObjectReference{
						{
							Name: "appmesh-ecr-secret",
						},
					},
				},
			},
		},
		{
			name: "inject image pull secret when it doesn't contain intended secret",
			fields: fields{
				enabled:    true,
				secretName: "appmesh-ecr-secret",
			},
			args: args{
				pod: &corev1.Pod{
					Spec: corev1.PodSpec{
						ImagePullSecrets: []corev1.LocalObjectReference{
							{
								Name: "other-secrets",
							},
						},
					},
				},
			},
			wantPod: &corev1.Pod{
				Spec: corev1.PodSpec{
					ImagePullSecrets: []corev1.LocalObjectReference{
						{
							Name: "other-secrets",
						},
						{
							Name: "appmesh-ecr-secret",
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &ecrSecretMutator{
				enabled:    tt.fields.enabled,
				secretName: tt.fields.secretName,
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
