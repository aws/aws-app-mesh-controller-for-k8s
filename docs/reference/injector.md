# Sidecar installation

In order to use AWS App Mesh in Kubernetes, pods in the mesh must be running the AWS App Mesh sidecar proxy (Envoy).

The following sections describe methods for injecting sidecar/Envoy into a pod for Virtual Nodes and Virtual Gateways.

## Envoy injection for virtual nodes

Sidecars can be automatically added to Kubernetes pods using a [mutating webhook admission controller](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/) as part of the App Mesh Kubernetes controller.

App Mesh uses namespace and/or pod annotations to determine if pods in a namespace will be marked for sidecar injection. There are two ways to achieve this:

`appmesh.k8s.aws/sidecarInjectorWebhook: enabled`: The sidecar injector will inject the sidecar into pods by default. Add the `appmesh.k8s.aws/sidecarInjectorWebhook` annotation with value `disabled` to the pod template spec to override the default and disable injection. For example:

```
apiVersion: v1
kind: Namespace
metadata:
  name: default-enabled
  labels:
    mesh: default-enabled
    appmesh.k8s.aws/sidecarInjectorWebhook: enabled
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: default-behavior // sidecar will be injected as namespace has sidecar injection enabled
  namespace: default-enabled
spec:
  template:
    spec:
      containers:
      - name: default-behavior
        image: tutum/curl
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: override-default-enabled
  namespace: default-enabled
spec:
  template:
    metadata:
      annotations:
        appmesh.k8s.aws/sidecarInjectorWebhook: disabled // this will override the default and disable inject sidecar
    spec:
      containers:
      - name: override-default-enabled
        image: tutum/curl
```

`appmesh.k8s.aws/sidecarInjectorWebhook: disabled`: The sidecar injector will not inject the sidecar into pods by default. Add the `appmesh.k8s.aws/sidecarInjectorWebhook` annotation with value `enabled` to the pod template spec to override the default and enable injection.

```
apiVersion: v1
kind: Namespace
metadata:
  name: default-disabled
  labels:
    mesh: default-disabled
    appmesh.k8s.aws/sidecarInjectorWebhook: disabled
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: default-behavior // sidecar will be not injected as namespace has sidecar injection disabled
  namespace: default-disabled
spec:
  template:
    spec:
      containers:
      - name: default-behavior
        image: tutum/curl
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: override-default-disabled
  namespace: default-disabled
spec:
  template:
    metadata:
      annotations:
        appmesh.k8s.aws/sidecarInjectorWebhook: enabled // this will override the default and inject sidecar
    spec:
      containers:
      - name: override-default-disabled
        image: tutum/curl
```


## Envoy injection for virtual gateways

AWS App Mesh supports virtual gateway resource to allow resources that are outside of your mesh to communicate to resources that are inside of your mesh. The virtual gateway represents an Envoy proxy running in the Kubernetes cluster. Unlike a virtual node, which represents Envoy running with an application, a virtual gateway represents Envoy deployed by itself. App Mesh Kubernetes controller supports injecting Envoy and virtual gateway configuration.

App Mesh Kubernetes controller uses podSelector to designate Virtual Gateway membership. If you create a pod with labels matching the pod selector labels in a virtual gateway spec, the controller will inject the Envoy configuration to the pod/envoy container and override the default container image by default.

Also, since a pod may contain multiple containers, the controller relies on the container name `envoy` to determine, which container to mutate for virtual gateway configuration.

To use the controller to inject virtual gateway configuration, add podSelector to your virtual gateway, add namespaceSelector label where you need to create the virtual gateway and set the container name to `envoy`:

```
apiVersion: appmesh.k8s.aws/v1beta2
kind: VirtualGateway
metadata:
  name: ingress-gw
  namespace: ns
spec:
  namespaceSelector:
    matchLabels:
      gateway: ingress-gw
  podSelector:
    matchLabels:
      app: ingress-gw
  listeners:
    - portMapping:
        port: 8088
        protocol: http
```

Add the labels in your virtual gateway pod spec:

```
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ingress-gw
  namespace: ns
spec:
  replicas: 1
  selector:
    matchLabels:
      app: ingress-gw
  template:
    metadata:
      labels:
        app: ingress-gw
    spec:
      containers:
        - name: envoy
          image: <envoy-image>
          ports:
            - containerPort: 8088
```

### Skip overriding Envoy image

If you wish to skip the Envoy image override, you can add the annotation `appmesh.k8s.aws/virtualGatewaySkipImageOverride` to your pod spec. This will make sure only virtual gateway configuration is added and Envoy image url override is skipped, allowing you to use custom image version.

```
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ingress-gw
  namespace: ns
spec:
  replicas: 1
  selector:
    matchLabels:
      app: ingress-gw
  template:
    metadata:
      annotations:
        appmesh.k8s.aws/virtualGatewaySkipImageOverride: enabled
      labels:
        app: ingress-gw
    spec:
      containers:
        - name: envoy
          image: <envoy-image>
          ports:
            - containerPort: 8088
```


## Custom Environment Variables For Envoy

Additional environment variables can be passed to the envoy sidecar container by
adding an `appmesh.k8s.aws/sidecarEnv` annotation to the application's
deployment:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ingress-gw
  namespace: ns
spec:
  template:
    metadata:
      annotations:
        appmesh.k8s.aws/sidecarEnv: "CUSTOM_VAR=value1"
```

Multiple variables can be set by passing a comma-delimited list -
`appmesh.k8s.aws/sidecarEnv: "CUSTOM_VAR_1=a, CUSTOM_VAR_2=b"`.
