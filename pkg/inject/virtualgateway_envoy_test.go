package inject

import (
	"testing"

	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_virtualGatewayEnvoyMutator_mutate(t *testing.T) {
	SDSVolumeType := corev1.HostPathSocket
	ms := &appmesh.Mesh{
		ObjectMeta: metav1.ObjectMeta{
			Name: "mesh",
		},
		Spec: appmesh.MeshSpec{
			AWSName: aws.String("my-mesh"),
		},
	}
	vg := &appmesh.VirtualGateway{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-ns",
			Name:      "my-vg",
		},
		Spec: appmesh.VirtualGatewaySpec{
			AWSName: aws.String("my-vg_my-ns"),
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
					Name:  "envoy",
					Image: "envoy:v2",
				},
			},
		},
	}

	podSkipOverride := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-ns",
			Name:      "my-pod",
			Annotations: map[string]string{
				"appmesh.k8s.aws/virtualGatewaySkipImageOverride": "enabled",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "envoy",
					Image: "envoy:custom_version",
				},
			},
		},
	}

	podMultipleContainers := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-ns",
			Name:      "my-pod",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "app1",
					Image: "app1:v2",
				},
				{
					Name:  "envoy",
					Image: "envoy:v2",
				},
				{
					Name:  "app2",
					Image: "app2:v1",
				},
			},
		},
	}

	podExistingEnv := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-ns",
			Name:      "my-pod",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "envoy",
					Image: "envoy:v2",
					Env: []corev1.EnvVar{
						{
							Name:  "TEST_ENV",
							Value: "test_val",
						}},
				},
			},
		},
	}

	podWithImageStub := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-ns",
			Name:      "my-pod",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "envoy",
					Image: "injector-envoy-image",
					Env: []corev1.EnvVar{
						{
							Name:  "TEST_ENV",
							Value: "test_val",
						}},
				},
			},
		},
	}

	podDuplicateEnv := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-ns",
			Name:      "my-pod",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "envoy",
					Image: "envoy:v2",
					Env: []corev1.EnvVar{
						{
							Name:  "TEST_ENV",
							Value: "test_val",
						},
						{
							Name:  "APPMESH_VIRTUAL_NODE_NAME",
							Value: "incorrect_node_name",
						},
						{
							Name:  "APPMESH_PREVIEW",
							Value: "1",
						},
					},
				},
			},
		},
	}

	podExistingProbe := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-ns",
			Name:      "my-pod",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "envoy",
					Image: "envoy:v2",
					ReadinessProbe: &corev1.Probe{
						Handler: corev1.Handler{

							Exec: &corev1.ExecAction{Command: []string{
								"sh", "-c", "curl -s http://localhost:8810/server_info | grep state | grep -q LIVE",
							}},
						},
						InitialDelaySeconds: 20,
						TimeoutSeconds:      1,
						PeriodSeconds:       30,
						SuccessThreshold:    2,
						FailureThreshold:    3,
					},
				},
			},
		},
	}

	podSDSDisabled := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-ns",
			Name:      "my-pod",
			Annotations: map[string]string{
				"appmesh.k8s.aws/sds": "disabled",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "envoy",
					Image: "envoy:v2",
					ReadinessProbe: &corev1.Probe{
						Handler: corev1.Handler{

							Exec: &corev1.ExecAction{Command: []string{
								"sh", "-c", "curl -s http://localhost:8810/server_info | grep state | grep -q LIVE",
							}},
						},
						InitialDelaySeconds: 20,
						TimeoutSeconds:      1,
						PeriodSeconds:       30,
						SuccessThreshold:    2,
						FailureThreshold:    3,
					},
				},
			},
		},
	}

	podWithExistingSDSVolume := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-ns",
			Name:      "my-pod",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "envoy",
					Image: "envoy:v2",
					ReadinessProbe: &corev1.Probe{
						Handler: corev1.Handler{

							Exec: &corev1.ExecAction{Command: []string{
								"sh", "-c", "curl -s http://localhost:8810/server_info | grep state | grep -q LIVE",
							}},
						},
						InitialDelaySeconds: 20,
						TimeoutSeconds:      1,
						PeriodSeconds:       30,
						SuccessThreshold:    2,
						FailureThreshold:    3,
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "appmesh-sds-socket-volume",
							MountPath: "/run/spire/sockets/agent.sock",
						},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "appmesh-sds-socket-volume",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: "/run/spire/sockets/agent.sock",
							Type: &SDSVolumeType,
						},
					},
				},
			},
		},
	}

	type fields struct {
		vg            *appmesh.VirtualGateway
		ms            *appmesh.Mesh
		mutatorConfig virtualGatwayEnvoyConfig
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
			name: "env append",
			fields: fields{
				vg: vg,
				ms: ms,
				mutatorConfig: virtualGatwayEnvoyConfig{
					awsRegion:                  "us-west-2",
					preview:                    false,
					logLevel:                   "debug",
					adminAccessPort:            9901,
					adminAccessLogFile:         "/tmp/envoy_admin_access.log",
					sidecarImage:               "envoy:v2",
					readinessProbeInitialDelay: 1,
					readinessProbePeriod:       10,
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
							Name:  "envoy",
							Image: "envoy:v2",
							Env: []corev1.EnvVar{
								{
									Name:  "ENVOY_LOG_LEVEL",
									Value: "debug",
								},
								{
									Name:  "ENVOY_ADMIN_ACCESS_PORT",
									Value: "9901",
								},
								{
									Name:  "ENVOY_ADMIN_ACCESS_LOG_FILE",
									Value: "/tmp/envoy_admin_access.log",
								},
								{
									Name:  "AWS_REGION",
									Value: "us-west-2",
								},
								{
									Name:  "APPMESH_VIRTUAL_NODE_NAME",
									Value: "mesh/my-mesh/virtualGateway/my-vg_my-ns",
								},
								{
									Name:  "APPMESH_PREVIEW",
									Value: "0",
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
						},
					},
				},
			},
		},
		{
			name: "skip image override",
			fields: fields{
				vg: vg,
				ms: ms,
				mutatorConfig: virtualGatwayEnvoyConfig{
					awsRegion:                  "us-west-2",
					preview:                    false,
					logLevel:                   "debug",
					adminAccessPort:            9901,
					adminAccessLogFile:         "/tmp/envoy_admin_access.log",
					sidecarImage:               "envoy:v2",
					readinessProbeInitialDelay: 3,
					readinessProbePeriod:       5,
				},
			},
			args: args{
				pod: podSkipOverride,
			},
			wantPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "my-ns",
					Name:      "my-pod",
					Annotations: map[string]string{
						"appmesh.k8s.aws/virtualGatewaySkipImageOverride": "enabled",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "envoy",
							Image: "envoy:custom_version",
							Env: []corev1.EnvVar{
								{
									Name:  "ENVOY_LOG_LEVEL",
									Value: "debug",
								},
								{
									Name:  "ENVOY_ADMIN_ACCESS_PORT",
									Value: "9901",
								},
								{
									Name:  "ENVOY_ADMIN_ACCESS_LOG_FILE",
									Value: "/tmp/envoy_admin_access.log",
								},
								{
									Name:  "AWS_REGION",
									Value: "us-west-2",
								},
								{
									Name:  "APPMESH_VIRTUAL_NODE_NAME",
									Value: "mesh/my-mesh/virtualGateway/my-vg_my-ns",
								},
								{
									Name:  "APPMESH_PREVIEW",
									Value: "0",
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
						},
					},
				},
			},
		},
		{
			name: "pod multiple containers",
			fields: fields{
				vg: vg,
				ms: ms,
				mutatorConfig: virtualGatwayEnvoyConfig{
					awsRegion:                  "us-west-2",
					preview:                    false,
					logLevel:                   "debug",
					adminAccessPort:            9901,
					adminAccessLogFile:         "/tmp/envoy_admin_access.log",
					sidecarImage:               "envoy:v2",
					readinessProbeInitialDelay: 1,
					readinessProbePeriod:       10,
				},
			},
			args: args{
				pod: podMultipleContainers,
			},
			wantPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "my-ns",
					Name:      "my-pod",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "app1",
							Image: "app1:v2",
						},
						{
							Name:  "app2",
							Image: "app2:v1",
						},
						{
							Name:  "envoy",
							Image: "envoy:v2",
							Env: []corev1.EnvVar{
								{
									Name:  "ENVOY_LOG_LEVEL",
									Value: "debug",
								},
								{
									Name:  "ENVOY_ADMIN_ACCESS_PORT",
									Value: "9901",
								},
								{
									Name:  "ENVOY_ADMIN_ACCESS_LOG_FILE",
									Value: "/tmp/envoy_admin_access.log",
								},
								{
									Name:  "AWS_REGION",
									Value: "us-west-2",
								},
								{
									Name:  "APPMESH_VIRTUAL_NODE_NAME",
									Value: "mesh/my-mesh/virtualGateway/my-vg_my-ns",
								},
								{
									Name:  "APPMESH_PREVIEW",
									Value: "0",
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
						},
					},
				},
			},
		},
		{
			name: "pod with existing env and image override",
			fields: fields{
				vg: vg,
				ms: ms,
				mutatorConfig: virtualGatwayEnvoyConfig{
					awsRegion:                  "us-west-2",
					preview:                    false,
					logLevel:                   "debug",
					adminAccessPort:            9901,
					adminAccessLogFile:         "/tmp/envoy_admin_access.log",
					sidecarImage:               "envoy:v3",
					readinessProbeInitialDelay: 1,
					readinessProbePeriod:       10,
				},
			},
			args: args{
				pod: podExistingEnv,
			},
			wantPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "my-ns",
					Name:      "my-pod",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "envoy",
							Image: "envoy:v3",
							Env: []corev1.EnvVar{
								{
									Name:  "TEST_ENV",
									Value: "test_val",
								},
								{
									Name:  "APPMESH_VIRTUAL_NODE_NAME",
									Value: "mesh/my-mesh/virtualGateway/my-vg_my-ns",
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
									Name:  "ENVOY_ADMIN_ACCESS_PORT",
									Value: "9901",
								},
								{
									Name:  "ENVOY_ADMIN_ACCESS_LOG_FILE",
									Value: "/tmp/envoy_admin_access.log",
								},
								{
									Name:  "AWS_REGION",
									Value: "us-west-2",
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
						},
					},
				},
			},
		},
		{
			name: "pod with image stub",
			fields: fields{
				vg: vg,
				ms: ms,
				mutatorConfig: virtualGatwayEnvoyConfig{
					awsRegion:                  "us-west-2",
					preview:                    false,
					logLevel:                   "debug",
					adminAccessPort:            9901,
					adminAccessLogFile:         "/tmp/envoy_admin_access.log",
					sidecarImage:               "envoy:v3",
					readinessProbeInitialDelay: 1,
					readinessProbePeriod:       10,
				},
			},
			args: args{
				pod: podWithImageStub,
			},
			wantPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "my-ns",
					Name:      "my-pod",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "envoy",
							Image: "envoy:v3",
							Env: []corev1.EnvVar{
								{
									Name:  "TEST_ENV",
									Value: "test_val",
								},
								{
									Name:  "APPMESH_VIRTUAL_NODE_NAME",
									Value: "mesh/my-mesh/virtualGateway/my-vg_my-ns",
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
									Name:  "ENVOY_ADMIN_ACCESS_PORT",
									Value: "9901",
								},
								{
									Name:  "ENVOY_ADMIN_ACCESS_LOG_FILE",
									Value: "/tmp/envoy_admin_access.log",
								},
								{
									Name:  "AWS_REGION",
									Value: "us-west-2",
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
						},
					},
				},
			},
		},
		{
			name: "pod with duplicate env",
			fields: fields{
				vg: vg,
				ms: ms,
				mutatorConfig: virtualGatwayEnvoyConfig{
					awsRegion:                  "us-west-2",
					preview:                    false,
					logLevel:                   "debug",
					adminAccessPort:            9901,
					adminAccessLogFile:         "/tmp/envoy_admin_access.log",
					sidecarImage:               "envoy:v2",
					readinessProbeInitialDelay: 1,
					readinessProbePeriod:       10,
				},
			},
			args: args{
				pod: podDuplicateEnv,
			},
			wantPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "my-ns",
					Name:      "my-pod",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "envoy",
							Image: "envoy:v2",
							Env: []corev1.EnvVar{
								{
									Name:  "TEST_ENV",
									Value: "test_val",
								},
								{
									Name:  "APPMESH_VIRTUAL_NODE_NAME",
									Value: "mesh/my-mesh/virtualGateway/my-vg_my-ns",
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
									Name:  "ENVOY_ADMIN_ACCESS_PORT",
									Value: "9901",
								},
								{
									Name:  "ENVOY_ADMIN_ACCESS_LOG_FILE",
									Value: "/tmp/envoy_admin_access.log",
								},
								{
									Name:  "AWS_REGION",
									Value: "us-west-2",
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
						},
					},
				},
			},
		},
		{
			name: "xray",
			fields: fields{
				vg: vg,
				ms: ms,
				mutatorConfig: virtualGatwayEnvoyConfig{
					awsRegion:                  "us-west-2",
					preview:                    false,
					logLevel:                   "debug",
					adminAccessPort:            9901,
					adminAccessLogFile:         "/tmp/envoy_admin_access.log",
					sidecarImage:               "envoy:v2",
					enableXrayTracing:          true,
					xrayDaemonPort:             2000,
					readinessProbeInitialDelay: 1,
					readinessProbePeriod:       10,
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
							Name:  "envoy",
							Image: "envoy:v2",
							Env: []corev1.EnvVar{
								{
									Name:  "ENVOY_LOG_LEVEL",
									Value: "debug",
								},
								{
									Name:  "ENVOY_ADMIN_ACCESS_PORT",
									Value: "9901",
								},
								{
									Name:  "ENVOY_ADMIN_ACCESS_LOG_FILE",
									Value: "/tmp/envoy_admin_access.log",
								},
								{
									Name:  "AWS_REGION",
									Value: "us-west-2",
								},
								{
									Name:  "APPMESH_VIRTUAL_NODE_NAME",
									Value: "mesh/my-mesh/virtualGateway/my-vg_my-ns",
								},
								{
									Name:  "APPMESH_PREVIEW",
									Value: "0",
								},
								{
									Name:  "ENABLE_ENVOY_XRAY_TRACING",
									Value: "1",
								},
								{
									Name:  "XRAY_DAEMON_PORT",
									Value: "2000",
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
						},
					},
				},
			},
		},
		{
			name: "jaeger",
			fields: fields{
				vg: vg,
				ms: ms,
				mutatorConfig: virtualGatwayEnvoyConfig{
					awsRegion:                  "us-west-2",
					preview:                    false,
					logLevel:                   "debug",
					adminAccessPort:            9901,
					adminAccessLogFile:         "/tmp/envoy_admin_access.log",
					sidecarImage:               "envoy:v2",
					enableJaegerTracing:        true,
					readinessProbeInitialDelay: 1,
					readinessProbePeriod:       10,
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
							Name:  "envoy",
							Image: "envoy:v2",
							Env: []corev1.EnvVar{
								{
									Name:  "ENVOY_LOG_LEVEL",
									Value: "debug",
								},
								{
									Name:  "ENVOY_ADMIN_ACCESS_PORT",
									Value: "9901",
								},
								{
									Name:  "ENVOY_ADMIN_ACCESS_LOG_FILE",
									Value: "/tmp/envoy_admin_access.log",
								},
								{
									Name:  "AWS_REGION",
									Value: "us-west-2",
								},
								{
									Name:  "APPMESH_VIRTUAL_NODE_NAME",
									Value: "mesh/my-mesh/virtualGateway/my-vg_my-ns",
								},
								{
									Name:  "APPMESH_PREVIEW",
									Value: "0",
								},
								{
									Name:  "ENABLE_ENVOY_JAEGER_TRACING",
									Value: "1",
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
						},
					},
				},
			},
		},
		{
			name: "pod existing readiness probe",
			fields: fields{
				vg: vg,
				ms: ms,
				mutatorConfig: virtualGatwayEnvoyConfig{
					awsRegion:                  "us-west-2",
					preview:                    false,
					logLevel:                   "debug",
					adminAccessPort:            9901,
					adminAccessLogFile:         "/tmp/envoy_admin_access.log",
					sidecarImage:               "envoy:v2",
					readinessProbeInitialDelay: 1,
					readinessProbePeriod:       10,
				},
			},
			args: args{
				pod: podExistingProbe,
			},
			wantPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "my-ns",
					Name:      "my-pod",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "envoy",
							Image: "envoy:v2",
							Env: []corev1.EnvVar{
								{
									Name:  "ENVOY_LOG_LEVEL",
									Value: "debug",
								},
								{
									Name:  "ENVOY_ADMIN_ACCESS_PORT",
									Value: "9901",
								},
								{
									Name:  "ENVOY_ADMIN_ACCESS_LOG_FILE",
									Value: "/tmp/envoy_admin_access.log",
								},
								{
									Name:  "AWS_REGION",
									Value: "us-west-2",
								},
								{
									Name:  "APPMESH_VIRTUAL_NODE_NAME",
									Value: "mesh/my-mesh/virtualGateway/my-vg_my-ns",
								},
								{
									Name:  "APPMESH_PREVIEW",
									Value: "0",
								},
							},
							ReadinessProbe: &corev1.Probe{
								Handler: corev1.Handler{

									Exec: &corev1.ExecAction{Command: []string{
										"sh", "-c", "curl -s http://localhost:8810/server_info | grep state | grep -q LIVE",
									}},
								},
								InitialDelaySeconds: 20,
								TimeoutSeconds:      1,
								PeriodSeconds:       30,
								SuccessThreshold:    2,
								FailureThreshold:    3,
							},
						},
					},
				},
			},
		},
		{
			name: "enable sds controller flag set + no annotation",
			fields: fields{
				vg: vg,
				ms: ms,
				mutatorConfig: virtualGatwayEnvoyConfig{
					awsRegion:                  "us-west-2",
					preview:                    false,
					logLevel:                   "debug",
					enableSDS:                  true,
					sdsUdsPath:                 "/run/spire/sockets/agent.sock",
					adminAccessPort:            9901,
					adminAccessLogFile:         "/tmp/envoy_admin_access.log",
					sidecarImage:               "envoy:v2",
					readinessProbeInitialDelay: 1,
					readinessProbePeriod:       10,
				},
			},
			args: args{
				pod: podExistingProbe,
			},
			wantPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "my-ns",
					Name:      "my-pod",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "envoy",
							Image: "envoy:v2",
							Env: []corev1.EnvVar{
								{
									Name:  "ENVOY_LOG_LEVEL",
									Value: "debug",
								},
								{
									Name:  "ENVOY_ADMIN_ACCESS_PORT",
									Value: "9901",
								},
								{
									Name:  "ENVOY_ADMIN_ACCESS_LOG_FILE",
									Value: "/tmp/envoy_admin_access.log",
								},
								{
									Name:  "AWS_REGION",
									Value: "us-west-2",
								},
								{
									Name:  "APPMESH_SDS_SOCKET_PATH",
									Value: "/run/spire/sockets/agent.sock",
								},
								{
									Name:  "APPMESH_VIRTUAL_NODE_NAME",
									Value: "mesh/my-mesh/virtualGateway/my-vg_my-ns",
								},
								{
									Name:  "APPMESH_PREVIEW",
									Value: "0",
								},
							},
							ReadinessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									Exec: &corev1.ExecAction{Command: []string{
										"sh", "-c", "curl -s http://localhost:8810/server_info | grep state | grep -q LIVE",
									}},
								},
								InitialDelaySeconds: 20,
								TimeoutSeconds:      1,
								PeriodSeconds:       30,
								SuccessThreshold:    2,
								FailureThreshold:    3,
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "appmesh-sds-socket-volume",
									MountPath: "/run/spire/sockets/agent.sock",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "appmesh-sds-socket-volume",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/run/spire/sockets/agent.sock",
									Type: &SDSVolumeType,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "enable sds controller flag set + pod annotation to disable sds",
			fields: fields{
				vg: vg,
				ms: ms,
				mutatorConfig: virtualGatwayEnvoyConfig{
					awsRegion:                  "us-west-2",
					preview:                    false,
					logLevel:                   "debug",
					enableSDS:                  true,
					sdsUdsPath:                 "/run/spire/sockets/agent.sock",
					adminAccessPort:            9901,
					adminAccessLogFile:         "/tmp/envoy_admin_access.log",
					sidecarImage:               "envoy:v2",
					readinessProbeInitialDelay: 1,
					readinessProbePeriod:       10,
				},
			},
			args: args{
				pod: podSDSDisabled,
			},
			wantPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "my-ns",
					Name:      "my-pod",
					Annotations: map[string]string{
						"appmesh.k8s.aws/sds": "disabled",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "envoy",
							Image: "envoy:v2",
							Env: []corev1.EnvVar{
								{
									Name:  "ENVOY_LOG_LEVEL",
									Value: "debug",
								},
								{
									Name:  "ENVOY_ADMIN_ACCESS_PORT",
									Value: "9901",
								},
								{
									Name:  "ENVOY_ADMIN_ACCESS_LOG_FILE",
									Value: "/tmp/envoy_admin_access.log",
								},
								{
									Name:  "AWS_REGION",
									Value: "us-west-2",
								},
								{
									Name:  "APPMESH_VIRTUAL_NODE_NAME",
									Value: "mesh/my-mesh/virtualGateway/my-vg_my-ns",
								},
								{
									Name:  "APPMESH_PREVIEW",
									Value: "0",
								},
							},
							ReadinessProbe: &corev1.Probe{
								Handler: corev1.Handler{

									Exec: &corev1.ExecAction{Command: []string{
										"sh", "-c", "curl -s http://localhost:8810/server_info | grep state | grep -q LIVE",
									}},
								},
								InitialDelaySeconds: 20,
								TimeoutSeconds:      1,
								PeriodSeconds:       30,
								SuccessThreshold:    2,
								FailureThreshold:    3,
							},
						},
					},
				},
			},
		},
		{
			name: "enable sds controller flag set + no annotation + SDS Volume already present",
			fields: fields{
				vg: vg,
				ms: ms,
				mutatorConfig: virtualGatwayEnvoyConfig{
					awsRegion:                  "us-west-2",
					preview:                    false,
					logLevel:                   "debug",
					enableSDS:                  true,
					sdsUdsPath:                 "/run/spire/sockets/agent.sock",
					adminAccessPort:            9901,
					adminAccessLogFile:         "/tmp/envoy_admin_access.log",
					sidecarImage:               "envoy:v2",
					readinessProbeInitialDelay: 1,
					readinessProbePeriod:       10,
				},
			},
			args: args{
				pod: podWithExistingSDSVolume,
			},
			wantPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "my-ns",
					Name:      "my-pod",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "envoy",
							Image: "envoy:v2",
							Env: []corev1.EnvVar{
								{
									Name:  "ENVOY_LOG_LEVEL",
									Value: "debug",
								},
								{
									Name:  "ENVOY_ADMIN_ACCESS_PORT",
									Value: "9901",
								},
								{
									Name:  "ENVOY_ADMIN_ACCESS_LOG_FILE",
									Value: "/tmp/envoy_admin_access.log",
								},
								{
									Name:  "AWS_REGION",
									Value: "us-west-2",
								},
								{
									Name:  "APPMESH_SDS_SOCKET_PATH",
									Value: "/run/spire/sockets/agent.sock",
								},
								{
									Name:  "APPMESH_VIRTUAL_NODE_NAME",
									Value: "mesh/my-mesh/virtualGateway/my-vg_my-ns",
								},
								{
									Name:  "APPMESH_PREVIEW",
									Value: "0",
								},
							},
							ReadinessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									Exec: &corev1.ExecAction{Command: []string{
										"sh", "-c", "curl -s http://localhost:8810/server_info | grep state | grep -q LIVE",
									}},
								},
								InitialDelaySeconds: 20,
								TimeoutSeconds:      1,
								PeriodSeconds:       30,
								SuccessThreshold:    2,
								FailureThreshold:    3,
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "appmesh-sds-socket-volume",
									MountPath: "/run/spire/sockets/agent.sock",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "appmesh-sds-socket-volume",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/run/spire/sockets/agent.sock",
									Type: &SDSVolumeType,
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
			m := &virtualGatewayEnvoyConfig{
				vg:            tt.fields.vg,
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
					cmpopts.SortSlices(func(lhs corev1.EnvVar, rhs corev1.EnvVar) bool {
						return lhs.Name < rhs.Name
					}),
					cmpopts.SortSlices(func(lhs corev1.Container, rhs corev1.Container) bool {
						return lhs.Name < rhs.Name
					}),
				}
				assert.True(t, cmp.Equal(tt.wantPod, pod, opts), "diff", cmp.Diff(tt.wantPod, pod, opts))
			}
		})
	}
}
