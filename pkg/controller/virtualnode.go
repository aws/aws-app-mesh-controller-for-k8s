package controller

import (
	"context"
	"fmt"
	"reflect"

	awssdk "github.com/aws/aws-sdk-go/aws"

	appmeshv1beta1 "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/apis/appmesh/v1beta1"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws"
	"github.com/aws/aws-sdk-go/service/appmesh"
	api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
)

const (
	attributeKeyAppMeshMeshName        = "appmesh.k8s.aws/mesh"
	attributeKeyAppMeshVirtualNodeName = "appmesh.k8s.aws/virtualNode"
	defaultHealthyThreshold            = 10
	defaultIntervalMillis              = 30000
	defaultTimeoutMillis               = 5000
	defaultUnhealthyThreshold          = 2
	defaultClientPolicyTlsEnforce      = true
)

func (c *Controller) handleVNode(key string) error {
	ctx := context.Background()

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}
	shared, err := c.virtualNodeLister.VirtualNodes(namespace).Get(name)
	if errors.IsNotFound(err) {
		klog.V(2).Infof("Virtual node %s has been deleted", key)
		return nil
	}
	if err != nil {
		return err
	}

	// Make copy here so we never update the shared copy
	vnode := shared.DeepCopy()
	//now mutate the node to adjust name and fill in default values
	c.mutateVirtualNodeForProcessing(vnode)

	// Make copy for updates so we don't save namespaced resource names
	copy := shared.DeepCopy()

	// Resources with finalizers are not deleted immediately,
	// instead the deletion timestamp is set when a client deletes them.
	if !vnode.DeletionTimestamp.IsZero() {
		c.stats.SetVirtualNodeInactive(vnode.Name, vnode.Spec.MeshName)
		// Resource is being deleted, process finalizers
		return c.handleVNodeDelete(ctx, vnode, copy)
	}

	// This is not a delete, add the deletion finalizer if it doesn't exist
	if yes, _ := containsFinalizer(copy, virtualNodeDeletionFinalizerName); !yes {
		if err := addFinalizer(copy, virtualNodeDeletionFinalizerName); err != nil {
			return fmt.Errorf("error adding finalizer %s to virtual node %s: %s", virtualNodeDeletionFinalizerName, vnode.Name, err)
		}
		if updated, err := c.updateVNodeResource(copy); err != nil {
			return fmt.Errorf("error adding finalizer %s to virtual node %s: %s", virtualNodeDeletionFinalizerName, vnode.Name, err)
		} else if updated != nil {
			copy = updated
		}
	}

	if processVNode := c.handleVNodeMeshDeleting(ctx, copy); !processVNode {
		klog.Infof("skipping processing virtual node %s", vnode.Name)
		return nil
	}

	// Get Mesh for virtual node
	meshName := vnode.Spec.MeshName
	if vnode.Spec.MeshName == "" {
		return fmt.Errorf("'MeshName' is a required field")
	}

	mesh, err := c.meshLister.Get(meshName)
	if errors.IsNotFound(err) {
		return fmt.Errorf("mesh %s for virtual node %s does not exist", meshName, name)
	}

	if !checkMeshActive(mesh) {
		return fmt.Errorf("mesh %s must be active for virtual node %s", meshName, name)
	}

	// Create virtual node if it does not exist
	targetNode, err := c.cloud.GetVirtualNode(ctx, vnode.Name, meshName)
	if err != nil {
		if aws.IsAWSErrNotFound(err) {
			if targetNode, err = c.cloud.CreateVirtualNode(ctx, vnode); err != nil {
				return fmt.Errorf("error creating virtual node: %s", err)
			}
			klog.Infof("Created virtual node %s", vnode.Name)
		} else {
			return fmt.Errorf("error describing virtual node: %s", err)
		}
	} else {
		if vnodeNeedsUpdate(vnode, targetNode) {
			if targetNode, err = c.cloud.UpdateVirtualNode(ctx, vnode); err != nil {
				return fmt.Errorf("error updating virtual node: %s", err)
			}
			klog.Infof("Updated virtual node %s", vnode.Name)
		}
	}

	c.stats.SetVirtualNodeActive(vnode.Name, vnode.Spec.MeshName)

	updated, err := c.updateVNodeStatus(copy, targetNode)
	if err != nil {
		return fmt.Errorf("error updating virtual node status: %s", err)
	} else if updated != nil {
		copy = updated
	}

	err = c.handleServiceDiscovery(ctx, vnode, copy)
	if err != nil {
		return fmt.Errorf("Error handling cloudmap service discovery for virtual node %s: %s", vnode.Name, err)
	}

	return nil
}

