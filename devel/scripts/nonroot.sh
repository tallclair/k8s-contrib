#!/bin/bash

REAL_IMG=gcr.io/stclair-k8s-ext/busybox-nonroot:real
FAKE_IMG=gcr.io/stclair-k8s-ext/busybox-nonroot:fake
PUSH_IMG=gcr.io/stclair-k8s-ext/busybox-nonroot:latest

i=0
while true; do

  (( i++ ))
  echo "[$i] Republishing real non-root"

  docker tag $REAL_IMG $PUSH_IMG
  docker push $PUSH_IMG

  echo "[$i] Republishing fake non-root"

  docker tag $FAKE_IMG $PUSH_IMG
  docker push $PUSH_IMG

done
