package aws

import "strings"

// This method is created to address the lack of native support of namespace within AppMesh API.
// If virtualNodeName in the CRD spec doesn't contain ".", we will construct the new virtualNodeName by appending "-defaultNamespace".
// If it does, the new virtualNodeName is constructed by converting the "." to "-" since "." isn't a valid character as AppMesh virtual node name.
// Example 1: virtualNodeName: "foo", defaultNamespace: "bar". The new virtualNodeName will be "foo-bar"
// Example 2: virtualNodeName: "foo.dummy", defaultNamespace: "bar". The new virtualNodeName will be "foo-dummy"
func ConstructAppMeshVNodeNameFromCRD(virtualNodeName string, defaultNamespace string) string {
	if strings.Contains(virtualNodeName, ".") {
		return strings.ReplaceAll(virtualNodeName, ".", "-")
	}
	// no "."
	return virtualNodeName + "-" + defaultNamespace
}
