#!/usr/bin/env bash

# A script that builds the appmesh controller, provisions a KinD
# Kubernetes cluster, installs appmesh CRDs and controller into that
# Kubernetes cluster and runs a set of tests

set -eo pipefail

SCRIPTS_DIR=$(cd "$(dirname "$0")" || exit 1; pwd)
ROOT_DIR="$SCRIPTS_DIR/.."
INT_TEST_DIR="$ROOT_DIR/test/integration"

AWS_ACCOUNT_ID=${AWS_ACCOUNT_ID:-""}
AWS_REGION=${AWS_REGION:-"us-west-2"}
IMAGE_NAME=amazon/appmesh-controller
ECR_URL=${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com
IMAGE=${ECR_URL}/${IMAGE_NAME}

CLUSTER_NAME="appmesh-test"

source "$SCRIPTS_DIR/lib/aws.sh"
source "$SCRIPTS_DIR/lib/common.sh"
source "$SCRIPTS_DIR/lib/k8s.sh"

function install_crds {
    echo "installing CRDs ... "
    make install
    echo "ok."
}

function install_controller {
       echo -n "installing appmesh controller ... "
       local __controller_name="appmesh-controller"
       local __ns="appmesh-system"
       kubectl create ns $__ns
       APPMESH_PREVIEW=y AWS_ACCOUNT=$AWS_ACCOUNT_ID AWS_REGION=$AWS_REGION make helm-deploy
       check_deployment_rollout $__controller_name $__ns
       echo -n "check the pods in appmesh-system namespace ... "
       kubectl get pod -n $__ns
       echo "ok."

}

function run_integration_tests {
  local __vpc_id=$( vpc_id )

  for __type in ${INT_TEST_DIR}/*
  do
    case `basename $__type` in
      test_app) # /test_app contains test app images
        continue
        ;;
    esac

    echo -n "running integration test type $__type ... "
    ginkgo -v -r $__type -- --cluster-kubeconfig=${KUBECONFIG} \
            --cluster-name=${CLUSTER_NAME} --aws-region=${AWS_REGION} \
            --aws-vpc-id=$__vpc_id
    echo "ok."
  done
}

export KUBECONFIG="$HOME/.kube/config"

# Generate and install CRDs
install_crds

# Install the controller
install_controller

# Show the installed CRDs
kubectl get crds

#FIXME sometimes the test controller "deployment" is ready but internally process is not ready
# leading to tests failing. Added this hack to workaround. Will be replaced with a better
# check later
sleep 15

run_integration_tests