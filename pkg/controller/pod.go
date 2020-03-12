package controller

import (
	"context"
	"fmt"
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

	virtualNodes, err := c.virtualNodeLister.List(labels.Everything())
	if err != nil {
		return
	}

	cloudmapConfigs := make(map[string]*appmeshv1beta1.CloudMapServiceDiscovery)
	virtualNodesMap := make(map[string]*appmeshv1beta1.VirtualNode)

	//We don't know all the cloudmap namespaces and services to query for.
	//So we do best effort in collecting this information by peaking at
	//all virtual-nodes. Note that if there is a namespace and service
	//exists with no virtual-node matching (post-delete), those services
	//will be ignored.
	for _, virtualNode := range virtualNodes {
		if virtualNode.Spec.ServiceDiscovery == nil ||
			virtualNode.Spec.ServiceDiscovery.CloudMap == nil {
			continue
		}
		virtualNodesMap[virtualNodeCacheKey(&virtualNodeInfo{
			virtualNodeName: virtualNode.Name,
			meshName:        virtualNode.Spec.MeshName,
		})] = virtualNode
		cloudmapConfig := virtualNode.Spec.ServiceDiscovery.CloudMap
		cloudmapConfigs[cloudmapServiceCacheKey(*cloudmapConfig)] = cloudmapConfig
	}

	//Now go over each namespace and service and find instances that
	//can be de-registered based on virtual-node information.
	for _, cloudmapConfig := range cloudmapConfigs {
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
			if instance.Attributes[ctrlaws.AttrK8sNamespace] == nil ||
				instance.Attributes[ctrlaws.AttrK8sPod] == nil {
				continue
			}
			podName := awssdk.StringValue(instance.Attributes[ctrlaws.AttrK8sPod])
			podNamespace := awssdk.StringValue(instance.Attributes[ctrlaws.AttrK8sNamespace])
			pod, err := c.podsLister.Pods(podNamespace).Get(podName)
			if errors.IsNotFound(err) {
				err = c.cloud.DeregisterInstance(ctx, awssdk.StringValue(instance.Id), appmeshCloudMapConfig)
				if err != nil {
					klog.Errorf("Unable to deregister instance from cloudmap %v", err)
				}
				continue
			}

			if err != nil {
				klog.Errorf("Error looking up pod with name %s is not found in namespsace %s, %v", podName, podNamespace, err)
				continue
			}

			//If pod exists we need to check its association with virtual-node and sync instance as per virtual-nodes specification
			vnodeInfo := podToVirtualNodeInfo(pod)
			if vnodeInfo.meshName == "" || vnodeInfo.virtualNodeName == "" {
				err = c.cloud.DeregisterInstance(ctx, awssdk.StringValue(instance.Id), appmeshCloudMapConfig)
				if err != nil {
					klog.Errorf("Unable to deregister instance from cloudmap %v", err)
				}
				continue
			}

			_, ok := virtualNodesMap[virtualNodeCacheKey(vnodeInfo)]
			if !ok {
				klog.Infof("Deregistering pod %s in namespace %s, because virtualNode %+v is not found or is not using CloudMap as service-discovery", podName, podNamespace, vnodeInfo)
				err = c.cloud.DeregisterInstance(ctx, awssdk.StringValue(instance.Id), appmeshCloudMapConfig)
				if err != nil {
					klog.Errorf("Unable to deregister instance from cloudmap %v", err)
				}
				continue
			}
		}
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

	vnodeInfo := podToVirtualNodeInfo(pod)
	if vnodeInfo.meshName == "" || vnodeInfo.virtualNodeName == "" {
		klog.V(4).Infof("No appmesh annotations found for pod %s", pod.Name)
		return nil
	}

	virtualNode, err := c.cloud.GetVirtualNode(ctx, vnodeInfo.virtualNodeName, vnodeInfo.meshName)
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

	klog.V(4).Infof("Registering instance %s under service %+v", pod.Name, cloudmapConfig)
	err = c.cloud.RegisterInstance(ctx, instanceID, pod, cloudmapConfig)
	if err != nil {
		return err
	}

	return nil
}

func podToVirtualNodeInfo(pod *corev1.Pod) *virtualNodeInfo {
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

	return &virtualNodeInfo{
		meshName:        meshName,
		virtualNodeName: virtualNodeName,
	}
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

type virtualNodeInfo struct {
	meshName        string
	virtualNodeName string
}

func virtualNodeCacheKey(vn *virtualNodeInfo) string {
	return fmt.Sprintf("%s@%s", vn.virtualNodeName, vn.meshName)
}
