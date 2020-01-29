package controller

import (
	"testing"

	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/appmesh"

	appmeshv1beta1 "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/apis/appmesh/v1beta1"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws"
)

func TestMeshNeedsUpdate(t *testing.T) {
	var (
		defaultSpec                 = appmeshv1beta1.MeshSpec{}
		specWithEgressFilterDropAll = appmeshv1beta1.MeshSpec{
			EgressFilter: &appmeshv1beta1.MeshEgressFilter{
				Type: appmeshv1beta1.MeshEgressFilterTypeDropAll,
			},
		}
		specWithEgressFilterAllowAll = appmeshv1beta1.MeshSpec{
			EgressFilter: &appmeshv1beta1.MeshEgressFilter{
				Type: appmeshv1beta1.MeshEgressFilterTypeAllowAll,
			},
		}

		defaultResult                 = &appmesh.MeshSpec{}
		resultWithEgressFilterDropAll = &appmesh.MeshSpec{
			EgressFilter: &appmesh.EgressFilter{
				Type: awssdk.String(appmesh.EgressFilterTypeDropAll),
			},
		}
		resultWithEgressFilterAllowAll = &appmesh.MeshSpec{
			EgressFilter: &appmesh.EgressFilter{
				Type: awssdk.String(appmesh.EgressFilterTypeAllowAll),
			},
		}
	)

	var meshtests = []struct {
		name        string
		spec        appmeshv1beta1.MeshSpec
		result      *appmesh.MeshSpec
		needsUpdate bool
	}{
		{"meshes are the same", defaultSpec, defaultResult, false},
		{"egressFilter same DROP_ALL", specWithEgressFilterDropAll, resultWithEgressFilterDropAll, false},
		{"egressFilter same ALLOW_ALL", specWithEgressFilterAllowAll, resultWithEgressFilterAllowAll, false},
		{"egressfilter mismatch DROP_ALL, ALLOW_ALL", specWithEgressFilterDropAll, resultWithEgressFilterAllowAll, true},
		{"egressfilter mismatch ALLOW_ALL, DROP_ALL", specWithEgressFilterAllowAll, resultWithEgressFilterDropAll, true},
	}

	for _, tt := range meshtests {
		t.Run(tt.name, func(t *testing.T) {
			ctrlMesh := &appmeshv1beta1.Mesh{
				Spec: tt.spec,
			}
			awsMesh := &aws.Mesh{
				Data: appmesh.MeshData{
					Spec: tt.result,
				},
			}
			ctrl := &Controller{}
			if res := ctrl.meshNeedsUpdate(ctrlMesh, awsMesh); res != tt.needsUpdate {
				t.Errorf("got %v, want %v", res, tt.needsUpdate)
			}
		})
	}
}
