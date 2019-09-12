# Design and Usage

## Background

App Mesh is an AWS service that provides an envoy based service mesh in which an envoy proxy (https://github.com/envoyproxy/envoy) is deployed alongside a user's applications, and facilitates communication between them, whether they are deployed in EKS, ECS, EC2, or any combination thereof.

In AWS App Mesh, the envoy proxies receive their configuration from the App Mesh service based on a number of API constructs that must be created that model the required connectivity between services.  The goal of this integration is the management of these API constructs on the cluster operator's behalf.

## App Mesh Concepts

### Mesh

A Mesh is a collection of App Mesh resources, and should include any resources for services that should communicate with one another, whether they run on EC2, ECS, or EKS.

### Virtual Service

An App Mesh virtual service is a named entity that represents a collection of endpoints.  For HTTP requests, the destination service is determined by the host header.  Each virtual service has a pointer to either a virtual router, or a virtual node in some special cases; these determine how the request to the service routed.

### Virtual Router

A virtual router encapsulates the routing logic for a virtual service.  The virtual router is a separate entity, so that a virtual service can be updated with entirely new routing logic with a single atomic update in which the virtual service's virtual router is swapped out. Multiple virtual services may share a single virtual router.

### Virtual Node

A virtual node represents a node in the mesh when viewed as a graph, where each node in the graph is an application instance or collection of instances, where the instance might be a client and/or a server, and each edge in the graph is a communication pathway between them.  Each envoy proxy is a member of a single virtual node, which must be defined on startup.  Multiple envoys can be (and usually are) members of the same virtual node.  This determines what set of configuration the envoy proxy receives.  Virtual node definitions include backends (the services that the virtual node wants to talk to), listeners (the ports on which it listens), and service discovery (how other services should find its endpoints).

### Route

A route is a set of rules that determines how HTTP traffic, bound for any of a set of virtual services which are managed by the route's containing virtual router, is directed to any of the virtual nodes that contain the service's endpoints.

## Design

The primary task of the controller is to watch for creation of and modification to three Custom Resources: `Mesh`, `VirtualNode` and `VirtualService`. First, the Mesh Resource will provide some initial bootstrap configuration and its creation will trigger the creation of the Mesh via the App Mesh API.

When you want to add an application to their service mesh, create a VirtualNode Custom Resource, and the controller will create a corresponding Virtual Node via the App Mesh API.  The Virtual Node contains listener, backend and service discovery configuration for a set of envoys.

Next (though order is not enforced), you can create a VirtualService custom resource, which contains route configuration, allowing requests from one application in the mesh to be routed to a number of virtual nodes that make up a service.  In App Mesh, this results in the creation of a virtual service, virtual router, and one or more routes.

It is highly recommended, though not required, to use the controller together with aws-app-mesh-inject, a webhook which performs envoy sidecar injection, to reduce the amount of manual configuration that you have to perform. The webhook will inject the sidecar into any namespace that is labelled with `appmesh.k8s.aws/sidecarInjectorWebhook=enabled`. See the [aws-app-mesh-inject](https://github.com/aws/aws-app-mesh-inject) repository for more details.

Each virtual node supports two types of service discovery, DNS and Cloud Map. When DNS service discovery is being used, then the provided hostname must resolve. The easiest way to configure that is to create a [service](https://kubernetes.io/docs/concepts/services-networking/service/) that corresponds to the virtual node.

## Virtual node naming convention

To support the use case of using the same application as virtual node across namespaces, the controller creates the virtual node name in App Mesh backend by appending the name of the `VirtualNode` custom resource with the namespace. The naming format is "VirtualNodeName-Namespace"

For example, in the VirtualNode definition below,
```
kind: VirtualNode
metadata:
  name: colorteller-black
  namespace: appmesh-demo

```
the virtual node name created in App Mesh backend will be "colorteller-black-appmesh-demo".

However, if the VirtualNode name in the custom resource contains `.`, i.e. `foo.bar`, the controller will assume namespace is `bar` by kubernetes convention. It will replace all `.` with `-` when creating the App Mesh virtual node because `.` isn't a valid character in App Mesh name.

For example, in the VirtualNode definition below,
```
kind: VirtualNode
metadata:
  name: colorteller-black.appmesh-demo
  namespace: appmesh-demo
```
the virtual node name created in App Mesh backend will be "colorteller-black-appmesh-demo".

Similarly, any references to VirtualNode name in other custom resources follow the same naming convention.

For example, in the example below,

```
kind: VirtualService
metadata:
  name: colorteller.appmesh-demo.svc.cluster.local
  namespace: appmesh-demo
spec:
  routes:
  - http:
      action:
        weightedTargets:
        - virtualNodeName: colorteller.appmesh-demo
          weight: 0
        - virtualNodeName: colorteller-blue
          weight: 3
        - virtualNodeName: colorteller-black.appmesh-demo
          weight: 3
```
the corresponding virtual node names in the App Mesh backend are, `colorteller-appmesh-demo`, `colorteller-blue-appmesh-demo`, and `colorteller-black-appmesh-demo` respectively.

Check out the [example](docs/example.md) application for more details.

## Cloud Map Service Discovery

Cloud Map service discovery can be used in place of DNS. See this [App Mesh road map item](https://github.com/aws/aws-app-mesh-roadmap/issues/47).  In order to use it, you must specify the service discovery type as "cloudMap" in your virtual node definition.  For example,

```
apiVersion: appmesh.k8s.aws/v1beta1
kind: VirtualNode
metadata:
  name: colorgateway
  namespace: appmesh-demo
spec:
  meshName: color-mesh
  listeners:
    - portMapping:
        port: 9080
        protocol: http
  serviceDiscovery:
    cloudMap:
      namespaceName: color-mesh-dns
      serviceName: colorteller-red
  backends:
    - virtualService:
        virtualServiceName: colorteller.appmesh-demo
    - virtualService:
        virtualServiceName: tcpecho.appmesh-demo
  logging:
    accessLog:
      file:
        path: /dev/stdout
```

The App Mesh controller will look for a Cloud Map namespace and service corresponding to the configuration in your virtual node.  The controller expects a namespace to preexist, but it will create a Cloud Map service if it needs to.  The controller watches for pod creates, and registers the endpoints (using [RegisterInstance](https://docs.aws.amazon.com/cloud-map/latest/api/API_RegisterInstance.html)) with the Cloud Map API.  If a pod belongs to a virtual node that uses Cloud Map service discovery, then that pod will be registered accordingly.
