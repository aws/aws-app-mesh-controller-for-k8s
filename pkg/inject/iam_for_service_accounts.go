package inject

import corev1 "k8s.io/api/core/v1"

const (
	// We don't want to make this configurable since users shouldn't rely on this
	// feature to set a fsGroup for them. This feature is just to protect innocent
	// users that are not aware of the limitation of iam-for-service-accounts:
	// https://github.com/aws/amazon-eks-pod-identity-webhook/issues/8
	// Users should set fsGroup on the pod spec directly if a specific fsGroup is desired.
	defaultIAMForSAFSGroup int64 = 1337
)

func newIAMForServiceAccountsMutator(enabled bool) *iamForServiceAccountsMutator {
	return &iamForServiceAccountsMutator{
		enabled:   enabled,
		fsGroupID: defaultIAMForSAFSGroup,
	}
}

var _ PodMutator = &iamForServiceAccountsMutator{}

// If enabled, a fsGroup will be injected in the absence of it within pod securityContext
// see https://github.com/aws/amazon-eks-pod-identity-webhook/issues/8 for more details
type iamForServiceAccountsMutator struct {
	enabled   bool
	fsGroupID int64
}

func (m *iamForServiceAccountsMutator) mutate(pod *corev1.Pod) error {
	if !m.enabled {
		return nil
	}
	if pod.Spec.SecurityContext != nil && pod.Spec.SecurityContext.FSGroup != nil {
		return nil
	}
	if pod.Spec.SecurityContext == nil {
		pod.Spec.SecurityContext = &corev1.PodSecurityContext{}
	}
	pod.Spec.SecurityContext.FSGroup = &m.fsGroupID
	return nil
}
