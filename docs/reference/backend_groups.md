### Backend Groups
Backend Groups are an experimental feature aimed at simplifying the process of defining backends for VirtualNodes.  
#### Enabling Backend Groups
To use Backend Groups, include the flag `enableBackendGroups=true` in your controller deployment.

#### BackendGroup Spec
A Backend Group defines a set of VirtualService backends that can be added to VirtualNodes.

Here is a sample spec with a Backend Group containing 3 VirtualServices. This group is applied to a VirtualNode.

```
apiVersion: appmesh.k8s.aws/v1beta2
kind: BackendGroup
metadata:
  name: color-group
  namespace: ${APP_NAMESPACE}
spec:
  virtualservices:
    - name: green
      namespace: namespace-1
    - name: red
      namespace: namespace-1
    - name: yellow
      namespace: namespace-2
---
apiVersion: appmesh.k8s.aws/v1beta2
kind: VirtualNode
metadata:
  name: color-node
  namespace: ${APP_NAMESPACE}
spec:
  backendGroups:
    - name: color-group
      namespace: ${APP_NAMESPACE}
---
```

VirtualNode `color-node` will be able to use `green`, `red`, and `yellow` as backends.

Additionally, in the VirtualNode `backendGroups` field, specifying `*` as the Backend Group name will automatically add all VirtualServices in the specified namespace.
```
backendGroups:
  - name: *
    namespace: color-namespace
```
This allows any VirtualService in `color-namespace` to be a backend of the VirtualNode.
