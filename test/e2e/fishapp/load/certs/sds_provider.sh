#!/bin/bash

set -e

deploySpire() {
    kubectl apply -f certs/spire_setup.yaml
}

deleteSpire() {
    kubectl delete -f certs/spire_setup.yaml
}

echo "Installing SPIRE Server and Agent"
if [ "$1" == "deploy" ]; then
  deploySpire
elif [ "$1" == "delete" ]; then
  deleteSpire
fi
