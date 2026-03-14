#!/usr/bin/env bats
# deploy.bats - E2E tests for vector deploy commands

load test_helper


# --- deploy list ---

@test "deploy list returns valid JSON (default non-TTY)" {
  create_credentials "test-token"
  run vector deploy list 01JTEST00000000000000000AA
  assert_success
  is_valid_json
}

@test "deploy list --no-json returns table output" {
  create_credentials "test-token"
  run vector deploy list 01JTEST00000000000000000AA --no-json
  assert_success
  assert_output_contains "ID"
  assert_output_contains "STATUS"
}

@test "deploy list --json returns valid JSON" {
  create_credentials "test-token"
  run vector deploy list 01JTEST00000000000000000AA --json
  assert_success
  is_valid_json
}


# --- deploy show ---

@test "deploy show returns valid JSON (default non-TTY)" {
  create_credentials "test-token"
  run vector deploy show 01JTEST00000000000000DEP01
  assert_success
  is_valid_json
}

@test "deploy show --no-json returns key-value output" {
  create_credentials "test-token"
  run vector deploy show 01JTEST00000000000000DEP01 --no-json
  assert_success
  assert_output_contains "ID"
  assert_output_contains "Status"
}


# --- deploy trigger ---

@test "deploy trigger succeeds" {
  create_credentials "test-token"
  run vector deploy trigger 01JTEST00000000000000000AA
  assert_success
  is_valid_json
}

@test "deploy trigger --no-json returns key-value output" {
  create_credentials "test-token"
  run vector deploy trigger 01JTEST00000000000000000AA --no-json
  assert_success
  assert_output_contains "ID"
  assert_output_contains "Status"
}

@test "deploy trigger with flags succeeds" {
  create_credentials "test-token"
  run vector deploy trigger 01JTEST00000000000000000AA --include-uploads --include-database=false
  assert_success
  is_valid_json
}


# --- deploy rollback ---

@test "deploy rollback succeeds" {
  create_credentials "test-token"
  run vector deploy rollback 01JTEST00000000000000000AA
  assert_success
  is_valid_json
}

@test "deploy rollback --no-json returns key-value output" {
  create_credentials "test-token"
  run vector deploy rollback 01JTEST00000000000000000AA --no-json
  assert_success
  assert_output_contains "ID"
  assert_output_contains "Status"
}

@test "deploy rollback with --target succeeds" {
  create_credentials "test-token"
  run vector deploy rollback 01JTEST00000000000000000AA --target 01JTEST00000000000000DEP01
  assert_success
  is_valid_json
}


# --- auth required ---

@test "deploy list without auth fails with exit code 2" {
  run vector deploy list 01JTEST00000000000000000AA
  assert_failure
  assert_exit_code 2
}

@test "deploy show without auth fails with exit code 2" {
  run vector deploy show 01JTEST00000000000000DEP01
  assert_failure
  assert_exit_code 2
}

@test "deploy trigger without auth fails with exit code 2" {
  run vector deploy trigger 01JTEST00000000000000000AA
  assert_failure
  assert_exit_code 2
}

@test "deploy rollback without auth fails with exit code 2" {
  run vector deploy rollback 01JTEST00000000000000000AA
  assert_failure
  assert_exit_code 2
}
