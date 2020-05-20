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

const apiPathValidateAppMeshGatewayRoute = "/validate-appmesh-k8s-aws-v1beta2-gatewayroute"

// NewGatewayRouteValidator returns a validator for GatewayRoute.
func NewGatewayRouteValidator() *gatewayRouteValidator {
	return &gatewayRouteValidator{}
}

var _ webhook.Validator = &gatewayRouteValidator{}

type gatewayRouteValidator struct {
}

func (v *gatewayRouteValidator) Prototype(req admission.Request) (runtime.Object, error) {
	return &appmesh.GatewayRoute{}, nil
}

func (v *gatewayRouteValidator) ValidateCreate(ctx context.Context, obj runtime.Object) error {
	return nil
}

func (v *gatewayRouteValidator) ValidateUpdate(ctx context.Context, obj runtime.Object, oldObj runtime.Object) error {
	newGR := obj.(*appmesh.GatewayRoute)
	oldGR := oldObj.(*appmesh.GatewayRoute)
	if err := v.enforceFieldsImmutability(newGR, oldGR); err != nil {
		return err
	}
	return nil
}

func (v *gatewayRouteValidator) ValidateDelete(ctx context.Context, obj runtime.Object) error {
	return nil
}

// enforceFieldsImmutability will enforce immutable fields are not changed.
func (v *gatewayRouteValidator) enforceFieldsImmutability(newGR *appmesh.GatewayRoute, oldGR *appmesh.GatewayRoute) error {
	var changedImmutableFields []string
	if !reflect.DeepEqual(newGR.Spec.AWSName, oldGR.Spec.AWSName) {
		changedImmutableFields = append(changedImmutableFields, "spec.awsName")
	}
	if !reflect.DeepEqual(newGR.Spec.MeshRef, oldGR.Spec.MeshRef) {
		changedImmutableFields = append(changedImmutableFields, "spec.meshRef")
	}
	if !reflect.DeepEqual(newGR.Spec.VirtualGatewayRef, oldGR.Spec.VirtualGatewayRef) {
		changedImmutableFields = append(changedImmutableFields, "spec.virtualGatewayRef")
	}
	if len(changedImmutableFields) != 0 {
		return errors.Errorf("%s update may not change these fields: %s", "GatewayRoute", strings.Join(changedImmutableFields, ","))
	}
	return nil
}

//// +kubebuilder:webhook:path=/validate-appmesh-k8s-aws-v1beta2-gatewayroute,mutating=false,failurePolicy=fail,groups=appmesh.k8s.aws,resources=gatewayroutes,verbs=create;update,versions=v1beta2,name=vgatewayroute.appmesh.k8s.aws

func (v *gatewayRouteValidator) SetupWithManager(mgr ctrl.Manager) {
	mgr.GetWebhookServer().Register(apiPathValidateAppMeshGatewayRoute, webhook.ValidatingWebhookForValidator(v))
}
