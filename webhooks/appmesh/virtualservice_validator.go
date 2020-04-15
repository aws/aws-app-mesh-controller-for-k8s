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

const apiPathValidateAppMeshVirtualService = "/validate-appmesh-k8s-aws-v1beta2-virtualservice"

// NewVirtualServiceValidator returns a validator for VirtualService.
func NewVirtualServiceValidator() *virtualServiceValidator {
	return &virtualServiceValidator{}
}

var _ webhook.Validator = &virtualServiceValidator{}

type virtualServiceValidator struct {
}

func (v *virtualServiceValidator) Prototype(req admission.Request) (runtime.Object, error) {
	return &appmesh.VirtualService{}, nil
}

func (v *virtualServiceValidator) ValidateCreate(ctx context.Context, obj runtime.Object) error {
	return nil
}

func (v *virtualServiceValidator) ValidateUpdate(ctx context.Context, obj runtime.Object, oldObj runtime.Object) error {
	vs := obj.(*appmesh.VirtualService)
	oldVS := oldObj.(*appmesh.VirtualService)
	if err := v.enforceFieldsImmutability(vs, oldVS); err != nil {
		return err
	}
	return nil
}

func (v *virtualServiceValidator) ValidateDelete(ctx context.Context, obj runtime.Object) error {
	return nil
}

// enforceFieldsImmutability will enforce immutable fields are not changed.
func (v *virtualServiceValidator) enforceFieldsImmutability(vs *appmesh.VirtualService, oldVS *appmesh.VirtualService) error {
	var changedImmutableFields []string
	if !reflect.DeepEqual(vs.Spec.AWSName, oldVS.Spec.AWSName) {
		changedImmutableFields = append(changedImmutableFields, "spec.awsName")
	}
	if !reflect.DeepEqual(vs.Spec.MeshRef, oldVS.Spec.MeshRef) {
		changedImmutableFields = append(changedImmutableFields, "spec.meshRef")
	}
	if len(changedImmutableFields) != 0 {
		return errors.Errorf("%s update may not change these fields: %s", "VirtualService", strings.Join(changedImmutableFields, ","))
	}
	return nil
}

// +kubebuilder:webhook:path=/validate-appmesh-k8s-aws-v1beta2-virtualservice,mutating=false,failurePolicy=fail,groups=appmesh.k8s.aws,resources=virtualservices,verbs=create;update,versions=v1beta2,name=vvirtualservice.appmesh.k8s.aws

func (v *virtualServiceValidator) SetupWithManager(mgr ctrl.Manager) {
	mgr.GetWebhookServer().Register(apiPathValidateAppMeshVirtualService, webhook.ValidatingWebhookForValidator(v))
}
