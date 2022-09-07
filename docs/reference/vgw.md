### GatewayRoute to VirtualGateway Association via (Yaml Spec)
A VirtualGateway can select GatewayRoute using following selectors  
#### namespaceSelector
VirtualGateway must specify namespaceSelector to associate GatewayRoutes belonging to a particular namespace.
An empty namespaceSelector would target GatewayRoutes in all namespaces. While nil or not specifying any namespace selector would not select any GatewayRoutes.

#### gatewayRouteSelector 
VirtualGateway can additionally specify gatewayRouteSelector to select subset of GatewayRoutes in a given namespace. 
An empty or not specifying this field (nil) will select all GatewayRoutes in a given namespace. If specified then it will select only those GatewayRoutes which have the matching labels. 

Here is a sample spec with 1 VirtualGateway and 2 GatewayRoutes. Here VirtualGateway specified a gatewayRouteSelector, based on which only one of the GatewayRoutes get selected.

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
  gatewayRouteSelector:
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
  labels:
    gateway: ingress-gw
spec:
  httpRoute:
    match:
      prefix: "/paths"
    action:
      target:
        virtualService:
          virtualServiceRef:
            name: color-paths
----
```

Since the GatewayRoute: gateway-route-headers doesn't have any matching VirtualGateway, customers will see following error message
```
failed to find matching virtualGateway for gatewayRoute: gateway-route-headers, expecting 1 but found 0
```

The above error message is to only notify the user that the GatewayRoute in the error message has not been associated with any VirtualGateway. So the user should either add matching gatewayRouteSelector to the unmatched gatewayRoute or completely remove the gatewayRouteSelector so that the VirtualGateway ignores this field and uses only the namespaceSelector. 