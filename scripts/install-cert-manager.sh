#!/usr/bin/env bash

# ./scripts/install-cert-manager.sh
#
# Installs a stable version of cert-manager

set -Eo pipefail

SCRIPTS_DIR=$(cd "$(dirname "$0")" || exit 1; pwd)
DEFAULT_CERT_MANAGER_VERSION="v0.14.3"

source "$SCRIPTS_DIR/lib/k8s.sh"
source "$SCRIPTS_DIR/lib/common.sh"

check_is_installed kubectl "You can install kubectl with the helper scripts/install-kubectl.sh"

__cert_manager_version="$1"
if [ "z$__cert_manager_version" == "z" ]; then
    __cert_manager_version=${CERT_MANAGER_VERSION:-$DEFAULT_CERT_MANAGER_VERSION}
fi

install() {
    echo -n "installing cert-manager ... "
    __cert_manager_url="https://github.com/jetstack/cert-manager/releases/download/${__cert_manager_version}/cert-manager.yaml"
    echo -n "installing cert-manager from $__cert_manager_url ... "
    kubectl apply --validate=false -f $__cert_manager_url
    echo "ok."
}

check() {
    echo -n "checking cert-manager deployments have rolled out ... "
    local __ns="cert-manager"
    local __timeout="4m"
    check_deployment_rollout cert-manager-webhook $__ns $__timeout
    check_deployment_rollout cert-manager $__ns
    check_deployment_rollout cert-manager-cainjector $__ns
    echo "ok."
}


install
check
