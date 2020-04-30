package appmeshinject

import (
	"bufio"
	"bytes"
	"encoding/json"
	corev1 "k8s.io/api/core/v1"
	"strconv"
	"strings"
	"text/template"
)

const tracingConfigVolumeName = "envoy-tracing-config"

func renderTemplate(name string, t string, meta interface{}) (string, error) {
	tmpl, err := template.New(name).Parse(t)
	if err != nil {
		return "", err
	}
	var data bytes.Buffer
	b := bufio.NewWriter(&data)

	if err := tmpl.Execute(b, meta); err != nil {
		return "", err
	}
	err = b.Flush()
	if err != nil {
		return "", err
	}
	return data.String(), nil
}

// encodes the Envoy config so it can used
// in the init container command
func escapeYaml(yaml string) (string, error) {
	i, err := json.Marshal(yaml)
	if err != nil {
		return "", err
	}
	s := string(i)
	if len(s) > 0 && s[0] == '"' {
		s = s[1:]
	}
	if len(s) > 0 && s[len(s)-1] == '"' {
		s = s[:len(s)-1]
	}
	return s, nil
}

// renderInitContainer helper creates inject-datadog-config or inject-jaeger-config
// container that writes the Envoy config in an empty dir volume
// the same volume is mounted in the Envoy container at /tmp/envoy/
// when Envoy starts it will load the tracing config
func renderInitContainer(name string, confTmpl string, injectTmpl string, meta interface{}) (*corev1.Container, error) {
	config, err := renderTemplate(name, confTmpl, meta)
	if err != nil {
		return nil, err
	}
	config, err = escapeYaml(config)
	if err != nil {
		return nil, err
	}
	initModel := struct {
		Config string
	}{
		config,
	}
	initContainer, err := renderTemplate("initConfig", injectTmpl, initModel)
	if err != nil {
		return nil, err
	}
	var container corev1.Container
	err = json.Unmarshal([]byte(initContainer), &container)
	if err != nil {
		return nil, err
	}
	return &container, nil

}

func renderConfigVolume() (*corev1.Volume, error) {
	vol := corev1.Volume{Name: tracingConfigVolumeName}
	return &vol, nil
}

// get all the ports from containers
func getPortsFromContainers(containers []corev1.Container) string {
	parts := make([]string, 0)
	for _, container := range containers {
		parts = append(parts, getPortsForContainer(container)...)
	}
	return strings.Join(parts, ",")
}

// get all the ports for that container
func getPortsForContainer(container corev1.Container) []string {
	parts := make([]string, 0)
	for _, p := range container.Ports {
		parts = append(parts, strconv.Itoa(int(p.ContainerPort)))
	}
	return parts
}

func shouldInject(pod *corev1.Pod) bool {
	if v, ok := pod.ObjectMeta.Annotations[AppMeshSidecarInjectAnnotation]; ok {
		return strings.ToLower(v) == "enabled"
	}
	return config.InjectDefault
}

func isAppMeshCNIEnabled(pod *corev1.Pod) bool {
	annotations := pod.GetAnnotations()
	if v, ok := annotations[AppMeshCNIAnnotation]; ok {
		return v == "enabled"
	}
	//Fargate platform has appmesh-cni enabled by default
	if v, ok := pod.GetLabels()[FargateProfileLabel]; ok {
		return len(v) > 0
	}
	return false
}

func updateConfigFromPodAnnotations(config Config, pod *corev1.Pod) Config {
	egressIgnoredPorts := "22"
	if v, ok := pod.ObjectMeta.Annotations[AppMeshEgressIgnoredPortsAnnotation]; ok {
		egressIgnoredPorts = v
	}
	config.EgressIgnoredPorts = egressIgnoredPorts
	if v, ok := pod.ObjectMeta.Annotations[AppMeshCpuRequestAnnotation]; ok {
		config.SidecarCpu = v
	}
	if v, ok := pod.ObjectMeta.Annotations[AppMeshMemoryRequestAnnotation]; ok {
		config.SidecarMemory = v
	}
	return config
}
