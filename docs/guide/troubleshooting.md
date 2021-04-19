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

## VirtualGateway - Common Issues
```
"Error from server (found multiple matching virtualGateways for namespace: namespace, expecting 1 but found N"
```
You will see an error similar to above if you try to create a gateway route in a namespace which has been associated with multiple virtual gateways.
Virtual Gateway selects namespace using namespace selector and it selects all GatewayRoutes present in that namespace. If there are 2 Virtual Gateways
for same GatewayRoute in a given namespace then you would see the above error. For more details refer [LiveDocs Virtual Gateway section](../reference/vgw.md)

## mTLS - Common Issues

### Envoy fails to boot up when SDS based mTLS is enabled

When SDS based mTLS is enabled at the controller level via `enable-sds` flag, controller expects to find SDS Provider’s UDS at path specified by `sds-uds-path`. It is set to a default value of `/run/spire/sockets/agent.sock` which is the default SPIRE Agent’s UDS path. Make sure that SDS Provider on the local node is up and running and UDS is active. Currently, SPIRE is the only supported SDS provider. Please check if SPIRE Agent is up and running on the same node as the problematic Envoy.

You can use the below command to figure out the exact reason of the envoy bootup issue. If the error is due to not being able to mount SDS provider's UDS socket then you would need to address that.

```bash
kubectl describe pod <pod-name> -n <namespace-name>
```

### Pod is up and running but Envoy doesn’t have any certs in SDS mode.

1. To begin with, check if `APPMESH_SDS_SOCKET_PATH` env variable is present under Envoy and if it has the correct UDS path value. If it is missing, then the controller didn’t inject the env variable. Check if `enable-sds` is set to `true` for the controller.

For example, when using SPIRE Agent you should see something like below in Envoy.

```
APPMESH_SDS_SOCKET_PATH:      /run/spire/sockets/agent.sock
```

2. If the above env variable is present with the correct value, then check if Envoy is able to communicate with the SDS Provider. Below command will help verify if the Envoy is able to reach out to the local SDS Provider via the UDS path passed in to the controller and also to see if it is healthy.

```bash
kubectl exec -it <pod-name> -n <namespace-name> -c envoy -- curl http://localhost:9901/clusters | grep -E '(static_cluster_sds.*cx_active|static_cluster_sds.*healthy)'

static_cluster_sds_unix_socket::/run/spire/sockets/agent.sock::cx_active::1
static_cluster_sds_unix_socket::/run/spire/sockets/agent.sock::health_flags::healthy
```

3. If the SDS cluster is healthy in Envoy, then check if the workload entry tied to this particular Pod/app container is registered with SPIRE Server and if the selectors match. Use the below command to list out all the registered workload entries.

```bash
kubectl exec -n spire spire-server-0 -- /opt/spire/bin/spire-server entry show
```
Once you have the list of entries, check for the entry that is tied to the app container and check if the selectors match. Default selectors that we currently use are pod’s service account, namespace and labels.

### Pod liveness and readiness probes fail when mTLS is enabled

HTTP and TCP health checks from the kubelet will not work as is, if mutual TLS is enabled as the kubelet doesn't have relevant certs.

**Workarounds:**

1. Expose the health check endpoint on a different port and skip mTLS for that port. You can then set `appmesh.k8s.aws/ports` annotation with the application port value on the deployment spec.

**Example**: If your main application port is 8080 and if health check endpoint is exposed on 8081, then `appmesh.k8s.aws/ports:8080` will help bypass mTLS for health check port.

### SDS cluster is present in Envoy's config even though corresponding VirtualNode doesn't have mTLS SDS config

Set `appmesh.k8s.aws/sds:disabled` for the deployments behind VirtualNodes without SDS config.
