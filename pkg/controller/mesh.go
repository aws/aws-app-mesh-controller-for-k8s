package controller

import (
	"context"
	"fmt"

	appmeshv1alpha1 "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/apis/appmesh/v1alpha1"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws"
	api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
)

func (c *Controller) handleMesh(key string) error {
	ctx := context.Background()

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}
	shared, err := c.meshLister.Meshes(namespace).Get(name)
	if errors.IsNotFound(err) {
		klog.V(2).Infof("Mesh %v has been deleted", key)

		// TODO: cleanup appmesh and cloudmap resources

		return nil
	}
	if err != nil {
		return err
	}

	// Make copy here so we never update the shared copy
	mesh := shared.DeepCopy()

	// Initialize status if empty
	if err = c.initMeshStatus(mesh); err != nil {
		return fmt.Errorf("error updating mesh status: %s", err)
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
		if err := c.updateMeshActive(mesh); err != nil {
			return fmt.Errorf("error updating mesh status: %s", err)
		}
	}

	// TODO: Check if cloudmap namespace exists

	return nil
}

func (c *Controller) updateMeshActive(mesh *appmeshv1alpha1.Mesh) error {
	return c.updateMeshCondition(mesh, appmeshv1alpha1.MeshActive, api.ConditionTrue)
}

func (c *Controller) updateMeshCondition(mesh *appmeshv1alpha1.Mesh, conditionType appmeshv1alpha1.MeshConditionType, status api.ConditionStatus) error {
	now := metav1.Now()

	condition := getMeshCondition(conditionType, mesh.Status)

	if condition == nil {
		// condition does not exist
		newCondition := appmeshv1alpha1.MeshCondition{
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

	_, err := c.meshclientset.AppmeshV1alpha1().Meshes(mesh.Namespace).UpdateStatus(mesh)
	return err
}

func (c *Controller) initMeshStatus(mesh *appmeshv1alpha1.Mesh) error {
	if mesh.Status == nil {
		mesh.Status = &appmeshv1alpha1.MeshStatus{
			Conditions: []appmeshv1alpha1.MeshCondition{},
		}
		_, err := c.meshclientset.AppmeshV1alpha1().Meshes(mesh.Namespace).UpdateStatus(mesh)
		return err
	}
	return nil
}

func checkMeshActive(mesh *appmeshv1alpha1.Mesh) bool {
	condition := getMeshCondition(appmeshv1alpha1.MeshActive, mesh.Status)
	return condition != nil && condition.Status == api.ConditionTrue
}

func getMeshCondition(conditionType appmeshv1alpha1.MeshConditionType, status *appmeshv1alpha1.MeshStatus) *appmeshv1alpha1.MeshCondition {
	for _, condition := range status.Conditions {
		if condition.Type == conditionType {
			return &condition
		}
	}
	return nil
}
