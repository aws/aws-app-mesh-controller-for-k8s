package controller

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
)

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

// parseMeshName returns meshName, namespace given the meshName reference and namespace of a virtual node or virtual
// service.  In order to support mesh references for meshes in different namespaces, we allow the format
// meshName.meshNamespace in the meshName reference field of a virtual node or virtual service.
func parseMeshName(meshName string, namespace string) (string, string) {
	meshNamespace := namespace
	meshParts := strings.Split(meshName, ".")
	if len(meshParts) > 1 {
		meshNamespace = strings.Join(meshParts[1:], ".")
		meshName = meshParts[0]
	}
	return meshName, meshNamespace
}