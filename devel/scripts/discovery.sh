#!/bin/bash

# Prints out all the resources for each API group using the preferred versions.

set -o errexit
set -o pipefail
set -o nounset

if ! APIS=$(curl -s localhost:8001/apis | jq -r '.groups[].name'); then
  echo "Failed to query discovery endpoint."
  echo "Is 'kubectl proxy' running?"
  exit 1
fi

for API in $APIS; do
  echo "${API}:"
  VERSION=$(curl -s localhost:8001/apis/${API} | jq -r '.preferredVersion.version')
  RESOURCES=$(curl -s localhost:8001/apis/${API}/${VERSION} | jq -r '.resources[].name')
  for RES in $RESOURCES; do
    echo "  $RES"
  done
done
