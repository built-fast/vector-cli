#!/usr/bin/env bash
# test_helper.bash - Shared test utilities for vector-cli E2E tests

# --- Setup / Teardown ---

setup() {
  # Create isolated temp config directory
  TEST_TEMP_DIR="$(mktemp -d)"
  TEST_CONFIG_DIR="$TEST_TEMP_DIR/config"
  mkdir -p "$TEST_CONFIG_DIR"

  # Point vector at the temp config directory
  export VECTOR_CONFIG_DIR="$TEST_CONFIG_DIR"

  # Set API URL to the Prism mock server
  export VECTOR_API_URL="$PRISM_URL"

  # Ensure the binary is on PATH
  export PATH="$(dirname "$VECTOR_BINARY"):$PATH"

  # Clear env vars that could interfere
  unset VECTOR_API_KEY
  unset XDG_CONFIG_HOME

  # Write config.json pointing at Prism
  create_config "$PRISM_URL"
}

teardown() {
  if [[ -d "${TEST_TEMP_DIR:-}" ]]; then
    rm -rf "$TEST_TEMP_DIR"
  fi
}

# --- Fixture helpers ---

# create_config API_URL
# Writes config.json with the given API URL.
create_config() {
  local api_url="${1:-$PRISM_URL}"
  cat > "$TEST_CONFIG_DIR/config.json" <<EOF
{
  "api_url": "$api_url"
}
EOF
}

# --- Assertions ---

assert_success() {
  if [[ "$status" -ne 0 ]]; then
    echo "Expected success (exit 0), got exit $status"
    echo "Output: $output"
    return 1
  fi
}

assert_failure() {
  if [[ "$status" -eq 0 ]]; then
    echo "Expected failure (non-zero exit), got exit 0"
    echo "Output: $output"
    return 1
  fi
}

assert_exit_code() {
  local expected="$1"
  if [[ "$status" -ne "$expected" ]]; then
    echo "Expected exit code $expected, got $status"
    echo "Output: $output"
    return 1
  fi
}

assert_output_contains() {
  local expected="$1"
  if [[ "$output" != *"$expected"* ]]; then
    echo "Expected output to contain: $expected"
    echo "Actual output: $output"
    return 1
  fi
}

assert_output_not_contains() {
  local unexpected="$1"
  if [[ "$output" == *"$unexpected"* ]]; then
    echo "Expected output NOT to contain: $unexpected"
    echo "Actual output: $output"
    return 1
  fi
}

is_valid_json() {
  echo "$output" | jq . &>/dev/null
}

assert_json_value() {
  local jq_path="$1"
  local expected="$2"
  local actual
  actual=$(echo "$output" | jq -r "$jq_path")

  if [[ "$actual" != "$expected" ]]; then
    echo "JSON path $jq_path: expected '$expected', got '$actual'"
    echo "Full output: $output"
    return 1
  fi
}
