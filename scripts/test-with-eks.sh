#!/usr/bin/env bash

# A script that builds the appmesh controller, provisions an EKS
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


#CLUSTER_NAME_BASE="test"
#CLUSTER_NAME=""
K8S_VERSION="1.23"
TMP_DIR=""

source "$SCRIPTS_DIR/lib/aws.sh"
source "$SCRIPTS_DIR/lib/common.sh"
source "$SCRIPTS_DIR/lib/k8s.sh"

check_is_installed curl
check_is_installed docker
check_is_installed jq
check_is_installed uuidgen
check_is_installed wget
check_is_installed kubectl "You can install kubectl with the helper scripts/install-kubectl.sh"
check_is_installed kustomize "You can install kustomize with the helper scripts/install-kustomize.sh"
check_is_installed controller-gen "You can install controller-gen with the helper scripts/install-controller-gen.sh"


function setup_eks_cluster {
    eksctl create cluster --name $CLUSTER_NAME --region us-west-2 --version $K8S_VERSION --nodes-min 2 --nodes-max 3 --nodes 2 --kubeconfig $KUBECONFIG --full-ecr-access --appmesh-access
    eksctl utils associate-iam-oidc-provider --region $AWS_REGION --cluster $CLUSTER_NAME --approve
}

function install_crds {
    echo "installing CRDs ... "
    make install
    echo "ok."
}

function build_and_publish_controller {
       echo -n "building and publishing appmesh controller  ... "
       AWS_ACCOUNT=$AWS_ACCOUNT_ID AWS_REGION=$AWS_REGION make docker-build
       AWS_ACCOUNT=$AWS_ACCOUNT_ID AWS_REGION=$AWS_REGION make docker-push
       echo "ok."
}

function install_controller {
       echo -n "installing appmesh controller ... "
       local __controller_name="appmesh-controller"
       local __ns="appmesh-system"
       kubectl create ns $__ns
       APPMESH_PREVIEW=y AWS_ACCOUNT=$AWS_ACCOUNT_ID AWS_REGION=$AWS_REGION ENABLE_BACKEND_GROUPS=true make helm-deploy
       check_deployment_rollout $__controller_name $__ns
       echo -n "check the pods in appmesh-system namespace ... "
       kubectl get pod -n $__ns
       echo "ok."

}

function run_integration_tests {
  local __vpc_id=$(aws eks describe-cluster --name $CLUSTER_NAME --query "cluster.resourcesVpcConfig.vpcId")

  for __type in ${INT_TEST_DIR}/*
  do
    case `basename $__type` in
      test_app) # /test_app contains test app images
        continue
        ;;
      sidecar)
       APPMESH_PREVIEW=y AWS_ACCOUNT=$AWS_ACCOUNT_ID AWS_REGION=$AWS_REGION make helm-deploy WAIT_PROXY_READY=true
       check_deployment_rollout appmesh-controller appmesh-system
       kubectl get pod -n appmesh-system
        ;;
      sidecar-v1.22.2.0)
       APPMESH_PREVIEW=y AWS_ACCOUNT=$AWS_ACCOUNT_ID AWS_REGION=$AWS_REGION make helm-deploy WAIT_PROXY_READY=true SIDECAR_IMAGE_TAG=v1.22.2.0-prod
       check_deployment_rollout appmesh-controller appmesh-system
       kubectl get pod -n appmesh-system
        ;;
    esac

    echo -n "running integration test type $__type ... "
    ginkgo -v -r ${INT_TEST_DIR}/virtualnode -- --cluster-kubeconfig=${KUBECONFIG} \
            --cluster-name=${CLUSTER_NAME} --aws-region=${AWS_REGION} \
            --aws-vpc-id=$__vpc_id
    echo "ok."
  done
}

aws_check_credentials

if [ -z "$AWS_ACCOUNT_ID" ]; then
    AWS_ACCOUNT_ID=$( aws_account_id )
fi

ecr_login $AWS_REGION $ECR_URL

# Install kubebuilder
curl -L https://github.com/kubernetes-sigs/kubebuilder/releases/download/v1.0.8/kubebuilder_1.0.8_linux_amd64.tar.gz | tar -xz -C /tmp/
sudo mv /tmp/kubebuilder_1.0.8_linux_amd64 /usr/local/kubebuilder

# Build and publish the controller image
build_and_publish_controller

export KUBECONFIG=$(pwd)"/.kube/config"
setup_eks_cluster

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

kubectl logs --follow -l app.kubernetes.io/name=appmesh-controller -n appmesh-system & run_integration_tests
