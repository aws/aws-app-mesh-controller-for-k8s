#!/bin/bash

# This script runs integration tests for the AWS AppMesh Controller

set -e

SECONDS=0
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
echo "Running AppMeshController integration test with the following variables
KUBE CONFIG: $KUBE_CONFIG_PATH
CLUSTER_NAME: $CLUSTER_NAME
REGION: $REGION
OS_OVERRIDE: $OS_OVERRIDE"

if [[ -z "${OS_OVERRIDE}" ]]; then
  OS_OVERRIDE=linux
fi

function toggle_windows_scheduling(){
  schedule=$1
  nodes=$(kubectl get nodes -l kubernetes.io/os=windows | tail -n +2 | cut -d' ' -f1)
  for n in $nodes
  do
    kubectl $schedule $n
  done
}

GET_CLUSTER_INFO_CMD="aws eks describe-cluster --name $CLUSTER_NAME --region $REGION"

if [[ -z "${ENDPOINT}" ]]; then
  CLUSTER_INFO=$($GET_CLUSTER_INFO_CMD)
else
  CLUSTER_INFO=$($GET_CLUSTER_INFO_CMD --endpoint $ENDPOINT)
fi

echo "Cordon off windows nodes"
toggle_windows_scheduling "cordon"

VPC_ID=$(echo $CLUSTER_INFO | jq -r '.cluster.resourcesVpcConfig.vpcId')
ACCOUNT_ID=$(aws sts get-caller-identity | jq -r '.Account')

echo "VPC ID: $VPC_ID"

ROLE_SEARCH_STR=$CLUSTER_NAME-.*-NodeInstanceRole
NODE_ROLE_NAME=$(aws iam list-roles | jq -r '.Roles[] | select(.RoleName|match('\"$ROLE_SEARCH_STR\"')) | .RoleName')

echo "Node Instance Role Name: $NODE_ROLE_NAME"

# Install appmesh CRDs
echo "Installing appmesh CRDs"
helm repo add eks https://aws.github.io/eks-charts
helm repo update
kubectl apply -k "github.com/aws/eks-charts/stable/appmesh-controller//crds?ref=master"

echo "Create namespace appmesh-system"
kubectl create ns appmesh-system || true

eksctl utils associate-iam-oidc-provider --region=$REGION \
    --cluster=$CLUSTER_NAME \
    --approve

curl -o controller-iam-policy.json https://raw.githubusercontent.com/aws/aws-app-mesh-controller-for-k8s/master/config/iam/controller-iam-policy.json

aws iam create-policy \
    --policy-name AWSAppMeshK8sControllerIAMPolicy \
    --policy-document file://controller-iam-policy.json || true

curl -o envoy-iam-policy.json https://raw.githubusercontent.com/aws/aws-app-mesh-controller-for-k8s/master/config/iam/envoy-iam-policy.json

aws iam create-policy \
    --policy-name AWSAppMeshEnvoyIAMPolicy \
    --policy-document file://envoy-iam-policy.json || true

echo "Creating service account"
eksctl create iamserviceaccount --cluster $CLUSTER_NAME \
    --namespace appmesh-system \
    --name appmesh-controller \
    --attach-policy-arn arn:aws:iam::$ACCOUNT_ID:policy/AWSAppMeshK8sControllerIAMPolicy  \
    --override-existing-serviceaccounts \
    --approve

echo "Attaching Envoy policy to Node Instance Role"
aws iam attach-role-policy \
    --policy-arn arn:aws:iam::$ACCOUNT_ID:policy/AWSAppMeshEnvoyIAMPolicy \
    --role-name "$NODE_ROLE_NAME" || true

echo "Deploying appmesh-controller"
helm upgrade -i appmesh-controller eks/appmesh-controller \
    --namespace appmesh-system \
    --set region=$REGION \
    --set serviceAccount.create=false \
    --set serviceAccount.name=appmesh-controller

#Start the test
echo "Starting the ginkgo test suite" 

(cd $SCRIPT_DIR && CGO_ENABLED=0 GOOS=$OS_OVERRIDE ginkgo -v -r -- --cluster-kubeconfig=$KUBE_CONFIG_PATH --cluster-name=$CLUSTER_NAME --aws-region=$REGION --aws-vpc-id=$VPC_ID || true)

#Tear down local resources
echo "Detaching the Envoy IAM Policy from Node Instance Role"
aws iam detach-role-policy \
    --policy-arn arn:aws:iam::$ACCOUNT_ID:policy/AWSAppMeshEnvoyIAMPolicy \
    --role-name $NODE_ROLE_NAME || true

echo "Delete iamserviceaccount"    
eksctl delete iamserviceaccount --name appmesh-controller --namespace appmesh-system --cluster $CLUSTER_NAME --timeout=10m || true

#Delete AppMesh CRDs
echo "Deleting appmesh CRD's"
kubectl delete -k "github.com/aws/eks-charts/stable/appmesh-controller//crds?ref=master" --timeout=10m || true

echo "Uninstall appmesh-controller"
helm delete appmesh-controller -n appmesh-system --timeout=10m || true

echo "Delete namespace appmesh-system"
kubectl delete ns appmesh-system --timeout=10m || true

echo "Uncordon windows nodes"
toggle_windows_scheduling "uncordon"

echo "Successfully finished the test suite $(($SECONDS / 60)) minutes and $(($SECONDS % 60)) seconds"