#!/bin/bash

FOCUS=""

# Build dependencies first
make ginkgo generated_files || exit 1

function labelHost() {
  echo -e "\n\n\n\n[[[[[[  $1  ]]]]]]\n\n"
}

ITER=0
while true; do
  ITER=$((ITER + 1))

  echo -e "\n\n\n\n"
  echo "!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!"
  echo "!!  ITERATION -- $ITER"
  echo "!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!"

  labelHost "ContainerVM"
  make test-e2e-node-no-build REMOTE=true FOCUS="${FOCUS}" IMAGES=e2e-node-containervm-v20160321-image IMAGE_PROJECT=kubernetes-node-e2e-images

  labelHost "Ubuntu docker9"
  make test-e2e-node-no-build REMOTE=true FOCUS="${FOCUS}" IMAGES=e2e-node-ubuntu-trusty-docker9-v1-image IMAGE_PROJECT=kubernetes-node-e2e-images

  labelHost "Ubuntu docker 10"
  make test-e2e-node-no-build REMOTE=true FOCUS="${FOCUS}" IMAGES=e2e-node-ubuntu-trusty-docker10-v1-image IMAGE_PROJECT=kubernetes-node-e2e-images

  labelHost "CoreOS"
  make test-e2e-node-no-build REMOTE=true FOCUS="${FOCUS}" IMAGES=coreos-alpha-1122-0-0-v20160727 IMAGE_PROJECT=coreos-cloud INSTANCE_METADATA="user-data<test/e2e_node/jenkins/coreos-init.json"

  labelHost "GCI"
  make test-e2e-node-no-build REMOTE=true FOCUS="${FOCUS}" IMAGES=gci-dev-54-8743-3-0 IMAGE_PROJECT=google-containers METADATA="user-data<test/e2e_node/jenkins/gci-init.yaml"

done |& tee ~/logs/node-e2e-soak/soak.log
