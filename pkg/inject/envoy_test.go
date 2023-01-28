package inject

import (
	"errors"
	"testing"

	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	envPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-ns",
			Name:      "my-pod",
			Annotations: map[string]string{
				"appmesh.k8s.aws/sidecarEnv": "DD_ENV=prod, TEST_ENV1=env_val1, TEST_ENV2=env_val2",
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

	duplicateEnvPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-ns",
			Name:      "my-pod",
			Annotations: map[string]string{
				"appmesh.k8s.aws/sidecarEnv": "DD_ENV=prod, APPMESH_VIRTUAL_NODE_NAME=random_val",
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

	podDisableSds := &corev1.Pod{
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
					Name:  "app",
					Image: "app/v1",
				},
			},
		},
	}

	certPod := &corev1.Pod{
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
	}

	podMultipleContainer := &corev1.Pod{
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
	}

	annotationCpuRequest, _ := resource.ParseQuantity("128Mi")
	annotationMemoryRequest, _ := resource.ParseQuantity("20m")
	annotationCpuLimit, _ := resource.ParseQuantity("256Mi")
	annotationMemoryLimit, _ := resource.ParseQuantity("80m")

	SDSVolumeType := corev1.HostPathSocket

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
					adminAccessPort:            9901,
					preStopDelay:               "20",
					readinessProbeInitialDelay: 1,
					readinessProbePeriod:       10,
					sidecarImageRepository:     "envoy",
					sidecarImageTag:            "v2",
					sidecarCPURequests:         cpuRequests.String(),
					sidecarMemoryRequests:      memoryRequests.String(),
					sidecarCPULimits:           cpuLimits.String(),
					sidecarMemoryLimits:        memoryLimits.String(),
					controllerVersion:          "v1.4.1",
					k8sVersion:                 "v1.20.1-eks-fdsedv",
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
								PreStop: &corev1.LifecycleHandler{
									Exec: &corev1.ExecAction{Command: []string{
										"sh", "-c", "sleep 20",
									}},
								},
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{

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
									Name:  "ENVOY_ADMIN_ACCESS_PORT",
									Value: "9901",
								},
								{
									Name:  "AWS_REGION",
									Value: "us-west-2",
								},
								{
									Name:  "APPMESH_PLATFORM_APP_MESH_CONTROLLER_VERSION",
									Value: "v1.4.1",
								},
								{
									Name:  "APPMESH_PLATFORM_K8S_VERSION",
									Value: "v1.20.1-eks-fdsedv",
								},
								{
									Name:  "APPMESH_DUALSTACK_ENDPOINT",
									Value: "0",
								},
								{
									Name:  "ENVOY_ADMIN_ACCESS_ENABLE_IPV6",
									Value: "false",
								},
								{
									Name:  "APPMESH_FIPS_ENDPOINT",
									Value: "0",
								},
								{
									Name:  "APPNET_AGENT_ADMIN_MODE",
									Value: "uds",
								},
								{
									Name:  "APPNET_AGENT_ADMIN_UDS_PATH",
									Value: "/tmp/agent.sock",
								},
								{
									Name:  "APPMESH_PLATFORM_K8S_POD_UID",
									Value: "",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.uid",
										},
									},
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
					adminAccessPort:            9901,
					preStopDelay:               "20",
					readinessProbeInitialDelay: 10,
					readinessProbePeriod:       2,
					sidecarImageRepository:     "envoy",
					sidecarImageTag:            "v2",
					sidecarCPURequests:         cpuRequests.String(),
					sidecarMemoryRequests:      memoryRequests.String(),
					controllerVersion:          "v1.4.1",
					k8sVersion:                 "v1.20.1-eks-fdsedv",
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
								PreStop: &corev1.LifecycleHandler{
									Exec: &corev1.ExecAction{Command: []string{
										"sh", "-c", "sleep 20",
									}},
								},
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{

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
									Name:  "ENVOY_ADMIN_ACCESS_PORT",
									Value: "9901",
								},
								{
									Name:  "AWS_REGION",
									Value: "us-west-2",
								},
								{
									Name:  "APPMESH_PLATFORM_APP_MESH_CONTROLLER_VERSION",
									Value: "v1.4.1",
								},
								{
									Name:  "APPMESH_PLATFORM_K8S_VERSION",
									Value: "v1.20.1-eks-fdsedv",
								},
								{
									Name:  "APPMESH_DUALSTACK_ENDPOINT",
									Value: "0",
								},
								{
									Name:  "ENVOY_ADMIN_ACCESS_ENABLE_IPV6",
									Value: "false",
								},
								{
									Name:  "APPMESH_FIPS_ENDPOINT",
									Value: "0",
								},
								{
									Name:  "APPNET_AGENT_ADMIN_MODE",
									Value: "uds",
								},
								{
									Name:  "APPNET_AGENT_ADMIN_UDS_PATH",
									Value: "/tmp/agent.sock",
								},
								{
									Name:  "APPMESH_PLATFORM_K8S_POD_UID",
									Value: "",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.uid",
										},
									},
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
					adminAccessPort:            9901,
					preStopDelay:               "20",
					readinessProbeInitialDelay: 3,
					readinessProbePeriod:       5,
					sidecarImageRepository:     "envoy",
					sidecarImageTag:            "v2",
					sidecarCPURequests:         cpuRequests.String(),
					sidecarMemoryRequests:      memoryRequests.String(),
					enableXrayTracing:          true,
					xrayDaemonPort:             2000,
					xraySamplingRate:           "0.01",
					controllerVersion:          "v1.4.1",
					k8sVersion:                 "v1.20.1-eks-fdsedv",
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
								PreStop: &corev1.LifecycleHandler{
									Exec: &corev1.ExecAction{Command: []string{
										"sh", "-c", "sleep 20",
									}},
								},
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{

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
									Name:  "ENVOY_ADMIN_ACCESS_PORT",
									Value: "9901",
								},
								{
									Name:  "AWS_REGION",
									Value: "us-west-2",
								},
								{
									Name:  "ENABLE_ENVOY_XRAY_TRACING",
									Value: "1",
								},
								{
									Name:  "XRAY_DAEMON_PORT",
									Value: "2000",
								},
								{
									Name:  "XRAY_SAMPLING_RATE",
									Value: "0.01",
								},
								{
									Name:  "APPMESH_PLATFORM_APP_MESH_CONTROLLER_VERSION",
									Value: "v1.4.1",
								},
								{
									Name:  "APPMESH_PLATFORM_K8S_VERSION",
									Value: "v1.20.1-eks-fdsedv",
								},
								{
									Name:  "APPMESH_DUALSTACK_ENDPOINT",
									Value: "0",
								},
								{
									Name:  "ENVOY_ADMIN_ACCESS_ENABLE_IPV6",
									Value: "false",
								},
								{
									Name:  "APPMESH_FIPS_ENDPOINT",
									Value: "0",
								},
								{
									Name:  "APPNET_AGENT_ADMIN_MODE",
									Value: "uds",
								},
								{
									Name:  "APPNET_AGENT_ADMIN_UDS_PATH",
									Value: "/tmp/agent.sock",
								},
								{
									Name:  "APPMESH_PLATFORM_K8S_POD_UID",
									Value: "",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.uid",
										},
									},
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
			name: "xray tracing with bad sampling rate",
			fields: fields{
				vn: vn,
				ms: ms,
				mutatorConfig: envoyMutatorConfig{
					awsRegion:                  "us-west-2",
					preview:                    false,
					logLevel:                   "debug",
					adminAccessPort:            9901,
					preStopDelay:               "20",
					readinessProbeInitialDelay: 3,
					readinessProbePeriod:       5,
					sidecarImageRepository:     "envoy",
					sidecarImageTag:            "v2",
					sidecarCPURequests:         cpuRequests.String(),
					sidecarMemoryRequests:      memoryRequests.String(),
					enableXrayTracing:          true,
					xrayDaemonPort:             2000,
					xraySamplingRate:           "5%",
				},
			},
			args: args{
				pod: pod,
			},
			wantErr: errors.New("tracing.samplingRate should be a decimal between 0 & 1.00, " +
				"but instead got 5% strconv.ParseFloat: parsing \"5%\": invalid syntax"),
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
					adminAccessPort:            9901,
					preStopDelay:               "20",
					readinessProbeInitialDelay: 1,
					readinessProbePeriod:       10,
					sidecarImageRepository:     "envoy",
					sidecarImageTag:            "v2",
					sidecarCPURequests:         cpuRequests.String(),
					sidecarMemoryRequests:      memoryRequests.String(),
					enableJaegerTracing:        true,
					jaegerPort:                 "8000",
					jaegerAddress:              "localhost",
					controllerVersion:          "v1.4.1",
					k8sVersion:                 "v1.20.1-eks-fdsedv",
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
								PreStop: &corev1.LifecycleHandler{
									Exec: &corev1.ExecAction{Command: []string{
										"sh", "-c", "sleep 20",
									}},
								},
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{

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
									Name:  "ENVOY_ADMIN_ACCESS_PORT",
									Value: "9901",
								},
								{
									Name:  "ENABLE_ENVOY_JAEGER_TRACING",
									Value: "1",
								},
								{
									Name:  "JAEGER_TRACER_PORT",
									Value: "8000",
								},
								{
									Name:  "JAEGER_TRACER_ADDRESS",
									Value: "localhost",
								},
								{
									Name:  "AWS_REGION",
									Value: "us-west-2",
								},
								{
									Name:  "APPMESH_PLATFORM_APP_MESH_CONTROLLER_VERSION",
									Value: "v1.4.1",
								},
								{
									Name:  "APPMESH_PLATFORM_K8S_VERSION",
									Value: "v1.20.1-eks-fdsedv",
								},
								{
									Name:  "APPMESH_DUALSTACK_ENDPOINT",
									Value: "0",
								},
								{
									Name:  "ENVOY_ADMIN_ACCESS_ENABLE_IPV6",
									Value: "false",
								},
								{
									Name:  "APPMESH_FIPS_ENDPOINT",
									Value: "0",
								},
								{
									Name:  "APPNET_AGENT_ADMIN_MODE",
									Value: "uds",
								},
								{
									Name:  "APPNET_AGENT_ADMIN_UDS_PATH",
									Value: "/tmp/agent.sock",
								},
								{
									Name:  "APPMESH_PLATFORM_K8S_POD_UID",
									Value: "",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.uid",
										},
									},
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
					adminAccessPort:            9901,
					preStopDelay:               "20",
					readinessProbeInitialDelay: 1,
					readinessProbePeriod:       10,
					sidecarImageRepository:     "envoy",
					sidecarImageTag:            "v2",
					sidecarCPURequests:         cpuRequests.String(),
					sidecarMemoryRequests:      memoryRequests.String(),
					enableDatadogTracing:       true,
					datadogTracerPort:          8126,
					datadogTracerAddress:       "127.0.0.1",
					controllerVersion:          "v1.4.1",
					k8sVersion:                 "v1.20.1-eks-fdsedv",
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
								PreStop: &corev1.LifecycleHandler{
									Exec: &corev1.ExecAction{Command: []string{
										"sh", "-c", "sleep 20",
									}},
								},
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
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
									Name:  "ENVOY_ADMIN_ACCESS_PORT",
									Value: "9901",
								},
								{
									Name:  "AWS_REGION",
									Value: "us-west-2",
								},
								{
									Name:  "ENABLE_ENVOY_DATADOG_TRACING",
									Value: "1",
								},
								{
									Name:  "DATADOG_TRACER_PORT",
									Value: "8126",
								},
								{
									Name:  "DATADOG_TRACER_ADDRESS",
									Value: "127.0.0.1",
								},
								{
									Name:  "APPMESH_PLATFORM_APP_MESH_CONTROLLER_VERSION",
									Value: "v1.4.1",
								},
								{
									Name:  "APPMESH_PLATFORM_K8S_VERSION",
									Value: "v1.20.1-eks-fdsedv",
								},
								{
									Name:  "APPMESH_DUALSTACK_ENDPOINT",
									Value: "0",
								},
								{
									Name:  "ENVOY_ADMIN_ACCESS_ENABLE_IPV6",
									Value: "false",
								},
								{
									Name:  "APPMESH_FIPS_ENDPOINT",
									Value: "0",
								},
								{
									Name:  "APPNET_AGENT_ADMIN_MODE",
									Value: "uds",
								},
								{
									Name:  "APPNET_AGENT_ADMIN_UDS_PATH",
									Value: "/tmp/agent.sock",
								},
								{
									Name:  "APPMESH_PLATFORM_K8S_POD_UID",
									Value: "",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.uid",
										},
									},
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
					adminAccessPort:            9901,
					preStopDelay:               "20",
					readinessProbeInitialDelay: 1,
					readinessProbePeriod:       10,
					sidecarImageRepository:     "envoy",
					sidecarImageTag:            "v2",
					sidecarCPURequests:         cpuRequests.String(),
					sidecarMemoryRequests:      memoryRequests.String(),
					enableStatsTags:            true,
					controllerVersion:          "v1.4.1",
					k8sVersion:                 "v1.20.1-eks-fdsedv",
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
								PreStop: &corev1.LifecycleHandler{
									Exec: &corev1.ExecAction{Command: []string{
										"sh", "-c", "sleep 20",
									}},
								},
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{

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
									Name:  "ENVOY_ADMIN_ACCESS_PORT",
									Value: "9901",
								},
								{
									Name:  "AWS_REGION",
									Value: "us-west-2",
								},
								{
									Name:  "ENABLE_ENVOY_STATS_TAGS",
									Value: "1",
								},
								{
									Name:  "APPMESH_PLATFORM_APP_MESH_CONTROLLER_VERSION",
									Value: "v1.4.1",
								},
								{
									Name:  "APPMESH_PLATFORM_K8S_VERSION",
									Value: "v1.20.1-eks-fdsedv",
								},
								{
									Name:  "APPMESH_DUALSTACK_ENDPOINT",
									Value: "0",
								},
								{
									Name:  "ENVOY_ADMIN_ACCESS_ENABLE_IPV6",
									Value: "false",
								},
								{
									Name:  "APPMESH_FIPS_ENDPOINT",
									Value: "0",
								},
								{
									Name:  "APPNET_AGENT_ADMIN_MODE",
									Value: "uds",
								},
								{
									Name:  "APPNET_AGENT_ADMIN_UDS_PATH",
									Value: "/tmp/agent.sock",
								},
								{
									Name:  "APPMESH_PLATFORM_K8S_POD_UID",
									Value: "",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.uid",
										},
									},
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
					adminAccessPort:            9901,
					preStopDelay:               "20",
					readinessProbeInitialDelay: 1,
					readinessProbePeriod:       10,
					sidecarImageRepository:     "envoy",
					sidecarImageTag:            "v2",
					sidecarCPURequests:         cpuRequests.String(),
					sidecarMemoryRequests:      memoryRequests.String(),
					enableStatsD:               true,
					statsDAddress:              "127.0.0.1",
					statsDPort:                 8125,
					statsDSocketPath:           "/var/run/datadog/dsd.socket",
					controllerVersion:          "v1.4.1",
					k8sVersion:                 "v1.20.1-eks-fdsedv",
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
								PreStop: &corev1.LifecycleHandler{
									Exec: &corev1.ExecAction{Command: []string{
										"sh", "-c", "sleep 20",
									}},
								},
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{

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
									Name:  "ENVOY_ADMIN_ACCESS_PORT",
									Value: "9901",
								},
								{
									Name:  "AWS_REGION",
									Value: "us-west-2",
								},
								{
									Name:  "ENABLE_ENVOY_DOG_STATSD",
									Value: "1",
								},
								{
									Name:  "STATSD_PORT",
									Value: "8125",
								},
								{
									Name:  "STATSD_ADDRESS",
									Value: "127.0.0.1",
								},
								{
									Name:  "STATSD_SOCKET_PATH",
									Value: "/var/run/datadog/dsd.socket",
								},
								{
									Name:  "APPMESH_PLATFORM_APP_MESH_CONTROLLER_VERSION",
									Value: "v1.4.1",
								},
								{
									Name:  "APPMESH_PLATFORM_K8S_VERSION",
									Value: "v1.20.1-eks-fdsedv",
								},
								{
									Name:  "APPMESH_DUALSTACK_ENDPOINT",
									Value: "0",
								},
								{
									Name:  "ENVOY_ADMIN_ACCESS_ENABLE_IPV6",
									Value: "false",
								},
								{
									Name:  "APPMESH_FIPS_ENDPOINT",
									Value: "0",
								},
								{
									Name:  "APPNET_AGENT_ADMIN_MODE",
									Value: "uds",
								},
								{
									Name:  "APPNET_AGENT_ADMIN_UDS_PATH",
									Value: "/tmp/agent.sock",
								},
								{
									Name:  "APPMESH_PLATFORM_K8S_POD_UID",
									Value: "",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.uid",
										},
									},
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
					adminAccessPort:            9901,
					preStopDelay:               "20",
					readinessProbeInitialDelay: 1,
					readinessProbePeriod:       10,
					sidecarImageRepository:     "envoy",
					sidecarImageTag:            "v2",
					sidecarCPURequests:         cpuRequests.String(),
					sidecarMemoryRequests:      memoryRequests.String(),
					controllerVersion:          "v1.4.1",
					k8sVersion:                 "v1.20.1-eks-fdsedv",
				},
			},
			args: args{
				pod: certPod,
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
								PreStop: &corev1.LifecycleHandler{
									Exec: &corev1.ExecAction{Command: []string{
										"sh", "-c", "sleep 20",
									}},
								},
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{

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
									Name:  "ENVOY_ADMIN_ACCESS_PORT",
									Value: "9901",
								},
								{
									Name:  "AWS_REGION",
									Value: "us-west-2",
								},
								{
									Name:  "APPMESH_PLATFORM_APP_MESH_CONTROLLER_VERSION",
									Value: "v1.4.1",
								},
								{
									Name:  "APPMESH_PLATFORM_K8S_VERSION",
									Value: "v1.20.1-eks-fdsedv",
								},
								{
									Name:  "APPMESH_DUALSTACK_ENDPOINT",
									Value: "0",
								},
								{
									Name:  "ENVOY_ADMIN_ACCESS_ENABLE_IPV6",
									Value: "false",
								},
								{
									Name:  "APPMESH_FIPS_ENDPOINT",
									Value: "0",
								},
								{
									Name:  "APPNET_AGENT_ADMIN_MODE",
									Value: "uds",
								},
								{
									Name:  "APPNET_AGENT_ADMIN_UDS_PATH",
									Value: "/tmp/agent.sock",
								},
								{
									Name:  "APPMESH_PLATFORM_K8S_POD_UID",
									Value: "",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.uid",
										},
									},
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
			name: "enable SDS controller flag + no pod annotation",
			fields: fields{
				vn: vn,
				ms: ms,
				mutatorConfig: envoyMutatorConfig{
					awsRegion:                  "us-west-2",
					preview:                    true,
					enableSDS:                  true,
					sdsUdsPath:                 "/run/spire/sockets/agent.sock",
					logLevel:                   "debug",
					adminAccessPort:            9901,
					preStopDelay:               "20",
					readinessProbeInitialDelay: 10,
					readinessProbePeriod:       2,
					sidecarImageRepository:     "envoy",
					sidecarImageTag:            "v2",
					sidecarCPURequests:         cpuRequests.String(),
					sidecarMemoryRequests:      memoryRequests.String(),
					controllerVersion:          "v1.4.1",
					k8sVersion:                 "v1.20.1-eks-fdsedv",
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
						},
					},
				}},
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
								PreStop: &corev1.LifecycleHandler{
									Exec: &corev1.ExecAction{Command: []string{
										"sh", "-c", "sleep 20",
									}},
								},
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{

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
									Name:  "APPMESH_SDS_SOCKET_PATH",
									Value: "/run/spire/sockets/agent.sock",
								},
								{
									Name:  "ENVOY_ADMIN_ACCESS_PORT",
									Value: "9901",
								},
								{
									Name:  "AWS_REGION",
									Value: "us-west-2",
								},
								{
									Name:  "APPMESH_PLATFORM_APP_MESH_CONTROLLER_VERSION",
									Value: "v1.4.1",
								},
								{
									Name:  "APPMESH_PLATFORM_K8S_VERSION",
									Value: "v1.20.1-eks-fdsedv",
								},
								{
									Name:  "APPMESH_DUALSTACK_ENDPOINT",
									Value: "0",
								},
								{
									Name:  "ENVOY_ADMIN_ACCESS_ENABLE_IPV6",
									Value: "false",
								},
								{
									Name:  "APPMESH_FIPS_ENDPOINT",
									Value: "0",
								},
								{
									Name:  "APPNET_AGENT_ADMIN_MODE",
									Value: "uds",
								},
								{
									Name:  "APPNET_AGENT_ADMIN_UDS_PATH",
									Value: "/tmp/agent.sock",
								},
								{
									Name:  "APPMESH_PLATFORM_K8S_POD_UID",
									Value: "",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.uid",
										},
									},
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
			name: "enable SDS controller flag + pod annotation to disable sds",
			fields: fields{
				vn: vn,
				ms: ms,
				mutatorConfig: envoyMutatorConfig{
					awsRegion:                  "us-west-2",
					preview:                    true,
					enableSDS:                  true,
					sdsUdsPath:                 "/run/spire/sockets/agent.sock",
					logLevel:                   "debug",
					adminAccessPort:            9901,
					preStopDelay:               "20",
					readinessProbeInitialDelay: 10,
					readinessProbePeriod:       2,
					sidecarImageRepository:     "envoy",
					sidecarImageTag:            "v2",
					sidecarCPURequests:         cpuRequests.String(),
					sidecarMemoryRequests:      memoryRequests.String(),
					controllerVersion:          "v1.4.1",
					k8sVersion:                 "v1.20.1-eks-fdsedv",
				},
			},
			args: args{
				pod: podDisableSds,
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
								PreStop: &corev1.LifecycleHandler{
									Exec: &corev1.ExecAction{Command: []string{
										"sh", "-c", "sleep 20",
									}},
								},
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{

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
									Name:  "ENVOY_ADMIN_ACCESS_PORT",
									Value: "9901",
								},
								{
									Name:  "AWS_REGION",
									Value: "us-west-2",
								},
								{
									Name:  "APPMESH_PLATFORM_APP_MESH_CONTROLLER_VERSION",
									Value: "v1.4.1",
								},
								{
									Name:  "APPMESH_PLATFORM_K8S_VERSION",
									Value: "v1.20.1-eks-fdsedv",
								},
								{
									Name:  "APPMESH_DUALSTACK_ENDPOINT",
									Value: "0",
								},
								{
									Name:  "ENVOY_ADMIN_ACCESS_ENABLE_IPV6",
									Value: "false",
								},
								{
									Name:  "APPMESH_FIPS_ENDPOINT",
									Value: "0",
								},
								{
									Name:  "APPNET_AGENT_ADMIN_MODE",
									Value: "uds",
								},
								{
									Name:  "APPNET_AGENT_ADMIN_UDS_PATH",
									Value: "/tmp/agent.sock",
								},
								{
									Name:  "APPMESH_PLATFORM_K8S_POD_UID",
									Value: "",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.uid",
										},
									},
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
				vn: vn,
				ms: ms,
				mutatorConfig: envoyMutatorConfig{
					awsRegion:                  "us-west-2",
					preview:                    false,
					logLevel:                   "debug",
					adminAccessPort:            9901,
					preStopDelay:               "20",
					readinessProbeInitialDelay: 1,
					readinessProbePeriod:       10,
					sidecarImageRepository:     "envoy",
					sidecarImageTag:            "v2",
					sidecarCPURequests:         cpuRequests.String(),
					sidecarMemoryRequests:      memoryRequests.String(),
					sidecarMemoryLimits:        memoryLimits.String(),
					enableStatsD:               true,
					statsDPort:                 8125,
					statsDAddress:              "127.0.0.1",
					controllerVersion:          "v1.4.1",
					k8sVersion:                 "v1.20.1-eks-fdsedv",
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
								PreStop: &corev1.LifecycleHandler{
									Exec: &corev1.ExecAction{Command: []string{
										"sh", "-c", "sleep 20",
									}},
								},
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{

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
									Name:  "ENVOY_ADMIN_ACCESS_PORT",
									Value: "9901",
								},
								{
									Name:  "AWS_REGION",
									Value: "us-west-2",
								},
								{
									Name:  "ENABLE_ENVOY_DOG_STATSD",
									Value: "1",
								},
								{
									Name:  "STATSD_PORT",
									Value: "8125",
								},
								{
									Name:  "STATSD_ADDRESS",
									Value: "127.0.0.1",
								},
								{
									Name:  "APPMESH_PLATFORM_APP_MESH_CONTROLLER_VERSION",
									Value: "v1.4.1",
								},
								{
									Name:  "APPMESH_PLATFORM_K8S_VERSION",
									Value: "v1.20.1-eks-fdsedv",
								},
								{
									Name:  "APPMESH_DUALSTACK_ENDPOINT",
									Value: "0",
								},
								{
									Name:  "ENVOY_ADMIN_ACCESS_ENABLE_IPV6",
									Value: "false",
								},
								{
									Name:  "APPMESH_FIPS_ENDPOINT",
									Value: "0",
								},
								{
									Name:  "APPNET_AGENT_ADMIN_MODE",
									Value: "uds",
								},
								{
									Name:  "APPNET_AGENT_ADMIN_UDS_PATH",
									Value: "/tmp/agent.sock",
								},
								{
									Name:  "APPMESH_PLATFORM_K8S_POD_UID",
									Value: "",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.uid",
										},
									},
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
					adminAccessPort:            9901,
					preStopDelay:               "20",
					readinessProbeInitialDelay: 1,
					readinessProbePeriod:       10,
					sidecarImageRepository:     "envoy",
					sidecarImageTag:            "v2",
					sidecarCPURequests:         cpuRequests.String(),
					sidecarMemoryRequests:      memoryRequests.String(),
					sidecarCPULimits:           cpuLimits.String(),
					enableStatsD:               true,
					statsDPort:                 8125,
					statsDAddress:              "127.0.0.1",
					controllerVersion:          "v1.4.1",
					k8sVersion:                 "v1.20.1-eks-fdsedv",
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
								PreStop: &corev1.LifecycleHandler{
									Exec: &corev1.ExecAction{Command: []string{
										"sh", "-c", "sleep 20",
									}},
								},
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{

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
									Name:  "ENVOY_ADMIN_ACCESS_PORT",
									Value: "9901",
								},
								{
									Name:  "AWS_REGION",
									Value: "us-west-2",
								},
								{
									Name:  "ENABLE_ENVOY_DOG_STATSD",
									Value: "1",
								},
								{
									Name:  "STATSD_PORT",
									Value: "8125",
								},
								{
									Name:  "STATSD_ADDRESS",
									Value: "127.0.0.1",
								},
								{
									Name:  "APPMESH_PLATFORM_APP_MESH_CONTROLLER_VERSION",
									Value: "v1.4.1",
								},
								{
									Name:  "APPMESH_PLATFORM_K8S_VERSION",
									Value: "v1.20.1-eks-fdsedv",
								},
								{
									Name:  "APPMESH_DUALSTACK_ENDPOINT",
									Value: "0",
								},
								{
									Name:  "ENVOY_ADMIN_ACCESS_ENABLE_IPV6",
									Value: "false",
								},
								{
									Name:  "APPMESH_FIPS_ENDPOINT",
									Value: "0",
								},
								{
									Name:  "APPNET_AGENT_ADMIN_MODE",
									Value: "uds",
								},
								{
									Name:  "APPNET_AGENT_ADMIN_UDS_PATH",
									Value: "/tmp/agent.sock",
								},
								{
									Name:  "APPMESH_PLATFORM_K8S_POD_UID",
									Value: "",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.uid",
										},
									},
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
					adminAccessPort:            9901,
					preStopDelay:               "20",
					readinessProbeInitialDelay: 1,
					readinessProbePeriod:       10,
					sidecarImageRepository:     "envoy",
					sidecarImageTag:            "v2",
					sidecarCPURequests:         cpuRequests.String(),
					sidecarMemoryRequests:      memoryRequests.String(),
					sidecarCPULimits:           cpuLimits.String(),
					sidecarMemoryLimits:        memoryLimits.String(),
					enableStatsD:               true,
					statsDAddress:              "127.0.0.1",
					statsDPort:                 8125,
					controllerVersion:          "v1.4.1",
					k8sVersion:                 "v1.20.1-eks-fdsedv",
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
								PreStop: &corev1.LifecycleHandler{
									Exec: &corev1.ExecAction{Command: []string{
										"sh", "-c", "sleep 20",
									}},
								},
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{

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
									Name:  "ENVOY_ADMIN_ACCESS_PORT",
									Value: "9901",
								},
								{
									Name:  "AWS_REGION",
									Value: "us-west-2",
								},
								{
									Name:  "ENABLE_ENVOY_DOG_STATSD",
									Value: "1",
								},
								{
									Name:  "STATSD_PORT",
									Value: "8125",
								},
								{
									Name:  "STATSD_ADDRESS",
									Value: "127.0.0.1",
								},
								{
									Name:  "APPMESH_PLATFORM_APP_MESH_CONTROLLER_VERSION",
									Value: "v1.4.1",
								},
								{
									Name:  "APPMESH_PLATFORM_K8S_VERSION",
									Value: "v1.20.1-eks-fdsedv",
								},
								{
									Name:  "APPMESH_DUALSTACK_ENDPOINT",
									Value: "0",
								},
								{
									Name:  "ENVOY_ADMIN_ACCESS_ENABLE_IPV6",
									Value: "false",
								},
								{
									Name:  "APPMESH_FIPS_ENDPOINT",
									Value: "0",
								},
								{
									Name:  "APPNET_AGENT_ADMIN_MODE",
									Value: "uds",
								},
								{
									Name:  "APPNET_AGENT_ADMIN_UDS_PATH",
									Value: "/tmp/agent.sock",
								},
								{
									Name:  "APPMESH_PLATFORM_K8S_POD_UID",
									Value: "",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.uid",
										},
									},
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
					adminAccessPort:            9901,
					preStopDelay:               "20",
					readinessProbeInitialDelay: 1,
					readinessProbePeriod:       10,
					sidecarImageRepository:     "envoy",
					sidecarImageTag:            "v2",
					sidecarCPURequests:         cpuRequests.String(),
					sidecarMemoryRequests:      memoryRequests.String(),
					controllerVersion:          "v1.4.1",
					k8sVersion:                 "v1.20.1-eks-fdsedv",
				},
			},
			args: args{
				pod: podMultipleContainer,
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
		{
			name: "base + enable Datadog tracing with hostIP ref",
			fields: fields{
				vn: vn,
				ms: ms,
				mutatorConfig: envoyMutatorConfig{
					awsRegion:                  "us-west-2",
					preview:                    false,
					logLevel:                   "debug",
					adminAccessPort:            9901,
					preStopDelay:               "20",
					readinessProbeInitialDelay: 1,
					readinessProbePeriod:       10,
					sidecarImageRepository:     "envoy",
					sidecarImageTag:            "v2",
					sidecarCPURequests:         cpuRequests.String(),
					sidecarMemoryRequests:      memoryRequests.String(),
					enableDatadogTracing:       true,
					datadogTracerPort:          8126,
					datadogTracerAddress:       "ref:status.hostIP",
					controllerVersion:          "v1.4.1",
					k8sVersion:                 "v1.20.1-eks-fdsedv",
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
								PreStop: &corev1.LifecycleHandler{
									Exec: &corev1.ExecAction{Command: []string{
										"sh", "-c", "sleep 20",
									}},
								},
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
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
									Name:  "ENVOY_ADMIN_ACCESS_PORT",
									Value: "9901",
								},
								{
									Name:  "AWS_REGION",
									Value: "us-west-2",
								},
								{
									Name:  "ENABLE_ENVOY_DATADOG_TRACING",
									Value: "1",
								},
								{
									Name:  "DATADOG_TRACER_PORT",
									Value: "8126",
								},
								{
									Name:  "DATADOG_TRACER_ADDRESS",
									Value: "",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.hostIP",
										},
									},
								},
								{
									Name:  "APPMESH_PLATFORM_APP_MESH_CONTROLLER_VERSION",
									Value: "v1.4.1",
								},
								{
									Name:  "APPMESH_PLATFORM_K8S_VERSION",
									Value: "v1.20.1-eks-fdsedv",
								},
								{
									Name:  "APPMESH_DUALSTACK_ENDPOINT",
									Value: "0",
								},
								{
									Name:  "ENVOY_ADMIN_ACCESS_ENABLE_IPV6",
									Value: "false",
								},
								{
									Name:  "APPMESH_FIPS_ENDPOINT",
									Value: "0",
								},
								{
									Name:  "APPNET_AGENT_ADMIN_MODE",
									Value: "uds",
								},
								{
									Name:  "APPNET_AGENT_ADMIN_UDS_PATH",
									Value: "/tmp/agent.sock",
								},
								{
									Name:  "APPMESH_PLATFORM_K8S_POD_UID",
									Value: "",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.uid",
										},
									},
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
			name: "base + enable Stats D hostIP",
			fields: fields{
				vn: vn,
				ms: ms,
				mutatorConfig: envoyMutatorConfig{
					awsRegion:                  "us-west-2",
					preview:                    false,
					logLevel:                   "debug",
					adminAccessPort:            9901,
					preStopDelay:               "20",
					readinessProbeInitialDelay: 1,
					readinessProbePeriod:       10,
					sidecarImageRepository:     "envoy",
					sidecarImageTag:            "v2",
					sidecarCPURequests:         cpuRequests.String(),
					sidecarMemoryRequests:      memoryRequests.String(),
					enableStatsD:               true,
					statsDAddress:              "ref:status.hostIP",
					statsDPort:                 8125,
					controllerVersion:          "v1.4.1",
					k8sVersion:                 "v1.20.1-eks-fdsedv",
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
								PreStop: &corev1.LifecycleHandler{
									Exec: &corev1.ExecAction{Command: []string{
										"sh", "-c", "sleep 20",
									}},
								},
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{

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
									Name:  "ENVOY_ADMIN_ACCESS_PORT",
									Value: "9901",
								},
								{
									Name:  "AWS_REGION",
									Value: "us-west-2",
								},
								{
									Name:  "ENABLE_ENVOY_DOG_STATSD",
									Value: "1",
								},
								{
									Name:  "STATSD_PORT",
									Value: "8125",
								},
								{
									Name:  "STATSD_ADDRESS",
									Value: "",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.hostIP",
										},
									},
								},
								{
									Name:  "APPMESH_PLATFORM_APP_MESH_CONTROLLER_VERSION",
									Value: "v1.4.1",
								},
								{
									Name:  "APPMESH_PLATFORM_K8S_VERSION",
									Value: "v1.20.1-eks-fdsedv",
								},
								{
									Name:  "APPMESH_DUALSTACK_ENDPOINT",
									Value: "0",
								},
								{
									Name:  "ENVOY_ADMIN_ACCESS_ENABLE_IPV6",
									Value: "false",
								},
								{
									Name:  "APPMESH_FIPS_ENDPOINT",
									Value: "0",
								},
								{
									Name:  "APPNET_AGENT_ADMIN_MODE",
									Value: "uds",
								},
								{
									Name:  "APPNET_AGENT_ADMIN_UDS_PATH",
									Value: "/tmp/agent.sock",
								},
								{
									Name:  "APPMESH_PLATFORM_K8S_POD_UID",
									Value: "",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.uid",
										},
									},
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
			name: "base + custom sidecar env variables",
			fields: fields{
				vn: vn,
				ms: ms,
				mutatorConfig: envoyMutatorConfig{
					awsRegion:                  "us-west-2",
					preview:                    false,
					logLevel:                   "debug",
					adminAccessPort:            9901,
					preStopDelay:               "20",
					readinessProbeInitialDelay: 1,
					readinessProbePeriod:       10,
					sidecarImageRepository:     "envoy",
					sidecarImageTag:            "v2",
					sidecarCPURequests:         cpuRequests.String(),
					sidecarMemoryRequests:      memoryRequests.String(),
					enableStatsD:               true,
					statsDAddress:              "ref:status.hostIP",
					statsDPort:                 8125,
					controllerVersion:          "v1.4.1",
					k8sVersion:                 "v1.20.1-eks-fdsedv",
				},
			},
			args: args{
				pod: envPod,
			},
			wantPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "my-ns",
					Name:      "my-pod",
					Annotations: map[string]string{
						"appmesh.k8s.aws/sidecarEnv": "DD_ENV=prod, TEST_ENV1=env_val1, TEST_ENV2=env_val2",
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
								PreStop: &corev1.LifecycleHandler{
									Exec: &corev1.ExecAction{Command: []string{
										"sh", "-c", "sleep 20",
									}},
								},
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{

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
									Name:  "ENVOY_ADMIN_ACCESS_PORT",
									Value: "9901",
								},
								{
									Name:  "AWS_REGION",
									Value: "us-west-2",
								},
								{
									Name:  "ENABLE_ENVOY_DOG_STATSD",
									Value: "1",
								},
								{
									Name:  "STATSD_PORT",
									Value: "8125",
								},
								{
									Name:  "DD_ENV",
									Value: "prod",
								},
								{
									Name:  "TEST_ENV1",
									Value: "env_val1",
								},
								{
									Name:  "TEST_ENV2",
									Value: "env_val2",
								},
								{
									Name:  "STATSD_ADDRESS",
									Value: "",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.hostIP",
										},
									},
								},
								{
									Name:  "APPMESH_PLATFORM_APP_MESH_CONTROLLER_VERSION",
									Value: "v1.4.1",
								},
								{
									Name:  "APPMESH_PLATFORM_K8S_VERSION",
									Value: "v1.20.1-eks-fdsedv",
								},
								{
									Name:  "APPMESH_DUALSTACK_ENDPOINT",
									Value: "0",
								},
								{
									Name:  "ENVOY_ADMIN_ACCESS_ENABLE_IPV6",
									Value: "false",
								},
								{
									Name:  "APPMESH_FIPS_ENDPOINT",
									Value: "0",
								},
								{
									Name:  "APPNET_AGENT_ADMIN_MODE",
									Value: "uds",
								},
								{
									Name:  "APPNET_AGENT_ADMIN_UDS_PATH",
									Value: "/tmp/agent.sock",
								},
								{
									Name:  "APPMESH_PLATFORM_K8S_POD_UID",
									Value: "",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.uid",
										},
									},
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
			name: "base + duplicate sidecar env ",
			fields: fields{
				vn: vn,
				ms: ms,
				mutatorConfig: envoyMutatorConfig{
					awsRegion:                  "us-west-2",
					preview:                    false,
					logLevel:                   "debug",
					adminAccessPort:            9901,
					preStopDelay:               "20",
					readinessProbeInitialDelay: 1,
					readinessProbePeriod:       10,
					sidecarImageRepository:     "envoy",
					sidecarImageTag:            "v2",
					sidecarCPURequests:         cpuRequests.String(),
					sidecarMemoryRequests:      memoryRequests.String(),
					sidecarCPULimits:           cpuLimits.String(),
					sidecarMemoryLimits:        memoryLimits.String(),
					controllerVersion:          "v1.4.1",
					k8sVersion:                 "v1.20.1-eks-fdsedv",
				},
			},
			args: args{
				pod: duplicateEnvPod,
			},
			wantPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "my-ns",
					Name:      "my-pod",
					Annotations: map[string]string{
						"appmesh.k8s.aws/sidecarEnv": "DD_ENV=prod, APPMESH_VIRTUAL_NODE_NAME=random_val",
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
								PreStop: &corev1.LifecycleHandler{
									Exec: &corev1.ExecAction{Command: []string{
										"sh", "-c", "sleep 20",
									}},
								},
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{

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
									Name:  "ENVOY_ADMIN_ACCESS_PORT",
									Value: "9901",
								},
								{
									Name:  "DD_ENV",
									Value: "prod",
								},
								{
									Name:  "AWS_REGION",
									Value: "us-west-2",
								},
								{
									Name:  "APPMESH_PLATFORM_APP_MESH_CONTROLLER_VERSION",
									Value: "v1.4.1",
								},
								{
									Name:  "APPMESH_PLATFORM_K8S_VERSION",
									Value: "v1.20.1-eks-fdsedv",
								},
								{
									Name:  "APPMESH_DUALSTACK_ENDPOINT",
									Value: "0",
								},
								{
									Name:  "ENVOY_ADMIN_ACCESS_ENABLE_IPV6",
									Value: "false",
								},
								{
									Name:  "APPMESH_FIPS_ENDPOINT",
									Value: "0",
								},
								{
									Name:  "APPMESH_PLATFORM_K8S_POD_UID",
									Value: "",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.uid",
										},
									},
								},
								{
									Name:  "APPNET_AGENT_ADMIN_MODE",
									Value: "uds",
								},
								{
									Name:  "APPNET_AGENT_ADMIN_UDS_PATH",
									Value: "/tmp/agent.sock",
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
			name: "enabled waitUntilProxyReady",
			fields: fields{
				vn: vn,
				ms: ms,
				mutatorConfig: envoyMutatorConfig{
					awsRegion:                  "us-west-2",
					preview:                    true,
					logLevel:                   "debug",
					adminAccessPort:            9901,
					preStopDelay:               "20",
					postStartInterval:          5,
					postStartTimeout:           60,
					readinessProbeInitialDelay: 10,
					readinessProbePeriod:       2,
					sidecarImageRepository:     "envoy",
					sidecarImageTag:            "v2",
					sidecarCPURequests:         cpuRequests.String(),
					sidecarMemoryRequests:      memoryRequests.String(),
					waitUntilProxyReady:        true,
					controllerVersion:          "v1.4.1",
					k8sVersion:                 "v1.20.1-eks-fdsedv",
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
								PostStart: &corev1.LifecycleHandler{
									Exec: &corev1.ExecAction{Command: []string{
										"sh", "-c", "if [[ $(/usr/bin/envoy --version) =~ ([0-9]+)\\.([0-9]+)\\.([0-9]+)-appmesh\\.([0-9]+) && " +
											"${BASH_REMATCH[1]} -ge 2 || (${BASH_REMATCH[1]} -ge 1 && ${BASH_REMATCH[2]} -gt 22) || (${BASH_REMATCH[1]} -ge 1 && " +
											"${BASH_REMATCH[2]} -ge 22 && ${BASH_REMATCH[3]} -gt 2) || (${BASH_REMATCH[1]} -ge 1 && ${BASH_REMATCH[2]} -ge 22 && " +
											"${BASH_REMATCH[3]} -ge 2 && ${BASH_REMATCH[4]} -gt 0) ]]; then APPNET_AGENT_POLL_ENVOY_READINESS_TIMEOUT_S=60 " +
											"APPNET_AGENT_POLL_ENVOY_READINESS_INTERVAL_S=5 /usr/bin/agent -envoyReadiness; else echo 'WaitUntilProxyReady " +
											"is not supported in Envoy version < 1.22.2.1'; fi",
									}},
								},
								PreStop: &corev1.LifecycleHandler{
									Exec: &corev1.ExecAction{Command: []string{
										"sh", "-c", "sleep 20",
									}},
								},
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{

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
									Name:  "ENVOY_ADMIN_ACCESS_PORT",
									Value: "9901",
								},
								{
									Name:  "AWS_REGION",
									Value: "us-west-2",
								},
								{
									Name:  "APPMESH_PLATFORM_APP_MESH_CONTROLLER_VERSION",
									Value: "v1.4.1",
								},
								{
									Name:  "APPMESH_PLATFORM_K8S_VERSION",
									Value: "v1.20.1-eks-fdsedv",
								},
								{
									Name:  "APPMESH_DUALSTACK_ENDPOINT",
									Value: "0",
								},
								{
									Name:  "ENVOY_ADMIN_ACCESS_ENABLE_IPV6",
									Value: "false",
								},
								{
									Name:  "APPMESH_FIPS_ENDPOINT",
									Value: "0",
								},
								{
									Name:  "APPNET_AGENT_ADMIN_MODE",
									Value: "uds",
								},
								{
									Name:  "APPNET_AGENT_ADMIN_UDS_PATH",
									Value: "/tmp/agent.sock",
								},
								{
									Name:  "APPMESH_PLATFORM_K8S_POD_UID",
									Value: "",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.uid",
										},
									},
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									"cpu":    cpuRequests,
									"memory": memoryRequests,
								},
							},
						},
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
					cmpopts.SortSlices(func(lhs corev1.EnvVar, rhs corev1.EnvVar) bool {
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

func Test_envoyMutator_getVolumeMounts(t *testing.T) {
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
			name: "pods with valid appmesh.k8s.aws/volumeMounts annotation",
			args: args{
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							"appmesh.k8s.aws/volumeMounts": "svc1-cert-chain-key:/certs/svc1, svc1-svc2-ca-bundle:/certs",
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
			name: "pods with no appmesh.k8s.aws/volumeMounts annotation",
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
			name: "pods with invalid appmesh.k8s.aws/volumeMounts annotation",
			args: args{
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							"appmesh.k8s.aws/volumeMounts": "svc1-cert-chain-ke",
						},
					},
				},
			},
			wantErr: errors.New("malformed annotation appmesh.k8s.aws/volumeMounts, expected format: volumeName:mountPath"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &envoyMutator{}
			got, err := m.getVolumeMounts(tt.args.pod)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, got, tt.want)
			}
		})
	}
}

