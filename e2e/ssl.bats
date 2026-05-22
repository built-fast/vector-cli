#!/usr/bin/env bats
# ssl.bats - E2E tests for vector ssl commands

load test_helper


# --- ssl status ---

@test "ssl status returns valid JSON (default non-TTY)" {
  create_credentials "test-token"
  run vector ssl status 01JTEST00000000000000000AA
  assert_success
  is_valid_json
}

@test "ssl status --no-json returns key-value output" {
  create_credentials "test-token"
  run vector ssl status 01JTEST00000000000000000AA --no-json
  assert_success
  assert_output_contains "Status"
  assert_output_contains "Production"
}

@test "ssl status --json returns valid JSON" {
  create_credentials "test-token"
  run vector ssl status 01JTEST00000000000000000AA --json
  assert_success
  is_valid_json
}


# --- ssl nudge ---

@test "ssl nudge returns valid JSON (default non-TTY)" {
  create_credentials "test-token"
  run vector ssl nudge 01JTEST00000000000000000AA
  assert_success
  is_valid_json
}

@test "ssl nudge --no-json returns key-value output" {
  create_credentials "test-token"
  run vector ssl nudge 01JTEST00000000000000000AA --no-json
  assert_success
  assert_output_contains "Status"
}

@test "ssl nudge with --retry succeeds" {
  create_credentials "test-token"
  run vector ssl nudge 01JTEST00000000000000000AA --retry
  assert_success
  is_valid_json
}


# --- auth required ---

@test "ssl status without auth fails with exit code 2" {
  run vector ssl status 01JTEST00000000000000000AA
  assert_failure
  assert_exit_code 2
}

@test "ssl nudge without auth fails with exit code 2" {
  run vector ssl nudge 01JTEST00000000000000000AA
  assert_failure
  assert_exit_code 2
}
