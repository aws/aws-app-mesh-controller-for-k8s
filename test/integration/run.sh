#!/bin/bash

# This script run integration tests on the AWS AppMesh Controller

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
echo "Running AppMeshController integration test with the following variables
KUBE CONFIG: $KUBE_CONFIG_PATH
CLUSTER_NAME: $CLUSTER_NAME
REGION: $REGION
OS_OVERRIDE: $OS_OVERRIDE"

if [[ -z "${OS_OVERRIDE}" ]]; then
  OS_OVERRIDE=linux
fi

CLUSTER_INFO=$(aws eks describe-cluster --name $CLUSTER_NAME --region $REGION)

VPC_ID=$(echo $CLUSTER_INFO | jq -r '.cluster.resourcesVpcConfig.vpcId')
SERVICE_ROLE_ARN=$(echo $CLUSTER_INFO | jq -r '.cluster.roleArn')
ROLE_NAME=${SERVICE_ROLE_ARN##*/}

ACCOUNT_ID=$(aws sts get-caller-identity | jq -r '.Account')

NODE_GROUP_NAME=${CLUSTER_NAME}"ng"

NODE_INSTANCE_ROLE_ARN=$(aws eks describe-nodegroup --cluster-name $CLUSTER_NAME --nodegroup-name $NODE_GROUP_NAME | jq -r '.nodegroup.nodeRole')
NODE_ROLE_NAME=${NODE_INSTANCE_ROLE_ARN##*/}

echo "VPC ID: $VPC_ID, Service Role ARN: $SERVICE_ROLE_ARN, Role Name: $ROLE_NAME"
echo "Node Instance Role: $NODE_INSTANCE_ROLE_ARN, NODE_ROLE_NAME: $NODE_ROLE_NAME"

# Set up local resources
echo "Attaching IAM Policy to Cluster Service Role"
aws iam attach-role-policy \
    --policy-arn arn:aws:iam::aws:policy/AmazonEKSVPCResourceController \
    --role-name "$ROLE_NAME" > /dev/null

echo "Enabling Pod ENI on aws-node"
kubectl set env daemonset aws-node -n kube-system ENABLE_POD_ENI=true

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

echo "Attaching Controller and Envoy policies to Node Instance Role"
aws iam attach-role-policy \
    --policy-arn arn:aws:iam::$ACCOUNT_ID:policy/AWSAppMeshK8sControllerIAMPolicy \
    --role-name "$NODE_ROLE_NAME" || true

aws iam attach-role-policy \
    --policy-arn arn:aws:iam::$ACCOUNT_ID:policy/AWSAppMeshEnvoyIAMPolicy \
    --role-name "$NODE_ROLE_NAME" || true

echo "Deploying appmesh-controller"
helm upgrade -i appmesh-controller eks/appmesh-controller \
    --namespace appmesh-system

#Start the test
echo "Starting the ginkgo test suite" 

(cd $SCRIPT_DIR && CGO_ENABLED=0 GOOS=$OS_OVERRIDE ginkgo -v -r -- --cluster-kubeconfig=$KUBE_CONFIG_PATH --cluster-name=$CLUSTER_NAME --aws-region=$REGION --aws-vpc-id=$VPC_ID)

echo "Successfully finished the test suite"

#Tear down local resources
echo "Detaching the IAM Policy from Cluster Service Role"
aws iam detach-role-policy \
    --policy-arn arn:aws:iam::aws:policy/AmazonEKSVPCResourceController \
    --role-name $ROLE_NAME || true

echo "Disabling Pod ENI on aws-node"
kubectl set env daemonset aws-node -n kube-system ENABLE_POD_ENI=false

echo "Detaching the IAM Policies from Node Instance Role"
aws iam detach-role-policy \
    --policy-arn arn:aws:iam::$ACCOUNT_ID:policy/AWSAppMeshK8sControllerIAMPolicy \
    --role-name $NODE_ROLE_NAME || true

aws iam detach-role-policy \
    --policy-arn arn:aws:iam::$ACCOUNT_ID:policy/AWSAppMeshEnvoyIAMPolicy \
    --role-name $NODE_ROLE_NAME || true

#Delete AppMesh CRDs
echo "Deleting appmesh CRD's"
kubectl delete -k "github.com/aws/eks-charts/stable/appmesh-controller//crds?ref=master" 

echo "Uninstall appmesh-controller"
helm delete appmesh-controller -n appmesh-system

echo "Delete namespace appmesh-system"
kubectl delete ns appmesh-system