#/bin/bash

set -e

register_server_entries() {
    kubectl exec -n spire spire-server-0 -c spire-server -- /opt/spire/bin/spire-server entry create $@
}

echo "Registering an entry for the virtualnode..."
register_server_entries \
  -parentID spiffe://mtls-e2e.aws/ns/spire/sa/spire-agent \
  -spiffeID spiffe://mtls-e2e.aws/$1 \
  -selector k8s:ns:mtls-e2e \
  -selector k8s:sa:default \
  -selector k8s:pod-label:app.kubernetes.io/instance:$1 \
  -selector k8s:container-name:envoy
