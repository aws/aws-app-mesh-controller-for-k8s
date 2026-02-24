#!/usr/bin/env bash

# ./scripts/test-with-kind.sh
#
# Runs integration tests against a kind cluster. This is mostly intended for
# automated build systems, but can be used by developers on appropriately configured
# linux machines as well.
#
# Configuration is done through environment variables. All of the following
# must be set:
#
#   AWS_ACCOUNT_ID: aws account id to use in testing
#   VPC_ID: aws vpc id to use for the tests
#   AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, AWS_SESSION_TOKEN: session tokens to use
#
# There are some additional options that may be set:
#
#   AWS_REGION: region to run tests in (defaults to us-west-2)
#   TEST_SUITES: space-separated set of test suites to run (e.g. "mesh sidecar virtualnode").
#                If unset, all test suites are run.
#   K8S_VERSION: kubernetes version to use with the tests (default 1.17)
#   CLUSTER_NAME, KUBECONFIG: can be set to an existing kind cluster. If unset,
#                             a new cluster will be created and used for the tests.
#

set -eo pipefail

SCRIPTS_DIR="$(cd "$(dirname "$0")" || exit 1; pwd)"
ROOT_DIR="$(cd "$SCRIPTS_DIR/.." || exit 1; pwd)"
INT_TEST_DIR="$ROOT_DIR/test/integration"

source "$SCRIPTS_DIR/lib/aws.sh"
source "$SCRIPTS_DIR/lib/common.sh"
source "$SCRIPTS_DIR/lib/k8s.sh"

check_is_installed curl
check_is_installed docker
check_is_installed jq
check_is_installed uuidgen
check_is_installed wget
check_is_installed kind "You can install kind with the helper scripts/install-kind.sh"
check_is_installed kubectl "You can install kubectl with the helper scripts/install-kubectl.sh"
check_is_installed kustomize "You can install kustomize with the helper scripts/install-kustomize.sh"
check_is_installed controller-gen "You can install controller-gen with the helper scripts/install-controller-gen.sh"

if [ -z "$AWS_ACCOUNT_ID" ]; then
  echo "Must set 'AWS_ACCOUNT_ID'" >&2
  exit 1
fi
AWS_REGION="${AWS_REGION:-"us-west-2"}"

if [[ -z "$VPC_ID" ]]; then
  echo "Must set 'VPC_ID'" >&2
  exit 1
fi

if [[ -z "$AWS_ACCESS_KEY_ID" || -z "$AWS_SECRET_ACCESS_KEY" || -z "$AWS_SESSION_TOKEN" ]]; then
  echo "AWS credentials must be set in env: 'AWS_ACCESS_KEY_ID', 'AWS_SECRET_ACCESS_KEY'," \
    " and 'AWS_SESSION_TOKEN' must all be set" >&2
  exit 1
fi

if [[ -z "$TEST_SUITES" ]]; then
  TEST_SUITES="$(ls "$INT_TEST_DIR")"
fi

K8S_VERSION=${K8S_VERSION:-"1.17"}

# Creates a kind test cluster if none was provided.
if [[ -z "$CLUSTER_NAME" || -z "$KUBECONFIG" ]]; then
  echo "No test cluster detected, creating one..."
  CLUSTER_NAME="appmesh-test-$(uuidgen | cut -d'-' -f1 | tr '[:upper:]' '[:lower:]')"
  TMP_DIR=$ROOT_DIR/build/tmp-$CLUSTER_NAME
  $SCRIPTS_DIR/provision-kind-cluster.sh "${CLUSTER_NAME}" -v "${K8S_VERSION}"
  KUBECONFIG="$TMP_DIR"/kubeconfig
else
  echo "Using existing cluster '$CLUSTER_NAME' for testing..."
fi

CONTROLLER_IMAGE="appmesh-controller"
CONTROLLER_TAG="local"

IMAGE_HOST="840364872350.dkr.ecr.us-west-2.amazonaws.com"

