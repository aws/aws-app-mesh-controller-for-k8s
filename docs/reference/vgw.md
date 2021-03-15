### GatewayRoute to VirtualGateway Association while creating via CRD (Yaml Spec)
The virtual gateway selects namespaces using the namespaceSelector, here you can select multiple namespaces based on the labels you specify. But from namespace point of view, it is expected to have only 1 virtual gateway and all gateway routes defined in that namespace get attached to that 1 virtual gateway. If there is more than 1 VirtualGateway per namespace then creation of GatewayRoute will fail with following error
```
"Error from server (found multiple matching virtualGateways for namespace: namespace, expecting 1 but found N"
```

You can use multiple GatewayRoutes to route traffic to different applications in one namespace but there can be only 1 VirtualGateway for that namespace. Below is the CRD for VirtualGateway creation which selects namespace with labels (gateway: ingress-gw). We also create 2 GatewayRoutes in this namespace which get associated with this VirtualGateway. For a complete example, please check [example](https://github.com/aws/aws-app-mesh-examples/tree/main/walkthroughs/howto-k8s-ingress-gateway)
```
apiVersion: appmesh.k8s.aws/v1beta2
kind: VirtualGateway
metadata:
  name: ingress-gw
  namespace: ${APP_NAMESPACE}
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
---
apiVersion: appmesh.k8s.aws/v1beta2
kind: GatewayRoute
metadata:
  name: gateway-route-headers
  namespace: ${APP_NAMESPACE}
spec:
  httpRoute:
    match:
      prefix: "/headers"
    action:
      target:
        virtualService:
          virtualServiceRef:
            name: color-headers
---
apiVersion: appmesh.k8s.aws/v1beta2
kind: GatewayRoute
metadata:
  name: gateway-route-paths
  namespace: ${APP_NAMESPACE}
spec:
  httpRoute:
    match:
      prefix: "/paths"
    action:
      target:
        virtualService:
          virtualServiceRef:
            name: color-paths
---
