#!/bin/bash

set -e

DIR=$(cd "$(dirname "$0")"; pwd)/..

if [ -z "${RELEASE}" ]; then
    MESH_NAME=${MESH_NAME:-"color-mesh"}
    APP_NAMESPACE=${APP_NAMESPACE:-"appmesh-demo"}
    COLOR_GATEWAY_IMAGE=${COLOR_GATEWAY_IMAGE:-"970805265562.dkr.ecr.us-west-2.amazonaws.com/gateway:latest"}
    COLOR_TELLER_IMAGE=${COLOR_TELLER_IMAGE:-"970805265562.dkr.ecr.us-west-2.amazonaws.com/colorteller:latest"}
    EXAMPLES_OUT_DIR=${EXAMPLES_OUT_DIR:-"${DIR}/_output/examples/"}
    mkdir -p ${EXAMPLES_OUT_DIR}
else
    MESH_NAME="color-mesh"
    APP_NAMESPACE="appmesh-demo"
    COLOR_GATEWAY_IMAGE="970805265562.dkr.ecr.us-west-2.amazonaws.com/gateway:latest"
    COLOR_TELLER_IMAGE="970805265562.dkr.ecr.us-west-2.amazonaws.com/colorteller:latest"
    EXAMPLES_OUT_DIR="${DIR}/examples/"
fi

eval "cat <<EOF
$(<${DIR}/examples/color.yaml.template)
EOF
" > ${EXAMPLES_OUT_DIR}/color.yaml

if [ -z "${RELEASE}" ]; then
    kubectl apply -f ${EXAMPLES_OUT_DIR}/color.yaml
    exit 0
fi

