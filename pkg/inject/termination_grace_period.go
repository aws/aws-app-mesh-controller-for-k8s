package inject

import (
	corev1 "k8s.io/api/core/v1"
	"strconv"
)

const defaultTerminationGracePeriod int64 = 30

// newTerminationGracePeriod constructs new terminationGracePeriod
func newTerminationGracePeriodMutator(preStopDelay string) *terminationGracePeriodMutator {
	preStopDelayDuration, _ := strconv.ParseInt(preStopDelay, 10, 64)
	gracePeriod := defaultTerminationGracePeriod
	needsToBeAdjusted := false
	if preStopDelayDuration > defaultTerminationGracePeriod {
		gracePeriod = preStopDelayDuration
		needsToBeAdjusted = true
	}

	return &terminationGracePeriodMutator{
		terminationGracePeriodSeconds: gracePeriod,
		needsToBeAdjusted:             needsToBeAdjusted,
	}
}

var _ PodMutator = &terminationGracePeriodMutator{}

// mutator to adjust terminationGracePeriodSeconds for pods selected by VirtualNode with preStop hook delay
// greater than the default 30 seconds.
type terminationGracePeriodMutator struct {
	terminationGracePeriodSeconds int64
	needsToBeAdjusted             bool
}

func (m *terminationGracePeriodMutator) mutate(pod *corev1.Pod) error {
	if !m.needsToBeAdjusted {
		return nil
	}

	pod.Spec.TerminationGracePeriodSeconds = &m.terminationGracePeriodSeconds
	return nil
}
