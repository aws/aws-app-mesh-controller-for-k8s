package appmesh

import (
	"context"
	"reflect"
	"strings"

	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/webhook"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const apiPathValidateAppMeshVirtualRouter = "/validate-appmesh-k8s-aws-v1beta2-virtualrouter"

// NewVirtualRouterValidator returns a validator for VirtualRouter.
func NewVirtualRouterValidator() *virtualRouterValidator {
	return &virtualRouterValidator{}
}

var _ webhook.Validator = &virtualRouterValidator{}

type virtualRouterValidator struct {
}

func (v *virtualRouterValidator) Prototype(req admission.Request) (runtime.Object, error) {
	return &appmesh.VirtualRouter{}, nil
}

func (v *virtualRouterValidator) ValidateCreate(ctx context.Context, obj runtime.Object) error {
	vr := obj.(*appmesh.VirtualRouter)
	if err := v.checkForDuplicateRouteEntries(vr); err != nil {
		return err
	}
	for _, route := range vr.Spec.Routes {
		if err := validateRoute(route); err != nil {
			return err
		}
	}
	return nil
}

func validateRoute(route appmesh.Route) error {
	if route.HTTPRoute != nil {
		return validateRouteMatch(route.HTTPRoute.Match)
	}

	if route.HTTP2Route != nil {
		return validateRouteMatch(route.HTTP2Route.Match)
	}
	return nil
}

func validateRouteMatch(route appmesh.HTTPRouteMatch) error {
	if route.Prefix == nil && route.Path == nil {
		return errors.New("Either Prefix or Path must be specified")
	}
	if route.Prefix != nil && route.Path != nil {
		return errors.New("Both Prefix and Path cannot be specified, only 1 allowed")
	}

	if route.Path != nil {
		return validatePathForVirtualRoute(route.Path)
	}
	return nil
}

func validatePathForVirtualRoute(path *appmesh.HTTPPathMatch) error {
	exact := path.Exact
	regex := path.Regex

	if exact == nil && regex == nil {
		return errors.New("Either exact or regex for path must be specified")
	}

	if exact != nil && regex != nil {
		return errors.New("Both exact and regex for path are not allowed. Only one must be specified")
	}
	return nil
}

func (v *virtualRouterValidator) ValidateUpdate(ctx context.Context, obj runtime.Object, oldObj runtime.Object) error {
	vr := obj.(*appmesh.VirtualRouter)
	oldVR := oldObj.(*appmesh.VirtualRouter)
	if err := v.enforceFieldsImmutability(vr, oldVR); err != nil {
		return err
	}
	if err := v.checkForDuplicateRouteEntries(vr); err != nil {
		return err
	}
	for _, route := range vr.Spec.Routes {
		if err := validateRoute(route); err != nil {
			return err
		}
	}
	return nil
}

func (v *virtualRouterValidator) ValidateDelete(ctx context.Context, obj runtime.Object) error {
	return nil
}

// enforceFieldsImmutability will enforce immutable fields are not changed.
func (v *virtualRouterValidator) enforceFieldsImmutability(vr *appmesh.VirtualRouter, oldVR *appmesh.VirtualRouter) error {
	var changedImmutableFields []string
	if !reflect.DeepEqual(vr.Spec.AWSName, oldVR.Spec.AWSName) {
		changedImmutableFields = append(changedImmutableFields, "spec.awsName")
	}
	if !reflect.DeepEqual(vr.Spec.MeshRef, oldVR.Spec.MeshRef) {
		changedImmutableFields = append(changedImmutableFields, "spec.meshRef")
	}
	if len(changedImmutableFields) != 0 {
		return errors.Errorf("%s update may not change these fields: %s", "VirtualRouter", strings.Join(changedImmutableFields, ","))
	}
	return nil
}

func (v *virtualRouterValidator) checkForDuplicateRouteEntries(vr *appmesh.VirtualRouter) error {
	routes := vr.Spec.Routes
	routeMap := make(map[string]bool, len(routes))
	for _, route := range routes {
		if _, ok := routeMap[route.Name]; ok {
			return errors.Errorf("%s-%s has duplicate route entries for %s", "VirtualRouter", vr.Name, route.Name)
		} else {
			routeMap[route.Name] = true
		}
	}
	return nil
}

// +kubebuilder:webhook:path=/validate-appmesh-k8s-aws-v1beta2-virtualrouter,mutating=false,failurePolicy=fail,groups=appmesh.k8s.aws,resources=virtualrouters,verbs=create;update,versions=v1beta2,name=vvirtualrouter.appmesh.k8s.aws,sideEffects=None,webhookVersions=v1beta1

func (v *virtualRouterValidator) SetupWithManager(mgr ctrl.Manager) {
	mgr.GetWebhookServer().Register(apiPathValidateAppMeshVirtualRouter, webhook.ValidatingWebhookForValidator(v))
}
