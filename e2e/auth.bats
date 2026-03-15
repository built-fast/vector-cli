#!/usr/bin/env bats
# auth.bats - E2E tests for vector auth commands

load test_helper


# --- auth login --token ---

@test "auth login --token stores credentials" {
  run vector auth login --token test-token-12345
  assert_success

  # Verify credentials file was created
  [[ -f "$TEST_CONFIG_DIR/credentials.json" ]]

  # Verify stored token
  local stored_key
  stored_key=$(jq -r '.api_key' "$TEST_CONFIG_DIR/credentials.json")
  [[ "$stored_key" == "test-token-12345" ]]
}

@test "auth login --token with --no-json shows success message" {
  run vector auth login --token test-token --no-json
  assert_success
  assert_output_contains "Authenticated as"
}

@test "auth login --token overwrites existing credentials" {
  create_credentials "old-token"
  run vector auth login --token new-token-67890
  assert_success

  local stored_key
  stored_key=$(jq -r '.api_key' "$TEST_CONFIG_DIR/credentials.json")
  [[ "$stored_key" == "new-token-67890" ]]
}

@test "auth login --token sets file permissions to 0600" {
  run vector auth login --token secret-token
  assert_success

  local perms
  if [[ "$(uname)" == "Darwin" ]]; then
    perms=$(stat -f '%Lp' "$TEST_CONFIG_DIR/credentials.json")
  else
    perms=$(stat -c '%a' "$TEST_CONFIG_DIR/credentials.json")
  fi
  [[ "$perms" == "600" ]]
}


# --- auth login without token (non-TTY) ---

@test "auth login without token and without TTY fails" {
  # In BATS, stdin is not a TTY. Provide empty input via /dev/null.
  run vector auth login < /dev/null
  assert_failure
  assert_exit_code 2
  assert_output_contains "No API token provided"
}


# --- auth status ---

@test "auth status with stored credentials shows logged-in state" {
  create_credentials "valid-token"
  run vector auth status
  assert_success
  assert_output_contains "stored credentials"
}

@test "auth status without credentials fails with exit code 2" {
  # No credentials created — config dir is empty (except config.json)
  run vector auth status
  assert_failure
  assert_exit_code 2
  assert_output_contains "Not logged in"
}

@test "auth status with --token flag shows token source" {
  run vector auth status --token some-token
  assert_success
  assert_output_contains "--token flag"
}


# --- auth logout ---

@test "auth logout removes credentials file" {
  create_credentials "token-to-remove"
  [[ -f "$TEST_CONFIG_DIR/credentials.json" ]]

  run vector auth logout
  assert_success
  assert_output_contains "Logged out successfully"

  # Credentials file should be gone
  [[ ! -f "$TEST_CONFIG_DIR/credentials.json" ]]
}

@test "auth logout without credentials succeeds (idempotent)" {
  # No credentials file exists
  [[ ! -f "$TEST_CONFIG_DIR/credentials.json" ]]

  run vector auth logout
  assert_success
  assert_output_contains "Logged out successfully"
}


# --- VECTOR_API_KEY env var ---

@test "VECTOR_API_KEY env var is used when no stored credentials exist" {
  export VECTOR_API_KEY="env-token-abc"
  run vector auth status
  assert_success
  assert_output_contains "VECTOR_API_KEY env"
}

@test "VECTOR_API_KEY env var is overridden by --token flag" {
  export VECTOR_API_KEY="env-token"
  run vector auth status --token flag-token
  assert_success
  assert_output_contains "--token flag"
  assert_output_not_contains "VECTOR_API_KEY"
}