func (c *Controller) updateVNodeResource(vnode *appmeshv1beta1.VirtualNode) (*appmeshv1beta1.VirtualNode, error) {
	return c.meshclientset.AppmeshV1beta1().VirtualNodes(vnode.Namespace).Update(vnode)
}

func (c *Controller) updateVNodeStatus(vnode *appmeshv1beta1.VirtualNode, target *aws.VirtualNode) (*appmeshv1beta1.VirtualNode, error) {
	vnode.Status.VirtualNodeArn = target.Data.Metadata.Arn
	switch target.Status() {
	case appmesh.VirtualNodeStatusCodeActive:
		return c.updateVNodeActive(vnode, api.ConditionTrue)
	case appmesh.VirtualNodeStatusCodeInactive:
		return c.updateVNodeActive(vnode, api.ConditionFalse)
	case appmesh.VirtualNodeStatusCodeDeleted:
		return c.updateVNodeActive(vnode, api.ConditionFalse)
	}

	return nil, nil
}

func (c *Controller) updateVNodeActive(vnode *appmeshv1beta1.VirtualNode, status api.ConditionStatus) (*appmeshv1beta1.VirtualNode, error) {
	return c.updateVNodeCondition(vnode, appmeshv1beta1.VirtualNodeActive, status)
}

func (c *Controller) updateVNodeCondition(vnode *appmeshv1beta1.VirtualNode, conditionType appmeshv1beta1.VirtualNodeConditionType, status api.ConditionStatus) (*appmeshv1beta1.VirtualNode, error) {
	condition := getVNodeCondition(conditionType, vnode.Status)
	if condition.Status == status {
		return nil, nil
	}

	now := metav1.Now()
	if condition == (appmeshv1beta1.VirtualNodeCondition{}) {
		// condition does not exist
		newCondition := appmeshv1beta1.VirtualNodeCondition{
			Type:               conditionType,
			Status:             status,
			LastTransitionTime: &now,
		}
		vnode.Status.Conditions = append(vnode.Status.Conditions, newCondition)
	} else {
		// condition exists and not set to status
		condition.Status = status
		condition.LastTransitionTime = &now
	}

	err := c.setVirtualNodeStatusConditions(vnode, vnode.Status.Conditions)
	return vnode, err
}

func (c *Controller) setVirtualNodeStatusConditions(vnode *appmeshv1beta1.VirtualNode, conditions []appmeshv1beta1.VirtualNodeCondition) error {
	firstTry := true
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		var getErr error
		if !firstTry {
			vnode, getErr = c.meshclientset.AppmeshV1beta1().VirtualNodes(vnode.Namespace).Get(vnode.GetName(), metav1.GetOptions{})
			if getErr != nil {
				return getErr
			}
		}
		vnodeCopy := vnode.DeepCopy()
		vnodeCopy.Status.Conditions = conditions
		_, err := c.meshclientset.AppmeshV1beta1().VirtualNodes(vnode.Namespace).UpdateStatus(vnodeCopy)
		firstTry = false
		return err
	})
}

func getVNodeCondition(conditionType appmeshv1beta1.VirtualNodeConditionType, status appmeshv1beta1.VirtualNodeStatus) appmeshv1beta1.VirtualNodeCondition {
	for _, condition := range status.Conditions {
		if condition.Type == conditionType {
			return condition
		}
	}

	return appmeshv1beta1.VirtualNodeCondition{}
}

