#!/usr/bin/env bash
set -euo pipefail
PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." >/dev/null 2>&1 && pwd)"

ENVTEST_ASSETS_DIR="${ENVTEST_ASSETS_DIR:-"$PROJECT_DIR/testbin"}"

mkdir -p "${ENVTEST_ASSETS_DIR}"

SETUP_ENVTEST_PATH="${ENVTEST_ASSETS_DIR}/setup-envtest.sh"
if [[ ! -f "$SETUP_ENVTEST_PATH" ]]; then
  SETUP_ENVTEST_VERSION="v0.8.2"
  SETUP_ENVTEST_URL="https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/$SETUP_ENVTEST_VERSION/hack/setup-envtest.sh"
  curl -sSLo "$SETUP_ENVTEST_PATH" "$SETUP_ENVTEST_URL"
fi

# shellcheck disable=SC1090
source "$SETUP_ENVTEST_PATH"
fetch_envtest_tools "$ENVTEST_ASSETS_DIR"
setup_envtest_env "$ENVTEST_ASSETS_DIR"

exec "$@"
