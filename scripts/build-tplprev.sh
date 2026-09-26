#!/bin/bash
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

# wasm_exec.js moved from misc/wasm to lib/wasm in Go 1.24
GOROOT="$(go env GOROOT)"
WASM_EXEC="$GOROOT/lib/wasm/wasm_exec.js"
if [ ! -f "$WASM_EXEC" ]; then
  WASM_EXEC="$GOROOT/misc/wasm/wasm_exec.js"
fi

install -m 644 "$WASM_EXEC" ./docs/assets/wasm_exec.js

GOARCH=wasm GOOS=js go build -o ./docs/assets/tplprev.wasm ./tplprev
