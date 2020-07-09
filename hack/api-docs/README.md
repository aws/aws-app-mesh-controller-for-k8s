Follow below instructions to generate docs

1. create a doc.go under apis/appmesh/v1beta2/ with below contents
```
// Package v1beta2 contains API Schema definitions for the appmesh v1beta2 API group
// +kubebuilder:object:generate=true
// +groupName=appmesh.k8s.aws
package v1beta2
```
2. add `// +genclient` to CRD object type declarations(VirtualNode/Mesh/etc) with a blank line before other comments. e.g.
```
// +genclient
// 
// +kubebuilder:object:root=true
```

3. generate doc with below commend
```
gen-crd-api-reference-docs \
    -template-dir=hack/api-docs/template/ \
    -config=hack/api-docs/config.json \
    -api-dir=github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2 \
    -out-file docs/reference/api_spec.md
```