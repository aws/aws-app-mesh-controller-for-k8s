package cloudmap

import (
	"context"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws/services"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/k8s"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

const (
	defaultHealthProbePeriod        = 4 * time.Second
	defaultHealthTransitionDuration = 20 * time.Second
)

// instancesHealthProber will probe health status
type instancesHealthProber interface {
	// Submit will submit probe task for serviceID and instances.
	Submit(ctx context.Context, service serviceSummary, subset serviceSubset, instanceInfoByID map[string]instanceInfo, timeout time.Duration) error
}

// newDefaultInstancesHealthProber constructs new instancesHealthProber
func newDefaultInstancesHealthProber(ctx context.Context, k8sClient client.Client, cloudMapSDK services.CloudMap, log logr.Logger) *defaultInstancesHealthProber {
	prober := &defaultInstancesHealthProber{
		k8sClient:          k8sClient,
		cloudMapSDK:        cloudMapSDK,
		probeRequestChan:   make(chan probeRequest),
		probePeriod:        defaultHealthProbePeriod,
		transitionDuration: defaultHealthTransitionDuration,
		log:                log,
	}
	go prober.probeLoop(ctx)
	return prober
}

var _ instancesHealthProber = &defaultInstancesHealthProber{}

type defaultInstancesHealthProber struct {
	k8sClient   client.Client
	cloudMapSDK services.CloudMap

	probeRequestChan chan probeRequest
	// how frequently to probe instance health
	probePeriod time.Duration
	// how long an instance should stay in specific healthyStatus before we update pod's condition.
	transitionDuration time.Duration

	log logr.Logger
}

type instanceHealthyStatusProbeFunc func(ctx context.Context, serviceID string, instanceIDs []string) (map[string]bool, error)

type probeRequest struct {
	serviceSubsetID  serviceSubsetID
	probeFunc        instanceHealthyStatusProbeFunc
	instanceInfoByID map[string]instanceInfo
	timeout          time.Duration
}

type probeConfig struct {
	probeFunc    instanceHealthyStatusProbeFunc
	probeEntries []instanceProbeEntry
	timeoutTime  time.Time
}

type instanceProbeEntry struct {
	instanceID   string
	instanceInfo instanceInfo

	// last time this pod have been changing healthyStatus
	lastTransitionTime time.Time
	// last HealthyStatus probed.
	lastHealthyStatus bool
}

func (p *defaultInstancesHealthProber) Submit(ctx context.Context, service serviceSummary, subset serviceSubset, instanceInfoByID map[string]instanceInfo, timeout time.Duration) error {
	serviceSubsetID := serviceSubsetID{serviceID: service.serviceID, subsetID: subset.SubsetID()}
	probeFunc := p.probeInstanceHealthyStatusWithoutCustomHC
	if service.healthCheckCustomConfig != nil {
		probeFunc = p.probeInstanceHealthyStatusWithCustomHC
	}

	instanceInfoByID, err := p.filterInstancesBlockedByCMHealthyReadinessGate(ctx, instanceInfoByID)
	if err != nil {
		return err
	}
	probeRequest := probeRequest{
		serviceSubsetID:  serviceSubsetID,
		probeFunc:        probeFunc,
		instanceInfoByID: instanceInfoByID,
		timeout:          timeout,
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case p.probeRequestChan <- probeRequest:
		return nil
	}
}

func (p *defaultInstancesHealthProber) probeLoop(ctx context.Context) {
	probeConfigByServiceSubset := make(map[serviceSubsetID]probeConfig)
	for {
		var timer <-chan time.Time
		if len(probeConfigByServiceSubset) > 0 {
			timer = time.After(p.probePeriod)
		}

		select {
		case <-ctx.Done():
			return
		case probeRequest := <-p.probeRequestChan:
			if len(probeRequest.instanceInfoByID) == 0 {
				delete(probeConfigByServiceSubset, probeRequest.serviceSubsetID)
			} else {
				probeEntries := make([]instanceProbeEntry, 0, len(probeRequest.instanceInfoByID))
				for instanceID, instanceInfo := range probeRequest.instanceInfoByID {
					probeEntries = append(probeEntries, instanceProbeEntry{
						instanceID:         instanceID,
						instanceInfo:       instanceInfo,
						lastTransitionTime: time.Time{},
						lastHealthyStatus:  false,
					})
				}
				probeConfigByServiceSubset[probeRequest.serviceSubsetID] = probeConfig{
					probeFunc:    probeRequest.probeFunc,
					probeEntries: probeEntries,
					timeoutTime:  time.Now().Add(probeRequest.timeout),
				}
			}
		case <-timer:
			for serviceSubsetID, instancesProbeConfig := range probeConfigByServiceSubset {
				if time.Now().After(instancesProbeConfig.timeoutTime) {
					p.log.Error(errors.New("timeout probe instances"),
						"serviceID", serviceSubsetID.serviceID,
						"subsetID", serviceSubsetID.subsetID)
					delete(probeConfigByServiceSubset, serviceSubsetID)
					continue
				}
				probeEntriesToContinue, err := p.probeInstances(ctx, serviceSubsetID.serviceID, instancesProbeConfig.probeFunc, instancesProbeConfig.probeEntries)
				if err != nil {
					p.log.Error(err, "failed to probe instances",
						"serviceID", serviceSubsetID.serviceID,
						"subsetID", serviceSubsetID.subsetID)
				} else if len(probeEntriesToContinue) == 0 {
					delete(probeConfigByServiceSubset, serviceSubsetID)
				} else {
					probeConfigByServiceSubset[serviceSubsetID] = probeConfig{
						probeFunc:    instancesProbeConfig.probeFunc,
						probeEntries: probeEntriesToContinue,
						timeoutTime:  instancesProbeConfig.timeoutTime,
					}
				}
			}
		}
	}
}

func (p *defaultInstancesHealthProber) probeInstances(ctx context.Context, serviceID string, probeFunc instanceHealthyStatusProbeFunc, probeEntries []instanceProbeEntry) ([]instanceProbeEntry, error) {
	var instanceIDs []string
	for _, probeEntry := range probeEntries {
		instanceIDs = append(instanceIDs, probeEntry.instanceID)
	}
	healthyStatusByInstanceID, err := probeFunc(ctx, serviceID, instanceIDs)
	if err != nil {
		return nil, err
	}
	p.log.V(1).Info("probed instance healthStatus",
		"healthyStatusByInstanceID", healthyStatusByInstanceID)

	var probeEntriesToContinue []instanceProbeEntry
	for _, probeEntry := range probeEntries {
		healthStatus := healthyStatusByInstanceID[probeEntry.instanceID]
		shouldContinue, err := p.updateInstanceProbeEntry(ctx, &probeEntry, healthStatus)
		if err != nil {
			p.log.Error(err, "failed to update pod HealthyCondition",
				"pod", k8s.NamespacedName(probeEntry.instanceInfo.pod))
			probeEntriesToContinue = append(probeEntriesToContinue, probeEntry)
		} else if shouldContinue {
			probeEntriesToContinue = append(probeEntriesToContinue, probeEntry)
		}
	}

	return probeEntriesToContinue, nil
}

func (p *defaultInstancesHealthProber) updateInstanceProbeEntry(ctx context.Context, probeEntry *instanceProbeEntry, healthyStatus bool) (bool, error) {
	if probeEntry.lastTransitionTime.IsZero() || probeEntry.lastHealthyStatus != healthyStatus {
		probeEntry.lastTransitionTime = time.Now()
		probeEntry.lastHealthyStatus = healthyStatus
	}
	if time.Since(probeEntry.lastTransitionTime) < p.transitionDuration {
		return true, nil
	}

	podHealthyConditionStatus := corev1.ConditionFalse
	if healthyStatus {
		podHealthyConditionStatus = corev1.ConditionTrue
	}

	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		pod := probeEntry.instanceInfo.pod.DeepCopy()
		if err := p.k8sClient.Get(ctx, k8s.NamespacedName(pod), pod); err != nil {
			return err
		}
		oldPod := pod.DeepCopy()
		if updated := k8s.UpdatePodCondition(pod, k8s.ConditionAWSCloudMapHealthy, podHealthyConditionStatus, nil, nil); updated {
			if err := p.k8sClient.Status().Patch(ctx, pod, client.MergeFrom(oldPod)); err != nil {
				return err
			}
		}
		probeEntry.instanceInfo.pod = pod
		return nil
	}); err != nil {
		return false, err
	}
	return podHealthyConditionStatus != corev1.ConditionTrue, nil
}

