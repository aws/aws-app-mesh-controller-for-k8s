#!/usr/bin/env bash

set -ueo pipefail

AWS_DEFAULT_REGION=${AWS_DEFAULT_REGION:-"us-west-2"}
K8S_TESTER_BINARY=${K8S_TESTER_BINARY:-"/tmp/appmesh-e2e/bin/aws-k8s-tester"}

#######################################
# Initialize cluster package
# Globals:
#   K8S_TESTER_BINARY
# Arguments:
#   tester_version      the version of aws-k8s-tester
#
# sample: cluster::init v0.7.4
#######################################
cluster::init() {
    declare -r tester_version="$1"

    local tester_os_arch="$(go env GOOS)-$(go env GOARCH)"
    local tester_download_url="https://github.com/aws/aws-k8s-tester/releases/download/${tester_version}/aws-k8s-tester-${tester_version}-${tester_os_arch}"
    local tester_binary_dir=$(dirname $K8S_TESTER_BINARY)

    if [[ ! -d ${tester_binary_dir} ]]; then
        echo "Creating aws-k8s-tester directory ${tester_binary_dir}"

        if ! mkdir -p "${tester_binary_dir}"; then
            echo "Unable to create aws-k8s-tester directory ${tester_binary_dir}" >&2
            return 1
        fi
    fi

    if [[ ! -f ${K8S_TESTER_BINARY} ]]; then
        echo "Downloading aws-k8s-tester from ${tester_download_url} to ${K8S_TESTER_BINARY}"

        if ! curl -L -X GET ${tester_download_url} -o ${K8S_TESTER_BINARY}; then
            echo "Unable to download aws-k8s-tester binary" >&2
            return 1
        fi
        if ! chmod u+x ${K8S_TESTER_BINARY}; then
            echo "Unable to add execute permission for aws-k8s-tester binary" >&2
            return 1
        fi
    fi

    return 0
}

#######################################
# Create k8s cluster.
# Globals:
#   AWS_DEFAULT_REGION
#   K8S_TESTER_BINARY
# Arguments:
#   cluster_config
#   kubeconfig
#   cluster_name
#   k8s_version
#   node_instance_type
#   node_count
#
# sample: cluster::create appmesh-e2e.yaml appmesh-e2e.kubeconfig appmesh-e2e 1.15 m5.xlarge 4
#######################################
cluster::create() {
    declare -r cluster_config="$1" kubeconfig="$2" cluster_name="$3" k8s_version="$4" node_instance_type="$5" node_count="$6"

    echo "Creating cluster config for ${cluster_config}"
    if ! AWS_K8S_TESTER_EKS_REGION=${AWS_DEFAULT_REGION} \
          AWS_K8S_TESTER_EKS_KUBECONFIG_PATH=${kubeconfig} \
          AWS_K8S_TESTER_EKS_PARAMETERS_VERSION=${k8s_version} \
          AWS_K8S_TESTER_EKS_PARAMETERS_ENCRYPTION_CMK_CREATE=false \
          AWS_K8S_TESTER_EKS_ADD_ON_MANAGED_NODE_GROUPS_ENABLE=true \
          AWS_K8S_TESTER_EKS_ADD_ON_MANAGED_NODE_GROUPS_MNGS={\"${cluster_name}-mng\":{\"name\":\"${cluster_name}-mng\",\"tags\":{\"group\":\"aws-app-mesh-controller-for-k8s\"},\"ami-type\":\"AL2_x86_64\",\"asg-min-size\":${node_count},\"asg-max-size\":${node_count},\"asg-desired-capacity\":${node_count},\"instance-types\":[\"${node_instance_type}\"]}} \
          AWS_K8S_TESTER_EKS_ADD_ON_APP_MESH_ENABLE=true \
          AWS_K8S_TESTER_EKS_S3_BUCKET_NAME="" \
          AWS_K8S_TESTER_EKS_S3_BUCKET_CREATE=false \
          ${K8S_TESTER_BINARY} eks create config \
            --path ${cluster_config}; then
        echo "Unable to create cluster config for ${cluster_config}"
        return 1
    fi

    cat ${cluster_config}

    echo "Creating cluster for ${cluster_config}"
    if ! ${K8S_TESTER_BINARY} eks create cluster \
            --path ${cluster_config}; then
      echo "Unable to create cluster for ${cluster_config}"
      return 1
    fi

    return 0
}

#######################################
# Delete k8s cluster.
# Globals:
#   AWS_DEFAULT_REGION
#   K8S_TESTER_BINARY
# Arguments:
#    cluster_config
#
# sample: cluster::delete appmesh-e2e.yaml
#######################################
cluster::delete() {
    declare -r cluster_config="$1"

    echo "Deleting cluster for ${cluster_config}"
    if ! ${K8S_TESTER_BINARY} eks delete cluster \
           --path ${cluster_config}; then
        echo "Unable to delete cluster for ${cluster_config}"
    fi
}