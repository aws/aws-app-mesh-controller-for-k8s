package inject

import corev1 "k8s.io/api/core/v1"

const (
	defaultECRSecretName = "appmesh-ecr-secret"
)

func newECRSecretMutator(enabled bool) *ecrSecretMutator {
	return &ecrSecretMutator{
		enabled:    enabled,
		secretName: defaultECRSecretName,
	}
}

var _ PodMutator = &ecrSecretMutator{}

// If enabled, additional image pull secret will be injected.
type ecrSecretMutator struct {
	enabled    bool
	secretName string
}

func (m *ecrSecretMutator) mutate(pod *corev1.Pod) error {
	if !m.enabled {
		return nil
	}
	for _, imagePullSecret := range pod.Spec.ImagePullSecrets {
		if imagePullSecret.Name == m.secretName {
			return nil
		}
	}
	secretRef := corev1.LocalObjectReference{Name: m.secretName}
	pod.Spec.ImagePullSecrets = append(pod.Spec.ImagePullSecrets, secretRef)
	return nil
}
