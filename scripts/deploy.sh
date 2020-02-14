#!/bin/bash

set -e
set -o pipefail

IMAGE_URL=$1

if [[ -z "${IMAGE_URL}" ]]; then
    echo "USAGE ./scripts/deploy.sh <image-repo>:<version>"
    exit 1
fi

kubectl set image deployment/appmesh-controller \
    -n appmesh-system \
    appmesh-controller=${IMAGE_URL}
