package appmesh

import (
	"context"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/webhook"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"strings"
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

// +kubebuilder:webhook:path=/validate-appmesh-k8s-aws-v1beta2-virtualrouter,mutating=false,failurePolicy=fail,groups=appmesh.k8s.aws,resources=virtualrouters,verbs=create;update,versions=v1beta2,name=vvirtualrouter.appmesh.k8s.aws

func (v *virtualRouterValidator) SetupWithManager(mgr ctrl.Manager) {
	mgr.GetWebhookServer().Register(apiPathValidateAppMeshVirtualRouter, webhook.ValidatingWebhookForValidator(v))
}
