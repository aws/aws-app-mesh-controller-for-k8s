#!/usr/bin/env bash

# ./scripts/install-cert-manager.sh
#
# Installs a stable version of cert-manager

set -Eo pipefail

DEFAULT_CERT_MANAGER_VERSION="v0.14.3"

check_is_installed kubectl "You can install kubectl with the helper scripts/install-kubectl.sh"

__cert_manager_version="$1"
if [ "z$__cert_manager_version" == "z" ]; then
    __cert_manager_version=${CERT_MANAGER_VERSION:-$DEFAULT_CERT_MANAGER_VERSION}
fi

echo -n "installing cert-manager ... "
__cert_manager_url="https://github.com/jetstack/cert-manager/releases/download/${__cert_manager_version}/cert-manager.yaml"
echo -n "installing cert-manager from $__cert_manager_url ... "
kubectl apply --validate=false -f $__cert_manager_url
echo "ok."
