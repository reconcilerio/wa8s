#!/bin/bash

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

melange build "${SCRIPT_DIR}/wrpc.yaml" \
    --empty-workspace \
    --apk-cache-dir "${SCRIPT_DIR}/packages" \
    --keyring-append https://packages.wolfi.dev/os/wolfi-signing.rsa.pub

# TODO sign locally built packages
apko publish "${SCRIPT_DIR}/apko.yaml" ghcr.io/reconcilerio/wa8s/wasmtime:latest --ignore-signatures --vcs false

# crane tag ghcr.io/reconcilerio/wa8s/wasmtime:latest 30
