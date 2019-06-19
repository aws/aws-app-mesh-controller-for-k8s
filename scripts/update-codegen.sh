#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
CODEGEN_PKG=${CODEGEN_PKG:-$(cd "${SCRIPT_ROOT}"; ls -d -1 ./vendor/k8s.io/code-generator 2>/dev/null || echo ../code-generator)}
CLIENT_PKG=github.com/aws/aws-app-mesh-controller-for-k8s/pkg/client
APIS_PKG=github.com/aws/aws-app-mesh-controller-for-k8s/pkg/apis

${CODEGEN_PKG}/generate-groups.sh all \
    ${CLIENT_PKG} \
    ${APIS_PKG} \
    appmesh:v1beta1 \
    --output-base "${SCRIPT_ROOT}/../../.." \
    --go-header-file ${SCRIPT_ROOT}/scripts/custom-boilerplate.go.txt
