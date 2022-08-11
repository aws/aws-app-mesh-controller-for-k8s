package manifest

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-sdk-go/aws"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type BGBuilder struct {
	Namespace string
}

func (b *BGBuilder) BuildBackendGroup(instanceName string, backendVirtualServices []types.NamespacedName) *appmesh.BackendGroup {
	var backends []appmesh.VirtualServiceReference

	for _, backendVS := range backendVirtualServices {
		backends = append(backends, appmesh.VirtualServiceReference{
			Namespace: aws.String(backendVS.Namespace),
			Name:      backendVS.Name,
		})
	}

	bg := &appmesh.BackendGroup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: b.Namespace,
			Name:      instanceName,
		},
		Spec: appmesh.BackendGroupSpec{
			VirtualServices: backends,
		},
	}
	return bg
}
