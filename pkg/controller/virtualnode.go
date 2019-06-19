package controller

import (
	"context"
	"fmt"

	awssdk "github.com/aws/aws-sdk-go/aws"

	appmeshv1beta1 "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/apis/appmesh/v1beta1"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws"
	"github.com/aws/aws-sdk-go/service/appmesh"
	set "github.com/deckarep/golang-set"
	api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
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

	// Make copy for updates so we don't save namespaced resource names
	copy := shared.DeepCopy()

	// Namespace resource names
	vnode.Name = namespacedResourceName(vnode.Name, vnode.Namespace)

	// Resources with finalizers are not deleted immediately,
	// instead the deletion timestamp is set when a client deletes them.
	if !vnode.DeletionTimestamp.IsZero() {
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

	if processVNode := c.handleVNodeMeshDeleting(ctx, vnode); !processVNode {
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

	updated, err := c.updateVNodeStatus(copy, targetNode)
	if err != nil {
		return fmt.Errorf("error updating virtual node status: %s", err)
	} else if updated != nil {
		copy = updated
	}

	return nil
}

func (c *Controller) updateVNodeResource(vnode *appmeshv1beta1.VirtualNode) (*appmeshv1beta1.VirtualNode, error) {
	return c.meshclientset.AppmeshV1beta1().VirtualNodes(vnode.Namespace).Update(vnode)
}

func (c *Controller) updateVNodeStatus(vnode *appmeshv1beta1.VirtualNode, target *aws.VirtualNode) (*appmeshv1beta1.VirtualNode, error) {
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
	now := metav1.Now()
	condition := getVNodeCondition(conditionType, vnode.Status)
	if condition == (appmeshv1beta1.VirtualNodeCondition{}) {
		// condition does not exist
		newCondition := appmeshv1beta1.VirtualNodeCondition{
			Type:               conditionType,
			Status:             status,
			LastTransitionTime: &now,
		}
		vnode.Status.Conditions = append(vnode.Status.Conditions, newCondition)
	} else if condition.Status == status {
		// Already is set to status
		return nil, nil
	} else {
		// condition exists and not set to status
		condition.Status = status
		condition.LastTransitionTime = &now
	}

	return c.meshclientset.AppmeshV1beta1().VirtualNodes(vnode.Namespace).UpdateStatus(vnode)
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
	if desired.Spec.ServiceDiscovery != nil &&
		desired.Spec.ServiceDiscovery.Dns != nil {
		// If Service discovery is desired, verify target is equal
		if desired.Spec.ServiceDiscovery.Dns.HostName != target.HostName() {
			return true
		}
	} else {
		// If no desired Service Discovery, verify target is not set
		if target.HostName() != "" {
			return true
		}
	}

	if desired.Spec.Listeners != nil {
		desiredSet := set.NewSet()
		for i := range desired.Spec.Listeners {
			desiredSet.Add(desired.Spec.Listeners[i])
		}
		currSet := target.ListenersSet()
		if !desiredSet.Equal(currSet) {
			return true
		}
	} else {
		// If the spec doesn't have any listeners, make sure target is not set
		if len(target.Listeners()) != 0 {
			return true
		}
	}

	// This needs to be updated since AppMesh VN name isn't the same as k8s VN name.
	if desired.Spec.Backends != nil {
		desiredSet := set.NewSet()
		for i := range desired.Spec.Backends {
			desiredSet.Add(desired.Spec.Backends[i])
		}
		currSet := target.BackendsSet()
		if !desiredSet.Equal(currSet) {
			return true
		}
	} else {
		// If the spec doesn't have any backends, make sure target is not set
		if len(target.Backends()) != 0 {
			return true
		}
	}

	if desired.Spec.Logging != nil {
		if target.Data.Spec.Logging == nil {
			return true
		}
		if desired.Spec.Logging.AccessLog != nil {
			if target.Data.Spec.Logging.AccessLog == nil {
				return true
			}
			if desired.Spec.Logging.AccessLog.File != nil {
				if target.Data.Spec.Logging.AccessLog.File == nil {
					return true
				}
				if len(desired.Spec.Logging.AccessLog.File.Path) > 0 {
					if target.Data.Spec.Logging.AccessLog.File.Path == nil {
						return true
					}
					if desired.Spec.Logging.AccessLog.File.Path != awssdk.StringValue(target.Data.Spec.Logging.AccessLog.File.Path) {
						return true
					}
				}
			}
		}
	} else if target.Data.Spec.Logging != nil {
		return true
	}

	return false
}

func (c *Controller) handleVNodeDelete(ctx context.Context, vnode *appmeshv1beta1.VirtualNode, copy *appmeshv1beta1.VirtualNode) error {
	if yes, _ := containsFinalizer(vnode, virtualNodeDeletionFinalizerName); yes {
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
		if _, err := c.cloud.DeleteVirtualNode(ctx, vnode.Name, vnode.Spec.MeshName); err != nil {
			if aws.IsAWSErrNotFound(err) {
				klog.Infof("virtual node %s not found", vnode.Name)
			} else {
				klog.Errorf("failed to clean up virtual node %s during mesh deletion: %s", vnode.Name, err)
			}
		} else {
			klog.Infof("Deleted virtual node %s because mesh %s is being deleted", vnode.Name, vnode.Spec.MeshName)
		}
		return false
	}
	return true
}
