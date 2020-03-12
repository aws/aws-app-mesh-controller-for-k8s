#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
GOPATH=${GOPATH:-$(go env GOPATH)}

go get github.com/vektra/mockery/.../

$GOPATH/bin/mockery -name CloudAPI \
    -dir ${SCRIPT_ROOT}/pkg/aws/ \
    -output ${SCRIPT_ROOT}/pkg/aws/mocks

$GOPATH/bin/mockery -name Interface \
    -dir ${SCRIPT_ROOT}/pkg/client/clientset/versioned \
    -output ${SCRIPT_ROOT}/pkg/client/clientset/versioned/mocks

$GOPATH/bin/mockery -name AppmeshV1beta1Interface \
    -dir ${SCRIPT_ROOT}/pkg/client/clientset/versioned/typed/appmesh/v1beta1 \
    -output ${SCRIPT_ROOT}/pkg/client/clientset/versioned/typed/appmesh/v1beta1/mocks

$GOPATH/bin/mockery -name VirtualNodeInterface \
    -dir ${SCRIPT_ROOT}/pkg/client/clientset/versioned/typed/appmesh/v1beta1 \
    -output ${SCRIPT_ROOT}/pkg/client/clientset/versioned/typed/appmesh/v1beta1/mocks

$GOPATH/bin/mockery -name VirtualServiceInterface \
    -dir ${SCRIPT_ROOT}/pkg/client/clientset/versioned/typed/appmesh/v1beta1 \
    -output ${SCRIPT_ROOT}/pkg/client/clientset/versioned/typed/appmesh/v1beta1/mocks

$GOPATH/bin/mockery -name VirtualNodeLister \
    -dir ${SCRIPT_ROOT}/pkg/client/listers/appmesh/v1beta1 \
    -output ${SCRIPT_ROOT}/pkg/client/listers/appmesh/v1beta1/mocks

# PodLister
PKG_ROOT="k8s.io/client-go"
PKG_PATH="/listers/core/v1"
PKG_VERSION=$(grep ${PKG_ROOT} go.sum | awk '{print $2}' | head -1)
PKG_ROOT_ABS_PATH=$(echo `go env GOPATH`"/pkg/mod/${PKG_ROOT}@${PKG_VERSION}")
$GOPATH/bin/mockery -name PodLister \
    -dir ${PKG_ROOT_ABS_PATH}${PKG_PATH} \
    -output ${SCRIPT_ROOT}/pkg/${PKG_ROOT}${PKG_PATH}/mocks
$GOPATH/bin/mockery -name PodNamespaceLister \
    -dir ${PKG_ROOT_ABS_PATH}${PKG_PATH} \
    -output ${SCRIPT_ROOT}/pkg/${PKG_ROOT}${PKG_PATH}/mocks
