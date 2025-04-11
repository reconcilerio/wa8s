#!/bin/bash

set -e;

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &> /dev/null && pwd)

# Useful for debugging:
# export CARGO_PROFILE_RELEASE_DEBUG=2
# export WIT_BINDGEN_DEBUG=1
export RUSTFLAGS="-Zoom=panic"

mkdir -p "${SCRIPT_DIR}/lib"

wasm-tools component wit --wasm "${SCRIPT_DIR}/wit" -o "${SCRIPT_DIR}/lib/package.wasm"

cargo build -p adapter --target wasm32-unknown-unknown --release -Z build-std=std,panic_abort -Z build-std-features=panic_immediate_abort
cp "${SCRIPT_DIR}/target/wasm32-unknown-unknown/release/adapter.wasm" "${SCRIPT_DIR}/lib"

cargo component build -p factory --target wasm32-unknown-unknown --release
cp "${SCRIPT_DIR}/target/wasm32-unknown-unknown/release/factory.wasm" "${SCRIPT_DIR}/lib"

cargo build --release

