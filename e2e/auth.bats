#!/usr/bin/env bats
# auth.bats - E2E tests for vector auth commands

load test_helper


# --- auth login --token ---

@test "auth login --token succeeds with valid token" {
  run vector auth login --token test-token-12345
  assert_success
}

@test "auth login --token with --no-json shows success message" {
  run vector auth login --token test-token --no-json
  assert_success
  assert_output_contains "Authenticated as"
}

@test "auth login --token overwrites existing token" {
  run vector auth login --token old-token
  assert_success

  run vector auth login --token new-token-67890
  assert_success
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

@test "auth status with --token flag shows token source" {
  run vector auth status --token some-token
  assert_success
  assert_output_contains "flag"
}

@test "auth status without credentials fails with exit code 2" {
  # No credentials — config dir is empty (except config.json)
  run vector auth status
  assert_failure
  assert_exit_code 2
  assert_output_contains "Not logged in"
}


# --- auth logout ---

@test "auth logout succeeds" {
  run vector auth logout
  assert_success
  assert_output_contains "Logged out successfully"
}

@test "auth logout without credentials succeeds (idempotent)" {
  run vector auth logout
  assert_success
  assert_output_contains "Logged out successfully"
}


# --- VECTOR_API_KEY env var ---

@test "VECTOR_API_KEY env var is used when no stored credentials exist" {
  export VECTOR_API_KEY="env-token-abc"
  run vector auth status
  assert_success
  assert_output_contains "env"
}

@test "VECTOR_API_KEY env var is overridden by --token flag" {
  export VECTOR_API_KEY="env-token"
  run vector auth status --token flag-token
  assert_success
  assert_output_contains "flag"
}
