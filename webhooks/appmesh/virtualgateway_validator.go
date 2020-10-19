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

const apiPathValidateAppMeshVirtualGateway = "/validate-appmesh-k8s-aws-v1beta2-virtualgateway"

// NewVirtualGatewayValidator returns a validator for VirtualGateway.
func NewVirtualGatewayValidator() *virtualGatewayValidator {
	return &virtualGatewayValidator{}
}

var _ webhook.Validator = &virtualGatewayValidator{}

type virtualGatewayValidator struct {
}

func (v *virtualGatewayValidator) Prototype(req admission.Request) (runtime.Object, error) {
	return &appmesh.VirtualGateway{}, nil
}

func (v *virtualGatewayValidator) ValidateCreate(ctx context.Context, obj runtime.Object) error {
	vg := obj.(*appmesh.VirtualGateway)

	if err := v.checkForConnectionPoolProtocols(vg); err != nil {
		return err
	}
	return nil
}

func (v *virtualGatewayValidator) ValidateUpdate(ctx context.Context, obj runtime.Object, oldObj runtime.Object) error {
	vg := obj.(*appmesh.VirtualGateway)
	oldVGateway := oldObj.(*appmesh.VirtualGateway)

	if err := v.enforceFieldsImmutability(vg, oldVGateway); err != nil {
		return err
	}

	if err := v.checkForConnectionPoolProtocols(vg); err != nil {
		return err
	}
	return nil
}

func (v *virtualGatewayValidator) ValidateDelete(ctx context.Context, obj runtime.Object) error {
	return nil
}

// enforceFieldsImmutability will enforce immutable fields are not changed.
func (v *virtualGatewayValidator) enforceFieldsImmutability(newVGateway *appmesh.VirtualGateway, oldVGateway *appmesh.VirtualGateway) error {
	var changedImmutableFields []string
	if !reflect.DeepEqual(newVGateway.Spec.AWSName, oldVGateway.Spec.AWSName) {
		changedImmutableFields = append(changedImmutableFields, "spec.awsName")
	}
	if !reflect.DeepEqual(newVGateway.Spec.MeshRef, oldVGateway.Spec.MeshRef) {
		changedImmutableFields = append(changedImmutableFields, "spec.meshRef")
	}
	if len(changedImmutableFields) != 0 {
		return errors.Errorf("%s update may not change these fields: %s", "VirtualGateway", strings.Join(changedImmutableFields, ","))
	}
	return nil
}

func (v *virtualGatewayValidator) checkForConnectionPoolProtocols(vg *appmesh.VirtualGateway) error {
	//App Mesh supports one type of connection pool at a time
	if vg.Spec.Listeners != nil {
		for _, listener := range vg.Spec.Listeners {
			err := v.checkListenerMultipleConnectionPools(listener)
			if err != nil {
				return err
			}

		}
	}
	return nil
}

func (v *virtualGatewayValidator) checkListenerMultipleConnectionPools(ln appmesh.VirtualGatewayListener) error {

	//App Mesh supports one type of connection pool at a time

	if ln.ConnectionPool == nil {
		return nil
	}
	poolCount := 0

	if ln.ConnectionPool.HTTP != nil {
		poolCount += 1
	}

	if ln.ConnectionPool.HTTP2 != nil {
		poolCount += 1
	}

	if ln.ConnectionPool.GRPC != nil {
		poolCount += 1
	}

	if poolCount > 1 {
		return errors.Errorf("Only one type of Virtual Gateway Connection Pool is allowed")
	}

	return nil
}

// +kubebuilder:webhook:path=/validate-appmesh-k8s-aws-v1beta2-virtualgateway,mutating=false,failurePolicy=fail,groups=appmesh.k8s.aws,resources=virtualgateways,verbs=create;update,versions=v1beta2,name=vvirtualgateway.appmesh.k8s.aws

func (v *virtualGatewayValidator) SetupWithManager(mgr ctrl.Manager) {
	mgr.GetWebhookServer().Register(apiPathValidateAppMeshVirtualGateway, webhook.ValidatingWebhookForValidator(v))
}
