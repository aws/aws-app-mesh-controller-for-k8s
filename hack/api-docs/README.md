Follow below instructions to generate docs

1. Setup the [gen-crd-api-reference-docs](https://github.com/ahmetb/gen-crd-api-reference-docs/) tool
The tool does not support kubebuilder v2 so we need to add manual hooks to generate the docs

2. Create a doc.go under apis/appmesh/v1beta2/ with below contents
```
// Package v1beta2 contains API Schema definitions for the appmesh v1beta2 API group
// +kubebuilder:object:generate=true
// +groupName=appmesh.k8s.aws
package v1beta2
```
3. Add `// +genclient` to CRD object type declarations(VirtualNode/Mesh/etc) with a blank line before other comments. e.g.
```
// +genclient
// 
// +kubebuilder:object:root=true
```

4. Generate API docs with command below
```
gen-crd-api-reference-docs \
    -template-dir=hack/api-docs/template/ \
    -config=hack/api-docs/config.json \
    -api-dir=github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2 \
    -out-file docs/reference/api_spec.md
```
