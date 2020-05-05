package inject

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func Test_envoyMutator_mutate(t *testing.T) {
	cpuRequests, _ := resource.ParseQuantity("32Mi")
	memoryRequests, _ := resource.ParseQuantity("10m")
	ms := &appmesh.Mesh{
		ObjectMeta: metav1.ObjectMeta{
			Name: "mesh",
		},
		Spec: appmesh.MeshSpec{
			AWSName: aws.String("my-mesh"),
		},
	}
	vn := &appmesh.VirtualNode{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-ns",
			Name:      "my-vn",
		},
		Spec: appmesh.VirtualNodeSpec{
			AWSName: aws.String("my-vn_my-ns"),
		},
	}
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
		vn            *appmesh.VirtualNode
		ms            *appmesh.Mesh
		mutatorConfig envoyMutatorConfig
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
			name: "no tracing",
			fields: fields{
				vn: vn,
				ms: ms,
				mutatorConfig: envoyMutatorConfig{
					awsRegion:             "us-west-2",
					preview:               false,
					logLevel:              "debug",
					sidecarImage:          "envoy:v2",
					sidecarCPURequests:    cpuRequests.String(),
					sidecarMemoryRequests: memoryRequests.String(),
				},
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
							Name:  "envoy",
							Image: "envoy:v2",
							SecurityContext: &corev1.SecurityContext{
								RunAsUser: aws.Int64(1337),
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "stats",
									ContainerPort: 9901,
									Protocol:      "TCP",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "APPMESH_VIRTUAL_NODE_NAME",
									Value: "mesh/my-mesh/virtualNode/my-vn_my-ns",
								},
								{
									Name:  "APPMESH_PREVIEW",
									Value: "0",
								},
								{
									Name:  "ENVOY_LOG_LEVEL",
									Value: "debug",
								},
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
		{
			name: "no tracing + enable preview",
			fields: fields{
				vn: vn,
				ms: ms,
				mutatorConfig: envoyMutatorConfig{
					awsRegion:             "us-west-2",
					preview:               true,
					logLevel:              "debug",
					sidecarImage:          "envoy:v2",
					sidecarCPURequests:    cpuRequests.String(),
					sidecarMemoryRequests: memoryRequests.String(),
				},
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
							Name:  "envoy",
							Image: "envoy:v2",
							SecurityContext: &corev1.SecurityContext{
								RunAsUser: aws.Int64(1337),
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "stats",
									ContainerPort: 9901,
									Protocol:      "TCP",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "APPMESH_VIRTUAL_NODE_NAME",
									Value: "mesh/my-mesh/virtualNode/my-vn_my-ns",
								},
								{
									Name:  "APPMESH_PREVIEW",
									Value: "1",
								},
								{
									Name:  "ENVOY_LOG_LEVEL",
									Value: "debug",
								},
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
		{
			name: "no tracing + enable xray tracing",
			fields: fields{
				vn: vn,
				ms: ms,
				mutatorConfig: envoyMutatorConfig{
					awsRegion:             "us-west-2",
					preview:               false,
					logLevel:              "debug",
					sidecarImage:          "envoy:v2",
					sidecarCPURequests:    cpuRequests.String(),
					sidecarMemoryRequests: memoryRequests.String(),
					enableXrayTracing:     true,
				},
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
							Name:  "envoy",
							Image: "envoy:v2",
							SecurityContext: &corev1.SecurityContext{
								RunAsUser: aws.Int64(1337),
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "stats",
									ContainerPort: 9901,
									Protocol:      "TCP",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "APPMESH_VIRTUAL_NODE_NAME",
									Value: "mesh/my-mesh/virtualNode/my-vn_my-ns",
								},
								{
									Name:  "APPMESH_PREVIEW",
									Value: "0",
								},
								{
									Name:  "ENVOY_LOG_LEVEL",
									Value: "debug",
								},
								{
									Name:  "AWS_REGION",
									Value: "us-west-2",
								},
								{
									Name:  "ENABLE_ENVOY_XRAY_TRACING",
									Value: "1",
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
		{
			name: "no tracing + enable Jaeger tracing",
			fields: fields{
				vn: vn,
				ms: ms,
				mutatorConfig: envoyMutatorConfig{
					awsRegion:             "us-west-2",
					preview:               false,
					logLevel:              "debug",
					sidecarImage:          "envoy:v2",
					sidecarCPURequests:    cpuRequests.String(),
					sidecarMemoryRequests: memoryRequests.String(),
					enableJaegerTracing:   true,
				},
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
							Name:  "envoy",
							Image: "envoy:v2",
							SecurityContext: &corev1.SecurityContext{
								RunAsUser: aws.Int64(1337),
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "stats",
									ContainerPort: 9901,
									Protocol:      "TCP",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "APPMESH_VIRTUAL_NODE_NAME",
									Value: "mesh/my-mesh/virtualNode/my-vn_my-ns",
								},
								{
									Name:  "APPMESH_PREVIEW",
									Value: "0",
								},
								{
									Name:  "ENVOY_LOG_LEVEL",
									Value: "debug",
								},
								{
									Name:  "ENVOY_STATS_CONFIG_FILE",
									Value: "/tmp/envoy/envoyconf.yaml",
								},
								{
									Name:  "AWS_REGION",
									Value: "us-west-2",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "envoy-tracing-config",
									MountPath: "/tmp/envoy",
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
		{
			name: "no tracing + enable Datadog tracing",
			fields: fields{
				vn: vn,
				ms: ms,
				mutatorConfig: envoyMutatorConfig{
					awsRegion:             "us-west-2",
					preview:               false,
					logLevel:              "debug",
					sidecarImage:          "envoy:v2",
					sidecarCPURequests:    cpuRequests.String(),
					sidecarMemoryRequests: memoryRequests.String(),
					enableDatadogTracing:  true,
				},
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
							Name:  "envoy",
							Image: "envoy:v2",
							SecurityContext: &corev1.SecurityContext{
								RunAsUser: aws.Int64(1337),
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "stats",
									ContainerPort: 9901,
									Protocol:      "TCP",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "APPMESH_VIRTUAL_NODE_NAME",
									Value: "mesh/my-mesh/virtualNode/my-vn_my-ns",
								},
								{
									Name:  "APPMESH_PREVIEW",
									Value: "0",
								},
								{
									Name:  "ENVOY_LOG_LEVEL",
									Value: "debug",
								},
								{
									Name:  "ENVOY_STATS_CONFIG_FILE",
									Value: "/tmp/envoy/envoyconf.yaml",
								},
								{
									Name:  "AWS_REGION",
									Value: "us-west-2",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "envoy-tracing-config",
									MountPath: "/tmp/envoy",
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
		{
			name: "no tracing + enable Stats tags",
			fields: fields{
				vn: vn,
				ms: ms,
				mutatorConfig: envoyMutatorConfig{
					awsRegion:             "us-west-2",
					preview:               false,
					logLevel:              "debug",
					sidecarImage:          "envoy:v2",
					sidecarCPURequests:    cpuRequests.String(),
					sidecarMemoryRequests: memoryRequests.String(),
					enableStatsTags:       true,
				},
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
							Name:  "envoy",
							Image: "envoy:v2",
							SecurityContext: &corev1.SecurityContext{
								RunAsUser: aws.Int64(1337),
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "stats",
									ContainerPort: 9901,
									Protocol:      "TCP",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "APPMESH_VIRTUAL_NODE_NAME",
									Value: "mesh/my-mesh/virtualNode/my-vn_my-ns",
								},
								{
									Name:  "APPMESH_PREVIEW",
									Value: "0",
								},
								{
									Name:  "ENVOY_LOG_LEVEL",
									Value: "debug",
								},
								{
									Name:  "AWS_REGION",
									Value: "us-west-2",
								},
								{
									Name:  "ENABLE_ENVOY_STATS_TAGS",
									Value: "1",
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
		{
			name: "no tracing + enable Stats D",
			fields: fields{
				vn: vn,
				ms: ms,
				mutatorConfig: envoyMutatorConfig{
					awsRegion:             "us-west-2",
					preview:               false,
					logLevel:              "debug",
					sidecarImage:          "envoy:v2",
					sidecarCPURequests:    cpuRequests.String(),
					sidecarMemoryRequests: memoryRequests.String(),
					enableStatsD:          true,
				},
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
							Name:  "envoy",
							Image: "envoy:v2",
							SecurityContext: &corev1.SecurityContext{
								RunAsUser: aws.Int64(1337),
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "stats",
									ContainerPort: 9901,
									Protocol:      "TCP",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "APPMESH_VIRTUAL_NODE_NAME",
									Value: "mesh/my-mesh/virtualNode/my-vn_my-ns",
								},
								{
									Name:  "APPMESH_PREVIEW",
									Value: "0",
								},
								{
									Name:  "ENVOY_LOG_LEVEL",
									Value: "debug",
								},
								{
									Name:  "AWS_REGION",
									Value: "us-west-2",
								},
								{
									Name:  "ENABLE_ENVOY_DOG_STATSD",
									Value: "1",
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
			m := &envoyMutator{
				vn:            tt.fields.vn,
				ms:            tt.fields.ms,
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

func Test_envoyMutator_getPreview(t *testing.T) {
	type fields struct {
		mutatorConfig envoyMutatorConfig
	}
	type args struct {
		pod *corev1.Pod
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{
			name: "enabled preview by annotation",
			fields: fields{
				mutatorConfig: envoyMutatorConfig{
					preview: false,
				},
			},
			args: args{
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							"appmesh.k8s.aws/preview": "true",
						},
					},
				},
			},
			want: "1",
		},
		{
			name: "disable preview by annotation",
			fields: fields{
				mutatorConfig: envoyMutatorConfig{
					preview: true,
				},
			},
			args: args{
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							"appmesh.k8s.aws/preview": "false",
						},
					},
				},
			},
			want: "0",
		},
		{
			name: "enabled preview by default",
			fields: fields{
				mutatorConfig: envoyMutatorConfig{
					preview: true,
				},
			},
			args: args{
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{},
				},
			},
			want: "1",
		},
		{
			name: "disable preview by default",
			fields: fields{
				mutatorConfig: envoyMutatorConfig{
					preview: false,
				},
			},
			args: args{
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{},
				},
			},
			want: "0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &envoyMutator{
				mutatorConfig: tt.fields.mutatorConfig,
			}
			got := m.getPreview(tt.args.pod)
			assert.Equal(t, tt.want, got)
		})
	}
}
