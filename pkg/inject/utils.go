package inject

import (
	"bufio"
	"bytes"
	"encoding/json"
	corev1 "k8s.io/api/core/v1"
	"text/template"
)

const (
	AppMeshSDSSocketVolume = "appmesh-sds-socket-volume"
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

func getSidecarCPULimit(defaultCPULimit string, pod *corev1.Pod) string {
	if v, ok := pod.ObjectMeta.Annotations[AppMeshCPULimitAnnotation]; ok {
		return v
	}
	return defaultCPULimit
}

func getSidecarMemoryLimit(defaultMemoryLimit string, pod *corev1.Pod) string {
	if v, ok := pod.ObjectMeta.Annotations[AppMeshMemoryLimitAnnotation]; ok {
		return v
	}
	return defaultMemoryLimit
}

// containsEnvoyContainer checks whether pod already contains "envoy" container and return the slice index
func containsEnvoyContainer(pod *corev1.Pod) (bool, int) {
	for idx, container := range pod.Spec.Containers {
		if container.Name == envoyContainerName {
			return true, idx
		}
	}
	return false, -1
}

func isSDSDisabled(pod *corev1.Pod) bool {
	if v, ok := pod.ObjectMeta.Annotations[AppMeshSDSAnnotation]; ok {
		if v == "disabled" {
			return true
		}
	}
	return false
}

func isSDSVolumePresent(pod *corev1.Pod, SdsUdsPath string) bool {
	for _, volume := range pod.Spec.Volumes {
		if volume.HostPath != nil && volume.HostPath.Path == SdsUdsPath {
			return true
		}
	}
	return false
}

func mutateSDSMounts(pod *corev1.Pod, envoyContainer *corev1.Container, SdsUdsPath string) {
	SDSVolumeType := corev1.HostPathSocket
	if isSDSVolumePresent(pod, SdsUdsPath) {
		return
	}
	volume := corev1.Volume{
		Name: AppMeshSDSSocketVolume,
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: SdsUdsPath,
				Type: &SDSVolumeType,
			},
		},
	}

	volumeMount := corev1.VolumeMount{
		Name:      AppMeshSDSSocketVolume,
		MountPath: SdsUdsPath,
	}

	envoyContainer.VolumeMounts = append(envoyContainer.VolumeMounts, volumeMount)
	pod.Spec.Volumes = append(pod.Spec.Volumes, volume)
}
