#!/bin/bash

DIR=$(cd "$(dirname "$0")"; pwd)/..

kubectl apply -f $DIR/examples/color.yaml

