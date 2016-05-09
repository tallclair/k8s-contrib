#!/bin/bash

# Copyright 2016 The Kubernetes Authors All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit
set -o nounset
set -o pipefail

OUTPUT_DIR="~/tmp/logs/housekeep/12-8-1900c62"
OUTPUT_LABEL="hk10s"

DURATION_SECONDS=300

MANIFEST="${HOME}/k8s-devel/manifests/pause-rc.yaml"
RC="pause"
ADDON_COUNT="10"
SCALES=(10 35 50 100)

# Check node count
if [ ! $(kubectl get nodes --no-headers | wc -l) -eq 1 ]; then
  >&2 echo "Can only profile single node cluster"
  exit 1
fi

# Check kubectl proxy is running
if ! curl -sfG http://localhost:8001/api/v1 -o /dev/null; then
  >&2 echo "`kubectl proxy` must be running"
  exit 1
fi

# Check RC is running
if ! kubectl get rc $RC &> /dev/null; then
  kubectl create -f $MANIFEST
fi

NODE=$(kubectl get nodes -o=jsonpath='{.items[0].metadata.name}')

for SCALE in "${SCALES[@]}"; do
  echo "Preparing to profile with $SCALE pods"
  kubectl scale rc $RC --replicas=$((SCALE-ADDON_COUNT))
  # Wait for scale to take effect
  while true; do
    COUNT=$(kubectl get pods --all-namespaces -o=jsonpath='{.items[?(@.status.phase == "Running")].metadata.uid}' | wc -w)
    [ $COUNT -eq $SCALE ] && break
    echo "Waiting for $SCALE pods to be running; currently $COUNT"
    sleep 5
  done
  echo "$COUNT pods running; beginning profiling"

  # Profile!
  OUTPUT="${OUTPUT_DIR}/profile_${DURATION_SECONDS}s_${OUTPUT_LABEL}_${SCALE}pods.pprof"
  PPROF_URL="http://localhost:8001/api/v1/proxy/nodes/${NODE}:10250/debug/pprof/profile?seconds=${DURATION_SECONDS}"
  CMD="curl -G $PPROF_URL > $OUTPUT"
  echo $CMD
  bash -c "$CMD"
done
