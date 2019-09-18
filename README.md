<p align="center">
    <img src="docs/product-icon_AWS_App_Mesh_icon_squid_ink.svg" alt="App Mesh Logo" width="200" />
</p>

## AWS App Mesh Controller For K8s

AWS App Mesh Controller For K8s is a controller to help manage [App Mesh](https://aws.amazon.com/app-mesh/) resources for a Kubernetes cluster.  The controller watches custom resources for changes and reflects those changes into the [App Mesh API](https://docs.aws.amazon.com/app-mesh/latest/APIReference/Welcome.html). It is accompanied by the deployment of three custom resource definitions ([CRDs](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)): meshes, virtualnodes, and virtualservices.  These map to App Mesh API objects which the controller manages for you.

## Getting started

Check out the [example](docs/example.md) application, and read [design and usage](docs/design.md).

To begin using it in your cluster, follow the [install instructions](docs/install.md).

## Custom Resource Examples

First, create a Mesh resource, which will trigger the controller to create a Mesh via the App Mesh API.  Then, define VirtualServices for each application that you want to route traffic to, and finally create VirtualNodes for each Deployment that will make up that Application.

### Mesh

    apiVersion: appmesh.k8s.aws/v1beta1
    kind: Mesh
    metadata:
      name: my-mesh

### Virtual Node

    apiVersion: appmesh.k8s.aws/v1beta1
    kind: VirtualNode
    metadata:
      name: my-app-a
      namespace: prod
    spec:
      meshName: my-mesh
      listeners:
        - portMapping:
            port: 9080
            protocol: http
      serviceDiscovery:
        dns:
          hostName: my-app-a.prod.svc.cluster.local
      backends:
        - virtualService:
            virtualServiceName: my-svc-a

### Virtual Service

    apiVersion: appmesh.k8s.aws/v1beta1
    kind: VirtualService
    metadata:
      name: my-svc-a
      namespace: prod
    spec:
      meshName: my-mesh
      virtualRouter:
        name: my-svc-a-router
        listeners:
          - portMapping:
              port: 9080
              protocol: http
      routes:
        - name: route-to-svc-a
          http:
            match:
              prefix: /
            action:
              weightedTargets:
                - virtualNodeName:
                  weight: 1


## Integrations

* [Weaveworks Flagger](https://github.com/weaveworks/flagger)

## Security disclosures

If you think you’ve found a potential security issue, please do not post it in the Issues.  Instead, please follow the instructions [here](https://aws.amazon.com/security/vulnerability-reporting/) or [email AWS security directly](mailto:aws-security@amazon.com).

## Contributing

Contributions welcome!  Please read our [guidelines](CONTRIBUTING.md) and [Code of Conduct](CODE_OF_CONDUCT.md).

## License

This library is licensed under the Apache 2.0 License.
