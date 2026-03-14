#!/usr/bin/env bash
# E2E test runner for vector-cli
# Builds the binary, starts Prism mock server, runs BATS tests, cleans up.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
SPEC_FILE="$SCRIPT_DIR/openapi.yaml"
BINARY="$PROJECT_ROOT/bin/vector"

# --- Dependency checks ---

if ! command -v bats &>/dev/null; then
  echo "Error: bats not found. Install with: brew install bats-core (macOS) or from https://github.com/bats-core/bats-core" >&2
  exit 1
fi

if ! command -v npx &>/dev/null; then
  echo "Error: npx not found. Install Node.js to use Prism mock server." >&2
  exit 1
fi

if ! command -v jq &>/dev/null; then
  echo "Error: jq not found. Install with: brew install jq (macOS) or apt-get install jq (Linux)" >&2
  exit 1
fi

# --- Build ---

if [[ "${VECTOR_E2E_SKIP_BUILD:-}" != "1" ]]; then
  echo "Building vector binary..."
  make -C "$PROJECT_ROOT" build
else
  if [[ ! -x "$BINARY" ]]; then
    echo "Error: VECTOR_E2E_SKIP_BUILD=1 but no binary at $BINARY" >&2
    exit 1
  fi
  echo "Using pre-built binary: $BINARY"
fi

# --- Find available port ---

find_available_port() {
  if command -v python3 &>/dev/null; then
    python3 -c 'import socket; s=socket.socket(); s.bind(("",0)); print(s.getsockname()[1]); s.close()'
  elif command -v python &>/dev/null; then
    python -c 'import socket; s=socket.socket(); s.bind(("",0)); print(s.getsockname()[1]); s.close()'
  else
    # Fallback: pick a random port in the dynamic range
    echo $(( (RANDOM % 16383) + 49152 ))
  fi
}

PRISM_PORT=$(find_available_port)

# --- Prism lifecycle ---

PRISM_PID=""
PRISM_LOG="$(mktemp)"

cleanup() {
  if [[ -n "$PRISM_PID" ]]; then
    kill "$PRISM_PID" 2>/dev/null || true
    wait "$PRISM_PID" 2>/dev/null || true
    PRISM_PID=""
  fi
  rm -f "$PRISM_LOG"
}

trap cleanup EXIT INT TERM

echo "Starting Prism mock server on port $PRISM_PORT..."
npx @stoplight/prism-cli mock "$SPEC_FILE" \
  --port "$PRISM_PORT" \
  --host 127.0.0.1 \
  --dynamic \
  >"$PRISM_LOG" 2>&1 &
PRISM_PID=$!

# Wait for Prism to be ready (up to 30 seconds)
TRIES=0
MAX_TRIES=60
while ! curl -so /dev/null -w '' "http://127.0.0.1:$PRISM_PORT/" 2>/dev/null; do
  if ! kill -0 "$PRISM_PID" 2>/dev/null; then
    echo "Error: Prism failed to start. Log:" >&2
    cat "$PRISM_LOG" >&2
    exit 1
  fi
  TRIES=$((TRIES + 1))
  if [[ "$TRIES" -ge "$MAX_TRIES" ]]; then
    echo "Error: Prism did not become ready within 30s. Log:" >&2
    cat "$PRISM_LOG" >&2
    exit 1
  fi
  sleep 0.5
done

echo "Prism ready on http://127.0.0.1:$PRISM_PORT"

# Export for test_helper.bash
export PRISM_URL="http://127.0.0.1:$PRISM_PORT"
export VECTOR_BINARY="$BINARY"

# --- Run BATS ---

BATS_FILES=("$SCRIPT_DIR"/*.bats)

# If no .bats files exist, the glob returns the literal pattern
if [[ ${#BATS_FILES[@]} -eq 0 ]] || [[ "${BATS_FILES[0]}" == "$SCRIPT_DIR/*.bats" ]]; then
  echo "No .bats test files found — exiting cleanly."
  exit 0
fi

echo "Running BATS tests..."
bats "${BATS_FILES[@]}"
