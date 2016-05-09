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

# Script must be sourced to work properly.
if [[ ! "$-" =~ i ]]; then
  >&2 echo "Run with \`. ${BASH_SOURCE}\`"
  exit 1
fi

# Sandbox
function __godep-test-dir() {

# Check that we're in a descendant of GOPATH
if [[ ! "$PWD" =~ ^"$GOPATH" ]]; then
  >&2 echo "$PWD is not under \$GOPATH ($GOPATH)"
  return 1
fi

# Check that we have Godeps
if [[ ! -d "Godeps" ]]; then
  >&2 echo "Project does not contain Godeps"
  return 1
fi

local -r TESTDIR="$HOME/tmp/godep-test"

if [[ "$PWD" =~ ^"$TESTDIR" ]]; then
  >&2 echo "Already in a godep-test directory!"
  return 1
fi

# Delete existing godep test directory
if [[ -d "$TESTDIR" ]]; then
  echo "deleting $TESTDIR ..."
  rm -rf "$TESTDIR"
else
  echo "NOT deleting $TESTDIR"
fi

# Create new godep test directory
mkdir -p "$TESTDIR"

# Link current directory into godep test path
local -r PROJECT="$(basename $PWD)"
local -r PARENT="$(dirname ${PWD#$GOPATH})"
local -r DST="${TESTDIR}${PWD#$GOPATH}"
mkdir -p "${TESTDIR}${PARENT}"
ln -s "$PWD" "$DST"

# Update GOPATH
export GOPATH="$TESTDIR"
export __G_PROJECT="${PROJECT}-TEST"
export __G_PROJECT_DIR="$DST"
cd "$DST"
echo "$DST"

}

__godep-test-dir
