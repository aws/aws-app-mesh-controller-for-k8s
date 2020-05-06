package mesh

import (
	"context"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

// MembershipDesignator designates mesh membership for namespaced AppMesh CRs.
type MembershipDesignator interface {
	// Designate will choose a mesh for given namespaced AppMesh CR.
	Designate(ctx context.Context, obj metav1.Object) (*appmesh.Mesh, error)
}

// NewMembershipDesignator creates new MembershipDesignator.
func NewMembershipDesignator(k8sClient client.Client) MembershipDesignator {
	return &membershipDesignator{k8sClient: k8sClient}
}

var _ MembershipDesignator = &membershipDesignator{}

// meshSelectorDesignator designates mesh membership based on selectors on mesh.
type membershipDesignator struct {
	k8sClient client.Client
}

// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch
// +kubebuilder:rbac:groups=appmesh.k8s.aws,resources=meshes,verbs=get;list;watch

func (d *membershipDesignator) Designate(ctx context.Context, obj metav1.Object) (*appmesh.Mesh, error) {
	objNS := corev1.Namespace{}
	if err := d.k8sClient.Get(ctx, types.NamespacedName{Name: obj.GetNamespace()}, &objNS); err != nil {
		return nil, errors.Wrapf(err, "failed to get namespace: %s", obj.GetNamespace())
	}
	meshList := appmesh.MeshList{}
	if err := d.k8sClient.List(ctx, &meshList); err != nil {
		return nil, errors.Wrap(err, "failed to list meshes in cluster")
	}

	var meshCandidates []*appmesh.Mesh
	for _, meshObj := range meshList.Items {
		selector, err := metav1.LabelSelectorAsSelector(meshObj.Spec.NamespaceSelector)
		if err != nil {
			return nil, err
		}
		if selector.Matches(labels.Set(objNS.Labels)) {
			meshCandidates = append(meshCandidates, meshObj.DeepCopy())
		}
	}
	if len(meshCandidates) == 0 {
		return nil, errors.Errorf("failed to find matching mesh for namespace: %s, expecting 1 but found %d",
			obj.GetNamespace(), 0)
	}
	if len(meshCandidates) > 1 {
		var meshCandidatesNames []string
		for _, mesh := range meshCandidates {
			meshCandidatesNames = append(meshCandidatesNames, mesh.Name)
		}
		return nil, errors.Errorf("found multiple matching meshes for namespace: %s, expecting 1 but found %d: %s",
			obj.GetNamespace(), len(meshCandidates), strings.Join(meshCandidatesNames, ","))
	}
	if !meshCandidates[0].DeletionTimestamp.IsZero() {
		return nil, errors.Errorf("unable to create new content in mesh %s because it is being terminated", meshCandidates[0].Name)
	}
	return meshCandidates[0], nil
}
