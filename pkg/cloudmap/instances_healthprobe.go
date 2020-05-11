package cloudmap

import (
	"context"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws/services"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/k8s"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

const (
	defaultHealthProbePeriod        = 4 * time.Second
	defaultHealthTransitionDuration = 20 * time.Second
)

type InstanceProbe struct {
	instanceID string
	pod        *corev1.Pod
}

// InstancesHealthProber will probe health status
type InstancesHealthProber interface {
	// SubmitProbe will submit probe task for serviceID and instance list.
	SubmitProbe(ctx context.Context, serviceID string, instances []InstanceProbe) error
}

// NewDefaultInstancesHealthProber constructs new InstancesHealthProber
func NewDefaultInstancesHealthProber(k8sClient client.Client, cloudMapSDK services.CloudMap, log logr.Logger, stopChan <-chan struct{}) *defaultInstancesHealthProber {
	prober := &defaultInstancesHealthProber{
		k8sClient:          k8sClient,
		cloudMapSDK:        cloudMapSDK,
		probeConfigChan:    make(chan probeConfig),
		probePeriod:        defaultHealthProbePeriod,
		transitionDuration: defaultHealthTransitionDuration,
		log:                log,
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		select {
		case <-stopChan:
			cancel()
		}
	}()
	go prober.probeLoop(ctx)
	return prober
}

var _ InstancesHealthProber = &defaultInstancesHealthProber{}

type defaultInstancesHealthProber struct {
	k8sClient   client.Client
	cloudMapSDK services.CloudMap

	probeConfigChan chan probeConfig
	// how frequently to probe instance health
	probePeriod time.Duration
	// how long an instance should stay in specific healthyStatus before we update pod's condition.
	transitionDuration time.Duration

	log logr.Logger
}

type probeConfig struct {
	serviceID string
	instances []InstanceProbe
}

type instanceProbeEntry struct {
	instance InstanceProbe

	// last time this pod have been changing healthStatus
	lastTransitionTime time.Time
	// last HealthStatus probed.
	lastHealthStatus string
}

func (p *defaultInstancesHealthProber) SubmitProbe(ctx context.Context, serviceID string, instances []InstanceProbe) error {
	instancesToProbe := p.filterUnhealthyInstances(instances)
	config := probeConfig{
		serviceID: serviceID,
		instances: instancesToProbe,
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case p.probeConfigChan <- config:
		return nil
	}
}

func (p *defaultInstancesHealthProber) probeLoop(ctx context.Context) {
	instanceProbeEntriesByService := make(map[string][]instanceProbeEntry)
	for {
		var timer <-chan time.Time
		if len(instanceProbeEntriesByService) > 0 {
			timer = time.After(p.probePeriod)
		}
		select {
		case <-ctx.Done():
			return
		case config := <-p.probeConfigChan:
			if len(config.instances) == 0 {
				delete(instanceProbeEntriesByService, config.serviceID)
			} else {
				instanceProbeEntries := make([]instanceProbeEntry, 0, len(config.instances))
				for _, instance := range config.instances {
					instanceProbeEntries = append(instanceProbeEntries, instanceProbeEntry{
						instance:           instance,
						lastTransitionTime: time.Time{},
						lastHealthStatus:   "",
					})
				}
				instanceProbeEntriesByService[config.serviceID] = instanceProbeEntries
			}
		case <-timer:
			for serviceID, instanceProbeEntries := range instanceProbeEntriesByService {
				instanceProbeEntriesContinue, err := p.probeInstances(ctx, serviceID, instanceProbeEntries)
				if err != nil {
					p.log.Error(err, "failed to probe instances",
						"serviceID", serviceID)
				} else if len(instanceProbeEntriesContinue) == 0 {
					delete(instanceProbeEntriesByService, serviceID)
				} else {
					instanceProbeEntriesByService[serviceID] = instanceProbeEntriesContinue
				}
			}
		}
	}
}

func (p *defaultInstancesHealthProber) probeInstances(ctx context.Context, serviceID string, instanceProbeEntries []instanceProbeEntry) ([]instanceProbeEntry, error) {
	var instanceIDs []*string
	for _, instanceProbeEntry := range instanceProbeEntries {
		instanceIDs = append(instanceIDs, aws.String(instanceProbeEntry.instance.instanceID))
	}

	input := &servicediscovery.GetInstancesHealthStatusInput{
		ServiceId: aws.String(serviceID),
		Instances: instanceIDs,
	}
	healthStatusByInstanceID := make(map[string]string, len(instanceIDs))
	err := p.cloudMapSDK.GetInstancesHealthStatusPagesWithContext(ctx, input, func(output *servicediscovery.GetInstancesHealthStatusOutput, b bool) bool {
		for instanceID, status := range output.Status {
			healthStatusByInstanceID[instanceID] = aws.StringValue(status)
		}
		return true
	})
	if err != nil {
		return nil, err
	}
	p.log.V(1).Info("probed instance healthStatus", "healthStatusByInstanceID", healthStatusByInstanceID)

	var instanceProbeEntriesContinue []instanceProbeEntry
	for _, instanceProbeEntry := range instanceProbeEntries {
		healthStatus := healthStatusByInstanceID[instanceProbeEntry.instance.instanceID]
		shouldContinue, err := p.updateInstanceProbeEntry(ctx, &instanceProbeEntry, healthStatus)
		if err != nil {
			p.log.Error(err, "failed to update pod HealthyCondition", "pod", k8s.NamespacedName(instanceProbeEntry.instance.pod))
			instanceProbeEntriesContinue = append(instanceProbeEntriesContinue, instanceProbeEntry)
		} else if shouldContinue {
			instanceProbeEntriesContinue = append(instanceProbeEntriesContinue, instanceProbeEntry)
		}
	}

	return instanceProbeEntriesContinue, nil
}

func (p *defaultInstancesHealthProber) updateInstanceProbeEntry(ctx context.Context, instanceProbeEntry *instanceProbeEntry, healthStatus string) (bool, error) {
	if instanceProbeEntry.lastTransitionTime.IsZero() || healthStatus != instanceProbeEntry.lastHealthStatus {
		instanceProbeEntry.lastTransitionTime = time.Now()
		instanceProbeEntry.lastHealthStatus = healthStatus
	}
	if time.Since(instanceProbeEntry.lastTransitionTime) < p.transitionDuration {
		return true, nil
	}

	var podHealthyConditionStatus corev1.ConditionStatus
	switch healthStatus {
	case servicediscovery.HealthStatusHealthy:
		podHealthyConditionStatus = corev1.ConditionTrue
	case servicediscovery.HealthStatusUnknown:
		podHealthyConditionStatus = corev1.ConditionUnknown
	default:
		podHealthyConditionStatus = corev1.ConditionFalse
	}
	oldPod := instanceProbeEntry.instance.pod.DeepCopy()
	if updated := k8s.UpdatePodCondition(instanceProbeEntry.instance.pod, k8s.ConditionAWSCloudMapHealthy, podHealthyConditionStatus, nil, nil); updated {
		if err := p.k8sClient.Status().Patch(ctx, instanceProbeEntry.instance.pod, client.MergeFrom(oldPod)); err != nil {
			return false, err
		}
	}
	return podHealthyConditionStatus != corev1.ConditionTrue, nil
}

// filterUnhealthyInstances returns unhealthy ones that needs to be probed
func (p *defaultInstancesHealthProber) filterUnhealthyInstances(instances []InstanceProbe) []InstanceProbe {
	var unhealthyInstances []InstanceProbe
	for _, instance := range instances {
		podHealthyCondition := k8s.GetPodCondition(instance.pod, k8s.ConditionAWSCloudMapHealthy)
		if podHealthyCondition == nil || podHealthyCondition.Status != corev1.ConditionTrue {
			unhealthyInstances = append(unhealthyInstances, instance)
		}
	}
	return unhealthyInstances
}
