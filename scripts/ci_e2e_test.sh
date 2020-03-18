#!/usr/bin/env bash

set -ueo pipefail

# AWS environment
AWS_DEFAULT_REGION=${AWS_DEFAULT_REGION:-"us-west-2"}
AWS_K8S_TESTER_VERSION="v0.7.6"
source $(dirname "${BASH_SOURCE}")/lib/cluster.sh
source $(dirname "${BASH_SOURCE}")/lib/ecr.sh

# Build environment
LOCAL_GIT_VERSION=$(git describe --tags --always --dirty)
IMAGE_TAG=$LOCAL_GIT_VERSION
IMAGE_REPO_APP_MESH_CONTROLLER="amazon/app-mesh-controller"

# Cluster settings
CLUSTER_ID=${CLUSTER_ID:-$RANDOM}
CLUSTER_NAME=appmesh-e2e-$CLUSTER_ID
CLUSTER_TEST_DIR=/tmp/appmesh-e2e/clusters/${CLUSTER_NAME}
CLUSTER_CONFIG=${CLUSTER_CONFIG:-${CLUSTER_TEST_DIR}/${CLUSTER_NAME}.yaml}
CLUSTER_KUBECONFIG=${CLUSTER_KUBECONFIG:-${CLUSTER_TEST_DIR}/${CLUSTER_NAME}.kubeconfig}
CLUSTER_VERSION=${CLUSTER_VERSION:-"1.14"}
CLUSTER_INSTANCE_TYPE=m5.xlarge
CLUSTER_NODE_COUNT=4

#######################################
# Build and push appMesh controller image
# Globals:
#   None
# Arguments:
#   image_repo
#   image_tag
#   image_name
#
#######################################
build_push_controller_image() {
    declare -r image_repo="$1" image_tag="$2" image_name="$3"

    ecr::ensure_repository "${image_repo}"
    if [[ $? -ne 0 ]]; then
        echo "Unable to ensure docker image repository" >&2
        return 1
    fi

    if [[ $(ecr::contains_image "$image_repo" "$image_tag") ]]; then
      echo "docker image ${image_name} already exists in repository. Skipping image build..."
      return 0
    fi

    echo "Building docker image"
    if ! docker build -t "${image_name}" ./; then
        echo "Unable to build docker image" >&2
        return 1
    fi

    echo "Pushing docker image ${image_name}"
    if ! ecr::push_image "${image_name}"; then
        echo "Unable to push docker image" >&2
        return 1
    fi
    return 0
}

#######################################
# Setup test cluster
# Globals:
#   AWS_K8S_TESTER_VERSION
#   CLUSTER_CONFIG
#   CLUSTER_KUBECONFIG
#   CLUSTER_NAME
#   CLUSTER_VERSION
#   CLUSTER_INSTANCE_TYPE
#   CLUSTER_NODE_COUNT
# Arguments:
#   None
#
#######################################
setup_cluster() {
    if ! cluster::init "${AWS_K8S_TESTER_VERSION}"; then
        ehco "Unable to init aws-k8s-tester" >&2
        exit 1
    fi

    mkdir -p "${CLUSTER_TEST_DIR}"
    if ! cluster::create "${CLUSTER_CONFIG}" "${CLUSTER_KUBECONFIG}" "${CLUSTER_NAME}" "${CLUSTER_VERSION}" "${CLUSTER_INSTANCE_TYPE}" "${CLUSTER_NODE_COUNT}"; then
        echo "Unable to create cluster" >&2
        exit 1
    fi
}

#######################################
# Cleanup test cluster
# Globals:
#   CLUSTER_CONFIG
# Arguments:
#   None
#
#######################################
cleanup_cluster() {
    if ! cluster::delete "${CLUSTER_CONFIG}"; then
        echo "Unable to delete cluster" >&2
    fi
}

#######################################
# Test appMesh controller image
# Globals:
#   CLUSTER_CONFIG
#   CLUSTER_KUBECONFIG
#   CLUSTER_NAME
#   AWS_DEFAULT_REGION
# Arguments:
#   image_name
#
#######################################
test_controller_image() {
    declare -r image_name="$1"

    go get github.com/mikefarah/yq/v3
    go get github.com/onsi/ginkgo/ginkgo

    local vpc_id=$(yq read "${CLUSTER_CONFIG}" parameters.vpc-id)
    ginkgo -v -r test/e2e/ -- \
        --kubeconfig=${CLUSTER_KUBECONFIG} \
        --cluster-name=${CLUSTER_NAME} \
        --aws-region=${AWS_DEFAULT_REGION} \
        --aws-vpc-id=${vpc_id} \
        --controller-image=${image_name}
}

#######################################
# Entry point
# Globals:
#   IMAGE_REPO_APP_MESH_CONTROLLER
#   IMAGE_TAG
# Arguments:
#   None
#
#######################################
main() {
  local image_name=$(ecr::name_image "${IMAGE_REPO_APP_MESH_CONTROLLER}" "${IMAGE_TAG}")
  if [[ $? -ne 0 ]]; then
      echo "Unable to name docker image" >&2
      return 1
  fi
  build_push_controller_image "${IMAGE_REPO_APP_MESH_CONTROLLER}" "${IMAGE_TAG}" "${image_name}"

  echo $image_name
  trap "cleanup_cluster" EXIT
  setup_cluster
  test_controller_image "${image_name}"
}

main $@