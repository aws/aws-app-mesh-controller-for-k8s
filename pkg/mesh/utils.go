package mesh

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	corev1 "k8s.io/api/core/v1"
)

// IsMeshActive tests whether given mesh is active.
// mesh is active when its MeshActive condition equals true.
func IsMeshActive(ms *appmesh.Mesh) bool {
	for _, condition := range ms.Status.Conditions {
		if condition.Type == appmesh.MeshActive {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

// IsMeshReferenced tests whether given mesh is referenced by meshReference
func IsMeshReferenced(ms *appmesh.Mesh, reference appmesh.MeshReference) bool {
	return ms.Name == reference.Name && ms.UID == reference.UID
}
