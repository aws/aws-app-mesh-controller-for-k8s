package appmesh

import (
	"context"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/references"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/webhook"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"strings"
)

const apiPathValidateAppMeshVirtualNode = "/validate-appmesh-k8s-aws-v1beta2-virtualnode"

// NewVirtualNodeValidator returns a validator for VirtualNode.
func NewVirtualNodeValidator() *virtualNodeValidator {
	return &virtualNodeValidator{}
}

var _ webhook.Validator = &virtualNodeValidator{}

type virtualNodeValidator struct {
}

func (v *virtualNodeValidator) Prototype(req admission.Request) (runtime.Object, error) {
	return &appmesh.VirtualNode{}, nil
}

func (v *virtualNodeValidator) ValidateCreate(ctx context.Context, obj runtime.Object) error {
	vn := obj.(*appmesh.VirtualNode)
	if err := v.checkVirtualNodeBackendsForDuplicates(vn); err != nil {
		return err
	}
	return nil
}

func (v *virtualNodeValidator) ValidateUpdate(ctx context.Context, obj runtime.Object, oldObj runtime.Object) error {
	vn := obj.(*appmesh.VirtualNode)
	oldVN := oldObj.(*appmesh.VirtualNode)
	if err := v.enforceFieldsImmutability(vn, oldVN); err != nil {
		return err
	}
	if err := v.checkVirtualNodeBackendsForDuplicates(vn); err != nil {
		return err
	}
	return nil
}

func (v *virtualNodeValidator) ValidateDelete(ctx context.Context, obj runtime.Object) error {
	return nil
}

// enforceFieldsImmutability will enforce immutable fields are not changed.
func (v *virtualNodeValidator) enforceFieldsImmutability(vn *appmesh.VirtualNode, oldVN *appmesh.VirtualNode) error {
	var changedImmutableFields []string
	if !reflect.DeepEqual(vn.Spec.AWSName, oldVN.Spec.AWSName) {
		changedImmutableFields = append(changedImmutableFields, "spec.awsName")
	}
	if !reflect.DeepEqual(vn.Spec.MeshRef, oldVN.Spec.MeshRef) {
		changedImmutableFields = append(changedImmutableFields, "spec.meshRef")
	}
	if oldVN.Spec.ServiceDiscovery.AWSCloudMap != nil && !reflect.DeepEqual(vn.Spec.ServiceDiscovery.AWSCloudMap, oldVN.Spec.ServiceDiscovery.AWSCloudMap) {
		changedImmutableFields = append(changedImmutableFields, "spec.serviceDiscovery.awsCloudMap")
	}
	if len(changedImmutableFields) != 0 {
		return errors.Errorf("%s update may not change these fields: %s", "VirtualNode", strings.Join(changedImmutableFields, ","))
	}
	return nil
}

func (v *virtualNodeValidator) checkVirtualNodeBackendsForDuplicates(vn *appmesh.VirtualNode) error {
	backends := vn.Spec.Backends
	backendMap := make(map[string]bool, len(backends))

	for _, backend := range backends {
		if backend.VirtualService.VirtualServiceRef != nil {
			backendNamespacedName := references.ObjectKeyForVirtualServiceReference(vn, *backend.VirtualService.VirtualServiceRef)
			backendIdentifier := backendNamespacedName.Name + "-" + backendNamespacedName.Namespace
			if _, ok := backendMap[backendIdentifier]; ok {
				return errors.Errorf("%s-%s has duplicate VirtualServiceReferences %s", "VirtualNode", vn.Name, backend.VirtualService.VirtualServiceRef.Name)
			} else {
				backendMap[backendIdentifier] = true
			}
		} else if backend.VirtualService.VirtualServiceARN != nil {
			if _, ok := backendMap[*backend.VirtualService.VirtualServiceARN]; ok {
				return errors.Errorf("%s-%s has duplicate VirtualServiceReferenceARNs %s", "VirtualNode", vn.Name, *backend.VirtualService.VirtualServiceARN)
			} else {
				backendMap[*backend.VirtualService.VirtualServiceARN] = true
			}
		}
	}
	return nil
}

// +kubebuilder:webhook:path=/validate-appmesh-k8s-aws-v1beta2-virtualnode,mutating=false,failurePolicy=fail,groups=appmesh.k8s.aws,resources=virtualnodes,verbs=create;update,versions=v1beta2,name=vvirtualnode.appmesh.k8s.aws

func (v *virtualNodeValidator) SetupWithManager(mgr ctrl.Manager) {
	mgr.GetWebhookServer().Register(apiPathValidateAppMeshVirtualNode, webhook.ValidatingWebhookForValidator(v))
}
