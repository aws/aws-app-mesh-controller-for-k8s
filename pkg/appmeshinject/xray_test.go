package appmeshinject

import (
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"testing"
)

func checkXrayDaemon(t *testing.T, sidecar *corev1.Container, meta XrayMutator) {
	if !config.InjectXraySidecar {
		t.Errorf("Xray daemon is added when InjectXraySidecar is false")
		return
	}

	if sidecar.Image != "amazon/aws-xray-daemon" {
		t.Errorf("Xray daemon container image is not set to amazon/aws-xray-daemon")
	}

	expectedEnvs := map[string]string{
		"AWS_REGION": config.Region,
	}
	assert.Equal(t, "10m", sidecar.Resources.Requests.Cpu().String(), "CPU request mismatch")
	assert.Equal(t, "32Mi", sidecar.Resources.Requests.Memory().String(), "Memory request mismatch")
	checkEnv(t, sidecar, expectedEnvs)
}

func Test_XrayInject(t *testing.T) {
	tests := []struct {
		name   string
		conf   Config
		expect bool
	}{
		{
			name: "Inject X-ray container",
			conf: getConfig(func(cnf Config) Config {
				cnf.InjectXraySidecar = true
				return cnf
			}),
			expect: true,
		},
		{
			name: "No X-ray inject configured",
			conf: getConfig(func(cnf Config) Config {
				cnf.InjectXraySidecar = false
				return cnf
			}),
			expect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			New(tt.conf)
			x := XrayMutator{}
			pod := getPod(nil)

			err := x.mutate(pod)
			assert.NoError(t, err, "Unexpected error")
			found := false
			for _, v := range pod.Spec.Containers {
				if v.Name == "xray-daemon" {
					checkXrayDaemon(t, &v, x)
					found = true
				}
			}
			assert.True(t, found == tt.expect, "Unexpected x-ray container")
		})
	}
}
