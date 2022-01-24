package inject

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
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
		xRayImage:             "public.ecr.aws/xray/aws-xray-daemon",
		xRayDaemonPort:        2000,
		xRayConfigRoleArn:     "",
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
							Image: "public.ecr.aws/xray/aws-xray-daemon",
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
			name: "no resource limits",
			fields: fields{
				enabled: true,
				mutatorConfig: xrayMutatorConfig{
					awsRegion:             "us-west-2",
					sidecarCPURequests:    cpuRequests.String(),
					sidecarMemoryRequests: memoryRequests.String(),
					xRayImage:             "public.ecr.aws/xray/aws-xray-daemon",
					xRayDaemonPort:        2000,
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
							Name:  "xray-daemon",
							Image: "public.ecr.aws/xray/aws-xray-daemon",
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
		{
			name: "no cpu limits",
			fields: fields{
				enabled: true,
				mutatorConfig: xrayMutatorConfig{
					awsRegion:             "us-west-2",
					sidecarCPURequests:    cpuRequests.String(),
					sidecarMemoryRequests: memoryRequests.String(),
					sidecarMemoryLimits:   memoryLimits.String(),
					xRayImage:             "public.ecr.aws/xray/aws-xray-daemon",
					xRayDaemonPort:        2000,
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
							Name:  "xray-daemon",
							Image: "public.ecr.aws/xray/aws-xray-daemon",
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
				enabled: true,
				mutatorConfig: xrayMutatorConfig{
					awsRegion:             "us-west-2",
					sidecarCPURequests:    cpuRequests.String(),
					sidecarMemoryRequests: memoryRequests.String(),
					sidecarCPULimits:      cpuLimits.String(),
					xRayImage:             "public.ecr.aws/xray/aws-xray-daemon",
					xRayDaemonPort:        2000,
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
							Name:  "xray-daemon",
							Image: "public.ecr.aws/xray/aws-xray-daemon",
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
									"cpu": cpuLimits,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "missing xray daemon port",
			fields: fields{
				enabled: true,
				mutatorConfig: xrayMutatorConfig{
					awsRegion:             "us-west-2",
					sidecarCPURequests:    cpuRequests.String(),
					sidecarMemoryRequests: memoryRequests.String(),
					xRayImage:             "public.ecr.aws/xray/aws-xray-daemon",
				},
			},
			args: args{
				pod: pod,
			},
			wantErr: errors.New("Missing configuration parameters: xRayDaemonPort"),
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
			name: "missing aws region",
			fields: fields{
				enabled: true,
				mutatorConfig: xrayMutatorConfig{
					sidecarCPURequests:    cpuRequests.String(),
					sidecarMemoryRequests: memoryRequests.String(),
					xRayImage:             "public.ecr.aws/xray/aws-xray-daemon",
					xRayDaemonPort:        2000,
				},
			},
			args: args{
				pod: pod,
			},
			wantErr: errors.New("Missing configuration parameters: AWSRegion"),
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
			name: "missing xray image",
			fields: fields{
				enabled: true,
				mutatorConfig: xrayMutatorConfig{
					awsRegion:             "us-west-2",
					sidecarCPURequests:    cpuRequests.String(),
					sidecarMemoryRequests: memoryRequests.String(),
					xRayDaemonPort:        2000,
				},
			},
			args: args{
				pod: pod,
			},
			wantErr: errors.New("Missing configuration parameters: xRayImage"),
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
			name: "missing aws region and xray image",
			fields: fields{
				enabled: true,
				mutatorConfig: xrayMutatorConfig{
					sidecarCPURequests:    cpuRequests.String(),
					sidecarMemoryRequests: memoryRequests.String(),
					xRayDaemonPort:        2000,
				},
			},
			args: args{
				pod: pod,
			},
			wantErr: errors.New("Missing configuration parameters: AWSRegion,xRayImage"),
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
			name: "missing aws region, xray image and xray daemon port",
			fields: fields{
				enabled: true,
				mutatorConfig: xrayMutatorConfig{
					sidecarCPURequests:    cpuRequests.String(),
					sidecarMemoryRequests: memoryRequests.String(),
				},
			},
			args: args{
				pod: pod,
			},
			wantErr: errors.New("Missing configuration parameters: AWSRegion,xRayImage,xRayDaemonPort"),
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
							Image: "public.ecr.aws/xray/aws-xray-daemon",
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
		{
			name: "xray args contain logLevel and roleArn",
			fields: fields{
				enabled: true,
				mutatorConfig: xrayMutatorConfig{
					awsRegion:             "us-west-2",
					sidecarCPURequests:    cpuRequests.String(),
					sidecarMemoryRequests: memoryRequests.String(),
					sidecarCPULimits:      cpuLimits.String(),
					sidecarMemoryLimits:   memoryLimits.String(),
					xRayImage:             "public.ecr.aws/xray/aws-xray-daemon",
					xRayDaemonPort:        2000,
					xRayLogLevel:          "dev",
					xRayConfigRoleArn:     "arn:aws:iam::123456789012:role/xray-cross-account",
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
							Name:  "xray-daemon",
							Image: "public.ecr.aws/xray/aws-xray-daemon",
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
							Args: []string{
								"--log-level",
								"dev",
								"--role-arn",
								"arn:aws:iam::123456789012:role/xray-cross-account",
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
			name: "xray args contain invalid logLevel",
			fields: fields{
				enabled: true,
				mutatorConfig: xrayMutatorConfig{
					awsRegion:             "us-west-2",
					sidecarCPURequests:    cpuRequests.String(),
					sidecarMemoryRequests: memoryRequests.String(),
					sidecarCPULimits:      cpuLimits.String(),
					sidecarMemoryLimits:   memoryLimits.String(),
					xRayImage:             "public.ecr.aws/xray/aws-xray-daemon",
					xRayDaemonPort:        2000,
					xRayLogLevel:          "stage",
					xRayConfigRoleArn:     "arn:aws:iam::123456789012:role/xray-cross-account",
				},
			},
			args: args{
				pod: pod,
			},
			wantErr: errors.New("tracing.logLevel: \"stage\" is not valid." +
				" Set one of dev, debug, info, prod, warn, error"),
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
			name: "xray args contain invalid roleArn",
			fields: fields{
				enabled: true,
				mutatorConfig: xrayMutatorConfig{
					awsRegion:             "us-west-2",
					sidecarCPURequests:    cpuRequests.String(),
					sidecarMemoryRequests: memoryRequests.String(),
					sidecarCPULimits:      cpuLimits.String(),
					sidecarMemoryLimits:   memoryLimits.String(),
					xRayImage:             "public.ecr.aws/xray/aws-xray-daemon",
					xRayDaemonPort:        2000,
					xRayLogLevel:          "dev",
					xRayConfigRoleArn:     "xray-cross-account arn",
				},
			},
			args: args{
				pod: pod,
			},
			wantErr: errors.New("tracing.role: \"xray-cross-account arn\" is not a valid `--role-arn`." +
				" Please refer to AWS X-Ray Documentation for more information"),
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
			name: "xray daemon config volume mount annotation",
			fields: fields{
				enabled: true,
				mutatorConfig: xrayMutatorConfig{
					awsRegion:             "us-west-2",
					sidecarCPURequests:    cpuRequests.String(),
					sidecarMemoryRequests: memoryRequests.String(),
					sidecarCPULimits:      cpuLimits.String(),
					sidecarMemoryLimits:   memoryLimits.String(),
					xRayImage:             "public.ecr.aws/xray/aws-xray-daemon",
					xRayDaemonPort:        2000,
					xRayLogLevel:          "debug",
					xRayConfigRoleArn:     "arn:aws:iam::123456789012:role/xray-cross-account",
				},
			},
			args: args{
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "my-ns",
						Name:      "my-pod",
						Annotations: map[string]string{
							"appmesh.k8s.aws/xrayAgentConfigMount": "xray-daemon-config:/tmp/",
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
						"appmesh.k8s.aws/xrayAgentConfigMount": "xray-daemon-config:/tmp/",
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
							Image: "public.ecr.aws/xray/aws-xray-daemon",
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
							Command: []string{
								"/xray",
								"--config",
								"/tmp/xray-daemon.yaml",
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
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "xray-daemon-config",
									MountPath: "/tmp/",
									ReadOnly:  true,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "xray daemon config with more than 1 volume mount annotation",
			fields: fields{
				enabled: true,
				mutatorConfig: xrayMutatorConfig{
					awsRegion:             "us-west-2",
					sidecarCPURequests:    cpuRequests.String(),
					sidecarMemoryRequests: memoryRequests.String(),
					sidecarCPULimits:      cpuLimits.String(),
					sidecarMemoryLimits:   memoryLimits.String(),
					xRayImage:             "public.ecr.aws/xray/aws-xray-daemon",
					xRayDaemonPort:        2000,
					xRayLogLevel:          "dev",
					xRayConfigRoleArn:     "arn:aws:iam::123456789012:role/xray-cross-account",
				},
			},
			args: args{
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "my-ns",
						Name:      "my-pod",
						Annotations: map[string]string{
							"appmesh.k8s.aws/xrayAgentConfigMount": "xray-daemon-config1:/tmp/,xray-daemon-config2:/tmp/,",
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
			wantErr: errors.New("provide only one config mount for annotation " +
				"\"appmesh.k8s.aws/xrayAgentConfigMount: xray-daemon-config1:/tmp/,xray-daemon-config2:/tmp/,\""),
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
			name: "xray daemon config with malformed annotation",
			fields: fields{
				enabled: true,
				mutatorConfig: xrayMutatorConfig{
					awsRegion:             "us-west-2",
					sidecarCPURequests:    cpuRequests.String(),
					sidecarMemoryRequests: memoryRequests.String(),
					sidecarCPULimits:      cpuLimits.String(),
					sidecarMemoryLimits:   memoryLimits.String(),
					xRayImage:             "public.ecr.aws/xray/aws-xray-daemon",
					xRayDaemonPort:        2000,
					xRayLogLevel:          "dev",
					xRayConfigRoleArn:     "arn:aws:iam::123456789012:role/xray-cross-account",
				},
			},
			args: args{
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "my-ns",
						Name:      "my-pod",
						Annotations: map[string]string{
							"appmesh.k8s.aws/xrayAgentConfigMount": "xray-daemon-config1-/tmp/",
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
			wantErr: errors.New("malformed annotation \"appmesh.k8s.aws/xrayAgentConfigMount\"," +
				" expected format: volumeName:mountPath"),
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
