package controller

import (
	"context"
	"fmt"

	appmeshv1alpha1 "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/apis/appmesh/v1alpha1"
	"github.com/aws/aws-sdk-go/aws"
	api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
)

func (c *Controller) handleVService(key string) error {
	ctx := context.Background()

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}
	shared, err := c.virtualServiceLister.VirtualServices(namespace).Get(name)
	if errors.IsNotFound(err) {
		klog.V(2).Infof("Virtual service %s has been deleted", key)

		// TODO(nic) cleanup VirtualNode

		return nil
	}
	if err != nil {
		return err
	}

	// Make copy here so we never update the shared copy
	vservice := shared.DeepCopy()

	// Initialize status if empty
	if err = c.initVServiceStatus(vservice); err != nil {
		return fmt.Errorf("error updating virtual service status: %s", err)
	}

	// Get Mesh for virtual service
	meshName := vservice.Spec.MeshName
	if vservice.Spec.MeshName == "" {
		return fmt.Errorf("'MeshName' is a required field")
	}

	mesh, err := c.meshLister.Meshes(namespace).Get(meshName)
	if errors.IsNotFound(err) {
		return fmt.Errorf("mesh %s for virtual service %s does not exist", meshName, name)
	}

	if !checkMeshActive(mesh) {
		return fmt.Errorf("mesh %s must be active for virtual service %s", meshName, name)
	}

	virtualRouter := getVirtualRouter(vservice)

	// Check if virtual router already exists
	targetRouter, err := c.cloud.GetVirtualRouter(ctx, virtualRouter.Name, meshName)

	if err != nil {
		return fmt.Errorf("error describing virtual router: %s", err)
	} else if targetRouter == nil {

		// Create virtual router if it doesn't exist
		targetRouter, err = c.cloud.CreateVirtualRouter(ctx, virtualRouter, meshName)
		if err != nil {
			return fmt.Errorf("error creating virtual router: %s", err)
		}
		klog.Infof("Created virtual router %s", aws.StringValue(targetRouter.Data.VirtualRouterName))
	} else {
		klog.Infof("Discovered virtual router %s", aws.StringValue(targetRouter.Data.VirtualRouterName))
	}

	// Check if virtual service already exists
	targetService, err := c.cloud.GetVirtualService(ctx, vservice.Name, meshName)

	if err != nil {
		return fmt.Errorf("error describing virtual service: %s", err)
	} else if targetService == nil {

		// Create virtual service if it doesn't exist
		targetService, err = c.cloud.CreateVirtualService(ctx, vservice)
		if err != nil {
			return fmt.Errorf("error creating virtual service: %s", err)
		}
		klog.Infof("Created virtual service %s", aws.StringValue(targetService.Data.VirtualServiceName))
	} else {
		klog.Infof("Discovered virtual service %s", aws.StringValue(targetService.Data.VirtualServiceName))
	}

	routes := getRoutes(vservice)
	for _, route := range routes {
		// Check if route already exists
		targetRoute, err := c.cloud.GetRoute(ctx, route.Name, virtualRouter.Name, meshName)

		if err != nil {
			return fmt.Errorf("error describing route: %s", err)
		} else if targetRoute == nil {

			// Create route if it doesn't exist
			targetRoute, err = c.cloud.CreateRoute(ctx, &route, virtualRouter.Name, meshName)
			if err != nil {
				return fmt.Errorf("error creating route: %s", err)
			}
			klog.Infof("Created route %s", aws.StringValue(targetRoute.Data.RouteName))
		} else {
			klog.Infof("Discovered route %s", aws.StringValue(targetRoute.Data.RouteName))
		}
	}

	return nil
}

func (c *Controller) updateVServiceActive(vservice *appmeshv1alpha1.VirtualService) error {
	return c.updateVServiceCondition(vservice, appmeshv1alpha1.VirtualServiceActive, api.ConditionTrue)
}

func (c *Controller) updateVServiceCondition(vservice *appmeshv1alpha1.VirtualService, conditionType appmeshv1alpha1.VirtualServiceConditionType, status api.ConditionStatus) error {
	now := metav1.Now()

	condition := getVServiceCondition(conditionType, vservice.Status)

	if condition == nil {
		// condition does not exist
		newCondition := appmeshv1alpha1.VirtualServiceCondition{
			Type:               conditionType,
			Status:             status,
			LastTransitionTime: &now,
		}
		vservice.Status.Conditions = append(vservice.Status.Conditions, newCondition)
	} else if condition.Status == status {
		// Already is set to status
		return nil
	} else {
		// condition exists and not set to status
		condition.Status = status
		condition.LastTransitionTime = &now
	}

	_, err := c.meshclientset.AppmeshV1alpha1().VirtualServices(vservice.Namespace).UpdateStatus(vservice)
	return err
}

func (c *Controller) initVServiceStatus(vservice *appmeshv1alpha1.VirtualService) error {
	if vservice.Status == nil {
		vservice.Status = &appmeshv1alpha1.VirtualServiceStatus{
			Conditions: []appmeshv1alpha1.VirtualServiceCondition{},
		}
		_, err := c.meshclientset.AppmeshV1alpha1().VirtualServices(vservice.Namespace).UpdateStatus(vservice)
		return err
	}
	return nil
}

func checkVServiceActive(vservice *appmeshv1alpha1.VirtualService) bool {
	condition := getVServiceCondition(appmeshv1alpha1.VirtualServiceActive, vservice.Status)
	return condition != nil && condition.Status == api.ConditionTrue
}

func getVServiceCondition(conditionType appmeshv1alpha1.VirtualServiceConditionType, status *appmeshv1alpha1.VirtualServiceStatus) *appmeshv1alpha1.VirtualServiceCondition {
	if status != nil {
		for _, condition := range status.Conditions {
			if condition.Type == conditionType {
				return &condition
			}
		}
	}
	return nil
}

func getVirtualRouter(vservice *appmeshv1alpha1.VirtualService) *appmeshv1alpha1.VirtualRouter {
	if vservice.Spec.VirtualRouter != nil {
		return vservice.Spec.VirtualRouter
	}
	return &appmeshv1alpha1.VirtualRouter{
		Name: vservice.Name,
	}
}

func getRoutes(vservice *appmeshv1alpha1.VirtualService) []appmeshv1alpha1.Route {
	if vservice.Spec.Routes != nil {
		return vservice.Spec.Routes
	}
	return []appmeshv1alpha1.Route{}
}
