package cloudmap

import (
	"context"
	"time"

	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws/services"
	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/servicediscovery"

	corev1 "k8s.io/api/core/v1"
)

const (
	SUCCESS = "SUCCESS"
	FAIL    = "FAIL"
	TIMEOUT = "TIMEOUT"
)

func ArePodContainersReady(pod *corev1.Pod) bool {
	conditions := (&pod.Status).Conditions
	for i := range conditions {
		if conditions[i].Type == corev1.ContainersReady && conditions[i].Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func ShouldPodBeInEndpoints(pod *corev1.Pod) bool {
	switch pod.Spec.RestartPolicy {
	case corev1.RestartPolicyNever:
		return pod.Status.Phase != corev1.PodFailed && pod.Status.Phase != corev1.PodSucceeded
	case corev1.RestartPolicyOnFailure:
		return pod.Status.Phase != corev1.PodSucceeded
	default:
		return true
	}
}

func AwaitOperationSuccess(timeoutSeconds time.Duration, tickerSeconds time.Duration, f func() string) string {
	operationStatus := make(chan string, 1)
	timeout := time.After(timeoutSeconds * time.Second)
	ticker := time.Tick(tickerSeconds * time.Second)

	go func() {
		select {
		case <-timeout:
			operationStatus <- TIMEOUT
			return
		}
	}()

	go func() {
		for {
			select {
			case <-ticker:
				status := f()
				if status == SUCCESS {
					operationStatus <- status
					return
				}
			}
		}
	}()

	return <-operationStatus
}

func GetCloudMapDnsTTL(ctx context.Context, svc services.CloudMap, listServicesInput *servicediscovery.ListServicesInput) int64 {
	var defaultCloudMapDnsTTL int64 = 300
	output, _ := svc.ListServicesWithContext(ctx, listServicesInput)
	for _, svc := range output.Services {
		DnsConfig := *svc.DnsConfig
		DnsRecords := *DnsConfig.DnsRecords[len(DnsConfig.DnsRecords)-1]
		return *DnsRecords.TTL
	}
	return defaultCloudMapDnsTTL
}

func GetListServicesInputForCloudMapNamespace(ctx context.Context, svc services.CloudMap, cmNamespace string) *servicediscovery.ListServicesInput {
	//Get namespace summary
	listNamespacesInput := &servicediscovery.ListNamespacesInput{}
	var nsSummary *servicediscovery.NamespaceSummary
	svc.ListNamespacesPagesWithContext(ctx, listNamespacesInput,
		func(listNamespacesOutput *servicediscovery.ListNamespacesOutput, lastPage bool) bool {
			for _, ns := range listNamespacesOutput.Namespaces {
				if awssdk.StringValue(ns.Name) == cmNamespace {
					nsSummary = ns
					return false
				}
			}
			return true
		},
	)
	listServicesInput := &servicediscovery.ListServicesInput{
		Filters: []*servicediscovery.ServiceFilter{
			{
				Name:   awssdk.String(servicediscovery.ServiceFilterNameNamespaceId),
				Values: []*string{nsSummary.Id},
			},
		},
	}
	return listServicesInput
}
