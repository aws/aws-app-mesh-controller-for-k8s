package controller

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
)

const inClusterNamespacePath = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"

func containsFinalizer(obj interface{}, finalizer string) (bool, error) {
	metaobj, err := meta.Accessor(obj)
	if err != nil {
		return false, fmt.Errorf("object has no meta: %v", err)
	}
	for _, f := range metaobj.GetFinalizers() {
		if f == finalizer {
			return true, nil
		}
	}
	return false, nil
}

func addFinalizer(obj interface{}, finalizer string) error {
	metaobj, err := meta.Accessor(obj)
	if err != nil {
		return fmt.Errorf("object has no meta: %v", err)
	}
	metaobj.SetFinalizers(append(metaobj.GetFinalizers(), finalizer))
	return nil
}

func removeFinalizer(obj interface{}, finalizer string) error {
	metaobj, err := meta.Accessor(obj)
	if err != nil {
		return fmt.Errorf("object has no meta: %v", err)
	}

	var finalizers []string
	for _, f := range metaobj.GetFinalizers() {
		if f == finalizer {
			continue
		}
		finalizers = append(finalizers, f)
	}
	metaobj.SetFinalizers(finalizers)
	return nil
}

// namespacedResourceName addresses the lack of native support of namespace within AppMesh API for virtual nodes, virtual
// routers, and routes. If the resource name doesn't contain ".", we will construct the new name by appending
// "-defaultResourceNamespace" where defaultResourceNamespace is the namespace of the resource. If it does, the new name
// is constructed by converting the "." to "-" since "." isn't a valid character in AppMesh virtual node, virtual router
// or route names.
//
// This results in a namespaced name to send to the App Mesh API to avoid collisions if there are multiple resources
// with the same name in different Kubernetes namespaces.
//
// Example 1: resourceName: "foo", defaultResourceNamespace: "bar". The App Mesh name will be "foo-bar"
// Example 2: resourceName: "foo.dummy", defaultResourceNamespace: "bar". The App Mesh name will be "foo-dummy"
func namespacedResourceName(resourceName string, defaultResourceNamespace string) string {
	if strings.Contains(resourceName, ".") {
		return strings.ReplaceAll(resourceName, ".", "-")
	}
	return resourceName + "-" + defaultResourceNamespace
}

// getInClusterNamespace returns the namespace of the controller pod.
func getInClusterNamespace() (string, error) {
	// Check whether the namespace file exists.
	// If not, we are not running in a cluster and can't get the namespace.
	_, err := os.Stat(inClusterNamespacePath)
	if os.IsNotExist(err) {
		return "", fmt.Errorf("not running in-cluster, please specify --election-namespace")
	} else if err != nil {
		return "", fmt.Errorf("error checking namespace file: %v", err)
	}

	// Load the namespace file and return its content
	namespace, err := ioutil.ReadFile(inClusterNamespacePath)
	if err != nil {
		return "", fmt.Errorf("error reading namespace file: %v", err)
	}
	return string(namespace), nil
}
