package controller

import (
	"context"
	"fmt"

	appmeshv1beta1 "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/apis/appmesh/v1beta1"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws"
	awssdk "github.com/aws/aws-sdk-go/aws"
	api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
)

func (c *Controller) handleMesh(key string) error {
	ctx := context.Background()

	_, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}
	shared, err := c.meshLister.Get(name)
	if errors.IsNotFound(err) {
		klog.V(2).Infof("Mesh %v has been deleted", key)
		return nil
	}
	if err != nil {
		return err
	}

	// Make copy here so we never update the shared copy
	mesh := shared.DeepCopy()

	// Resources with finalizers are not deleted immediately,
	// instead the deletion timestamp is set when a client deletes them.
	if !mesh.DeletionTimestamp.IsZero() {
		c.stats.SetMeshInactive(mesh.Name)
		// Resource is being deleted, process finalizers
		return c.handleMeshDelete(ctx, mesh)
	}

	// This is not a delete, add the deletion finalizer if it doesn't exist
	if yes, _ := containsFinalizer(mesh, meshDeletionFinalizerName); !yes {
		if err = addFinalizer(mesh, meshDeletionFinalizerName); err != nil {
			return fmt.Errorf("error adding finalizer %s to mesh %s: %s", meshDeletionFinalizerName, mesh.Name, err)
		}
		if err := c.updateMeshResource(mesh); err != nil {
			return fmt.Errorf("error adding finalizer %s to mesh %s: %s", meshDeletionFinalizerName, mesh.Name, err)
		}
	}

	// Create mesh if it does not exist
	if targetMesh, err := c.cloud.GetMesh(ctx, mesh.Name); err != nil {
		if aws.IsAWSErrNotFound(err) {
			if targetMesh, err = c.cloud.CreateMesh(ctx, mesh); err != nil {
				return fmt.Errorf("error creating mesh: %s", err)
			}
			klog.Infof("Created mesh %s", targetMesh.Name())
		} else {
			return fmt.Errorf("error describing mesh: %s", err)
		}
	} else {
		if c.meshNeedsUpdate(mesh, targetMesh) {
			if targetMesh, err = c.cloud.UpdateMesh(ctx, mesh); err != nil {
				return fmt.Errorf("error updating mesh: %s", err)
			}
			klog.Infof("Updated mesh %s", mesh.Name)
		}
		if err := c.updateMeshActive(mesh); err != nil {
			return fmt.Errorf("error updating mesh status: %s", err)
		}
	}

	c.stats.SetMeshActive(mesh.Name)

	return nil
}

func (c *Controller) meshNeedsUpdate(desired *appmeshv1beta1.Mesh, target *aws.Mesh) bool {
	if desired.Spec.EgressFilter != nil {
		if target.Data.Spec.EgressFilter == nil {
			return true
		}
		if desired.Spec.EgressFilter.Type != awssdk.StringValue(target.Data.Spec.EgressFilter.Type) {
			return true
		}
	} else if target.Data.Spec.EgressFilter != nil {
		return true
	}

	return false
}

func (c *Controller) updateMeshResource(mesh *appmeshv1beta1.Mesh) error {
	_, err := c.meshclientset.AppmeshV1beta1().Meshes().Update(mesh)
	return err
}

func (c *Controller) updateMeshActive(mesh *appmeshv1beta1.Mesh) error {
	return c.updateMeshCondition(mesh, appmeshv1beta1.MeshActive, api.ConditionTrue)
}

func (c *Controller) updateMeshCondition(mesh *appmeshv1beta1.Mesh, conditionType appmeshv1beta1.MeshConditionType, status api.ConditionStatus) error {
	now := metav1.Now()
	condition := getMeshCondition(conditionType, mesh.Status)
	if condition == (appmeshv1beta1.MeshCondition{}) {
		// condition does not exist
		newCondition := appmeshv1beta1.MeshCondition{
			Type:               conditionType,
			Status:             status,
			LastTransitionTime: &now,
		}
		mesh.Status.Conditions = append(mesh.Status.Conditions, newCondition)
	} else if condition.Status == status {
		// Already is set to status
		return nil
	} else {
		// condition exists and not set to status
		condition.Status = status
		condition.LastTransitionTime = &now
	}

	_, err := c.meshclientset.AppmeshV1beta1().Meshes().UpdateStatus(mesh)
	return err
}

