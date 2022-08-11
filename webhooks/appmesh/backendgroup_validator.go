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

const apiPathValidateAppMeshBackendGroup = "/validate-appmesh-k8s-aws-v1beta2-backendgroup"

// NewBackendGroupValidator returns a validator for BackendGroup.
func NewBackendGroupValidator() *backendGroupValidator {
	return &backendGroupValidator{}
}

var _ webhook.Validator = &backendGroupValidator{}

type backendGroupValidator struct {
}

func (v *backendGroupValidator) Prototype(req admission.Request) (runtime.Object, error) {
	return &appmesh.BackendGroup{}, nil
}

func (v *backendGroupValidator) ValidateCreate(ctx context.Context, obj runtime.Object) error {
	return nil
}

func (v *backendGroupValidator) ValidateUpdate(ctx context.Context, obj runtime.Object, oldObj runtime.Object) error {
	bg := obj.(*appmesh.BackendGroup)
	oldVS := oldObj.(*appmesh.BackendGroup)
	if err := v.enforceFieldsImmutability(bg, oldVS); err != nil {
		return err
	}
	return nil
}

func (v *backendGroupValidator) ValidateDelete(ctx context.Context, obj runtime.Object) error {
	return nil
}

// enforceFieldsImmutability will enforce immutable fields are not changed.
func (v *backendGroupValidator) enforceFieldsImmutability(bg *appmesh.BackendGroup, oldVS *appmesh.BackendGroup) error {
	var changedImmutableFields []string
	if !reflect.DeepEqual(bg.Spec.MeshRef, oldVS.Spec.MeshRef) {
		changedImmutableFields = append(changedImmutableFields, "spec.meshRef")
	}
	if len(changedImmutableFields) != 0 {
		return errors.Errorf("%s update may not change these fields: %s", "BackendGroup", strings.Join(changedImmutableFields, ","))
	}
	return nil
}

// +kubebuilder:webhook:path=/validate-appmesh-k8s-aws-v1beta2-backendgroup,mutating=false,failurePolicy=fail,groups=appmesh.k8s.aws,resources=backendgroups,verbs=create;update,versions=v1beta2,name=vbackendgroup.appmesh.k8s.aws,sideEffects=None,webhookVersions=v1beta1

func (v *backendGroupValidator) SetupWithManager(mgr ctrl.Manager) {
	mgr.GetWebhookServer().Register(apiPathValidateAppMeshBackendGroup, webhook.ValidatingWebhookForValidator(v))
}
