package controllers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-logr/logr"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewJobReconciler(client client.Client, logger logr.Logger) *JobReconciler {
	return &JobReconciler{
		K8sClient: client,
		log:       logger,
	}
}

type JobReconciler struct {
	K8sClient client.Client
	log       logr.Logger
}

func (r *JobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&batchv1.Job{}).
		Owns(&corev1.Pod{}).
		Complete(r)
}

func (r *JobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	job := &batchv1.Job{}
	err := r.K8sClient.Get(ctx, req.NamespacedName, job)
	if err != nil {
		return ctrl.Result{}, err
	}

	pods := &corev1.PodList{}
	err = r.K8sClient.List(ctx, pods, client.InNamespace(req.Namespace), client.MatchingLabels(job.Spec.Template.Labels))
	if err != nil {
		return ctrl.Result{}, err
	}

	for _, p := range pods.Items {
		// skip if containers are not initialized
		if len(p.Status.ContainerStatuses) == 0 {
			continue
		}

		allContainersTerminated := true

		for _, c := range p.Status.ContainerStatuses {

			if c.Name == "envoy" {
				// if envoy has already been terminated exit early
				if c.State.Terminated != nil {
					return ctrl.Result{}, nil
				}

				continue
			}

			// do not quit sidecar if any app container is not terminated
			if c.State.Terminated == nil {
				allContainersTerminated = false
				break
			}
		}

		if allContainersTerminated {
			r.log.Info("Quitting sidecar for pod")
			r.log.V(5).Info("pod: %v", p)

			// quit envoy
			req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://%s:9901/quitquitquit", p.Status.PodIP), nil)
			req.Header.Add("Connection", "close")
			if err != nil {
				return ctrl.Result{}, err
			}

			httpClient := &http.Client{Timeout: 250 * time.Millisecond}

			_, err = httpClient.Do(req)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}
