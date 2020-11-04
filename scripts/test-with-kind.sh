#!/usr/bin/env bash

# A script that builds the appmesh controller, provisions a KinD
# Kubernetes cluster, installs appmesh CRDs and controller into that
# Kubernetes cluster and runs a set of tests

set -Eo pipefail

SCRIPTS_DIR=$(cd "$(dirname "$0")" || exit 1; pwd)
ROOT_DIR="$SCRIPTS_DIR/.."
INT_TEST_DIR="$ROOT_DIR/test/integration"

CLUSTER_NAME_BASE="test"
K8S_VERSION="1.17"
TMP_DIR=""

source "$SCRIPTS_DIR/lib/common.sh"

check_is_installed curl
check_is_installed docker
check_is_installed jq
check_is_installed uuidgen
check_is_installed wget
check_is_installed kind "You can install kind with the helper scripts/install-kind.sh"
check_is_installed kubectl "You can install kubectl with the helper scripts/install-kubectl.sh"
check_is_installed kustomize "You can install kustomize with the helper scripts/install-kustomize.sh"
check_is_installed controller-gen "You can install controller-gen with the helper scripts/install-controller-gen.sh"


function setup_kind_cluster {
    TEST_ID=$(uuidgen | cut -d'-' -f1 | tr '[:upper:]' '[:lower:]')
    CLUSTER_NAME_BASE=$(uuidgen | cut -d'-' -f1 | tr '[:upper:]' '[:lower:]')
    CLUSTER_NAME="appmesh-test-$CLUSTER_NAME_BASE"-"${TEST_ID}"
    TMP_DIR=$ROOT_DIR/build/tmp-$CLUSTER_NAME
    $SCRIPTS_DIR/provision-kind-cluster.sh "${CLUSTER_NAME}" -v "${K8S_VERSION}"
}

function install_crds {
    make install 
}

function build_and_publish_controller {
        echo "Not implemented"
}

function install_controller {
        echo "Not implemented"
}

function run_integration_tests {
	echo "Not implemented"
}

setup_kind_cluster
export KUBECONFIG="${TMP_DIR}/kubeconfig"

# Generate and install CRDs
install_crds

# Install cert-manager
$SCRIPTS_DIR/install-cert-manager.sh


# Show the installed CRDs
kubectl get crds