func (p *defaultInstancesHealthProber) probeInstanceHealthyStatusWithoutCustomHC(ctx context.Context, serviceID string, instanceIDs []string) (map[string]bool, error) {
	healthyStatusByInstanceID := make(map[string]bool, len(instanceIDs))
	for _, instanceID := range instanceIDs {
		_, err := p.cloudMapSDK.GetInstanceWithContext(ctx, &servicediscovery.GetInstanceInput{
			ServiceId:  aws.String(serviceID),
			InstanceId: aws.String(instanceID),
		})
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == servicediscovery.ErrCodeInstanceNotFound {
				healthyStatusByInstanceID[instanceID] = false
				continue
			}
			return nil, err
		}
		healthyStatusByInstanceID[instanceID] = true
	}
	return healthyStatusByInstanceID, nil
}

func (p *defaultInstancesHealthProber) probeInstanceHealthyStatusWithCustomHC(ctx context.Context, serviceID string, instanceIDs []string) (map[string]bool, error) {
	healthyStatusByInstanceID := make(map[string]bool, len(instanceIDs))
	if err := p.cloudMapSDK.GetInstancesHealthStatusPagesWithContext(ctx, &servicediscovery.GetInstancesHealthStatusInput{
		ServiceId: aws.String(serviceID),
		Instances: aws.StringSlice(instanceIDs),
	}, func(output *servicediscovery.GetInstancesHealthStatusOutput, b bool) bool {
		for instanceID, status := range output.Status {
			healthyStatusByInstanceID[instanceID] = aws.StringValue(status) == servicediscovery.HealthStatusHealthy
		}
		return true
	}); err != nil {
		return nil, err
	}
	return healthyStatusByInstanceID, nil
}

// filterInstancesBlockedByCMHealthyReadinessGate returns unhealthy ones that needs are blocked by ConditionAWSCloudMapHealthy readinessGate.
func (p *defaultInstancesHealthProber) filterInstancesBlockedByCMHealthyReadinessGate(ctx context.Context, instanceInfoByID map[string]instanceInfo) (map[string]instanceInfo, error) {
	blockedInstanceByID := make(map[string]instanceInfo, len(instanceInfoByID))
	for instanceID, instanceInfo := range instanceInfoByID {
		if err := p.k8sClient.Get(ctx, k8s.NamespacedName(instanceInfo.pod), instanceInfo.pod); err != nil {
			return nil, err
		}
		podHealthyCondition := k8s.GetPodCondition(instanceInfo.pod, k8s.ConditionAWSCloudMapHealthy)
		if podHealthyCondition == nil || podHealthyCondition.Status != corev1.ConditionTrue {
			blockedInstanceByID[instanceID] = instanceInfo
		}
	}
	return blockedInstanceByID, nil
}
