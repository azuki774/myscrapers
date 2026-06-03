#!/usr/bin/env bash
set -euo pipefail

nix develop -c bash -lc '
  command -v go >/dev/null
  command -v node >/dev/null
  test -n "${PLAYWRIGHT_BROWSERS_PATH:-}"
'
