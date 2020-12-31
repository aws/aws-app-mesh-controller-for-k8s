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
	if err := v.checkForRequiredFields(vn); err != nil {
		return err
	}
	if err := v.checkVirtualNodeBackendsForDuplicates(vn); err != nil {
		return err
	}
	if err := v.checkForConnectionPoolProtocols(vn); err != nil {
		return err
	}
	return nil
}

func (v *virtualNodeValidator) ValidateUpdate(ctx context.Context, obj runtime.Object, oldObj runtime.Object) error {
	vn := obj.(*appmesh.VirtualNode)
	oldVN := oldObj.(*appmesh.VirtualNode)
	if err := v.checkForRequiredFields(vn); err != nil {
		return err
	}
	if err := v.enforceFieldsImmutability(vn, oldVN); err != nil {
		return err
	}
	if err := v.checkVirtualNodeBackendsForDuplicates(vn); err != nil {
		return err
	}
	if err := v.checkForConnectionPoolProtocols(vn); err != nil {
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
	if oldVN.Spec.ServiceDiscovery != nil && oldVN.Spec.ServiceDiscovery.AWSCloudMap != nil &&
		!reflect.DeepEqual(vn.Spec.ServiceDiscovery.AWSCloudMap, oldVN.Spec.ServiceDiscovery.AWSCloudMap) {
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

func (v *virtualNodeValidator) checkForRequiredFields(vn *appmesh.VirtualNode) error {
	//ServiceDiscovery is mandatory if a listener is specified
	if vn.Spec.Listeners != nil && vn.Spec.ServiceDiscovery == nil {
		return errors.Errorf("ServiceDiscovery missing for %s-%s. ServiceDiscovery must be specified when a listener is specified.", "VirtualNode", vn.Name)
	}
	return nil
}

func (v *virtualNodeValidator) checkForConnectionPoolProtocols(vn *appmesh.VirtualNode) error {
	//App Mesh supports one type of connection pool at a time
	if vn.Spec.Listeners != nil {
		for _, listener := range vn.Spec.Listeners {
			err := v.checkListenerMultipleConnectionPools(listener)
			if err != nil {
				return err
			}

		}
	}
	return nil
}

func (v *virtualNodeValidator) checkListenerMultipleConnectionPools(ln appmesh.Listener) error {

	//App Mesh supports one type of connection pool at a time

	if ln.ConnectionPool == nil {
		return nil
	}
	poolCount := 0

	if ln.ConnectionPool.TCP != nil {
		poolCount += 1
	}

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
		return errors.Errorf("Only one type of Virtual Node Connection Pool is allowed")
	}

	return nil
}

// +kubebuilder:webhook:path=/validate-appmesh-k8s-aws-v1beta2-virtualnode,mutating=false,failurePolicy=fail,groups=appmesh.k8s.aws,resources=virtualnodes,verbs=create;update,versions=v1beta2,name=vvirtualnode.appmesh.k8s.aws,sideEffects=None,webhookVersions=v1beta1

func (v *virtualNodeValidator) SetupWithManager(mgr ctrl.Manager) {
	mgr.GetWebhookServer().Register(apiPathValidateAppMeshVirtualNode, webhook.ValidatingWebhookForValidator(v))
}
