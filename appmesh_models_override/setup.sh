#!/bin/bash

set -e

mkdir -p ./vendor/github.com/aws

SDK_VENDOR_PATH=./vendor/github.com/aws/aws-sdk-go
rm -rf $SDK_VENDOR_PATH
git clone --depth 1 https://github.com/aws/aws-sdk-go.git $SDK_VENDOR_PATH
API_PATH=$SDK_VENDOR_PATH/models/apis/appmesh/2019-01-25
SERVICE_NAME=$([ "$APPMESH_PREVIEW" == "1" ] && echo "appmesh-preview" || echo "appmesh" )
cat appmesh_models_override/api-2.json | jq "(.metadata | .endpointPrefix?, .signingName?) |= \"$SERVICE_NAME\"" > $API_PATH/api-2.json
cp appmesh_models_override/docs-2.json $API_PATH/docs-2.json
cp appmesh_models_override/examples-1.json $API_PATH/examples-1.json
cp appmesh_models_override/paginators-1.json $API_PATH/paginators-1.json

pushd ./vendor/github.com/aws/aws-sdk-go
make generate
popd

# Use the vendored version of aws-sdk-go
go mod edit -replace github.com/aws/aws-sdk-go=./vendor/github.com/aws/aws-sdk-go
go mod tidy
