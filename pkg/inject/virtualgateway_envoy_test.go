package inject

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func Test_virtualGatewayEnvoyMutator_mutate(t *testing.T) {
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
					awsRegion:    "us-west-2",
					preview:      false,
					logLevel:     "debug",
					sidecarImage: "envoy:v2",
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
					awsRegion:    "us-west-2",
					preview:      false,
					logLevel:     "debug",
					sidecarImage: "envoy:v2",
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
					awsRegion:    "us-west-2",
					preview:      false,
					logLevel:     "debug",
					sidecarImage: "envoy:v2",
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
					awsRegion:    "us-west-2",
					preview:      false,
					logLevel:     "debug",
					sidecarImage: "envoy:v3",
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
									Name:  "AWS_REGION",
									Value: "us-west-2",
								},
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
					awsRegion:    "us-west-2",
					preview:      false,
					logLevel:     "debug",
					sidecarImage: "envoy:v3",
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
									Name:  "AWS_REGION",
									Value: "us-west-2",
								},
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
					awsRegion:    "us-west-2",
					preview:      false,
					logLevel:     "debug",
					sidecarImage: "envoy:v2",
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
									Name:  "AWS_REGION",
									Value: "us-west-2",
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
