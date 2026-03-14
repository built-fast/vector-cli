#!/usr/bin/env bats
# event.bats - E2E tests for vector event commands

load test_helper


# --- event list ---

@test "event list returns valid JSON (default non-TTY)" {
  create_credentials "test-token"
  run vector event list
  assert_success
  is_valid_json
}

@test "event list --no-json returns table output" {
  create_credentials "test-token"
  run vector event list --no-json
  assert_success
  assert_output_contains "ID"
  assert_output_contains "EVENT"
}

@test "event list --json returns valid JSON" {
  create_credentials "test-token"
  run vector event list --json
  assert_success
  is_valid_json
}


# --- auth required ---

@test "event list without auth fails with exit code 2" {
  run vector event list
  assert_failure
  assert_exit_code 2
}