ENVOY_IMAGE="$IMAGE_HOST/aws-appmesh-envoy"
ENVOY_LATEST_TAG="v1.34.12.1-prod"
ENVOY_1_22_TAG="v1.22.2.0-prod"

PROXY_ROUTE_IMAGE="$IMAGE_HOST/aws-appmesh-proxy-route-manager"
PROXY_ROUTE_TAG="v7-prod"

function install_crds {
    echo "installing CRDs ... "
    make install
    echo "ok."
}

# Fetches and builds all required images, and loads them into kind. Kind does
# not inherit the docker registration settings for the host, so preloading
# the images is necessary to make sure it has the required values.
function build_and_load_images {
  docker buildx build --platform linux/amd64 --build-arg GOPROXY="$GOPROXY" -t "$CONTROLLER_IMAGE:$CONTROLLER_TAG" . --load
  kind load docker-image --name "$CLUSTER_NAME" "$CONTROLLER_IMAGE:$CONTROLLER_TAG"

  ecr_login "$AWS_REGION" "$IMAGE_HOST"

  local __images=(
      "$PROXY_ROUTE_IMAGE:$PROXY_ROUTE_TAG"
      "$ENVOY_IMAGE:$ENVOY_LATEST_TAG"
      "$ENVOY_IMAGE:$ENVOY_1_22_TAG"
  )
  for image in "${__images[@]}"; do
      docker pull "$image"
      kind load docker-image --name "$CLUSTER_NAME" "$image"
  done
}

function run_integration_tests {
  echo running tests

  kubectl delete ns appmesh-system || :
  kubectl create ns appmesh-system

  for __test in ${TEST_SUITES[@]}; do
    local __test_dir="test/integration/$__test"
    local __envoy_version="$ENVOY_LATEST_TAG"

    # /test_app contains test app images, and is not itself an integration test
    if [[ "$__test" == "test_app" ]]; then
      continue
    fi

    # This test specifically tests behavior for an older version of envoy
    if [[ "$__test" == "sidecar-v1.22" ]]; then
        __envoy_version="$ENVOY_1_22_TAG"
    fi

    # Try to delete the controller. Sometimes, carrying over controllers
    # between tests can cause some minor issues.
    helm delete -n=appmesh-system appmesh-controller || :

    helm upgrade -i appmesh-controller config/helm/appmesh-controller --namespace appmesh-system \
      --set image.repository="$CONTROLLER_IMAGE" \
      --set image.tag="$CONTROLLER_TAG" \
      --set sidecar.image.repository="$ENVOY_IMAGE" \
      --set sidecar.image.tag="$__envoy_version" \
      --set enableBackendGroups=true \
      --set sidecar.waitUntilProxyReady=true \
      --set region="$AWS_REGION" \
      --set accountId="$AWS_ACCOUNT_ID" \
      --set "env.AWS_DEFAULT_REGION='$AWS_REGION'" \
      --set "env.AWS_ACCESS_KEY_ID='$AWS_ACCESS_KEY_ID'" \
      --set "env.AWS_SECRET_ACCESS_KEY='$AWS_SECRET_ACCESS_KEY'" \
      --set "env.AWS_SESSION_TOKEN='$AWS_SESSION_TOKEN'" \
      --set sidecar.envoyAwsAccessKeyId="$AWS_ACCESS_KEY_ID" \
      --set sidecar.envoyAwsSecretAccessKey="$AWS_SECRET_ACCESS_KEY" \
      --set sidecar.envoyAwsSessionToken="$AWS_SESSION_TOKEN"

    check_deployment_rollout appmesh-controller appmesh-system
    kubectl get pod -n appmesh-system

    echo -n "running integration test type $__test ... "
    ginkgo --flakeAttempts=2 -vv -r $__test_dir -- \
      --cluster-kubeconfig="$KUBECONFIG" \
      --cluster-name="$CLUSTER_NAME" \
      --aws-region="$AWS_REGION" \
      --aws-vpc-id="$VPC_ID"
    echo "ok."
  done
}

aws_check_credentials

build_and_load_images

# Generate and install CRDs
install_crds
kubectl get crds

run_integration_tests
