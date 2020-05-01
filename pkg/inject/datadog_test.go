package inject

import (
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"strings"
	"testing"
)

func Test_RenderDatadogInitContainer(t *testing.T) {
	tests := []struct {
		name string
		conf Config
		pod  *corev1.Pod
		want bool
	}{
		{
			name: "Enable Jaeger inject",
			conf: getConfig(func(cnf Config) Config {
				cnf.EnableDatadogTracing = true
				cnf.DatadogAddress = "datadog.appmesh-system"
				cnf.DatadogPort = "8126"
				return cnf
			}),
			pod:  getPod(nil),
			want: true,
		},
		{
			name: "No Datadog inject",
			conf: getConfig(nil),
			pod:  getPod(nil),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDatadogMutator(&tt.conf)
			before := len(tt.pod.Spec.InitContainers)
			err := d.mutate(tt.pod)
			var init *corev1.Container
			assert.NoError(t, err, "Unexpected error")
			found := false
			for _, v := range tt.pod.Spec.InitContainers {
				if v.Name == "inject-datadog-config" {
					found = true
					init = &v
				}
			}
			assert.True(t, found == tt.want, "Unexpected datadog container")
			if tt.want {
				assert.NotNil(t, init)
				assert.Equal(t, "busybox", init.Image)
				if len(init.Command) < 1 {
					t.Error("Datadog init container does not contain command")
				}
				allCommands := strings.Join(init.Command, " ")
				if !strings.Contains(allCommands, tt.conf.DatadogPort) {
					t.Errorf("Datadog port did not get configured correctly")
				}
				if !strings.Contains(allCommands, tt.conf.DatadogAddress) {
					t.Errorf("Datadog address did not get configured correctly")
				}
				assert.True(t, len(tt.pod.Spec.Volumes) > 0)
				assert.Greater(t, len(tt.pod.Spec.InitContainers), before)
			} else {
				assert.Nil(t, init)
				assert.Equal(t, before, len(tt.pod.Spec.InitContainers))
			}
		})
	}
}
