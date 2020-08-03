package inject

import (
	corev1 "k8s.io/api/core/v1"
)

func envoyReadinessProbe(initialDelaySeconds int32, periodSeconds int32) *corev1.Probe {
	return &corev1.Probe{
		Handler: corev1.Handler{

			// server_info returns the following struct:
			// {
			//	"version": "...",
			//	"state": "...",
			//	"uptime_current_epoch": "{...}",
			//	"uptime_all_epochs": "{...}",
			//	"hot_restart_version": "...",
			//      "command_line_options": "{...}"
			//  }
			// server_info->state supports the following states: LIVE, DRAINING, PRE_INITIALIZING, and INITIALIZING
			// LIVE: Server is live and serving traffic
			// DRAINING:  ⁣Server is draining listeners in response to external health checks failing
			// PRE_INITIALIZING: Server has not yet completed cluster manager initialization
			// INITIALIZING: Server is running the cluster manager initialization callbacks
			Exec: &corev1.ExecAction{Command: []string{
				"sh", "-c", "curl -s http://localhost:9901/server_info | grep state | grep -q LIVE",
			}},
		},

		// Number of seconds after the container has started before readiness probes are initiated
		InitialDelaySeconds: initialDelaySeconds,

		// Number of seconds after which the probe times out
		// This is a call to the local Envoy endpoint. 1 second is more than enough for timeout
		TimeoutSeconds: 1,

		// How often (in seconds) to perform the probe
		PeriodSeconds: periodSeconds,

		// Minimum consecutive successes for the probe to be considered successful after having failed
		// If Envoy shows LIVE status once, we're good to call it a success
		SuccessThreshold: 1,

		// Minimum consecutive failures for the probe to be considered failed after having succeeded
		// Keeping the failure threshold to 3 to not fail preemptively
		FailureThreshold: 3,
	}
}
