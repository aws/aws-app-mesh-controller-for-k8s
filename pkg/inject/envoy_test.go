package inject

import (
	"errors"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func Test_envoyMutator_mutate(t *testing.T) {
	cpuRequests, _ := resource.ParseQuantity("32Mi")
	memoryRequests, _ := resource.ParseQuantity("10m")

	cpuLimits, _ := resource.ParseQuantity("64Mi")
	memoryLimits, _ := resource.ParseQuantity("30m")

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
					awsRegion:                  "us-west-2",
					preview:                    false,
					logLevel:                   "debug",
					preStopDelay:               "20",
					readinessProbeInitialDelay: 1,
					readinessProbePeriod:       10,
					sidecarImage:               "envoy:v2",
					sidecarCPURequests:         cpuRequests.String(),
					sidecarMemoryRequests:      memoryRequests.String(),
					sidecarCPULimits:           cpuLimits.String(),
					sidecarMemoryLimits:        memoryLimits.String(),
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
							Lifecycle: &corev1.Lifecycle{
								PostStart: nil,
								PreStop: &corev1.Handler{
									Exec: &corev1.ExecAction{Command: []string{
										"sh", "-c", "sleep 20",
									}},
								},
							},
							ReadinessProbe: &corev1.Probe{
								Handler: corev1.Handler{

									Exec: &corev1.ExecAction{Command: []string{
										"sh", "-c", "curl -s http://localhost:9901/server_info | grep state | grep -q LIVE",
									}},
								},
								InitialDelaySeconds: 1,
								TimeoutSeconds:      1,
								PeriodSeconds:       10,
								SuccessThreshold:    1,
								FailureThreshold:    3,
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
			name: "no tracing + enable preview",
			fields: fields{
				vn: vn,
				ms: ms,
				mutatorConfig: envoyMutatorConfig{
					awsRegion:                  "us-west-2",
					preview:                    true,
					logLevel:                   "debug",
					preStopDelay:               "20",
					readinessProbeInitialDelay: 10,
					readinessProbePeriod:       2,
					sidecarImage:               "envoy:v2",
					sidecarCPURequests:         cpuRequests.String(),
					sidecarMemoryRequests:      memoryRequests.String(),
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
							Lifecycle: &corev1.Lifecycle{
								PostStart: nil,
								PreStop: &corev1.Handler{
									Exec: &corev1.ExecAction{Command: []string{
										"sh", "-c", "sleep 20",
									}},
								},
							},
							ReadinessProbe: &corev1.Probe{
								Handler: corev1.Handler{

									Exec: &corev1.ExecAction{Command: []string{
										"sh", "-c", "curl -s http://localhost:9901/server_info | grep state | grep -q LIVE",
									}},
								},
								InitialDelaySeconds: 10,
								TimeoutSeconds:      1,
								PeriodSeconds:       2,
								SuccessThreshold:    1,
								FailureThreshold:    3,
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
					awsRegion:                  "us-west-2",
					preview:                    false,
					logLevel:                   "debug",
					preStopDelay:               "20",
					readinessProbeInitialDelay: 3,
					readinessProbePeriod:       5,
					sidecarImage:               "envoy:v2",
					sidecarCPURequests:         cpuRequests.String(),
					sidecarMemoryRequests:      memoryRequests.String(),
					enableXrayTracing:          true,
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
							Lifecycle: &corev1.Lifecycle{
								PostStart: nil,
								PreStop: &corev1.Handler{
									Exec: &corev1.ExecAction{Command: []string{
										"sh", "-c", "sleep 20",
									}},
								},
							},
							ReadinessProbe: &corev1.Probe{
								Handler: corev1.Handler{

									Exec: &corev1.ExecAction{Command: []string{
										"sh", "-c", "curl -s http://localhost:9901/server_info | grep state | grep -q LIVE",
									}},
								},
								InitialDelaySeconds: 3,
								TimeoutSeconds:      1,
								PeriodSeconds:       5,
								SuccessThreshold:    1,
								FailureThreshold:    3,
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
					awsRegion:                  "us-west-2",
					preview:                    false,
					logLevel:                   "debug",
					preStopDelay:               "20",
					readinessProbeInitialDelay: 1,
					readinessProbePeriod:       10,
					sidecarImage:               "envoy:v2",
					sidecarCPURequests:         cpuRequests.String(),
					sidecarMemoryRequests:      memoryRequests.String(),
					enableJaegerTracing:        true,
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
							Lifecycle: &corev1.Lifecycle{
								PostStart: nil,
								PreStop: &corev1.Handler{
									Exec: &corev1.ExecAction{Command: []string{
										"sh", "-c", "sleep 20",
									}},
								},
							},
							ReadinessProbe: &corev1.Probe{
								Handler: corev1.Handler{

									Exec: &corev1.ExecAction{Command: []string{
										"sh", "-c", "curl -s http://localhost:9901/server_info | grep state | grep -q LIVE",
									}},
								},
								InitialDelaySeconds: 1,
								TimeoutSeconds:      1,
								PeriodSeconds:       10,
								SuccessThreshold:    1,
								FailureThreshold:    3,
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
					awsRegion:                  "us-west-2",
					preview:                    false,
					logLevel:                   "debug",
					preStopDelay:               "20",
					readinessProbeInitialDelay: 1,
					readinessProbePeriod:       10,
					sidecarImage:               "envoy:v2",
					sidecarCPURequests:         cpuRequests.String(),
					sidecarMemoryRequests:      memoryRequests.String(),
					enableDatadogTracing:       true,
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
							Lifecycle: &corev1.Lifecycle{
								PostStart: nil,
								PreStop: &corev1.Handler{
									Exec: &corev1.ExecAction{Command: []string{
										"sh", "-c", "sleep 20",
									}},
								},
							},
							ReadinessProbe: &corev1.Probe{
								Handler: corev1.Handler{

									Exec: &corev1.ExecAction{Command: []string{
										"sh", "-c", "curl -s http://localhost:9901/server_info | grep state | grep -q LIVE",
									}},
								},
								InitialDelaySeconds: 1,
								TimeoutSeconds:      1,
								PeriodSeconds:       10,
								SuccessThreshold:    1,
								FailureThreshold:    3,
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
					awsRegion:                  "us-west-2",
					preview:                    false,
					logLevel:                   "debug",
					preStopDelay:               "20",
					readinessProbeInitialDelay: 1,
					readinessProbePeriod:       10,
					sidecarImage:               "envoy:v2",
					sidecarCPURequests:         cpuRequests.String(),
					sidecarMemoryRequests:      memoryRequests.String(),
					enableStatsTags:            true,
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
							Lifecycle: &corev1.Lifecycle{
								PostStart: nil,
								PreStop: &corev1.Handler{
									Exec: &corev1.ExecAction{Command: []string{
										"sh", "-c", "sleep 20",
									}},
								},
							},
							ReadinessProbe: &corev1.Probe{
								Handler: corev1.Handler{

									Exec: &corev1.ExecAction{Command: []string{
										"sh", "-c", "curl -s http://localhost:9901/server_info | grep state | grep -q LIVE",
									}},
								},
								InitialDelaySeconds: 1,
								TimeoutSeconds:      1,
								PeriodSeconds:       10,
								SuccessThreshold:    1,
								FailureThreshold:    3,
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
					awsRegion:                  "us-west-2",
					preview:                    false,
					logLevel:                   "debug",
					preStopDelay:               "20",
					readinessProbeInitialDelay: 1,
					readinessProbePeriod:       10,
					sidecarImage:               "envoy:v2",
					sidecarCPURequests:         cpuRequests.String(),
					sidecarMemoryRequests:      memoryRequests.String(),
					enableStatsD:               true,
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
							Lifecycle: &corev1.Lifecycle{
								PostStart: nil,
								PreStop: &corev1.Handler{
									Exec: &corev1.ExecAction{Command: []string{
										"sh", "-c", "sleep 20",
									}},
								},
							},
							ReadinessProbe: &corev1.Probe{
								Handler: corev1.Handler{

									Exec: &corev1.ExecAction{Command: []string{
										"sh", "-c", "curl -s http://localhost:9901/server_info | grep state | grep -q LIVE",
									}},
								},
								InitialDelaySeconds: 1,
								TimeoutSeconds:      1,
								PeriodSeconds:       10,
								SuccessThreshold:    1,
								FailureThreshold:    3,
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
		{
			name: "no tracing + secretMounts",
			fields: fields{
				vn: vn,
				ms: ms,
				mutatorConfig: envoyMutatorConfig{
					awsRegion:                  "us-west-2",
					preview:                    false,
					logLevel:                   "debug",
					preStopDelay:               "20",
					readinessProbeInitialDelay: 1,
					readinessProbePeriod:       10,
					sidecarImage:               "envoy:v2",
					sidecarCPURequests:         cpuRequests.String(),
					sidecarMemoryRequests:      memoryRequests.String(),
				},
			},
			args: args{
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "my-ns",
						Name:      "my-pod",
						Annotations: map[string]string{
							"appmesh.k8s.aws/secretMounts": "svc1-cert-chain-key:/certs/svc1, svc1-svc2-ca-bundle:/certs",
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
				},
			},
			wantPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "my-ns",
					Name:      "my-pod",
					Annotations: map[string]string{
						"appmesh.k8s.aws/secretMounts": "svc1-cert-chain-key:/certs/svc1, svc1-svc2-ca-bundle:/certs",
					},
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
							Lifecycle: &corev1.Lifecycle{
								PostStart: nil,
								PreStop: &corev1.Handler{
									Exec: &corev1.ExecAction{Command: []string{
										"sh", "-c", "sleep 20",
									}},
								},
							},
							ReadinessProbe: &corev1.Probe{
								Handler: corev1.Handler{

									Exec: &corev1.ExecAction{Command: []string{
										"sh", "-c", "curl -s http://localhost:9901/server_info | grep state | grep -q LIVE",
									}},
								},
								InitialDelaySeconds: 1,
								TimeoutSeconds:      1,
								PeriodSeconds:       10,
								SuccessThreshold:    1,
								FailureThreshold:    3,
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
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "svc1-cert-chain-key",
									MountPath: "/certs/svc1",
									ReadOnly:  true,
								},
								{
									Name:      "svc1-svc2-ca-bundle",
									MountPath: "/certs",
									ReadOnly:  true,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "svc1-cert-chain-key",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "svc1-cert-chain-key",
								},
							},
						},
						{
							Name: "svc1-svc2-ca-bundle",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "svc1-svc2-ca-bundle",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "no cpu limits",
			fields: fields{
				vn: vn,
				ms: ms,
				mutatorConfig: envoyMutatorConfig{
					awsRegion:                  "us-west-2",
					preview:                    false,
					logLevel:                   "debug",
					preStopDelay:               "20",
					readinessProbeInitialDelay: 1,
					readinessProbePeriod:       10,
					sidecarImage:               "envoy:v2",
					sidecarCPURequests:         cpuRequests.String(),
					sidecarMemoryRequests:      memoryRequests.String(),
					sidecarMemoryLimits:        memoryLimits.String(),
					enableStatsD:               true,
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
							Lifecycle: &corev1.Lifecycle{
								PostStart: nil,
								PreStop: &corev1.Handler{
									Exec: &corev1.ExecAction{Command: []string{
										"sh", "-c", "sleep 20",
									}},
								},
							},
							ReadinessProbe: &corev1.Probe{
								Handler: corev1.Handler{

									Exec: &corev1.ExecAction{Command: []string{
										"sh", "-c", "curl -s http://localhost:9901/server_info | grep state | grep -q LIVE",
									}},
								},
								InitialDelaySeconds: 1,
								TimeoutSeconds:      1,
								PeriodSeconds:       10,
								SuccessThreshold:    1,
								FailureThreshold:    3,
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
								Limits: corev1.ResourceList{
									"memory": memoryLimits,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "no memory limits",
			fields: fields{
				vn: vn,
				ms: ms,
				mutatorConfig: envoyMutatorConfig{
					awsRegion:                  "us-west-2",
					preview:                    false,
					logLevel:                   "debug",
					preStopDelay:               "20",
					readinessProbeInitialDelay: 1,
					readinessProbePeriod:       10,
					sidecarImage:               "envoy:v2",
					sidecarCPURequests:         cpuRequests.String(),
					sidecarMemoryRequests:      memoryRequests.String(),
					sidecarCPULimits:           cpuLimits.String(),
					enableStatsD:               true,
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
							Lifecycle: &corev1.Lifecycle{
								PostStart: nil,
								PreStop: &corev1.Handler{
									Exec: &corev1.ExecAction{Command: []string{
										"sh", "-c", "sleep 20",
									}},
								},
							},
							ReadinessProbe: &corev1.Probe{
								Handler: corev1.Handler{

									Exec: &corev1.ExecAction{Command: []string{
										"sh", "-c", "curl -s http://localhost:9901/server_info | grep state | grep -q LIVE",
									}},
								},
								InitialDelaySeconds: 1,
								TimeoutSeconds:      1,
								PeriodSeconds:       10,
								SuccessThreshold:    1,
								FailureThreshold:    3,
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
								Limits: corev1.ResourceList{
									"cpu": cpuLimits,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "resource requests and limits annotation override",
			fields: fields{
				vn: vn,
				ms: ms,
				mutatorConfig: envoyMutatorConfig{
					awsRegion:                  "us-west-2",
					preview:                    false,
					logLevel:                   "debug",
					preStopDelay:               "20",
					readinessProbeInitialDelay: 1,
					readinessProbePeriod:       10,
					sidecarImage:               "envoy:v2",
					sidecarCPURequests:         cpuRequests.String(),
					sidecarMemoryRequests:      memoryRequests.String(),
					sidecarCPULimits:           cpuLimits.String(),
					sidecarMemoryLimits:        memoryLimits.String(),
					enableStatsD:               true,
				},
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
							Lifecycle: &corev1.Lifecycle{
								PostStart: nil,
								PreStop: &corev1.Handler{
									Exec: &corev1.ExecAction{Command: []string{
										"sh", "-c", "sleep 20",
									}},
								},
							},
							ReadinessProbe: &corev1.Probe{
								Handler: corev1.Handler{

									Exec: &corev1.ExecAction{Command: []string{
										"sh", "-c", "curl -s http://localhost:9901/server_info | grep state | grep -q LIVE",
									}},
								},
								InitialDelaySeconds: 1,
								TimeoutSeconds:      1,
								PeriodSeconds:       10,
								SuccessThreshold:    1,
								FailureThreshold:    3,
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
		{
			name: "no-op when already injected",
			fields: fields{
				vn: vn,
				ms: ms,
				mutatorConfig: envoyMutatorConfig{
					awsRegion:                  "us-west-2",
					preview:                    false,
					logLevel:                   "debug",
					preStopDelay:               "20",
					readinessProbeInitialDelay: 1,
					readinessProbePeriod:       10,
					sidecarImage:               "envoy:v2",
					sidecarCPURequests:         cpuRequests.String(),
					sidecarMemoryRequests:      memoryRequests.String(),
				},
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
								Name: "envoy",
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
							Name: "envoy",
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
				opts := cmp.Options{
					cmpopts.SortSlices(func(lhs corev1.Volume, rhs corev1.Volume) bool {
						return lhs.Name < rhs.Name
					}),
					cmpopts.SortSlices(func(lhs corev1.VolumeMount, rhs corev1.VolumeMount) bool {
						return lhs.Name < rhs.Name
					}),
				}
				assert.True(t, cmp.Equal(tt.wantPod, pod, opts), "diff", cmp.Diff(tt.wantPod, pod, opts))
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
							"appmesh.k8s.aws/preview": "enabled",
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
							"appmesh.k8s.aws/preview": "disabled",
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

func Test_containsEnvoyTracingConfigVolume(t *testing.T) {
	type args struct {
		pod *corev1.Pod
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "contains envoy tracing config volume",
			args: args{
				pod: &corev1.Pod{
					Spec: corev1.PodSpec{
						Volumes: []corev1.Volume{
							{
								Name: "envoy-tracing-config",
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "doesn't contains envoy tracing config volume",
			args: args{
				pod: &corev1.Pod{
					Spec: corev1.PodSpec{
						Volumes: []corev1.Volume{
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
			got := containsEnvoyTracingConfigVolume(tt.args.pod)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_envoyMutator_getSecretMounts(t *testing.T) {
	type args struct {
		pod *corev1.Pod
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]string
		wantErr error
	}{
		{
			name: "pods with valid appmesh.k8s.aws/secretMounts annotation",
			args: args{
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							"appmesh.k8s.aws/secretMounts": "svc1-cert-chain-key:/certs/svc1, svc1-svc2-ca-bundle:/certs",
						},
					},
				},
			},
			want: map[string]string{
				"svc1-cert-chain-key": "/certs/svc1",
				"svc1-svc2-ca-bundle": "/certs",
			},
			wantErr: nil,
		},
		{
			name: "pods with no appmesh.k8s.aws/secretMounts annotation",
			args: args{
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{},
					},
				},
			},
			want:    map[string]string{},
			wantErr: nil,
		},
		{
			name: "pods with invalid appmesh.k8s.aws/secretMounts annotation",
			args: args{
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							"appmesh.k8s.aws/secretMounts": "svc1-cert-chain-ke",
						},
					},
				},
			},
			wantErr: errors.New("malformed annotation appmesh.k8s.aws/secretMounts, expected format: secretName:mountPath"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &envoyMutator{}
			got, err := m.getSecretMounts(tt.args.pod)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, got, tt.want)
			}
		})
	}
}

func Test_envoyMutator_getAugmentedMeshName(t *testing.T) {
	type fields struct {
		ms            *appmesh.Mesh
		mutatorConfig envoyMutatorConfig
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "virtualNode's resourceOwner is same as meshOwner - meshOwner unset",
			fields: fields{
				ms: &appmesh.Mesh{
					Spec: appmesh.MeshSpec{
						AWSName: aws.String("my-mesh"),
					},
				},
				mutatorConfig: envoyMutatorConfig{
					accountID: "000000000000",
				},
			},
			want: "my-mesh",
		},
		{
			name: "virtualNode's resourceOwner is same as meshOwner - meshOwner set",
			fields: fields{
				ms: &appmesh.Mesh{
					Spec: appmesh.MeshSpec{
						AWSName:   aws.String("my-mesh"),
						MeshOwner: aws.String("000000000000"),
					},
				},
				mutatorConfig: envoyMutatorConfig{
					accountID: "000000000000",
				},
			},
			want: "my-mesh",
		},
		{
			name: "virtualNode's resourceOwner is different than meshOwner",
			fields: fields{
				ms: &appmesh.Mesh{
					Spec: appmesh.MeshSpec{
						AWSName:   aws.String("my-mesh"),
						MeshOwner: aws.String("111111111111"),
					},
				},
				mutatorConfig: envoyMutatorConfig{
					accountID: "000000000000",
				},
			},
			want: "my-mesh@111111111111",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &envoyMutator{
				ms:            tt.fields.ms,
				mutatorConfig: tt.fields.mutatorConfig,
			}
			got := m.getAugmentedMeshName()
			assert.Equal(t, tt.want, got)
		})
	}
}
