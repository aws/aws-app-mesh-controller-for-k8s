#!/bin/bash

set -e

if [ -z "${AWS_ACCOUNT}" ]; then
    echo "AWS_ACCOUNT must be set."
    exit 1
fi

if [ -z "${AWS_REGION}" ]; then
    echo "AWS_REGION must be set."
    exit 1
fi

if [ -z "${MESH_NAME}" ]; then
    echo "MESH_NAME must be set."
    exit 1
fi

APP_NAMESPACE=${APP_NAMESPACE:-"appmesh-demo"}

DIR=$(cd "$(dirname "$0")"; pwd)/..
EXAMPLES_OUT_DIR="${DIR}/_output/examples/"
mkdir -p ${EXAMPLES_OUT_DIR}

COLOR_GATEWAY_IMAGE=${COLOR_GATEWAY_IMAGE:-"970805265562.dkr.ecr.us-west-2.amazonaws.com/gateway:latest"}
COLOR_TELLER_IMAGE=${COLOR_TELLER_IMAGE:-"970805265562.dkr.ecr.us-west-2.amazonaws.com/colorteller:latest"}

eval "cat <<EOF
$(<${DIR}/examples/color.yaml.template)
EOF
" > ${EXAMPLES_OUT_DIR}/color.yaml

kubectl apply -f ${EXAMPLES_OUT_DIR}/color.yaml