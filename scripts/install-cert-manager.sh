#!/usr/bin/env bash

# ./scripts/install-cert-manager.sh
#
# Installs a stable version of cert-manager

set -Eo pipefail

CERT_MANAGER_VERSION="v0.14.3"

kubectl apply --validate=false -f https://github.com/jetstack/cert-manager/releases/download/${CERT_MANAGER_VERSION}/cert-manager.yaml
