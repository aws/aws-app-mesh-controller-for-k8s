package inject

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_xrayMutator_mutate(t *testing.T) {
	cpuRequests, _ := resource.ParseQuantity("32Mi")
	memoryRequests, _ := resource.ParseQuantity("10m")

	cpuLimits, _ := resource.ParseQuantity("64Mi")
	memoryLimits, _ := resource.ParseQuantity("30m")

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

	annotationCpuRequest, _ := resource.ParseQuantity("128Mi")
	annotationMemoryRequest, _ := resource.ParseQuantity("20m")
	annotationCpuLimit, _ := resource.ParseQuantity("256Mi")
	annotationMemoryLimit, _ := resource.ParseQuantity("80m")

	podWithResourceAnnotations := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-ns",
			Name:      "my-pod",
			Annotations: map[string]string{
				AppMeshCPURequestAnnotation:    annotationCpuRequest.String(),
				AppMeshMemoryRequestAnnotation: annotationMemoryRequest.String(),
				AppMeshCPULimitAnnotation:      annotationCpuLimit.String(),
				AppMeshMemoryLimitAnnotation:   annotationMemoryLimit.String(),
			},
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
		sidecarCPULimits:      cpuLimits.String(),
		sidecarMemoryLimits:   memoryLimits.String(),
		xRayImage:             "amazon/aws-xray-daemon",
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
			name: "no-op when already contains xray daemon container",
			fields: fields{
				enabled:       true,
				mutatorConfig: mutatorConfig,
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
							{
								Name: "xray-daemon",
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
					Containers: []corev1.Container{
						{
							Name:  "app",
							Image: "app/v1",
						},
						{
							Name: "xray-daemon",
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
								Limits: corev1.ResourceList{
									"cpu":    cpuLimits,
									"memory": memoryLimits,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "resource requests and limits annotations",
			fields: fields{
				enabled:       true,
				mutatorConfig: mutatorConfig,
			},
			args: args{
				pod: podWithResourceAnnotations,
			},
			wantPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "my-ns",
					Name:      "my-pod",
					Annotations: map[string]string{
						AppMeshCPURequestAnnotation:    annotationCpuRequest.String(),
						AppMeshMemoryRequestAnnotation: annotationMemoryRequest.String(),
						AppMeshCPULimitAnnotation:      annotationCpuLimit.String(),
						AppMeshMemoryLimitAnnotation:   annotationMemoryLimit.String(),
					},
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
									"cpu":    annotationCpuRequest,
									"memory": annotationMemoryRequest,
								},
								Limits: corev1.ResourceList{
									"cpu":    annotationCpuLimit,
									"memory": annotationMemoryLimit,
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

func Test_containsXRAYDaemonContainer(t *testing.T) {
	type args struct {
		pod *corev1.Pod
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "contains xray daemon container",
			args: args{
				pod: &corev1.Pod{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name: "xray-daemon",
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "doesn't contains xray daemon container",
			args: args{
				pod: &corev1.Pod{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name: "other",
							},
						},
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsXRAYDaemonContainer(tt.args.pod)
			assert.Equal(t, tt.want, got)
		})
	}
}
