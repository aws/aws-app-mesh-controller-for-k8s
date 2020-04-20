package controller

import (
	"context"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
	"strings"
	"time"

	ctrlaws "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws"

	appmeshv1beta1 "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/apis/appmesh/v1beta1"
	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/appmesh"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
)

const (
	envAppMeshVirtualNodeName        = "APPMESH_VIRTUAL_NODE_NAME"
	annotationAppMeshMeshName        = "appmesh.k8s.aws/mesh"
	annotationAppMeshVirtualNodeName = "appmesh.k8s.aws/virtualNode"
)


type cloudMapInstanceCacheItem struct {
	key             string
	instanceSummary map[string]bool
}

func (c *Controller) handlePod(key string) error {
	begin := time.Now()
	defer func() {
		c.stats.RecordOperationDuration("podctl", "", "handlePod", time.Since(begin))
	}()

	ctx := context.Background()

	klog.V(4).Infof("processing pod %s", key)

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	pod, err := c.podsLister.Pods(namespace).Get(name)
	if errors.IsNotFound(err) {
		klog.V(2).Infof("Pod %s has been deleted", key)
		return nil
	}
	if err != nil {
		return err
	}

	return c.syncPod(ctx, pod)
}

func (c *Controller) reconcileInstances(ctx context.Context) {
	c.syncPods(ctx)
	c.syncInstances(ctx)
}

func (c *Controller) syncPods(ctx context.Context) {
	begin := time.Now()
	defer func() {
		c.stats.RecordOperationDuration("podctl", "", "syncPods", time.Since(begin))
	}()
	pods, err := c.podsLister.List(labels.Everything())
	if err != nil {
		klog.Errorf("Error listing pods %v", err)
		return
	}

	for _, pod := range pods {
		err = c.syncPod(ctx, pod)
		if err != nil {
			klog.Errorf("Error syncing pod %s, %v", pod.Name, err)
		}
	}
}

func (c *Controller) syncInstances(ctx context.Context) {
	begin := time.Now()
	defer func() {
		c.stats.RecordOperationDuration("podctl", "", "syncInstances", time.Since(begin))
	}()

	syncedServices := make(map[string]bool)

	virtualNodes, err := c.virtualNodeLister.List(labels.Everything())
	if err != nil {
		return
	}

	for _, virtualNode := range virtualNodes {
		if virtualNode.Spec.ServiceDiscovery == nil ||
			virtualNode.Spec.ServiceDiscovery.CloudMap == nil {
			continue
		}
		cloudmapConfig := virtualNode.Spec.ServiceDiscovery.CloudMap
		key := cloudmapServiceCacheKey(*cloudmapConfig)
		if _, ok := syncedServices[key]; ok {
			continue
		}

		appmeshCloudMapConfig := &appmesh.AwsCloudMapServiceDiscovery{
			NamespaceName: awssdk.String(cloudmapConfig.NamespaceName),
			ServiceName:   awssdk.String(cloudmapConfig.ServiceName),
		}

		instances, err := c.cloud.ListInstances(ctx, appmeshCloudMapConfig)
		if err != nil {
			klog.Errorf("Error syncing instances for cloudmapConfig %v, %v", cloudmapConfig, err)
			continue
		}

		for _, instance := range instances {
			podName := awssdk.StringValue(instance.Attributes[ctrlaws.AttrK8sPod])
			podNamespace := awssdk.StringValue(instance.Attributes[ctrlaws.AttrK8sNamespace])
			_, err := c.podsLister.Pods(podNamespace).Get(podName)
			if errors.IsNotFound(err) {
				err = c.cloud.DeregisterInstance(ctx, awssdk.StringValue(instance.Id), appmeshCloudMapConfig)
				if err != nil {
					klog.Errorf("Unable to deregister instance from cloudmap %v", err)
				}
			}
		}

		syncedServices[key] = true
	}
}

