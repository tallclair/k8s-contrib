#!/bin/bash

set -o errexit

for ARCH in amd64 arm arm64 ppc64le; do

  # docker pull gcr.io/google_containers/debian-iptables-${ARCH}:v5
  # docker tag gcr.io/google_containers/debian-iptables-${ARCH}:v5 gcr.io/google_containers/debian-iptables-${ARCH}:v5.0
  # gcloud docker -- push gcr.io/google_containers/debian-iptables-${ARCH}:v5.0

  make push ARCH=$ARCH
  docker tag gcr.io/google_containers/debian-iptables-${ARCH}:v5.1 gcr.io/google_containers/debian-iptables-${ARCH}:v5
  gcloud docker -- push gcr.io/google_containers/debian-iptables-${ARCH}:v5

done
