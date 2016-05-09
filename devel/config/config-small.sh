#!/bin/bash

export KUBE_GCE_INSTANCE_PREFIX=k8s-stclair
export NUM_NODES=1

export KUBE_GCE_ZONE=us-east1-b
export MASTER_SIZE=n1-standard-1

# export KUBE_ENABLE_CLUSTER_MONITORING=none

export K8S_ENV=${KUBE_GCE_INSTANCE_PREFIX}-small