func (c *Controller) syncPod(ctx context.Context, pod *corev1.Pod) error {
	begin := time.Now()
	defer func() {
		c.stats.RecordOperationDuration("podctl", "", "syncPod", time.Since(begin))
	}()

	instanceID := podToInstanceID(pod)
	if instanceID == "" {
		klog.V(4).Infof("Skipping pod %s with no instanceID mapping", pod.Name)
		return nil
	}

	if pod.Status.Phase != corev1.PodRunning {
		klog.V(4).Infof("Pod is in %s phase, skipping", pod.Status.Phase)
		return nil
	}

	var meshName string
	var virtualNodeName string

	//TODO: Remove this hack and always expect proper annotations to be injected into the pod
	for _, container := range pod.Spec.Containers {
		for _, envvar := range container.Env {
			if envvar.Name == envAppMeshVirtualNodeName {
				fqNodeName := envvar.Value
				//e.g. "mesh/eks-mesh/virtualNode/colorgateway-color"
				splits := strings.Split(fqNodeName, "/")
				if len(splits) == 4 && splits[0] == "mesh" && splits[2] == "virtualNode" {
					meshName = splits[1]
					virtualNodeName = splits[3]
				} else {
					klog.Errorf("skipping virtualNode because name %v is not well formed for pod %s", splits, pod.Name)
				}
				break
			}
		}
	}

	for k, v := range pod.Annotations {
		if k == annotationAppMeshMeshName {
			if meshName == "" {
				meshName = v
			}
		}
		if k == annotationAppMeshVirtualNodeName {
			if virtualNodeName == "" {
				virtualNodeName = v
			}
		}
	}

	if meshName == "" || virtualNodeName == "" {
		klog.V(4).Infof("No appmesh annotations found for pod %s", pod.Name)
		return nil
	}

	virtualNode, err := c.cloud.GetVirtualNode(ctx, virtualNodeName, meshName)
	if errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}

	if virtualNode.Data.Spec.ServiceDiscovery == nil {
		return nil
	}

	if virtualNode.Data.Spec.ServiceDiscovery.AwsCloudMap == nil {
		return nil
	}

	cloudmapConfig := virtualNode.Data.Spec.ServiceDiscovery.AwsCloudMap

	if !pod.DeletionTimestamp.IsZero() {
		klog.V(4).Infof("Deregistering instance %s under service %+v", pod.Name, cloudmapConfig)
		err = c.cloud.DeregisterInstance(ctx, instanceID, cloudmapConfig)
		if err != nil {
			return err
		}
	}

	cloudMapServiceKey := *cloudmapConfig.NamespaceName + "-" + *cloudmapConfig.ServiceName

	existingItem, exists, _ := c.cloudMapInstanceCache.Get(&cloudMapInstanceCacheItem{
		key: cloudMapServiceKey,
	})

	var serviceInstanceSummary map[string]bool

	if exists {
		if healthy, ok := existingItem.(*cloudMapInstanceCacheItem).instanceSummary[pod.Status.PodIP]; ok {
			//Right now we're not logging pod/instance health info to CloudMap. So, if it is in cache then it is
			//considered healthy. *TODO - v1.0* */
			klog.V(4).Info("Pod Instance already registered. Nothing to do for this pod; Healthy?: ", healthy)
			return nil
		}
		serviceInstanceSummary = existingItem.(*cloudMapInstanceCacheItem).instanceSummary
		if serviceInstanceSummary == nil { serviceInstanceSummary = make(map[string]bool) }
		serviceInstanceSummary[pod.Status.PodIP] = true
	} else {
		//Retrieve CloudMap instances for this service
		serviceInstanceSummary := make(map[string]bool)
		registered := false

		appmeshCloudMapConfig := &appmesh.AwsCloudMapServiceDiscovery{
			NamespaceName: cloudmapConfig.NamespaceName,
			ServiceName:   cloudmapConfig.ServiceName,
		}

		if serviceInstances, _ := c.getServiceInstancesFromCloudMap(ctx, appmeshCloudMapConfig); len(serviceInstances) > 0 {
			for _, instance := range serviceInstances {
				podName := awssdk.StringValue(instance.Attributes[ctrlaws.AttrK8sPod])
				podNamespace := awssdk.StringValue(instance.Attributes[ctrlaws.AttrK8sNamespace])
				instanceID := awssdk.StringValue(instance.Id)
				serviceName := awssdk.StringValue(instance.Attributes[ctrlaws.AttrK8sApp])

				if instanceID == pod.Status.PodIP {
					klog.V(4).Info("Instance already registered for ServiceName: ", podName, serviceName)
					registered = true
				}

				cloudMapServiceKey = podNamespace + ".pvt.aws.local" + "-" + serviceName
				serviceInstanceSummary[instanceID] = true
			}

			serviceItem := &cloudMapInstanceCacheItem{
				key:             cloudMapServiceKey,
				instanceSummary: serviceInstanceSummary,
			}
			_ = c.cloudMapInstanceCache.Add(serviceItem)

			if registered {
				klog.V(4).Info("Instance already registered. Cache Refreshed. Move on")
				return nil
			}
		}
	}

	klog.Info("Registering instance %s under service %s", pod.Name, *cloudmapConfig.ServiceName)
	err = c.cloud.RegisterInstance(ctx, instanceID, pod, cloudmapConfig)
	if err != nil {
		return err
	}

	serviceItem := &cloudMapInstanceCacheItem{
		key: cloudMapServiceKey,
		instanceSummary: serviceInstanceSummary,
	}

	_ = c.cloudMapInstanceCache.Add(serviceItem)

	return nil
}

func (c *Controller) getServiceInstancesFromCloudMap(ctx context.Context,
	           appmeshCloudMapConfig *appmesh.AwsCloudMapServiceDiscovery)([]*servicediscovery.InstanceSummary, error) {

	klog.V(4).Info("Reach out to CloudMap for service: ", *appmeshCloudMapConfig.ServiceName)
	instances, err := c.cloud.ListInstances(ctx, appmeshCloudMapConfig)
	if err != nil {
		klog.Errorf("Error obtaining instances for cloudmapConfig %v, %v", appmeshCloudMapConfig, err)
		return instances, err
	}

	return instances, nil
}

func podToInstanceID(pod *corev1.Pod) string {
	if pod.Status.PodIP == "" {
		return ""
	}

	return pod.Status.PodIP
}

func cloudmapServiceCacheKey(cloudmapConfig appmeshv1beta1.CloudMapServiceDiscovery) string {
	return cloudmapConfig.ServiceName + "@" + cloudmapConfig.NamespaceName
}
