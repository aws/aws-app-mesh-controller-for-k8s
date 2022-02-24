package appmesh

import (
	"context"
	"fmt"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/mesh"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/webhook"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const apiPathMutateAppMeshVirtualNode = "/mutate-appmesh-k8s-aws-v1beta2-virtualnode"

// NewVirtualNodeMutator returns a mutator for VirtualNode.
func NewVirtualNodeMutator(meshMembershipDesignator mesh.MembershipDesignator, ipFamily string) *virtualNodeMutator {
	return &virtualNodeMutator{
		meshMembershipDesignator: meshMembershipDesignator,
		ipFamily:                 ipFamily,
	}
}

var _ webhook.Mutator = &virtualNodeMutator{}

type virtualNodeMutator struct {
	meshMembershipDesignator mesh.MembershipDesignator
	ipFamily                 string
}

func (m *virtualNodeMutator) Prototype(req admission.Request) (runtime.Object, error) {
	return &appmesh.VirtualNode{}, nil
}

func (m *virtualNodeMutator) MutateCreate(ctx context.Context, obj runtime.Object) (runtime.Object, error) {
	vn := obj.(*appmesh.VirtualNode)
	if err := m.designateMeshMembership(ctx, vn); err != nil {
		return nil, err
	}
	if err := m.defaultingAWSName(vn); err != nil {
		return nil, err
	}
	if err := m.defaultingIpPreference(vn); err != nil {
		return nil, err
	}

	return vn, nil
}

func (m *virtualNodeMutator) MutateUpdate(ctx context.Context, obj runtime.Object, oldObj runtime.Object) (runtime.Object, error) {
	return obj, nil
}

func (m *virtualNodeMutator) defaultingAWSName(vn *appmesh.VirtualNode) error {
	if vn.Spec.AWSName == nil || len(*vn.Spec.AWSName) == 0 {
		awsName := fmt.Sprintf("%s_%s", vn.Name, vn.Namespace)
		vn.Spec.AWSName = &awsName
	}
	return nil
}

func (m *virtualNodeMutator) designateMeshMembership(ctx context.Context, vn *appmesh.VirtualNode) error {
	if vn.Spec.MeshRef != nil {
		return errors.Errorf("%s create may not specify read-only field: %s", "VirtualNode", "spec.meshRef")
	}
	mesh, err := m.meshMembershipDesignator.Designate(ctx, vn)
	if err != nil {
		return err
	}
	vn.Spec.MeshRef = &appmesh.MeshReference{
		Name: mesh.Name,
		UID:  mesh.UID,
	}
	return nil
}

func setDefaultIpPreference(ipPreference string) *string {
	if ipPreference == IPv6 {
		return aws.String(appmesh.IpPreferenceIPv6)
	} else {
		return aws.String(appmesh.IpPreferenceIPv4)
	}
}

func (m *virtualNodeMutator) defaultingIpPreference(vn *appmesh.VirtualNode) error {
	if vn.Spec.ServiceDiscovery.DNS != nil {
		ipPreferenceGiven := vn.Spec.ServiceDiscovery.DNS.IpPreference
		if ipPreferenceGiven == nil || len(*ipPreferenceGiven) == 0 {
			vn.Spec.ServiceDiscovery.DNS.IpPreference = setDefaultIpPreference(m.ipFamily)
		}
	}
	if vn.Spec.ServiceDiscovery.AWSCloudMap != nil {
		ipPreferenceGiven := vn.Spec.ServiceDiscovery.AWSCloudMap.IpPreference
		if ipPreferenceGiven == nil || len(*ipPreferenceGiven) == 0 {
			vn.Spec.ServiceDiscovery.AWSCloudMap.IpPreference = setDefaultIpPreference(m.ipFamily)
		}
	}
	return nil
}

// +kubebuilder:webhook:path=/mutate-appmesh-k8s-aws-v1beta2-virtualnode,mutating=true,failurePolicy=fail,groups=appmesh.k8s.aws,resources=virtualnodes,verbs=create;update,versions=v1beta2,name=mvirtualnode.appmesh.k8s.aws,sideEffects=None,webhookVersions=v1beta1

func (m *virtualNodeMutator) SetupWithManager(mgr ctrl.Manager) {
	mgr.GetWebhookServer().Register(apiPathMutateAppMeshVirtualNode, webhook.MutatingWebhookForMutator(m))
}