func checkMeshActive(mesh *appmeshv1beta1.Mesh) bool {
	condition := getMeshCondition(appmeshv1beta1.MeshActive, mesh.Status)
	return condition.Status == api.ConditionTrue
}

func getMeshCondition(conditionType appmeshv1beta1.MeshConditionType, status appmeshv1beta1.MeshStatus) appmeshv1beta1.MeshCondition {

	for _, condition := range status.Conditions {
		if condition.Type == conditionType {
			return condition
		}
	}

	return appmeshv1beta1.MeshCondition{}
}

func (c *Controller) handleMeshDelete(ctx context.Context, mesh *appmeshv1beta1.Mesh) error {
	if yes, _ := containsFinalizer(mesh, meshDeletionFinalizerName); yes {

		if err := c.markResourcesForMeshDeletion(mesh.Name); err != nil {
			// Log, but we will still attempt to delete the mesh
			klog.Error(err)
		}

		if _, err := c.cloud.DeleteMesh(ctx, mesh.Name); err != nil {
			if !aws.IsAWSErrNotFound(err) {
				// Don't remove the finalizer if the mesh still exists
				return fmt.Errorf("failed to clean up mesh %s during deletion finalizer: %s", mesh.Name, err)
			}
		}
		if err := removeFinalizer(mesh, meshDeletionFinalizerName); err != nil {
			return fmt.Errorf("error removing finalizer %s to mesh %s during deletion: %s", meshDeletionFinalizerName, mesh.Name, err)
		}
		if err := c.updateMeshResource(mesh); err != nil {
			return fmt.Errorf("error removing finalizer %s to mesh %s during deletion: %s", meshDeletionFinalizerName, mesh.Name, err)
		}
	}
	return nil
}

func (c *Controller) markResourcesForMeshDeletion(name string) error {
	wasError := false

	if objects, err := c.virtualNodeIndex.ByIndex("meshName", name); err != nil {
		return fmt.Errorf("meshName index error for %s: %s", name, err)
	} else {
		for _, obj := range objects {
			vnode, ok := obj.(*appmeshv1beta1.VirtualNode)
			if !ok {
				continue
			}

			if _, err := c.updateVNodeCondition(vnode, appmeshv1beta1.VirtualNodeMeshMarkedForDeletion, api.ConditionTrue); err != nil {
				klog.Errorf("Error marking node service %s for mesh deletion: %s", vnode.Name, err)
				wasError = true
				continue
			}
			klog.Infof("Marked virtual node for mesh deletion: %s", vnode.Name)
		}
		klog.Infof("Marked virtual nodes for mesh deletion")
	}

	if objects, err := c.virtualServiceIndex.ByIndex("meshName", name); err != nil {
		return fmt.Errorf("meshName index error for %s: %s", name, err)
	} else {
		for _, obj := range objects {
			vservice, ok := obj.(*appmeshv1beta1.VirtualService)
			if !ok {
				continue
			}

			if _, err := c.updateVServiceCondition(vservice, appmeshv1beta1.VirtualServiceMeshMarkedForDeletion, api.ConditionTrue); err != nil {
				klog.Errorf("Error marking virtual service %s for mesh deletion: %s", vservice.Name, err)
				wasError = true
				continue
			}
			klog.Infof("Marked virtual service for mesh deletion: %s", vservice.Name)
		}
		klog.Infof("Marked virtual services for mesh deletion")
	}

	if wasError {
		return fmt.Errorf("error marking resources for mesh deletion")
	}
	return nil
}
