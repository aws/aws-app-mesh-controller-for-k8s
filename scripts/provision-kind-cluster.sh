#!/usr/bin/env bash

# A script that provisions a KinD Kubernetes cluster for local development and
# testing

set -Eo pipefail

SCRIPTS_DIR=$(cd "$(dirname "$0")"; pwd)
ROOT_DIR="$SCRIPTS_DIR/.."

source "$SCRIPTS_DIR"/lib/common.sh

check_is_installed uuidgen
check_is_installed wget
check_is_installed docker
check_is_installed kind "You can install kind with the helper scripts/install-kind.sh"

OVERRIDE_PATH=0
KIND_CONFIG_FILE="$SCRIPTS_DIR/kind-two-node-cluster.yaml"

K8_1_26="kindest/node:v1.26.3@sha256:94eb63275ad6305210041cdb5aca87c8562cc50fa152dbec3fef8c58479db4ff"
K8_1_25="kindest/node:v1.25.8@sha256:b5ce984f5651f44457edf263c1fe93459df8d5d63db7f108ccf5ea4b8d4d9820"
K8_1_24="kindest/node:v1.24.12@sha256:0bdca26bd7fe65c823640b14253ea7bac4baad9336b332c94850f84d8102f873"
K8_1_23="kindest/node:v1.23.17@sha256:f935044f60483d33648d8c13decd79557cf3e916363a3c9ba7e82332cb249cba"
K8_1_22="kindest/node:v1.22.17@sha256:ed0f6a1cd1dcc0ff8b66257b3867e4c9e6a54adeb9ca31005f62638ad555315c"
K8_1_21="kindest/node:v1.21.14@sha256:75047f07ef306beff928fdc1f171a8b81fae1628f7515bdabc4fc9c31b698d6b"
K8_1_20="kindest/node:v1.20.15@sha256:cceb1bfeb9fe7c65e344a6372ef6ef4a349d6d7aefba621b822f44cff0070cfd"
K8_1_19="kindest/node:v1.19.4@sha256:796d09e217d93bed01ecf8502633e48fd806fe42f9d02fdd468b81cd4e3bd40b"

K8_VERSION="$K8_1_23"

USAGE="
Usage:
  $(basename "$0") CLUSTER_NAME [-v K8S_VERSION]

Provisions a KinD cluster for local development and testing.

Example: $(basename "$0") my-test -v 1.16

      Optional:
        -v          K8s version to use in this test
"

cluster_name="$1"
if [[ -z "$cluster_name" ]]; then
    echo "FATAL: required cluster name argument missing."
    echo "${USAGE}" 1>&2
    exit 1
fi

shift

# Process our input arguments
while getopts "b:i:v:k:" opt; do
  case ${opt} in
    v ) # K8s version to provision
        OPTARG="K8_$(echo "${OPTARG}" | sed 's/\./\_/g')"
        if [ ! -z ${OPTARG+x} ]; then
            K8_VERSION=${!OPTARG}
        else
            echo "K8s version not supported" 1>&2
            exit 2
        fi
      ;;
    \? )
        echo "${USAGE}" 1>&2
        exit
      ;;
  esac
done

TMP_DIR=$ROOT_DIR/build/tmp-$cluster_name
mkdir -p "${TMP_DIR}"

debug_msg "kind: using Kubernetes $K8_VERSION"
echo -n "creating kind cluster $cluster_name ... "
for i in $(seq 0 5); do
  if [[ -z $(kind get clusters 2>/dev/null | grep $cluster_name) ]]; then
      kind create cluster -q --name "$cluster_name" --image $K8_VERSION --config "$KIND_CONFIG_FILE" --kubeconfig $TMP_DIR/kubeconfig 1>&2 || :
  else
      break
  fi
done
echo "ok."

echo "$cluster_name" > $TMP_DIR/clustername

