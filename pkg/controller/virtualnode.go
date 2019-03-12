package controller

import (
	"context"
	"fmt"
	"strings"

	appmeshv1alpha1 "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/apis/appmesh/v1alpha1"
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

		// TODO(nic) cleanup VirtualNode

		return nil
	}
	if err != nil {
		return err
	}

	// Make copy here so we never update the shared copy
	vnode := shared.DeepCopy()

	// Initialize status if empty
	if err = c.initVNodeStatus(vnode); err != nil {
		return fmt.Errorf("error updating virtual node status: %s", err)
	}

	// Get Mesh for virtual node
	meshName := vnode.Spec.MeshName
	if vnode.Spec.MeshName == "" {
		return fmt.Errorf("'MeshName' is a required field")
	}

	// Extract namespace from Mesh name
	meshNamespace := namespace
	meshParts := strings.Split(meshName, ".")
	if len(meshParts) > 1 {
		meshNamespace = strings.Join(meshParts[1:], ".")
		meshName = meshParts[0]
	}

	mesh, err := c.meshLister.Meshes(meshNamespace).Get(meshName)
	if errors.IsNotFound(err) {
		return fmt.Errorf("mesh %s for virtual node %s does not exist", meshName, name)
	}

	if !checkMeshActive(mesh) {
		return fmt.Errorf("mesh %s must be active for virtual node %s", meshName, name)
	}

	// Check if virtual node already exists
	targetNode, err := c.cloud.GetVirtualNode(ctx, vnode.Name, meshName)

	if err != nil {
		return fmt.Errorf("error describing virtual node: %s", err)
	} else if targetNode == nil {

		// Create virtual node if it doesn't exist
		targetNode, err = c.cloud.CreateVirtualNode(ctx, vnode)
		if err != nil {
			return fmt.Errorf("error creating virtual node: %s", err)
		}
		klog.Infof("Created virtual node %s", vnode.Name)
	} else {
		klog.Infof("Discovered virtual node %s", vnode.Name)
	}

	return nil
}

func (c *Controller) updateVNodeActive(vnode *appmeshv1alpha1.VirtualNode) error {
	return c.updateVNodeCondition(vnode, appmeshv1alpha1.VirtualNodeActive, api.ConditionTrue)
}

func (c *Controller) updateVNodeCondition(vnode *appmeshv1alpha1.VirtualNode, conditionType appmeshv1alpha1.VirtualNodeConditionType, status api.ConditionStatus) error {
	now := metav1.Now()

	condition := getVNodeCondition(conditionType, vnode.Status)

	if condition == nil {
		// condition does not exist
		newCondition := appmeshv1alpha1.VirtualNodeCondition{
			Type:               conditionType,
			Status:             status,
			LastTransitionTime: &now,
		}
		vnode.Status.Conditions = append(vnode.Status.Conditions, newCondition)
	} else if condition.Status == status {
		// Already is set to status
		return nil
	} else {
		// condition exists and not set to status
		condition.Status = status
		condition.LastTransitionTime = &now
	}

	_, err := c.meshclientset.AppmeshV1alpha1().VirtualNodes(vnode.Namespace).UpdateStatus(vnode)
	return err
}

func (c *Controller) initVNodeStatus(vnode *appmeshv1alpha1.VirtualNode) error {
	if vnode.Status == nil {
		vnode.Status = &appmeshv1alpha1.VirtualNodeStatus{
			Conditions: []appmeshv1alpha1.VirtualNodeCondition{},
		}
		_, err := c.meshclientset.AppmeshV1alpha1().VirtualNodes(vnode.Namespace).UpdateStatus(vnode)
		return err
	}
	return nil
}

func checkVNodeActive(vnode *appmeshv1alpha1.VirtualNode) bool {
	condition := getVNodeCondition(appmeshv1alpha1.VirtualNodeActive, vnode.Status)
	return condition != nil && condition.Status == api.ConditionTrue
}

func getVNodeCondition(conditionType appmeshv1alpha1.VirtualNodeConditionType, status *appmeshv1alpha1.VirtualNodeStatus) *appmeshv1alpha1.VirtualNodeCondition {
	if status != nil {
		for _, condition := range status.Conditions {
			if condition.Type == conditionType {
				return &condition
			}
		}
	}
	return nil
}
