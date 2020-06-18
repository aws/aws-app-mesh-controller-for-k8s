# AppMesh Troubleshooting Guide

## Common Errors

### Exceeded pod count per VirtualNode limit
AppMesh limits pod count per virtualNode. By default the limit is 10.

Your can adjust this limit by adjust the "Connected Envoy processes per virtual node" [service quota](https://docs.aws.amazon.com/app-mesh/latest/userguide/service-quotas.html).

### Namespaces is not labeled correctly
Namespaces must be labeled with two kind of labels:

  * `appmesh.k8s.aws/sidecarInjectorWebhook: enabled` is required on namespaces where pod should be injected with envoy sidecars.
  * customized labels to make `mesh` CustomResource selects the namespace via `mesh.spec.namespaceSelector`. (optional if you have a single Mesh selects all namespaces)

## Troubleshooting

Tail the controller logs:

```bash
export APPMESH_SYSTEM_NAMESPACE=appmesh-system
kubectl logs -n "${APPMESH_SYSTEM_NAMESPACE}" -f --since 10s \
    $(kubectl get pods -n "${APPMESH_SYSTEM_NAMESPACE}" -o name | grep controller)
```

Tail envoy logs:

```bash
export APPLICATION_NAMESPACE=<your namespace>
export APPLICATION=<your pod or deployment> # i.e. deploy/my-app
kubectl logs -n "${APPLICATION_NAMESPACE} "${APPLICATION_POD}" envoy -f --since 10s
```

View envoy configuration:

```bash
export APPLICATION_NAMESPACE=<your namespace>
export APPLICATION=<your pod>
kubectl port-forward -n "${APPLICATION_NAMESPACE}" \
    $(kubectl get pod -n "${APPLICATION_NAMESPACE}" | grep "${APPLICATION}" |awk '{print $1}') \
    9901
```

Then navigate to `localhost:9901/` for the index or `localhost:9901/config_dump` for the envoy config.