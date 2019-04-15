#!/bin/bash

while true; do
  for pod in $(kubectl get pods | grep -i running | awk '{print $1}'); do
    echo "Found running pod $pod"
    kubectl logs $pod
  done
  sleep 10s
done