// vnodeNeedsUpdate compares the App Mesh API result (target) with the desired spec (desired) and
// determines if there is any drift that requires an update.
func vnodeNeedsUpdate(desired *appmeshv1beta1.VirtualNode, target *aws.VirtualNode) bool {
	if desired.Spec.ServiceDiscovery != nil {
		if target.Data.Spec.ServiceDiscovery == nil {
			return true
		}
		if desired.Spec.ServiceDiscovery.Dns != nil {
			// If Service discovery is desired, verify target is equal
			if desired.Spec.ServiceDiscovery.Dns.HostName != target.HostName() {
				return true
			}
		} else if desired.Spec.ServiceDiscovery.CloudMap != nil {
			if target.Data.Spec.ServiceDiscovery.AwsCloudMap == nil {
				return true
			}
			if desired.Spec.ServiceDiscovery.CloudMap.ServiceName !=
				awssdk.StringValue(target.Data.Spec.ServiceDiscovery.AwsCloudMap.ServiceName) {
				return true
			}
			if desired.Spec.ServiceDiscovery.CloudMap.NamespaceName !=
				awssdk.StringValue(target.Data.Spec.ServiceDiscovery.AwsCloudMap.NamespaceName) {
				return true
			}
			if len(desired.Spec.ServiceDiscovery.CloudMap.Attributes) != len(target.Data.Spec.ServiceDiscovery.AwsCloudMap.Attributes) {
				return true
			}
			for _, attr := range target.Data.Spec.ServiceDiscovery.AwsCloudMap.Attributes {
				val, ok := desired.Spec.ServiceDiscovery.CloudMap.Attributes[awssdk.StringValue(attr.Key)]
				if !ok {
					return true
				}
				if val != awssdk.StringValue(attr.Value) {
					return true
				}
			}
		} else {
			klog.Errorf("Unknown servicediscovery %v is defined for virtual-node %s", desired.Spec.ServiceDiscovery, target.Name())
		}
	} else {
		// If no desired Service Discovery, verify target is not set
		if target.Data.Spec.ServiceDiscovery != nil {
			return true
		}
	}

	if desired.Spec.Listeners != nil {
		if !reflect.DeepEqual(desired.Spec.Listeners, target.Listeners()) {
			return true
		}
	} else {
		// If the spec doesn't have any listeners, make sure target is not set
		if len(target.Listeners()) != 0 {
			return true
		}
	}

	// Since API may not guarantee order, convert backends to a map for equality checking
	if desired.Spec.Backends != nil {
		desiredBackendMap := make(map[string]appmeshv1beta1.Backend, len(desired.Spec.Backends))
		for _, val := range desired.Spec.Backends {
			desiredBackendMap[val.VirtualService.VirtualServiceName] = val
		}
		targetBackends := target.Backends()
		targetBackendMap := make(map[string]appmeshv1beta1.Backend, len(targetBackends))
		for _, val := range targetBackends {
			targetBackendMap[val.VirtualService.VirtualServiceName] = val
		}
		if !reflect.DeepEqual(desiredBackendMap, targetBackendMap) {
			return true
		}
	} else {
		// If the spec doesn't have any backends, make sure target is not set
		if len(target.Backends()) != 0 {
			return true
		}
	}

	if !reflect.DeepEqual(desired.Spec.BackendDefaults, target.BackendDefaults()) {
		return true
	}

	if vnodeLoggingNeedsUpdate(desired, target) {
		return true
	}

	return false
}

