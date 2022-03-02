package k8s

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
)

const (
	unknownK8sVersion = "UNKNOWN_K8S_VERSION"
)

// NamespacedName returns the namespaced name for k8s objects
func NamespacedName(obj metav1.Object) types.NamespacedName {
	return types.NamespacedName{
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
	}
}

// ServerVersion returns the version for k8s server
// the server version will be used to help App Mesh team to identify
// the platform Envoy is running on to provide better user experience
func ServerVersion(client discovery.ServerVersionInterface) string {
	versionInfo, err := client.ServerVersion()
	if err != nil {
		return unknownK8sVersion
	}
	return versionInfo.GitVersion
}
