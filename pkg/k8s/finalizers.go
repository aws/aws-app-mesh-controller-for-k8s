package k8s

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	FinalizerMeshMembers  = "finalizers.appmesh.k8s.aws/mesh-members"
	FinalizerAWSAppMeshResources = "finalizers.appmesh.k8s.aws/aws-appmesh-resources"
	FinalizerAWSCloudMapResources = "finalizers.appmesh.k8s.aws/aws-cloudmap-resources"
)

// HasFinalizer tests whether k8s object has specified finalizer
func HasFinalizer(obj metav1.Object, finalizer string) bool {
	f := obj.GetFinalizers()
	for _, e := range f {
		if e == finalizer {
			return true
		}
	}
	return false
}