func Test_envoyMutator_getCustomEnv(t *testing.T) {
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
			name: "pods with valid appmesh.k8s.aws/sidecarEnv annotation",
			args: args{
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							"appmesh.k8s.aws/sidecarEnv": "DD_ENV=prod, TEST_ENV=env_val",
						},
					},
				},
			},
			want: map[string]string{
				"DD_ENV":   "prod",
				"TEST_ENV": "env_val",
			},
			wantErr: nil,
		},
		{
			name: "pods with no appmesh.k8s.aws/sidecarEnv annotation",
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
			name: "pods with invalid appmesh.k8s.aws/sidecarEnv annotation",
			args: args{
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							"appmesh.k8s.aws/sidecarEnv": "DD_ENV=prod, TEST_ENV",
						},
					},
				},
			},
			wantErr: errors.New("malformed annotation appmesh.k8s.aws/sidecarEnv, expected format: EnvVariableKey=EnvVariableValue"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &envoyMutator{}
			got, err := m.getCustomEnv(tt.args.pod)
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

func Test_envoyMutator_getUseDualStackEndpoints(t *testing.T) {
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
			name: "enable using dualstack endpoint",
			fields: fields{
				ms: &appmesh.Mesh{
					Spec: appmesh.MeshSpec{
						AWSName: aws.String("my-mesh"),
					},
				},
				mutatorConfig: envoyMutatorConfig{
					accountID:            "000000000000",
					useDualStackEndpoint: false,
				},
			},
			want: "0",
		},
		{
			name: "disable using dualstack endpoint",
			fields: fields{
				ms: &appmesh.Mesh{
					Spec: appmesh.MeshSpec{
						AWSName:   aws.String("my-mesh"),
						MeshOwner: aws.String("000000000000"),
					},
				},
				mutatorConfig: envoyMutatorConfig{
					accountID:            "000000000000",
					useDualStackEndpoint: true,
				},
			},
			want: "1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &envoyMutator{
				ms:            tt.fields.ms,
				mutatorConfig: tt.fields.mutatorConfig,
			}
			got := m.getUseDualStackEndpoint(m.mutatorConfig.useDualStackEndpoint)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_envoyMutator_getUseFipsEndpoints(t *testing.T) {
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
			name: "disable using fips endpoint",
			fields: fields{
				ms: &appmesh.Mesh{
					Spec: appmesh.MeshSpec{
						AWSName: aws.String("my-mesh"),
					},
				},
				mutatorConfig: envoyMutatorConfig{
					accountID:       "000000000000",
					useFipsEndpoint: false,
				},
			},
			want: "0",
		},
		{
			name: "enable using fips endpoint",
			fields: fields{
				ms: &appmesh.Mesh{
					Spec: appmesh.MeshSpec{
						AWSName:   aws.String("my-mesh"),
						MeshOwner: aws.String("000000000000"),
					},
				},
				mutatorConfig: envoyMutatorConfig{
					accountID:       "000000000000",
					useFipsEndpoint: true,
				},
			},
			want: "1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &envoyMutator{
				ms:            tt.fields.ms,
				mutatorConfig: tt.fields.mutatorConfig,
			}
			got := m.getUseFipsEndpoint(m.mutatorConfig.useFipsEndpoint)
			assert.Equal(t, tt.want, got)
		})
	}
}
