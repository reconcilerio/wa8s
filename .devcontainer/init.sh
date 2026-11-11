!/usr/bin/env bash

# Setup script run once for new devcontainers to init the environment.

set -euo pipefail

mkdir -p "${HOME}/.cargo/bin"
PATH="${HOME}/.cargo/bin:${PATH}"
echo "PATH=\"${HOME}/.cargo/bin:\${PATH}\"" >> ~/.bashrc
echo "PATH=\"${HOME}/.cargo/bin:\${PATH}\"" >> ~/.zshrc

curl -L --proto '=https' --tlsv1.2 -sSf https://raw.githubusercontent.com/cargo-bins/cargo-binstall/main/install-from-binstall-release.sh | bash

cargo-binstall -y wkg
cargo-binstall -y wasm-tools
