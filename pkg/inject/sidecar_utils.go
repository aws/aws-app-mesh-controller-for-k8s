package inject

import (
	"bufio"
	"bytes"
	"encoding/json"
	corev1 "k8s.io/api/core/v1"
	"strings"
	"text/template"
)

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

func shouldInject(defaultShouldInject bool, pod *corev1.Pod) bool {
	if v, ok := pod.ObjectMeta.Annotations[AppMeshSidecarInjectAnnotation]; ok {
		return strings.ToLower(v) == "enabled"
	}
	return defaultShouldInject
}

func getSidecarCPURequest(defaultCPURequest string, pod *corev1.Pod) string {
	if v, ok := pod.ObjectMeta.Annotations[AppMeshCPURequestAnnotation]; ok {
		return v
	}
	return defaultCPURequest
}

func getSidecarMemoryRequest(defaultMemoryRequest string, pod *corev1.Pod) string {
	if v, ok := pod.ObjectMeta.Annotations[AppMeshMemoryRequestAnnotation]; ok {
		return v
	}
	return defaultMemoryRequest
}
