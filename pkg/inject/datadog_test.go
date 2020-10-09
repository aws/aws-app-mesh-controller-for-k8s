package inject

import (
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"testing"
)

func Test_datadogMutator_mutate(t *testing.T) {
	cpuLimits, _ := resource.ParseQuantity("100m")
	cpuRequests, _ := resource.ParseQuantity("10m")
	memoryLimits, _ := resource.ParseQuantity("64Mi")
	memoryRequests, _ := resource.ParseQuantity("32Mi")
	type fields struct {
		mutatorConfig datadogMutatorConfig
		enabled       bool
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
				mutatorConfig: datadogMutatorConfig{
					datadogAddress: "127.0.0.1",
					datadogPort:    "8080",
				},
				enabled: false,
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
			name: "no-op when already contain envoy tracing config volume",
			fields: fields{
				mutatorConfig: datadogMutatorConfig{
					datadogAddress: "127.0.0.1",
					datadogPort:    "8080",
				},
				enabled: true,
			},
			args: args{
				pod: &corev1.Pod{
					Spec: corev1.PodSpec{
						Volumes: []corev1.Volume{
							{
								Name: envoyTracingConfigVolumeName,
								VolumeSource: corev1.VolumeSource{
									EmptyDir: &corev1.EmptyDirVolumeSource{},
								},
							},
						},
					},
				},
			},
			wantPod: &corev1.Pod{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: envoyTracingConfigVolumeName,
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
		},
		{
			name: "inject sidecar and volume",
			fields: fields{
				mutatorConfig: datadogMutatorConfig{
					datadogAddress: "127.0.0.1",
					datadogPort:    "8080",
				},
				enabled: true,
			},
			args: args{
				pod: &corev1.Pod{
					Spec: corev1.PodSpec{},
				},
			},
			wantPod: &corev1.Pod{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{
							Name:            "inject-datadog-config",
							Image:           "busybox",
							ImagePullPolicy: "IfNotPresent",
							Command: []string{
								"sh",
								"-c",
								`cat <<EOF >> /tmp/envoy/envoyconf.yaml
tracing:
  http:
    name: envoy.tracers.datadog
    config:
      collector_cluster: datadog_agent
      service_name: envoy
static_resources:
  clusters:
  - name: datadog_agent
    connect_timeout: 1s
    type: STRICT_DNS
    lb_policy: ROUND_ROBIN
    load_assignment:
      cluster_name: datadog_agent
      endpoints:
      - lb_endpoints:
        - endpoint:
           address:
            socket_address:
             address: 127.0.0.1
             port_value: 8080
EOF

cat /tmp/envoy/envoyconf.yaml
`,
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      envoyTracingConfigVolumeName,
									MountPath: "/tmp/envoy",
								},
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									"cpu":    cpuLimits,
									"memory": memoryLimits,
								},
								Requests: corev1.ResourceList{
									"cpu":    cpuRequests,
									"memory": memoryRequests,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: envoyTracingConfigVolumeName,
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &datadogMutator{
				mutatorConfig: tt.fields.mutatorConfig,
				enabled:       tt.fields.enabled,
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
