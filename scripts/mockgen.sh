#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_ROOT=$(dirname "${BASH_SOURCE[0]}")/..

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