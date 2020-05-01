package inject

import (
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"testing"
)

func Test_CNIMutator(t *testing.T) {
	tests := []struct {
		name   string
		conf   Config
		pod    *corev1.Pod
		expect bool
	}{
		{
			name:   "CNI Mutator enabled",
			conf:   getConfig(nil),
			expect: true,
			pod: getPod(map[string]string{
				AppMeshCNIAnnotation: "enabled",
			}),
		},
		{
			name:   "CNI Mutator disabled via pod annotation",
			conf:   getConfig(nil),
			expect: false,
			pod: getPod(map[string]string{
				AppMeshCNIAnnotation: "Disabled",
			}),
		},
		{
			name:   "CNI Mutator disabled no pod annotation",
			conf:   getConfig(nil),
			expect: false,
			pod:    getPod(nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewAppMeshCNIMutator(&tt.conf)
			pod := tt.pod
			before := len(pod.Spec.InitContainers)
			err := p.mutate(pod)
			assert.NoError(t, err, "CNI MUtator failed to to mutate pod")
			if tt.expect {
				assert.Equal(t, len(pod.Spec.InitContainers), before)
				annotations := pod.GetAnnotations()
				assert.Equal(t, "enabled", annotations[AppMeshSidecarInjectAnnotation])
				assert.Equal(t, annotations[AppMeshEgressIgnoredIPsAnnotation], tt.conf.IgnoredIPs)
				assert.Equal(t, annotations[AppMeshEgressIgnoredPortsAnnotation], GetEgressIgnoredPorts(pod))
				assert.Equal(t, annotations[AppMeshPortsAnnotation], GetPortsFromContainers(pod.Spec.Containers))
				assert.Equal(t, annotations[AppMeshIgnoredUIDAnnotation], AppMeshProxyUID)
				assert.Equal(t, annotations[AppMeshProxyEgressPortAnnotation], AppMeshProxyEgressPort)
				assert.Equal(t, annotations[AppMeshProxyIngressPortAnnotation], AppMeshProxyIngressPort)
			} else {
				annotations := pod.GetAnnotations()
				if _, ok := annotations[AppMeshSidecarInjectAnnotation]; ok {
					t.Errorf("Unexpected pod annotation")
				}
				assert.Equal(t, len(pod.Spec.InitContainers), before)
			}
		})
	}
}
