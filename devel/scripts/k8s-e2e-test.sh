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

if [ $# -ne 2 ]; then
  echo "Usage: $0 [suite] ([directory prefix])"
  exit 1
fi

SUITE="${1:-none}"
DIRTAG="${2}"
SUITES=(
  "default"
  "serial"
#  "slow"
)
if [[ "${SUITES[*]}" =~ "${SUITE:-none}" ]]; then
  SUITES=("$SUITE")
elif [ "$SUITE" != "all" ]; then
  echo "Unknown test suite '$SUITE'"
  exit 1
fi

# Check that we're running in a kubernetes repository.
if [ ! -f LICENSE -o "$(md5sum LICENSE)" != "d6177d11bbe618cf4ac8a63a16d30f83  LICENSE" ]; then
  echo "It doesn't look like you're running in a kubernetes source repository."
  exit 1
fi

# Check that the e2e cluster is up.
if ! ISUP="$(go run hack/e2e.go -isup 2>&1)"; then
  echo "e2e cluster not up!"
  echo "$ISUP"
  exit 1
fi

OUTPUT_BASE="$HOME/logs/k8s-e2e"
DATE_STR="$(date +%y.%m.%d.%H%M)"
OUTPUT_DIR="${OUTPUT_BASE}/${DATE_STR}"
BRANCH="$(git rev-parse --abbrev-ref HEAD 2> /dev/null)"
OUTPUT_DIR="${OUTPUT_BASE}/${DIRTAG}_${DATE_STR}_${BRANCH}"

if [ -d $OUTPUT_DIR ]; then
  echo "Output directory exists! $OUTPUT_DIR"
  exit 1
fi

mkdir -p "$OUTPUT_DIR"

INFO_FILE="$OUTPUT_DIR/info.txt"
( # Write info log.
  echo "Running Kubernetes E2E: $SUITE"
  date
  echo
  echo "PWD = $PWD"
  echo "BRANCH = $BRANCH"
  echo; echo ----; echo
  echo "$ git status"
  git status
  echo; echo ----; echo
  echo "$ git log -n 5"
  git log -n 5
  echo; echo ----; echo
  echo "e2e status:"
  hack/e2e-internal/e2e-status.sh
) |& tee "$OUTPUT_DIR/info.txt"

if [ $? -ne 0 ]; then
  echo "Error gathering info"
  exit 1
fi

for SUITE in ${SUITES[@]}; do
  echo -e "\n\nRunning '$SUITE' test suite ..."

  case $SUITE in

    default)
      FOCUS=
      SKIP='\[Slow\]|\[Serial\]|\[Disruptive\]|\[Flaky\]|\[Feature:.+\]'
      ;;

    serial)
      FOCUS='\[Serial\]|\[Disruptive\]'
      SKIP='\[Flaky\]|\[Feature:.+\]'
      ;;

    slow)
      FOCUS='\[Slow\]'
      SKIP='\[Serial\]|\[Disruptive\]|\[Flaky\]|\[Feature:.+\]'
      ;;

    *)
      echo "Unexpected suite '$SUITE'"
      SKIP='.*'

  esac

  GINKGO_TEST_ARGS=""
  if [ "$FOCUS" != "" ]; then
    GINKGO_TEST_ARGS="${GINKGO_TEST_ARGS} --ginkgo.focus=$FOCUS"
  fi
  if [ "$SKIP" != "" ]; then
    GINKGO_TEST_ARGS="${GINKGO_TEST_ARGS} --ginkgo.skip=$SKIP"
  fi

  OUTPUT_FILE="${OUTPUT_DIR}/${SUITE}.log"
  if [ "$GINKGO_TEST_ARGS" != "" ]; then
    CMD="go run hack/e2e.go -v --test --test_args='$GINKGO_TEST_ARGS'"
    echo "$CMD" > "$OUTPUT_FILE"
    eval $CMD |& tee -a "$OUTPUT_FILE"
  fi

done
