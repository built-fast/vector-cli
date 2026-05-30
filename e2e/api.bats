#!/usr/bin/env bats
# api.bats - E2E tests for the vector api passthrough command

load test_helper


# --- GET passthrough ---

@test "api sites returns valid JSON" {
  create_credentials "test-token"
  run vector api sites
  assert_success
  is_valid_json
}

@test "api with bare endpoint prepends the vector base path" {
  create_credentials "test-token"
  run vector api sites
  assert_success
  # The standard envelope carries a data array for list endpoints.
  assert_output_contains "data"
}

@test "api with leading-slash endpoint is sent verbatim" {
  create_credentials "test-token"
  run vector api /api/v1/vector/sites
  assert_success
  is_valid_json
}

@test "api sites --jq filters the full envelope" {
  create_credentials "test-token"
  run vector api sites --jq '.data'
  assert_success
}


# --- auth required ---

@test "api without auth fails with exit code 2" {
  run vector api sites
  assert_failure
  assert_exit_code 2
}
