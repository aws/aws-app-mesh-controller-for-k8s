package inject

import (
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func Test_cniProxyMutator_mutate(t *testing.T) {
	type fields struct {
		proxyConfig proxyConfig
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
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{},
					},
				},
			},
			wantPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
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
			name: "normal case + exists other annotation",
			fields: fields{
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
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							"k8s.io/application-name": "my-application",
						},
					},
				},
			},
			wantPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"appmesh.k8s.aws/ports":                  "80,443",
						"appmesh.k8s.aws/egressIgnoredIPs":       "192.168.0.1",
						"appmesh.k8s.aws/egressIgnoredPorts":     "22",
						"appmesh.k8s.aws/proxyEgressPort":        "15001",
						"appmesh.k8s.aws/proxyIngressPort":       "15000",
						"appmesh.k8s.aws/ignoredUID":             "1337",
						"appmesh.k8s.aws/sidecarInjectorWebhook": "enabled",
						"k8s.io/application-name":                "my-application",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &cniProxyMutator{
				proxyConfig: tt.fields.proxyConfig,
			}
			err := m.mutate(tt.args.pod)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantPod, tt.args.pod)
		})
	}
}