func vnodeLoggingNeedsUpdate(desired *appmeshv1beta1.VirtualNode, target *aws.VirtualNode) bool {
	if desired.Spec.Logging != nil {
		//target is missing logging so update is required
		if target.Data.Spec.Logging == nil {
			return true
		}
		if desired.Spec.Logging.AccessLog != nil {
			//target is missing access-log config so update is required
			if target.Data.Spec.Logging.AccessLog == nil {
				return true
			}
			if desired.Spec.Logging.AccessLog.File != nil {
				//target is missing access-log file config so update is required
				if target.Data.Spec.Logging.AccessLog.File == nil ||
					target.Data.Spec.Logging.AccessLog.File.Path == nil {
					return true
				}
				//path exists but differs
				if desired.Spec.Logging.AccessLog.File.Path != awssdk.StringValue(target.Data.Spec.Logging.AccessLog.File.Path) {
					return true
				}
			}
		}
	} else if target.Data.Spec.Logging != nil {
		//target has logging config but desired spec doesn't
		return true
	}
	return false
}

func (c *Controller) handleVNodeDelete(ctx context.Context, vnode *appmeshv1beta1.VirtualNode, copy *appmeshv1beta1.VirtualNode) error {
	if yes, _ := containsFinalizer(vnode, virtualNodeDeletionFinalizerName); yes {
		if err := c.deregisterInstancesForVirtualNode(ctx, copy); err != nil {
			return err
		}

		if _, err := c.cloud.DeleteVirtualNode(ctx, vnode.Name, vnode.Spec.MeshName); err != nil {
			if !aws.IsAWSErrNotFound(err) {
				return fmt.Errorf("failed to clean up virtual node %s during deletion finalizer: %s", vnode.Name, err)
			}
		}
		if err := removeFinalizer(copy, virtualNodeDeletionFinalizerName); err != nil {
			return fmt.Errorf("error removing finalizer %s to virtual node %s during deletion: %s", virtualNodeDeletionFinalizerName, vnode.Name, err)
		}
		if _, err := c.updateVNodeResource(copy); err != nil {
			return fmt.Errorf("error removing finalizer %s to virtual node %s during deletion: %s", virtualNodeDeletionFinalizerName, vnode.Name, err)
		}
	}
	return nil
}

// deregisterInstancesForVirtualNode uses serviceDiscovery configuration
// from virtualNode spec to deregister instances from AWS CloudMap
func (c *Controller) deregisterInstancesForVirtualNode(ctx context.Context, vnode *appmeshv1beta1.VirtualNode) error {
	if vnode.Spec.ServiceDiscovery == nil ||
		vnode.Spec.ServiceDiscovery.CloudMap == nil {
		return nil
	}
	cloudmapConfig := vnode.Spec.ServiceDiscovery.CloudMap
	appmeshCloudMapConfig := &appmesh.AwsCloudMapServiceDiscovery{
		NamespaceName: awssdk.String(cloudmapConfig.NamespaceName),
		ServiceName:   awssdk.String(cloudmapConfig.ServiceName),
	}

	instances, err := c.cloud.ListInstances(ctx, appmeshCloudMapConfig)
	if err != nil {
		return fmt.Errorf("Error getting list of instances under virtual node %s to deregister: %s", vnode.Name, err)
	}
	for _, instance := range instances {
		meshName := awssdk.StringValue(instance.Attributes[attributeKeyAppMeshMeshName])
		virtualNodeName := awssdk.StringValue(instance.Attributes[attributeKeyAppMeshVirtualNodeName])
		if meshName != vnode.Spec.MeshName ||
			virtualNodeName != namespacedResourceName(vnode.Name, vnode.Namespace) {
			continue
		}
		err = c.cloud.DeregisterInstance(ctx, awssdk.StringValue(instance.Id), appmeshCloudMapConfig)
		if err != nil {
			return err
		}
	}
	return nil
}

// handleVNodeMeshDeleting deletes virtualNode when mesh is deleted (cascade)
func (c *Controller) handleVNodeMeshDeleting(ctx context.Context, vnode *appmeshv1beta1.VirtualNode) (processVNode bool) {
	mesh, err := c.meshLister.Get(vnode.Spec.MeshName)

	if err != nil {
		if errors.IsNotFound(err) {
			// If mesh doesn't exist, do nothing
			klog.Infof("mesh doesn't exist, skipping processing virtual node %s", vnode.Name)
		} else {
			klog.Errorf("error getting mesh: %s", err)
		}
		return false
	}

	// if mesh DeletionTimestamp is set, clean up virtual node via App Mesh API
	if !mesh.DeletionTimestamp.IsZero() {
		if err := c.meshclientset.AppmeshV1beta1().VirtualNodes(vnode.Namespace).Delete(vnode.Name, &metav1.DeleteOptions{}); err != nil {
			klog.Errorf("Deletion failed for virtual node: %s - %s", vnode.Name, err)
			return false
		}
		klog.Infof("Deleted App Mesh virtual node %s because mesh %s is being deleted", vnode.Name, vnode.Spec.MeshName)
	}

	return true
}

