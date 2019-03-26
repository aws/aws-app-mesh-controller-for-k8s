#!/bin/bash

set -e
set -o pipefail

DIR=$(cd "$(dirname "$0")"; pwd)/..

if [[ -z "${AWS_ACCOUNT}" ]]; then
    echo "AWS_ACCOUNT must be set."
    exit 1
fi

if [[ -z "${AWS_REGION}" ]]; then
    echo "AWS_REGION must be set."
    exit 1
fi

eval "cat <<EOF
$(<${DIR}/deploy/controller-deployment.yaml.template)
EOF
" > _output/controller-deployment.yaml


kubectl apply -f ${DIR}/deploy
kubectl apply -f ${DIR}/_output