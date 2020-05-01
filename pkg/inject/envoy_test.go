package inject

import (
	"fmt"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/k8s"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"testing"
)

func Test_Sidecar(t *testing.T) {
	sc := getConfig(nil)
	checkSidecars(t, sc)
}

func Test_Sidecar_WithXray(t *testing.T) {
	sc := getConfig(nil)
	sc.EnableXrayTracing = true
	checkSidecars(t, sc)
}

func Test_Sidecar_WithStatsTags(t *testing.T) {
	sc := getConfig(nil)
	sc.EnableStatsTags = true
	checkSidecars(t, sc)
}

func Test_Sidecar_WithStatsD(t *testing.T) {
	sc := getConfig(nil)
	sc.EnableStatsD = true
	checkSidecars(t, sc)
}

func Test_Sidecar_WithDatadog(t *testing.T) {
	sc := getConfig(nil)
	sc.EnableDatadogTracing = true
	sc.DatadogAddress = "datadog.appmesh-system"
	sc.DatadogPort = "8126"
	checkSidecars(t, sc)
}

func Test_Sidecar_WithJaeger(t *testing.T) {
	sc := getConfig(nil)
	sc.EnableJaegerTracing = true
	sc.JaegerAddress = "appmesh-jaeger.appmesh-system"
	sc.JaegerPort = "9411"
	checkSidecars(t, sc)
}

func checkSidecars(t *testing.T, cfg Config) {
	x := NewEnvoyMutator(&cfg, *getVn())
	pod := getPod(nil)
	assert.NoError(t, x.mutate(pod))
	var sidecar *corev1.Container
	for _, v := range pod.Spec.Containers {
		if v.Name == "envoy" {
			sidecar = &v
		}
	}
	assert.NotNil(t, sidecar)
	assert.Equal(t, "envoy", sidecar.Name, "Unexpected container found with name %s", sidecar.Name)
	assert.Equal(t, k8s.NamespacedName(&x.vn).String(), pod.Annotations[AppMeshVirtualNodeNameAnnotation])
	checkEnvoy(t, sidecar, x)
}

func checkEnvoy(t *testing.T, sidecar *corev1.Container, m *EnvoyMutator) {
	expectedEnvs := map[string]string{
		"APPMESH_VIRTUAL_NODE_NAME": fmt.Sprintf("mesh/%s/virtualNode/%s", m.vn.Spec.MeshRef.Name, k8s.NamespacedName(&m.vn)),
		"AWS_REGION":                m.config.Region,
		"ENVOY_LOG_LEVEL":           m.config.LogLevel,
		"APPMESH_PREVIEW":           "0",
	}

	if m.config.EnableJaegerTracing || m.config.EnableDatadogTracing {
		expectedEnvs["ENVOY_STATS_CONFIG_FILE"] = "/tmp/envoy/envoyconf.yaml"

		mounts := sidecar.VolumeMounts
		if len(mounts) < 1 {
			t.Errorf("no volume mounts found")
		}

		mount := mounts[0]
		mountName := mount.Name
		expectedMountName := "envoy-tracing-config"
		if mountName != expectedMountName {
			t.Errorf("volume mount name is set to %s instead of %s", mountName, expectedMountName)
		}

		mountPath := mount.MountPath
		expectedMountPath := "/tmp/envoy"
		if mountPath != expectedMountPath {
			t.Errorf("volume mount path is set to %s instead of %s", mountPath, expectedMountPath)
		}
	}

	if m.config.EnableXrayTracing {
		expectedEnvs["ENABLE_ENVOY_XRAY_TRACING"] = "1"
	}

	if m.config.EnableStatsTags {
		expectedEnvs["ENABLE_ENVOY_STATS_TAGS"] = "1"
	}

	if m.config.EnableStatsD {
		expectedEnvs["ENABLE_ENVOY_DOG_STATSD"] = "1"
	}

	if sidecar.Image != m.config.SidecarImage {
		t.Errorf("Envoy container image is not set to %s", m.config.SidecarImage)
	}
	assert.Equal(t, "10m", sidecar.Resources.Requests.Cpu().String(), "CPU request mismatch")
	assert.Equal(t, "32Mi", sidecar.Resources.Requests.Memory().String(), "Memory request mismatch")

	checkEnv(t, sidecar, expectedEnvs)
}

func checkEnv(t *testing.T, sidecar *corev1.Container, expectedEnvs map[string]string) {
	envs := sidecar.Env
	for _, u := range envs {
		name := u.Name
		if expected, ok := expectedEnvs[name]; ok {
			val := u.Value
			if val != expected {
				t.Errorf("%s env is set %s instead of %s", name, val, expected)
			} else {
				delete(expectedEnvs, name)
			}
		}
	}

	for k := range expectedEnvs {
		t.Errorf("%s env is not set", k)
	}
}