// handleServiceDiscovery uses serviceDiscovery configuration on virtualNode
// to setup external resources. For now only Cloud Map _service_ is setup
func (c *Controller) handleServiceDiscovery(ctx context.Context, vnode *appmeshv1beta1.VirtualNode, copyForUpdate *appmeshv1beta1.VirtualNode) error {
	if vnode.Spec.ServiceDiscovery == nil || vnode.Spec.ServiceDiscovery.CloudMap == nil {
		klog.V(4).Infof("Virtual-node %s is not using Cloud Map as servicediscovery", vnode.Name)
		return nil
	}

	cloudmapNamespaceName := vnode.Spec.ServiceDiscovery.CloudMap.NamespaceName
	cloudmapServiceName := vnode.Spec.ServiceDiscovery.CloudMap.ServiceName

	if cloudmapNamespaceName == "" {
		return fmt.Errorf("CloudMap servicediscovery is missing NamespaceName value for virtual node %s", vnode.Name)
	}

	if cloudmapServiceName == "" {
		return fmt.Errorf("CloudMap servicediscovery is missing ServiceName value for virtual node %s", vnode.Name)
	}

	cloudmapConfig := &appmesh.AwsCloudMapServiceDiscovery{
		NamespaceName: awssdk.String(cloudmapNamespaceName),
		ServiceName:   awssdk.String(cloudmapServiceName),
	}

	//It is okay to call Create multiple times for same service-name.
	//It is also cheaper than calling get and then figuring out to create.
	cloudmapService, err := c.cloud.CloudMapCreateService(ctx, cloudmapConfig, c.name)
	if err != nil {
		return err
	}

	klog.V(4).Infof("Created CloudMap service %s (id:%s)", cloudmapServiceName, cloudmapService.ServiceID)

	statusErr := c.setVirtualNodeStatusCloudMapService(copyForUpdate, &appmeshv1beta1.CloudMapServiceStatus{
		NamespaceID: awssdk.String(cloudmapService.NamespaceID),
		ServiceID:   awssdk.String(cloudmapService.ServiceID),
	})

	if statusErr != nil {
		//these are informational fields, so will be saved when reconciling
		klog.Errorf("Error updating CloudMapServiceStatus on virtual node %s: %s", copyForUpdate.Name, statusErr)
	}

	return nil
}

// setVirtualNodeStatusCloudMapService updates the status of virtualNode with CloudMap service details
func (c *Controller) setVirtualNodeStatusCloudMapService(vnode *appmeshv1beta1.VirtualNode, cloudmapService *appmeshv1beta1.CloudMapServiceStatus) error {
	firstTry := true
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		var getErr error
		if !firstTry {
			vnode, getErr = c.meshclientset.AppmeshV1beta1().VirtualNodes(vnode.Namespace).Get(vnode.GetName(), metav1.GetOptions{})
			if getErr != nil {
				return getErr
			}
		}
		vnodeCopy := vnode.DeepCopy()
		vnodeCopy.Status.CloudMapService = cloudmapService
		_, err := c.meshclientset.AppmeshV1beta1().VirtualNodes(vnode.Namespace).UpdateStatus(vnodeCopy)
		firstTry = false
		return err
	})
}

