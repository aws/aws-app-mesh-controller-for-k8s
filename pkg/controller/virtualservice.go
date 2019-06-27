package controller

import (
	"context"
	"fmt"
	"strings"

	appmeshv1beta1 "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/apis/appmesh/v1beta1"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws"
	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/appmesh"
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
		return nil
	}
	if err != nil {
		return err
	}

	// Make copy here so we never update the shared copy
	vservice := shared.DeepCopy()
	// Make copy for updates so we don't save namespaced resource names
	copy := shared.DeepCopy()
	copy.Spec.VirtualRouter = getVirtualRouter(copy)

	// Namespace resource names for use against App Mesh API
	if vservice.Spec.VirtualRouter == nil {
		vservice.Spec.VirtualRouter = &appmeshv1beta1.VirtualRouter{
			Name: getNamespacedVirtualRouterName(vservice),
		}
	} else {
		vservice.Spec.VirtualRouter.Name = getNamespacedVirtualRouterName(vservice)
	}

	for i := range vservice.Spec.Routes {
		route := vservice.Spec.Routes[i]
		route.Name = namespacedResourceName(route.Name, vservice.Namespace)
		var targets []appmeshv1beta1.WeightedTarget
		if route.Http != nil {
			targets = route.Http.Action.WeightedTargets
		} else if route.Tcp != nil {
			targets = route.Tcp.Action.WeightedTargets
		}
		for j := range targets {
			targets[j].VirtualNodeName = namespacedResourceName(targets[j].VirtualNodeName, vservice.Namespace)
		}
	}

	// Add the deletion finalizer if it doesn't exist
	if yes, _ := containsFinalizer(copy, virtualServiceDeletionFinalizerName); !yes {
		if err := addFinalizer(copy, virtualServiceDeletionFinalizerName); err != nil {
			return fmt.Errorf("error adding finalizer %s to virtual service %s: %s", virtualServiceDeletionFinalizerName, vservice.Name, err)
		}
		if updated, err := c.updateVServiceResource(copy); err != nil {
			return fmt.Errorf("error updating resource while adding finalizer %s to virtual service %s: %s", virtualServiceDeletionFinalizerName, vservice.Name, err)
		} else if updated != nil {
			copy = updated
		}
	}

	// Resources with finalizers are not deleted immediately,
	// instead the deletion timestamp is set when a client deletes them.
	if !vservice.DeletionTimestamp.IsZero() {
		// Resource is being deleted, process finalizers
		return c.handleVServiceDelete(ctx, vservice, copy)
	}

	if processVService := c.handleVServiceMeshDeleting(ctx, vservice); !processVService {
		klog.Infof("skipping processing virtual service %s", vservice.Name)
		return nil
	}

	// Get Mesh for virtual service
	meshName := vservice.Spec.MeshName
	if vservice.Spec.MeshName == "" {
		return fmt.Errorf("'MeshName' is a required field")
	}

	mesh, err := c.meshLister.Get(meshName)
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
		if updated, err := c.updateVRouterStatus(copy, targetRouter); err != nil {
			return fmt.Errorf("error updating virtual service status for virtual router: %s", err)
		} else if updated != nil {
			copy = updated
		}
	} else {
		if vrouterNeedsUpdate(virtualRouter, targetRouter) {
			if targetRouter, err = c.cloud.UpdateVirtualRouter(ctx, virtualRouter, meshName); err != nil {
				return fmt.Errorf("error updating virtual router: %s", err)
			}
			klog.Infof("Updated virtual router %s", virtualRouter.Name)
		}
	}

	desiredRoutes := getRoutes(vservice)
	existingRoutes, err := c.cloud.GetRoutesForVirtualRouter(ctx, virtualRouter.Name, meshName)
	if err != nil {
		return fmt.Errorf("error getting routes for virtual service %s: %s", vservice.Name, err)
	}
	if err = c.updateRoutes(ctx, meshName, virtualRouter.Name, desiredRoutes, existingRoutes); err != nil {
		return fmt.Errorf("error updating routes for virtual service %s: %s", vservice.Name, err)
	}

	routes, err := c.cloud.GetRoutesForVirtualRouter(ctx, virtualRouter.Name, meshName)
	if err != nil {
		klog.Errorf("Unable to check status of routes for virtual router %s: %s", virtualRouter.Name, err)
	} else {
		var status api.ConditionStatus
		if allRoutesActive(routes) {
			status = api.ConditionTrue
		} else {
			status = api.ConditionFalse
		}
		if updated, err := c.updateRoutesActive(copy, status); err != nil {
			return fmt.Errorf("error updating routes status: %s", err)
		} else if updated != nil {
			copy = updated
		}
	}

	// Create virtual service if it does not exist
	targetService, err := c.cloud.GetVirtualService(ctx, vservice.Name, meshName)
	if err != nil {
		if aws.IsAWSErrNotFound(err) {
			if targetService, err = c.cloud.CreateVirtualService(ctx, vservice); err != nil {
				return fmt.Errorf("error creating virtual service: %s", err)
			}
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

	if updated, err := c.updateVServiceStatus(copy, targetService); err != nil {
		return fmt.Errorf("error updating virtual service status: %s", err)
	} else if updated != nil {
		copy = updated
	}

	// TODO(nic) Need to determine if we need to clean up the old router here.  This needs to happen if we switched
	// routers for the service.  For now, the old router will be orphaned if the user changes a router name.

	return nil
}

func (c *Controller) updateVServiceResource(vservice *appmeshv1beta1.VirtualService) (*appmeshv1beta1.VirtualService, error) {
	return c.meshclientset.AppmeshV1beta1().VirtualServices(vservice.Namespace).Update(vservice)
}

func (c *Controller) updateVServiceStatus(vservice *appmeshv1beta1.VirtualService, target *aws.VirtualService) (*appmeshv1beta1.VirtualService, error) {
	switch target.Status() {
	case appmesh.VirtualServiceStatusCodeActive:
		return c.updateVServiceActive(vservice, api.ConditionTrue)
	case appmesh.VirtualServiceStatusCodeInactive:
		return c.updateVServiceActive(vservice, api.ConditionFalse)
	case appmesh.VirtualServiceStatusCodeDeleted:
		return c.updateVServiceActive(vservice, api.ConditionFalse)
	}
	return nil, nil
}

func (c *Controller) updateVRouterStatus(vservice *appmeshv1beta1.VirtualService, target *aws.VirtualRouter) (*appmeshv1beta1.VirtualService, error) {
	switch target.Status() {
	case appmesh.VirtualRouterStatusCodeActive:
		return c.updateVRouterActive(vservice, api.ConditionTrue)
	case appmesh.VirtualRouterStatusCodeInactive:
		return c.updateVRouterActive(vservice, api.ConditionFalse)
	case appmesh.VirtualRouterStatusCodeDeleted:
		return c.updateVRouterActive(vservice, api.ConditionFalse)
	}
	return nil, nil
}

func (c *Controller) updateVServiceActive(vservice *appmeshv1beta1.VirtualService, status api.ConditionStatus) (*appmeshv1beta1.VirtualService, error) {
	return c.updateVServiceCondition(vservice, appmeshv1beta1.VirtualServiceActive, status)
}

func (c *Controller) updateVRouterActive(vservice *appmeshv1beta1.VirtualService, status api.ConditionStatus) (*appmeshv1beta1.VirtualService, error) {
	return c.updateVServiceCondition(vservice, appmeshv1beta1.VirtualRouterActive, status)
}

func (c *Controller) updateRoutesActive(vservice *appmeshv1beta1.VirtualService, status api.ConditionStatus) (*appmeshv1beta1.VirtualService, error) {
	return c.updateVServiceCondition(vservice, appmeshv1beta1.RoutesActive, status)
}

func (c *Controller) updateVServiceCondition(vservice *appmeshv1beta1.VirtualService, conditionType appmeshv1beta1.VirtualServiceConditionType, status api.ConditionStatus) (*appmeshv1beta1.VirtualService, error) {
	now := metav1.Now()
	condition := getVServiceCondition(conditionType, vservice.Status)
	if condition == (appmeshv1beta1.VirtualServiceCondition{}) {
		// condition does not exist
		newCondition := appmeshv1beta1.VirtualServiceCondition{
			Type:               conditionType,
			Status:             status,
			LastTransitionTime: &now,
		}
		vservice.Status.Conditions = append(vservice.Status.Conditions, newCondition)
	} else if condition.Status == status {
		// Already is set to status
		return nil, nil
	} else {
		// condition exists and not set to status
		condition.Status = status
		condition.LastTransitionTime = &now
	}

	return c.meshclientset.AppmeshV1beta1().VirtualServices(vservice.Namespace).UpdateStatus(vservice)
}

func getVServiceCondition(conditionType appmeshv1beta1.VirtualServiceConditionType, status appmeshv1beta1.VirtualServiceStatus) appmeshv1beta1.VirtualServiceCondition {
	for _, condition := range status.Conditions {
		if condition.Type == conditionType {
			return condition
		}
	}

	return appmeshv1beta1.VirtualServiceCondition{}
}

func getVirtualRouter(vservice *appmeshv1beta1.VirtualService) *appmeshv1beta1.VirtualRouter {
	if vservice.Spec.VirtualRouter != nil {
		return vservice.Spec.VirtualRouter
	}
	return &appmeshv1beta1.VirtualRouter{
		Name: vservice.Name,
	}
}

func getNamespacedVirtualRouterName(vservice *appmeshv1beta1.VirtualService) string {
	var name string
	if vservice.Spec.VirtualRouter != nil {
		name = strings.TrimSpace(vservice.Spec.VirtualRouter.Name)
	}
	if len(name) == 0 {
		name = vservice.Name
	}
	return namespacedResourceName(name, vservice.Namespace)
}

func getRoutes(vservice *appmeshv1beta1.VirtualService) []appmeshv1beta1.Route {
	if vservice.Spec.Routes != nil {
		return vservice.Spec.Routes
	}
	return []appmeshv1beta1.Route{}
}

// vserviceNeedsUpdate compares the App Mesh API result (target) with the desired spec (desired) and
// determines if there is any drift that requires an update.
func vserviceNeedsUpdate(desired *appmeshv1beta1.VirtualService, target *aws.VirtualService) bool {
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

func vrouterNeedsUpdate(desired *appmeshv1beta1.VirtualRouter, target *aws.VirtualRouter) bool {
	if desired.Name != target.Name() {
		return true
	}

	if len(desired.Listeners) != len(target.Data.Spec.Listeners) {
		return true
	}

	if len(desired.Listeners) > 0 {
		//there should be only one listener
		desiredListener := desired.Listeners[0]
		targetListener := target.Data.Spec.Listeners[0]
		if desiredListener.PortMapping.Port != awssdk.Int64Value(targetListener.PortMapping.Port) {
			return true
		}

		if desiredListener.PortMapping.Protocol != awssdk.StringValue(targetListener.PortMapping.Protocol) {
			return true
		}
	}
	return false
}

func (c *Controller) updateRoutes(ctx context.Context, meshName string, routerName string, desired []appmeshv1beta1.Route, existing aws.Routes) error {
	routeNamesWithErrors := []string{}
	existingNames := existing.RouteNamesSet()
	desiredNames := set.NewSet()

	for _, d := range desired {
		desiredNames.Add(d.Name)
	}

	// Needs update for virtual node name convention
	for _, d := range desired {
		if existingNames.Contains(d.Name) {
			// There exists a route by the desired name, check if it needs to be updated
			e := existing.RouteByName(d.Name)
			if routeNeedsUpdate(d, e) {
				if _, err := c.cloud.UpdateRoute(ctx, &d, routerName, meshName); err != nil {
					routeNamesWithErrors = append(routeNamesWithErrors, d.Name)
					klog.Errorf("Error updating route %s: %s", d.Name, err)
				}
			}
		} else {
			// Create route because no existing route exists by the desired name
			if _, err := c.cloud.CreateRoute(ctx, &d, routerName, meshName); err != nil {
				routeNamesWithErrors = append(routeNamesWithErrors, d.Name)
				klog.Errorf("Error creating route %s: %s", d.Name, err)
			}
		}
	}

	for _, ex := range existing {
		if !desiredNames.Contains(ex.Name()) {
			if _, err := c.cloud.DeleteRoute(ctx, ex.Name(), routerName, meshName); err != nil {
				routeNamesWithErrors = append(routeNamesWithErrors, ex.Name())
				klog.Errorf("Error deleting route %s: %s", ex.Name(), err)
			}
		}
	}

	if len(routeNamesWithErrors) > 0 {
		return fmt.Errorf("error updating routes: %s", strings.Join(routeNamesWithErrors, " "))
	}
	return nil
}

func allRoutesActive(routes aws.Routes) bool {
	for _, r := range routes {
		if r.Status() != appmesh.RouteStatusCodeActive {
			return false
		}
	}
	return true
}

func routeNeedsUpdate(desired appmeshv1beta1.Route, target aws.Route) bool {
	if desired.Http != nil {
		if desired.Http.Action.WeightedTargets != nil {
			desiredSet := set.NewSet()
			for _, target := range desired.Http.Action.WeightedTargets {
				desiredSet.Add(appmeshv1beta1.WeightedTarget{VirtualNodeName: target.VirtualNodeName, Weight: target.Weight})
			}
			currSet := target.WeightedTargetSet()
			if !desiredSet.Equal(currSet) {
				return true
			}
		}
		if desired.Http.Match.Prefix != target.Prefix() {
			return true
		}
	}

	if desired.Tcp != nil {
		if desired.Tcp.Action.WeightedTargets != nil {
			desiredSet := set.NewSet()
			for _, target := range desired.Tcp.Action.WeightedTargets {
				desiredSet.Add(appmeshv1beta1.WeightedTarget{VirtualNodeName: target.VirtualNodeName, Weight: target.Weight})
			}
			currSet := target.WeightedTargetSet()
			if !desiredSet.Equal(currSet) {
				return true
			}
		}
	}
	return false
}

func (c *Controller) handleVServiceDelete(ctx context.Context, vservice *appmeshv1beta1.VirtualService, copy *appmeshv1beta1.VirtualService) error {
	if yes, _ := containsFinalizer(vservice, virtualServiceDeletionFinalizerName); yes {

		if err := c.deleteVServiceResources(ctx, vservice); err != nil {
			return err
		}

		if err := removeFinalizer(copy, virtualServiceDeletionFinalizerName); err != nil {
			return fmt.Errorf("error removing finalizer %s to virtual service %s during deletion: %s", virtualServiceDeletionFinalizerName, vservice.Name, err)
		}
		if _, err := c.updateVServiceResource(copy); err != nil {
			return fmt.Errorf("error removing finalizer %s to virtual service %s during deletion: %s", virtualServiceDeletionFinalizerName, vservice.Name, err)
		}
	}
	return nil
}

func (c *Controller) handleVServiceMeshDeleting(ctx context.Context, vservice *appmeshv1beta1.VirtualService) (processVService bool) {
	mesh, err := c.meshLister.Get(vservice.Spec.MeshName)

	if err != nil {
		if errors.IsNotFound(err) {
			// If mesh doesn't exist, do nothing
			klog.Infof("mesh doesn't exist, skipping processing virtual service %s", vservice.Name)
		} else {
			klog.Errorf("error getting mesh: %s", err)
		}
		return false
	}

	// if mesh DeletionTimestamp is set, clean up virtual service via App Mesh API
	if !mesh.DeletionTimestamp.IsZero() {
		if err := c.deleteVServiceResources(ctx, vservice); err != nil {
			klog.Error(err)
		} else {
			klog.Infof("Deleted resources for virtual service %s because mesh %s is being deleted", vservice.Name, vservice.Spec.MeshName)
		}
		return false
	}
	return true
}

func (c *Controller) deleteVServiceResources(ctx context.Context, vservice *appmeshv1beta1.VirtualService) error {
	// Cleanup routes
	for _, r := range vservice.Spec.Routes {
		if _, err := c.cloud.DeleteRoute(ctx, r.Name, vservice.Spec.VirtualRouter.Name, vservice.Spec.MeshName); err != nil {
			if !aws.IsAWSErrNotFound(err) {
				return fmt.Errorf("failed to clean up route %s for virtual service %s during deletion: %s", r.Name, vservice.Name, err)
			}
		}
	}

	// TODO(nic): if we support a force delete, we can delete the rest of the routes attached to the virtual router here

	// Cleanup virtual service
	if _, err := c.cloud.DeleteVirtualService(ctx, vservice.Name, vservice.Spec.MeshName); err != nil {
		if !aws.IsAWSErrNotFound(err) {
			return fmt.Errorf("failed to clean up virtual service %s during deletion: %s", vservice.Name, err)
		}
	}

	// Cleanup virtual router
	if _, err := c.cloud.DeleteVirtualRouter(ctx, vservice.Spec.VirtualRouter.Name, vservice.Spec.MeshName); err != nil {
		if aws.IsAWSErrNotFound(err) || aws.IsAWSErrResourceInUse(err) {
			klog.Warningf("Virtual router %s was not deleted during cleanup: %s", vservice.Spec.VirtualRouter.Name, err)
		} else {
			return fmt.Errorf("failed to clean up virtual router %s for virtual service %s during deletion: %s", vservice.Spec.VirtualRouter.Name, vservice.Name, err)
		}
	}
	return nil
}
