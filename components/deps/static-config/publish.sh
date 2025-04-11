#!/bin/bash

set -e;

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &> /dev/null && pwd)

repository="${1}"
version="${2}"
# replace forbidden characters for the tag
tag=$(echo "${version}" | sed 's/[^a-zA-Z0-9_.\-]/--/g')
revision=$(git rev-parse HEAD)
if [[ $(git status --porcelain) ]] ; then
  revision="${revision}+dirty"
fi

publish_oci() {
    local file="${1}"
    local component=$(basename "${file}" .wasm)

    wkg oci push \
        --annotation "org.opencontainers.image.title=${component}" \
        --annotation "org.opencontainers.image.version=${version}" \
        --annotation "org.opencontainers.image.source=https://github.com/reconcilerio/static-config" \
        --annotation "org.opencontainers.image.revision=${revision}" \
        --annotation "org.opencontainers.image.licenses=UNLICENSED" \
        "${repository}/${component}:${tag}" \
        "${file}"
}

publish_oci "${SCRIPT_DIR}/lib/factory.wasm"
