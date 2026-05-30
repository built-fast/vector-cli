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


# --- method selection & request body ---

@test "api POST with typed fields auto-selects POST and creates a site" {
  create_credentials "test-token"
  run vector api sites -f your_customer_id=cust_123 -f dev_php_version=8.3
  assert_success
  is_valid_json
}

@test "api POST with a raw body from stdin succeeds" {
  create_credentials "test-token"
  run bash -c 'echo "{\"your_customer_id\":\"cust_123\",\"dev_php_version\":\"8.3\"}" | '"$VECTOR_BINARY"' api sites -X POST --input -'
  assert_success
  is_valid_json
}

@test "api with both --input and -f fails with exit code 3" {
  create_credentials "test-token"
  run vector api sites --input - -f name=a
  assert_failure
  assert_exit_code 3
}


# --- custom request headers ---

@test "api sends a custom request header" {
  create_credentials "test-token"
  run vector api sites -H 'X-Custom: hello'
  assert_success
  is_valid_json
}

@test "api with a malformed header fails with exit code 3" {
  create_credentials "test-token"
  run vector api sites -H no-colon-here
  assert_failure
  assert_exit_code 3
}


# --- auth required ---

@test "api without auth fails with exit code 2" {
  run vector api sites
  assert_failure
  assert_exit_code 2
}
