package controller

import (
	"context"
	"fmt"
	"strings"

	appmeshv1alpha1 "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/apis/appmesh/v1alpha1"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws"
	set "github.com/deckarep/golang-set"
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

		// TODO(nic) cleanup VirtualService

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

	// Extract namespace from Mesh name
	meshNamespace := namespace
	meshParts := strings.Split(meshName, ".")
	if len(meshParts) > 1 {
		meshNamespace = strings.Join(meshParts[1:], ".")
		meshName = meshParts[0]
		vservice.Spec.MeshName = meshParts[0]
	}

	mesh, err := c.meshLister.Meshes(meshNamespace).Get(meshName)
	if errors.IsNotFound(err) {
		return fmt.Errorf("mesh %s for virtual service %s does not exist", meshName, name)
	}

	if !checkMeshActive(mesh) {
		return fmt.Errorf("mesh %s must be active for virtual service %s", meshName, name)
	}

	virtualRouter := getVirtualRouter(vservice)

	// Create virtual router if it does not exist
	if targetRouter, err := c.cloud.GetVirtualRouter(ctx, virtualRouter.Name, meshName); err != nil {
		if aws.IsAWSErrNotFound(err) {
			if targetRouter, err = c.cloud.CreateVirtualRouter(ctx, virtualRouter, meshName); err != nil {
				return fmt.Errorf("error creating virtual router: %s", err)
			}
			klog.Infof("Created virtual router %s", targetRouter.Name())
		} else {
			return fmt.Errorf("error describing virtual router: %s", err)
		}
	}

	desiredRoutes := getRoutes(vservice)
	existingRoutes, err := c.cloud.GetRoutesForVirtualRouter(ctx, virtualRouter.Name, meshName)
	if err = c.updateRoutes(ctx, meshName, virtualRouter.Name, desiredRoutes, existingRoutes); err != nil {
		return fmt.Errorf("error updating routes for service %s: %s", vservice.Name, err)
	}

	// Create virtual service if it does not exist
	if targetService, err := c.cloud.GetVirtualService(ctx, vservice.Name, meshName); err != nil {
		if aws.IsAWSErrNotFound(err) {
			if targetService, err = c.cloud.CreateVirtualService(ctx, vservice); err != nil {
				return fmt.Errorf("error creating virtual service: %s", err)
			}
			klog.Infof("Created virtual service %s", targetService.Name())
		} else {
			return fmt.Errorf("error describing virtual service: %s", err)
		}
	} else {
		if vserviceNeedsUpdate(vservice, targetService) {
			if targetService, err = c.cloud.UpdateVirtualService(ctx, vservice); err != nil {
				return fmt.Errorf("error updating virtual service: %s", err)
			}
			klog.Infof("Updated virtual service %s", vservice.Name)
		}
	}

	// TODO(nic) Need to determine if we need to clean up the old router here.  This needs to happen if we switched
	// routers for the service.  For now, the old router will be orphaned if the user changes a router name.

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

// vserviceNeedsUpdate compares the App Mesh API result (target) with the desired spec (desired) and
// determines if there is any drift that requires an update.
func vserviceNeedsUpdate(desired *appmeshv1alpha1.VirtualService, target *aws.VirtualService) bool {
	if desired.Spec.VirtualRouter != nil {
		// If we specify the virtual router name, verify the target is equal
		if desired.Spec.VirtualRouter.Name != target.VirtualRouterName() {
			return true
		}
	} else {
		// If no desired virtual router name, verify target is not set
		if target.VirtualRouterName() != "" {
			return true
		}
	}
	return false
}

func (c *Controller) updateRoutes(ctx context.Context, meshName string, routerName string, desired []appmeshv1alpha1.Route, existing aws.Routes) error {
	routeNamesWithErrors := []string{}
	existingNames := existing.RouteNamesSet()
	desiredNames := set.NewSet()

	for _, d := range desired {
		desiredNames.Add(d.Name)
	}

	for _, d := range desired {
		if existingNames.Contains(d.Name) {
			// There exists a route by the desired name, check if it needs to be updated
			e := existing.RouteByName(d.Name)
			if routeNeedsUpdate(d, e) {
				if _, err := c.cloud.UpdateRoute(ctx, &d, routerName, meshName); err != nil {
					routeNamesWithErrors = append(routeNamesWithErrors, d.Name)
				}
			}
		} else {
			// Create route because no existing route exists by the desired name
			if _, err := c.cloud.CreateRoute(ctx, &d, routerName, meshName); err != nil {
				routeNamesWithErrors = append(routeNamesWithErrors, d.Name)
			}
		}
	}

	for _, ex := range existing {
		if !desiredNames.Contains(ex.Name()) {
			if _, err := c.cloud.DeleteRoute(ctx, ex.Name(), routerName, meshName); err != nil {
				routeNamesWithErrors = append(routeNamesWithErrors, ex.Name())
			}
		}
	}
	if len(routeNamesWithErrors) > 0 {
		return fmt.Errorf("error updating routes: %s", strings.Join(routeNamesWithErrors, " "))
	}
	return nil
}

func routeNeedsUpdate(desired appmeshv1alpha1.Route, target aws.Route) bool {
	if desired.Http.Action.WeightedTargets != nil {
		desiredSet := set.NewSet()
		for _, target := range desired.Http.Action.WeightedTargets {
			desiredSet.Add(target)
		}
		currSet := target.WeightedTargetSet()
		if !desiredSet.Equal(currSet) {
			return true
		}
	}
	if desired.Http.Match.Prefix != target.Prefix() {
		return true
	}
	return false
}
