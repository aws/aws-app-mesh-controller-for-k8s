package inject

import corev1 "k8s.io/api/core/v1"

func NewAppMeshCNIMutator(Config *Config) *AppMeshCNIMutator {
	return &AppMeshCNIMutator{config: Config}
}

type AppMeshCNIMutator struct {
	config *Config
}

func (c *AppMeshCNIMutator) mutate(pod *corev1.Pod) error {
	if !IsAppMeshCNIEnabled(pod) {
		return nil
	}
	annotations := pod.GetAnnotations()
	annotations[AppMeshEgressIgnoredIPsAnnotation] = c.config.IgnoredIPs
	annotations[AppMeshEgressIgnoredPortsAnnotation] = GetEgressIgnoredPorts(pod)
	annotations[AppMeshPortsAnnotation] = GetPortsFromContainers(pod.Spec.Containers)
	annotations[AppMeshSidecarInjectAnnotation] = "enabled"
	annotations[AppMeshIgnoredUIDAnnotation] = AppMeshProxyUID
	annotations[AppMeshProxyEgressPortAnnotation] = AppMeshProxyEgressPort
	annotations[AppMeshProxyIngressPortAnnotation] = AppMeshProxyIngressPort
	pod.SetAnnotations(annotations)
	return nil
}
