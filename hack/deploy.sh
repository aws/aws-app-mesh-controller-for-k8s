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

mkdir -p _output/
sed \
    -e "s/{{AWS_ACCOUNT}}/${AWS_ACCOUNT}/g" \
    -e "s/{{AWS_REGION}}/${AWS_REGION}/g" \
    ${DIR}/deploy/controller-deployment.yaml.template \
    > ${DIR}/_output/controller-deployment.yaml

kubectl apply -f ${DIR}/deploy
kubectl apply -f ${DIR}/_output