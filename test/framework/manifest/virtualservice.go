package manifest

import (
	"fmt"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-sdk-go/aws"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type VSBuilder struct {
	Namespace string
}

func (b *VSBuilder) BuildVirtualServiceWithRouterBackend(instanceName string, vrName string) *appmesh.VirtualService {
	vsName := b.buildServiceName(instanceName)

	vsDNS := fmt.Sprintf("%s.%s.svc.cluster.local", vsName, b.Namespace)

	vs := &appmesh.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: b.Namespace,
			Name:      vsName,
		},
		Spec: appmesh.VirtualServiceSpec{
			AWSName: aws.String(vsDNS),
			Provider: &appmesh.VirtualServiceProvider{
				VirtualRouter: &appmesh.VirtualRouterServiceProvider{
					VirtualRouterRef: &appmesh.VirtualRouterReference{
						Namespace: aws.String(b.Namespace),
						Name:      vrName,
					},
				},
			},
		},
	}
	return vs
}

func (b *VSBuilder) BuildVirtualServiceWithNodeBackend(instanceName string, vnName string) *appmesh.VirtualService {
	vsName := b.buildServiceName(instanceName)

	vsDNS := fmt.Sprintf("%s.%s.svc.cluster.local", vsName, b.Namespace)

	vs := &appmesh.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: b.Namespace,
			Name:      vsName,
		},
		Spec: appmesh.VirtualServiceSpec{
			AWSName: aws.String(vsDNS),
			Provider: &appmesh.VirtualServiceProvider{
				VirtualNode: &appmesh.VirtualNodeServiceProvider{
					VirtualNodeRef: &appmesh.VirtualNodeReference{
						Namespace: aws.String(b.Namespace),
						Name:      vnName,
					},
				},
			},
		},
	}
	return vs
}

func (b *VSBuilder) BuildVirtualServiceNoBackend(instanceName string) *appmesh.VirtualService {
	vsName := b.buildServiceName(instanceName)

	vsDNS := fmt.Sprintf("%s.%s.svc.cluster.local", vsName, b.Namespace)

	vs := &appmesh.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: b.Namespace,
			Name:      vsName,
		},
		Spec: appmesh.VirtualServiceSpec{
			AWSName: aws.String(vsDNS),
		},
	}
	return vs
}

func (b *VSBuilder) buildName(instanceName string) string {
	return instanceName
}

func (b *VSBuilder) buildServiceName(instanceName string) string {
	return instanceName
}
