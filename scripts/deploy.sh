#!/bin/bash

set -e
set -o pipefail

if [[ -z "${AWS_ACCOUNT}" ]]; then
    echo "AWS_ACCOUNT must be set."
    exit 1
fi

if [[ -z "${AWS_REGION}" ]]; then
    echo "AWS_REGION must be set."
    exit 1
fi

DIR=$(cd "$(dirname "$0")"; pwd)/..
OUT_DIR="${DIR}/_output/deploy/"
mkdir -p ${OUT_DIR}

kubectl apply -f ${DIR}/deploy/all.yaml

# override controller deployment with dev deployment
eval "cat <<EOF
$(<${DIR}/deploy/controller-deployment.yaml.template)
EOF
" > ${OUT_DIR}/controller-deployment.yaml

kubectl apply -f ${OUT_DIR}/controller-deployment.yaml
