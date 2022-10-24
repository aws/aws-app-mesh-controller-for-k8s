package inject

import (
	"testing"

	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_proxyMutator_mutate(t *testing.T) {
	cpuRequests, _ := resource.ParseQuantity("32Mi")
	memoryRequests, _ := resource.ParseQuantity("10m")
	vn := &appmesh.VirtualNode{
		Spec: appmesh.VirtualNodeSpec{
			Listeners: []appmesh.Listener{
				{
					PortMapping: appmesh.PortMapping{
						Port:     80,
						Protocol: "http",
					},
				},
				{
					PortMapping: appmesh.PortMapping{
						Port:     443,
						Protocol: "http",
					},
				},
			},
		},
	}
	vnWithoutListener := &appmesh.VirtualNode{
		Spec: appmesh.VirtualNodeSpec{},
	}
	mutatorConfig := proxyMutatorConfig{
		initProxyMutatorConfig: initProxyMutatorConfig{
			containerImage: "840364872350.dkr.ecr.us-west-2.amazonaws.com/aws-appmesh-proxy-route-manager:v6-prod",
			cpuRequests:    cpuRequests.String(),
			memoryRequests: memoryRequests.String(),
		},
		egressIgnoredIPs: "192.168.0.1",
	}
	type fields struct {
		mutatorConfig proxyMutatorConfig
		vn            *appmesh.VirtualNode
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
			name: "mutate using init container",
			fields: fields{
				mutatorConfig: mutatorConfig,
				vn:            vn,
			},
			args: args{
				pod: &corev1.Pod{},
			},
			wantPod: &corev1.Pod{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{
							Name:  "proxyinit",
							Image: "840364872350.dkr.ecr.us-west-2.amazonaws.com/aws-appmesh-proxy-route-manager:v6-prod",
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
			name: "mutate using init container with no listener",
			fields: fields{
				mutatorConfig: mutatorConfig,
				vn:            vnWithoutListener,
			},
			args: args{
				pod: &corev1.Pod{},
			},
			wantPod: &corev1.Pod{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{
							Name:  "proxyinit",
							Image: "840364872350.dkr.ecr.us-west-2.amazonaws.com/aws-appmesh-proxy-route-manager:v6-prod",
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
									Value: "",
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
			name: "mutate using init container with ipv6 disabled annotation",
			fields: fields{
				mutatorConfig: mutatorConfig,
				vn:            vnWithoutListener,
			},
			args: args{
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							"appmesh.k8s.aws/ipv6": "disabled",
						},
					},
				},
			},
			wantPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"appmesh.k8s.aws/ipv6": "disabled",
					},
				},
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{
							Name:  "proxyinit",
							Image: "840364872350.dkr.ecr.us-west-2.amazonaws.com/aws-appmesh-proxy-route-manager:v6-prod",
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
									Value: "",
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
									Value: "0",
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
			name: "mutate using appMesh CNI",
			fields: fields{
				mutatorConfig: mutatorConfig,
				vn:            vn,
			},
			args: args{
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							"appmesh.k8s.aws/appmeshCNI": "enabled",
						},
					},
				},
			},
			wantPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"appmesh.k8s.aws/appmeshCNI":             "enabled",
						"appmesh.k8s.aws/ports":                  "80,443",
						"appmesh.k8s.aws/egressIgnoredIPs":       "192.168.0.1",
						"appmesh.k8s.aws/egressIgnoredPorts":     "22",
						"appmesh.k8s.aws/proxyEgressPort":        "15001",
						"appmesh.k8s.aws/proxyIngressPort":       "15000",
						"appmesh.k8s.aws/ignoredUID":             "1337",
						"appmesh.k8s.aws/sidecarInjectorWebhook": "enabled",
					},
				},
			},
		},
		{
			name: "mutate using appMesh CNI with no listeners",
			fields: fields{
				mutatorConfig: mutatorConfig,
				vn:            vnWithoutListener,
			},
			args: args{
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							"appmesh.k8s.aws/appmeshCNI": "enabled",
						},
					},
				},
			},
			wantPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"appmesh.k8s.aws/appmeshCNI":             "enabled",
						"appmesh.k8s.aws/ports":                  "",
						"appmesh.k8s.aws/egressIgnoredIPs":       "192.168.0.1",
						"appmesh.k8s.aws/egressIgnoredPorts":     "22",
						"appmesh.k8s.aws/proxyEgressPort":        "15001",
						"appmesh.k8s.aws/proxyIngressPort":       "15000",
						"appmesh.k8s.aws/ignoredUID":             "1337",
						"appmesh.k8s.aws/sidecarInjectorWebhook": "enabled",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &proxyMutator{
				mutatorConfig: tt.fields.mutatorConfig,
				vn:            tt.fields.vn,
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

func Test_proxyMutator_getAppPorts(t *testing.T) {
	type fields struct {
		vn *appmesh.VirtualNode
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
			name: "get AppPorts from annotation",
			fields: fields{
				vn: &appmesh.VirtualNode{
					Spec: appmesh.VirtualNodeSpec{
						Listeners: []appmesh.Listener{
							{
								PortMapping: appmesh.PortMapping{
									Port:     80,
									Protocol: "http",
								},
							},
							{
								PortMapping: appmesh.PortMapping{
									Port:     443,
									Protocol: "http",
								},
							},
						},
					},
				},
			},
			args: args{
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							"appmesh.k8s.aws/ports": "8080",
						},
					},
				},
			},
			want: "8080",
		},
		{
			name: "get AppPorts from VirtualNode with multiple listener",
			fields: fields{
				vn: &appmesh.VirtualNode{
					Spec: appmesh.VirtualNodeSpec{
						Listeners: []appmesh.Listener{
							{
								PortMapping: appmesh.PortMapping{
									Port:     80,
									Protocol: "http",
								},
							},
							{
								PortMapping: appmesh.PortMapping{
									Port:     443,
									Protocol: "http",
								},
							},
						},
					},
				},
			},
			args: args{
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{},
					},
				},
			},
			want: "80,443",
		},
		{
			name: "get AppPorts from VirtualNode with no listener",
			fields: fields{
				vn: &appmesh.VirtualNode{
					Spec: appmesh.VirtualNodeSpec{
						Listeners: nil,
					},
				},
			},
			args: args{
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{},
					},
				},
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &proxyMutator{
				vn: tt.fields.vn,
			}
			got := m.getAppPorts(tt.args.pod)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_proxyMutator_getEgressIgnoredPorts(t *testing.T) {
	type args struct {
		pod *corev1.Pod
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "get EgressIgnoredPorts from annotation",
			args: args{
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							"appmesh.k8s.aws/egressIgnoredPorts": "8443,9090",
						},
					},
				},
			},
			want: "8443,9090",
		},
		{
			name: "get EgressIgnoredPorts by default",
			args: args{
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{},
					},
				},
			},
			want: "22",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &proxyMutator{}
			got := m.getEgressIgnoredPorts(tt.args.pod)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_proxyMutator_isAppMeshCNIEnabled(t *testing.T) {
	type args struct {
		pod *corev1.Pod
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "cni is enabled when annotation appmesh.k8s.aws/appmeshCNI: enabled presents",
			args: args{
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							"appmesh.k8s.aws/appmeshCNI": "enabled",
						},
					},
				},
			},
			want: true,
		},
		{
			name: "cni is enabled when fargate label presents",
			args: args{
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"eks.amazonaws.com/fargate-profile": "my-fp",
						},
					},
				},
			},
			want: true,
		},
		{
			name: "cni is not enabled",
			args: args{
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &proxyMutator{}
			got := m.isAppMeshCNIEnabled(tt.args.pod)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_proxyMutator_isIPV6Enabled(t *testing.T) {
	type args struct {
		pod *corev1.Pod
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "IPV6 disabled from annotation",
			args: args{
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							"appmesh.k8s.aws/ipv6": "disabled",
						},
					},
				},
			},
			want: false,
		},
		{
			name: "no ipv6 annotation",
			args: args{
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{},
					},
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &proxyMutator{}
			got := m.isIPV6Enabled(tt.args.pod)
			assert.Equal(t, tt.want, got)
		})
	}
}