// reconcileServices reconciles the external _service_ resources corresponding to virtualNode
// using its serviceDiscovery configuration.
func (c *Controller) reconcileServices(ctx context.Context) error {
	virtualNodes, err := c.virtualNodeLister.List(labels.Everything())
	if err != nil {
		return err
	}

	for _, originalVNode := range virtualNodes {
		vnode := originalVNode.DeepCopy()
		copyForUpdate := originalVNode.DeepCopy()
		c.handleServiceDiscovery(ctx, vnode, copyForUpdate)
	}

	//TODO: delete unused cloudmap services
	return nil
}

func (c *Controller) mutateVirtualNodeForProcessing(vnode *appmeshv1beta1.VirtualNode) {
	vnode.Name = namespacedResourceName(vnode.Name, vnode.Namespace)
	if vnode.Spec.ServiceDiscovery != nil && vnode.Spec.ServiceDiscovery.CloudMap != nil {
		if vnode.Spec.ServiceDiscovery.CloudMap.Attributes == nil {
			vnode.Spec.ServiceDiscovery.CloudMap.Attributes = map[string]string{}
		}
		vnode.Spec.ServiceDiscovery.CloudMap.Attributes[attributeKeyAppMeshMeshName] = vnode.Spec.MeshName
		vnode.Spec.ServiceDiscovery.CloudMap.Attributes[attributeKeyAppMeshVirtualNodeName] = vnode.Name
	}

	if vnode.Spec.Listeners != nil {
		for _, listener := range vnode.Spec.Listeners {
			if listener.HealthCheck != nil {
				//if port-mapping is not set for health-check we default it to listener's port-mapping
				listener.HealthCheck.Port = defaultInt64(listener.HealthCheck.Port, listener.PortMapping.Port)
				listener.HealthCheck.Protocol = defaultString(listener.HealthCheck.Protocol, listener.PortMapping.Protocol)
				//below are some sane defaults for majority of applications
				listener.HealthCheck.HealthyThreshold = defaultInt64(listener.HealthCheck.HealthyThreshold, defaultHealthyThreshold)
				listener.HealthCheck.IntervalMillis = defaultInt64(listener.HealthCheck.IntervalMillis, defaultIntervalMillis)
				listener.HealthCheck.TimeoutMillis = defaultInt64(listener.HealthCheck.TimeoutMillis, defaultTimeoutMillis)
				listener.HealthCheck.UnhealthyThreshold = defaultInt64(listener.HealthCheck.UnhealthyThreshold, defaultUnhealthyThreshold)
			}
		}
	}

	// The following overrides facilitate matching the spec to the API response.
	// They can be removed if the API response matches the input spec.
	if vnode.Spec.Backends != nil {
		for _, backend := range vnode.Spec.Backends {
			if backend.VirtualService.ClientPolicy != nil {
				clientPolicy := backend.VirtualService.ClientPolicy
				if clientPolicy.TLS != nil {
					mergeTlsClientPolicyDefaults(clientPolicy.TLS)
				}
			}
		}
	}
	if vnode.Spec.BackendDefaults != nil {
		if vnode.Spec.BackendDefaults.ClientPolicy != nil {
			clientPolicy := vnode.Spec.BackendDefaults.ClientPolicy
			if clientPolicy.TLS != nil {
				mergeTlsClientPolicyDefaults(clientPolicy.TLS)
			}
		}
	}
}

func mergeTlsClientPolicyDefaults(specClientPolicyTls *appmeshv1beta1.ClientPolicyTls) {
	// API currently returns the default of true if no flag is set in spec
	if specClientPolicyTls.Enforce == nil {
		specClientPolicyTls.Enforce = awssdk.Bool(defaultClientPolicyTlsEnforce)
	}
	// API currently returns an empty array in response if no ports are set in spec
	if specClientPolicyTls.Ports == nil {
		specClientPolicyTls.Ports = []int64{}
	}
}

func defaultInt64(v *int64, defaultVal int64) *int64 {
	if v != nil {
		return v
	}
	return awssdk.Int64(defaultVal)
}

func defaultString(v *string, defaultVal string) *string {
	if v != nil {
		return v
	}
	return awssdk.String(defaultVal)
}
