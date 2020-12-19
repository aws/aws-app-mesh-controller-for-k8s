#/bin/bash

set -e

register_server_entries() {
    kubectl exec -n spire spire-server-0 -c spire-server -- /opt/spire/bin/spire-server entry create $@
}


echo "Registering an entry for spire agent..."
register_server_entries \
  -spiffeID spiffe://mtls-e2e.aws/ns/spire/sa/spire-agent \
  -selector k8s_sat:cluster:eks-cluster \
  -selector k8s_sat:agent_ns:spire \
  -selector k8s_sat:agent_sa:spire-agent \
  -node
