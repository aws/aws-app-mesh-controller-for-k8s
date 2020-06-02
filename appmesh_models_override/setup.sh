#!/bin/bash

mkdir -p ./vendor/github.com/aws

git clone --depth 1 git@github.com:aws/aws-sdk-go.git ./vendor/github.com/aws/aws-sdk-go/
API_PATH=./vendor/github.com/aws/aws-sdk-go/models/apis/appmesh/2019-01-25
cp appmesh_models_override/api-2.json $API_PATH/api-2.json
cp appmesh_models_override/docs-2.json $API_PATH/docs-2.json
cp appmesh_models_override/examples-1.json $API_PATH/examples-1.json
cp appmesh_models_override/paginators-1.json $API_PATH/paginators-1.json

pushd ./vendor/github.com/aws/aws-sdk-go
make generate
popd

# Use the vendored version of aws-sdk-go
go mod edit -replace github.com/aws/aws-sdk-go=./vendor/github.com/aws/aws-sdk-go
go mod tidy
