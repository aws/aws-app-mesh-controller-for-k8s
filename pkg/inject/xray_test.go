package inject

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func Test_xrayMutator_mutate(t *testing.T) {
	cpuRequests, _ := resource.ParseQuantity("32Mi")
	memoryRequests, _ := resource.ParseQuantity("10m")
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
	mutatorConfig := xrayMutatorConfig{
		awsRegion:             "us-west-2",
		sidecarCPURequests:    cpuRequests.String(),
		sidecarMemoryRequests: memoryRequests.String(),
	}
	type fields struct {
		enabled       bool
		mutatorConfig xrayMutatorConfig
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
				enabled:       false,
				mutatorConfig: mutatorConfig,
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
			name: "inject sidecar when enabled",
			fields: fields{
				enabled:       true,
				mutatorConfig: mutatorConfig,
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
					Containers: []corev1.Container{
						{
							Name:  "app",
							Image: "app/v1",
						},
						{
							Name:  "xray-daemon",
							Image: "amazon/aws-xray-daemon",
							SecurityContext: &corev1.SecurityContext{
								RunAsUser: aws.Int64(1337),
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "xray",
									ContainerPort: 2000,
									Protocol:      "UDP",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "AWS_REGION",
									Value: "us-west-2",
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									"cpu":    cpuRequests,
									"memory": memoryRequests,
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &xrayMutator{
				enabled:       tt.fields.enabled,
				mutatorConfig: tt.fields.mutatorConfig,
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
