package inject

import (
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"testing"
)

func Test_initProxyMutator_mutate(t *testing.T) {
	cpuRequests, _ := resource.ParseQuantity("32Mi")
	memoryRequests, _ := resource.ParseQuantity("10m")

	cpuLimits, _ := resource.ParseQuantity("64Mi")
	memoryLimits, _ := resource.ParseQuantity("30m")
	type fields struct {
		mutatorConfig initProxyMutatorConfig
		proxyConfig   proxyConfig
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
			name: "normal case",
			fields: fields{
				mutatorConfig: initProxyMutatorConfig{
					containerImage: "840364872350.dkr.ecr.us-west-2.amazonaws.com/aws-appmesh-proxy-route-manager:v4-prod",
					cpuRequests:    cpuRequests.String(),
					memoryRequests: memoryRequests.String(),
					cpuLimits:      cpuLimits.String(),
					memoryLimits:   memoryLimits.String(),
				},
				proxyConfig: proxyConfig{
					appPorts:           "80,443",
					egressIgnoredIPs:   "192.168.0.1",
					egressIgnoredPorts: "22",
					proxyEgressPort:    15001,
					proxyIngressPort:   15000,
					proxyUID:           1337,
				},
			},
			args: args{
				pod: &corev1.Pod{
					Spec: corev1.PodSpec{
						InitContainers: nil,
					},
				},
			},
			wantPod: &corev1.Pod{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{
							Name:  "proxyinit",
							Image: "840364872350.dkr.ecr.us-west-2.amazonaws.com/aws-appmesh-proxy-route-manager:v4-prod",
							SecurityContext: &corev1.SecurityContext{
								Capabilities: &corev1.Capabilities{
									Add: []corev1.Capability{
										"NET_ADMIN",
									},
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "APPMESH_START_ENABLED",
									Value: "1",
								},
								{
									Name:  "APPMESH_IGNORE_UID",
									Value: "1337",
								},
								{
									Name:  "APPMESH_ENVOY_INGRESS_PORT",
									Value: "15000",
								},
								{
									Name:  "APPMESH_ENVOY_EGRESS_PORT",
									Value: "15001",
								},
								{
									Name:  "APPMESH_APP_PORTS",
									Value: "80,443",
								},
								{
									Name:  "APPMESH_EGRESS_IGNORED_IP",
									Value: "192.168.0.1",
								},
								{
									Name:  "APPMESH_EGRESS_IGNORED_PORTS",
									Value: "22",
								},
								{
									Name:  "APPMESH_ENABLE_IPV6",
									Value: "1",
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
			name: "normal case without resource limits",
			fields: fields{
				mutatorConfig: initProxyMutatorConfig{
					containerImage: "840364872350.dkr.ecr.us-west-2.amazonaws.com/aws-appmesh-proxy-route-manager:v4-prod",
					cpuRequests:    cpuRequests.String(),
					memoryRequests: memoryRequests.String(),
				},
				proxyConfig: proxyConfig{
					appPorts:           "80,443",
					egressIgnoredIPs:   "192.168.0.1",
					egressIgnoredPorts: "22",
					proxyEgressPort:    15001,
					proxyIngressPort:   15000,
					proxyUID:           1337,
				},
			},
			args: args{
				pod: &corev1.Pod{
					Spec: corev1.PodSpec{
						InitContainers: nil,
					},
				},
			},
			wantPod: &corev1.Pod{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{
							Name:  "proxyinit",
							Image: "840364872350.dkr.ecr.us-west-2.amazonaws.com/aws-appmesh-proxy-route-manager:v4-prod",
							SecurityContext: &corev1.SecurityContext{
								Capabilities: &corev1.Capabilities{
									Add: []corev1.Capability{
										"NET_ADMIN",
									},
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "APPMESH_START_ENABLED",
									Value: "1",
								},
								{
									Name:  "APPMESH_IGNORE_UID",
									Value: "1337",
								},
								{
									Name:  "APPMESH_ENVOY_INGRESS_PORT",
									Value: "15000",
								},
								{
									Name:  "APPMESH_ENVOY_EGRESS_PORT",
									Value: "15001",
								},
								{
									Name:  "APPMESH_APP_PORTS",
									Value: "80,443",
								},
								{
									Name:  "APPMESH_EGRESS_IGNORED_IP",
									Value: "192.168.0.1",
								},
								{
									Name:  "APPMESH_EGRESS_IGNORED_PORTS",
									Value: "22",
								},
								{
									Name:  "APPMESH_ENABLE_IPV6",
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
			name: "normal case + exists other init container",
			fields: fields{
				mutatorConfig: initProxyMutatorConfig{
					containerImage: "840364872350.dkr.ecr.us-west-2.amazonaws.com/aws-appmesh-proxy-route-manager:v4-prod",
					cpuRequests:    cpuRequests.String(),
					memoryRequests: memoryRequests.String(),
				},
				proxyConfig: proxyConfig{
					appPorts:           "80,443",
					egressIgnoredIPs:   "192.168.0.1",
					egressIgnoredPorts: "22",
					proxyEgressPort:    15001,
					proxyIngressPort:   15000,
					proxyUID:           1337,
				},
			},
			args: args{
				pod: &corev1.Pod{
					Spec: corev1.PodSpec{
						InitContainers: []corev1.Container{
							{
								Name:  "custominit",
								Image: "custominit:v1",
							},
						},
					},
				},
			},
			wantPod: &corev1.Pod{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{
							Name:  "custominit",
							Image: "custominit:v1",
						},
						{
							Name:  "proxyinit",
							Image: "840364872350.dkr.ecr.us-west-2.amazonaws.com/aws-appmesh-proxy-route-manager:v4-prod",
							SecurityContext: &corev1.SecurityContext{
								Capabilities: &corev1.Capabilities{
									Add: []corev1.Capability{
										"NET_ADMIN",
									},
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "APPMESH_START_ENABLED",
									Value: "1",
								},
								{
									Name:  "APPMESH_IGNORE_UID",
									Value: "1337",
								},
								{
									Name:  "APPMESH_ENVOY_INGRESS_PORT",
									Value: "15000",
								},
								{
									Name:  "APPMESH_ENVOY_EGRESS_PORT",
									Value: "15001",
								},
								{
									Name:  "APPMESH_APP_PORTS",
									Value: "80,443",
								},
								{
									Name:  "APPMESH_EGRESS_IGNORED_IP",
									Value: "192.168.0.1",
								},
								{
									Name:  "APPMESH_EGRESS_IGNORED_PORTS",
									Value: "22",
								},
								{
									Name:  "APPMESH_ENABLE_IPV6",
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
			name: "no-op when already contains proxyInit container",
			fields: fields{
				mutatorConfig: initProxyMutatorConfig{
					containerImage: "840364872350.dkr.ecr.us-west-2.amazonaws.com/aws-appmesh-proxy-route-manager:v4-prod",
					cpuRequests:    cpuRequests.String(),
					memoryRequests: memoryRequests.String(),
				},
				proxyConfig: proxyConfig{
					appPorts:           "80,443",
					egressIgnoredIPs:   "192.168.0.1",
					egressIgnoredPorts: "22",
					proxyEgressPort:    15001,
					proxyIngressPort:   15000,
					proxyUID:           1337,
				},
			},
			args: args{
				pod: &corev1.Pod{
					Spec: corev1.PodSpec{
						InitContainers: []corev1.Container{
							{
								Name: "proxyinit",
							},
						},
					},
				},
			},
			wantPod: &corev1.Pod{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{
							Name: "proxyinit",
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &initProxyMutator{
				mutatorConfig: tt.fields.mutatorConfig,
				proxyConfig:   tt.fields.proxyConfig,
			}
			err := m.mutate(tt.args.pod)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.True(t, cmp.Equal(tt.wantPod, tt.args.pod), "diff", cmp.Diff(tt.wantPod, tt.args.pod))
			}
		})
	}
}

func Test_containsProxyInitContainer(t *testing.T) {
	type args struct {
		pod *corev1.Pod
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "contains proxy init container",
			args: args{
				pod: &corev1.Pod{
					Spec: corev1.PodSpec{
						InitContainers: []corev1.Container{
							{
								Name: "proxyinit",
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "doesn't contains proxy init container",
			args: args{
				pod: &corev1.Pod{
					Spec: corev1.PodSpec{
						InitContainers: []corev1.Container{
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
			got := containsProxyInitContainer(tt.args.pod)
			assert.Equal(t, tt.want, got)
		})
	}
}
