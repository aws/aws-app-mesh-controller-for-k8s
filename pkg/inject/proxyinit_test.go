package inject

import (
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"testing"
)

func Test_RenderInitContainer(t *testing.T) {
	tests := []struct {
		name   string
		conf   Config
		pod    *corev1.Pod
		expect bool
	}{
		{
			name:   "Inject ProxyInit",
			conf:   getConfig(nil),
			expect: true,
			pod:    getPod(nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewProxyInitMutator(&tt.conf)
			pod := tt.pod

			err := p.mutate(pod)
			assert.NoError(t, err, "Unable to render init container")
			if tt.expect {
				var init *corev1.Container
				for _, v := range pod.Spec.InitContainers {
					if v.Name == "proxyinit" {
						init = &v
					}
				}
				assert.NotNil(t, init)
				assert.Equal(t, init.Name, "proxyinit")
				assert.Equal(t, init.Image, tt.conf.InitImage)
				expected := map[string]string{
					"APPMESH_APP_PORTS":            "80,443",
					"APPMESH_EGRESS_IGNORED_PORTS": GetEgressIgnoredPorts(pod),
					"APPMESH_EGRESS_IGNORED_IP":    tt.conf.IgnoredIPs,
				}
				for _, v := range init.Env {
					if val, ok := expected[v.Name]; ok {
						assert.Equal(t, val, v.Value)
					}
				}
				assert.Equal(t, "10m", init.Resources.Requests.Cpu().String(), "CPU request mismatch")
				assert.Equal(t, "32Mi", init.Resources.Requests.Memory().String(), "Memory request mismatch")
			}
		})
	}
}
