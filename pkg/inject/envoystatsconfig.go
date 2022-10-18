package inject

import (
	corev1 "k8s.io/api/core/v1"
)

func newEnvoyStatsConfigMutator() *envoyStatsConfigMutator {
	return &envoyStatsConfigMutator{}
}

var _ PodMutator = &envoyStatsConfigMutator{}

type envoyStatsConfigMutator struct {
}

func (m envoyStatsConfigMutator) mutate(pod *corev1.Pod) error {
	pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{
		Name: "envoy-stats-config",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "", // TODO: how to determine the name
				},
			},
		},
	})
	return nil
}
