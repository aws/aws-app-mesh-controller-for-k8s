package virtualnode

import (
	"context"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/k8s"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

// MembershipDesignator designates VirtualNode membership for pods.
type MembershipDesignator interface {
	// Designate will choose a VirtualNode for given pod or nil if it don't belong to any VirtualNode.
	Designate(ctx context.Context, pod *corev1.Pod) (*appmesh.VirtualNode, error)
}

// NewMembershipDesignator creates new MembershipDesignator.
func NewMembershipDesignator(k8sClient client.Client) MembershipDesignator {
	return &membershipDesignator{k8sClient: k8sClient}
}

var _ MembershipDesignator = &membershipDesignator{}

// meshSelectorDesignator designates VirtualNode membership based on selectors on VirtualNode.
type membershipDesignator struct {
	k8sClient client.Client
}

// +kubebuilder:rbac:groups=appmesh.k8s.aws,resources=virtualnodes,verbs=get;list;watch

func (d *membershipDesignator) Designate(ctx context.Context, pod *corev1.Pod) (*appmesh.VirtualNode, error) {
	vnList := appmesh.VirtualNodeList{}
	if err := d.k8sClient.List(ctx, &vnList, client.InNamespace(pod.Namespace)); err != nil {
		return nil, errors.Wrap(err, "failed to list VirtualNodes in cluster")
	}

	var vnCandidates []*appmesh.VirtualNode
	for _, vnObj := range vnList.Items {
		selector, err := metav1.LabelSelectorAsSelector(vnObj.Spec.PodSelector)
		if err != nil {
			return nil, err
		}
		if selector.Matches(labels.Set(pod.Labels)) {
			vnCandidates = append(vnCandidates, vnObj.DeepCopy())
		}
	}
	if len(vnCandidates) == 0 {
		return nil, nil
	}
	if len(vnCandidates) > 1 {
		var vnCandidatesNames []string
		for _, vn := range vnCandidates {
			vnCandidatesNames = append(vnCandidatesNames, k8s.NamespacedName(vn).String())
		}
		return nil, errors.Errorf("found multiple matching VirtualNodes for pod %s: %s",
			k8s.NamespacedName(pod).String(), strings.Join(vnCandidatesNames, ","))
	}
	return vnCandidates[0], nil
}
