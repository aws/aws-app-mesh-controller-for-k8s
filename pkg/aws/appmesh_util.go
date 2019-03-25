package aws

import "strings"

// This method is created to address the lack of native support of namespace within AppMesh API.
// If virtualNodeName in the CRD spec doesn't contain ".", we will construct the new virtualNodeName by appending "-defaultVirtualNodeNamespace".
// If it does, the new virtualNodeName is constructed by converting the "." to "-" since "." isn't a valid character as AppMesh virtual node name.
// Example 1: virtualNodeName: "foo", defaultVirtualNodeNamespace: "bar". The new virtualNodeName will be "foo-bar"
// Example 2: virtualNodeName: "foo.dummy", defaultVirtualNodeNamespace: "bar". The new virtualNodeName will be "foo-dummy"
func ConstructAppMeshVNodeNameFromCRD(virtualNodeName string, defaultVirtualNodeNamespace string) string {
	if strings.Contains(virtualNodeName, ".") {
		return strings.ReplaceAll(virtualNodeName, ".", "-")
	}
	// no "."
	return virtualNodeName + "-" + defaultVirtualNodeNamespace
}
