package controller

import (
	"context"
	"fmt"
	appmeshv1alpha1 "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/apis/appmesh/v1alpha1"
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

	// Resources with finalizers are not deleted immediately,
	// instead the deletion timestamp is set when a client deletes them.
	if !vnode.DeletionTimestamp.IsZero() {
		// Resource is being deleted, process finalizers
		return c.handleVNodeDelete(ctx, vnode)
	}

	// This is not a delete, add the deletion finalizer if it doesn't exist
	if yes, _ := containsFinalizer(vnode, virtualNodeDeletionFinalizerName); !yes {
		if err := addFinalizer(vnode, virtualNodeDeletionFinalizerName); err != nil {
			return fmt.Errorf("error adding finalizer %s to virtual node %s: %s", virtualNodeDeletionFinalizerName, vnode.Name, err)
		}
		if err := c.updateVNodeResource(vnode); err != nil {
			return fmt.Errorf("error adding finalizer %s to virtual node %s: %s", virtualNodeDeletionFinalizerName, vnode.Name, err)
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

	// Extract namespace from Mesh name
	meshName, meshNamespace := parseMeshName(meshName, vnode.Namespace)

	mesh, err := c.meshLister.Meshes(meshNamespace).Get(meshName)
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

	updated, err := c.updateVNodeStatus(vnode, targetNode)
	if err != nil {
		return fmt.Errorf("error updating virtual service status: %s", err)
	} else if updated != nil {
		vnode = updated
	}

	return nil
}

func (c *Controller) updateVNodeResource(vnode *appmeshv1alpha1.VirtualNode) error {
	_, err := c.meshclientset.AppmeshV1alpha1().VirtualNodes(vnode.Namespace).Update(vnode)
	return err
}

func (c *Controller) updateVNodeStatus(vnode *appmeshv1alpha1.VirtualNode, target *aws.VirtualNode) (*appmeshv1alpha1.VirtualNode, error) {
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

func (c *Controller) updateVNodeActive(vnode *appmeshv1alpha1.VirtualNode, status api.ConditionStatus) (*appmeshv1alpha1.VirtualNode, error) {
	return c.updateVNodeCondition(vnode, appmeshv1alpha1.VirtualNodeActive, status)
}

func (c *Controller) updateVNodeCondition(vnode *appmeshv1alpha1.VirtualNode, conditionType appmeshv1alpha1.VirtualNodeConditionType, status api.ConditionStatus) (*appmeshv1alpha1.VirtualNode, error) {
	now := metav1.Now()
	condition := getVNodeCondition(conditionType, vnode.Status)
	if condition == (appmeshv1alpha1.VirtualNodeCondition{}) {
		// condition does not exist
		newCondition := appmeshv1alpha1.VirtualNodeCondition{
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

	return c.meshclientset.AppmeshV1alpha1().VirtualNodes(vnode.Namespace).UpdateStatus(vnode)
}

func getVNodeCondition(conditionType appmeshv1alpha1.VirtualNodeConditionType, status appmeshv1alpha1.VirtualNodeStatus) appmeshv1alpha1.VirtualNodeCondition {
	for _, condition := range status.Conditions {
		if condition.Type == conditionType {
			return condition
		}
	}

	return appmeshv1alpha1.VirtualNodeCondition{}
}

// vnodeNeedsUpdate compares the App Mesh API result (target) with the desired spec (desired) and
// determines if there is any drift that requires an update.
func vnodeNeedsUpdate(desired *appmeshv1alpha1.VirtualNode, target *aws.VirtualNode) bool {
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
	return false
}

func (c *Controller) handleVNodeDelete(ctx context.Context, vnode *appmeshv1alpha1.VirtualNode) error {
	if yes, _ := containsFinalizer(vnode, virtualNodeDeletionFinalizerName); yes {
		if _, err := c.cloud.DeleteVirtualNode(ctx, vnode.Name, vnode.Spec.MeshName); err != nil {
			if !aws.IsAWSErrNotFound(err) {
				return fmt.Errorf("failed to clean up virtual node %s during deletion finalizer: %s", vnode.Name, err)
			}
		}
		if err := removeFinalizer(vnode, virtualNodeDeletionFinalizerName); err != nil {
			return fmt.Errorf("error removing finalizer %s to virtual node %s during deletion: %s", virtualNodeDeletionFinalizerName, vnode.Name, err)
		}
		if err := c.updateVNodeResource(vnode); err != nil {
			return fmt.Errorf("error removing finalizer %s to virtual node %s during deletion: %s", virtualNodeDeletionFinalizerName, vnode.Name, err)
		}
	}
	return nil
}

func (c *Controller) handleVNodeMeshDeleting(ctx context.Context, vnode *appmeshv1alpha1.VirtualNode) (processVNode bool) {
	meshName, meshNamespace := parseMeshName(vnode.Spec.MeshName, vnode.Namespace)
	mesh, err := c.meshLister.Meshes(meshNamespace).Get(meshName)

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
