package inject

import (
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func Test_terminationGracePeriodMutator_mutate(t *testing.T) {
	var higherPreStopDelay int64 = 45
	var lowerPreStopDelay int64 = 15

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-ns",
			Name:      "my-pod",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: "app/v1",
				},
			},
		},
	}

	type fields struct {
		terminationGracePeriodSeconds int64
		needsToBeAdjusted             bool
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
			name: "no-op when preStop delay is less than or equal to 30 seconds",
			fields: fields{
				needsToBeAdjusted:             false,
				terminationGracePeriodSeconds: lowerPreStopDelay,
			},
			args: args{
				pod: pod,
			},
			wantPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "my-ns",
					Name:      "my-pod",
				},
				Spec: corev1.PodSpec{
					TerminationGracePeriodSeconds: nil,
					Containers: []corev1.Container{
						{
							Name:  "app",
							Image: "app/v1",
						},
					},
				},
			},
		},
		{
			name: "adjust terminationGracePeriod when preStopDelay is more than 30 seconds",
			fields: fields{
				needsToBeAdjusted:             true,
				terminationGracePeriodSeconds: higherPreStopDelay,
			},
			args: args{
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "my-ns",
						Name:      "my-pod",
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "app",
								Image: "app/v1",
							},
						},
					},
				},
			},
			wantPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "my-ns",
					Name:      "my-pod",
				},
				Spec: corev1.PodSpec{
					TerminationGracePeriodSeconds: &higherPreStopDelay,
					Containers: []corev1.Container{
						{
							Name:  "app",
							Image: "app/v1",
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &terminationGracePeriodMutator{
				needsToBeAdjusted:             tt.fields.needsToBeAdjusted,
				terminationGracePeriodSeconds: tt.fields.terminationGracePeriodSeconds,
			}
			pod := tt.args.pod.DeepCopy()
			err := m.mutate(pod)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.True(t, cmp.Equal(tt.wantPod, pod), "diff", cmp.Diff(tt.wantPod, pod))
			}
		})
	}
}
