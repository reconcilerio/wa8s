#!/bin/bash

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

# TODO sign locally built packages
apko publish "${SCRIPT_DIR}/apko.yaml" ghcr.io/reconcilerio/wa8s/wasmtime:latest --ignore-signatures --vcs=false

# crane tag ghcr.io/reconcilerio/wa8s/wasmtime:latest <version>
