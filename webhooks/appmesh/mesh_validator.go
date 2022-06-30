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

const apiPathValidateAppMeshMesh = "/validate-appmesh-k8s-aws-v1beta2-mesh"

// NewMeshValidator returns a validator for Mesh.
func NewMeshValidator(ipFamily string) *meshValidator {
	return &meshValidator{
		ipFamily: ipFamily,
	}
}

var _ webhook.Validator = &meshValidator{}

type meshValidator struct {
	ipFamily string
}

func (v *meshValidator) Prototype(req admission.Request) (runtime.Object, error) {
	return &appmesh.Mesh{}, nil
}

func (v *meshValidator) ValidateCreate(ctx context.Context, obj runtime.Object) error {
	mesh := obj.(*appmesh.Mesh)
	if err := v.checkIpPreference(mesh); err != nil {
		return err
	}
	return nil
}

func (v *meshValidator) ValidateUpdate(ctx context.Context, obj runtime.Object, oldObj runtime.Object) error {
	mesh := obj.(*appmesh.Mesh)
	oldMesh := oldObj.(*appmesh.Mesh)
	if err := v.enforceFieldsImmutability(mesh, oldMesh); err != nil {
		return err
	}
	if err := v.checkIpPreference(mesh); err != nil {
		return err
	}
	return nil
}

func (v *meshValidator) ValidateDelete(ctx context.Context, obj runtime.Object) error {
	return nil
}

// enforceFieldsImmutability will enforce immutable fields are not changed.
func (v *meshValidator) enforceFieldsImmutability(mesh *appmesh.Mesh, oldMesh *appmesh.Mesh) error {
	var changedImmutableFields []string
	if !reflect.DeepEqual(mesh.Spec.AWSName, oldMesh.Spec.AWSName) {
		changedImmutableFields = append(changedImmutableFields, "spec.awsName")
	}
	if len(changedImmutableFields) != 0 {
		return errors.Errorf("%s update may not change these fields: %s", "Mesh", strings.Join(changedImmutableFields, ","))
	}
	return nil
}

func (v *meshValidator) checkIpPreference(mesh *appmesh.Mesh) error {
	if mesh.Spec.ServiceDiscovery == nil {
		if v.ipFamily == IPv4 {
			return nil
		}
	} else {
		ipPreference := mesh.Spec.ServiceDiscovery.IpPreference
		if *ipPreference == appmesh.IpPreferenceIPv4 || *ipPreference == appmesh.IpPreferenceIPv6 {
			return nil
		} else {
			return errors.Errorf("Only non-empty values allowed are %s or %s", appmesh.IpPreferenceIPv4, appmesh.IpPreferenceIPv6)
		}
	}
	return nil
}

// +kubebuilder:webhook:path=/validate-appmesh-k8s-aws-v1beta2-mesh,mutating=false,failurePolicy=fail,groups=appmesh.k8s.aws,resources=meshes,verbs=create;update,versions=v1beta2,name=vmesh.appmesh.k8s.aws,sideEffects=None,webhookVersions=v1beta1

func (v *meshValidator) SetupWithManager(mgr ctrl.Manager) {
	mgr.GetWebhookServer().Register(apiPathValidateAppMeshMesh, webhook.ValidatingWebhookForValidator(v))
}
