#!/usr/bin/env bats
# restore.bats - E2E tests for vector restore commands

load test_helper


# --- restore list ---

@test "restore list returns valid JSON (default non-TTY)" {
  create_credentials "test-token"
  run vector restore list
  assert_success
  is_valid_json
}

@test "restore list --no-json returns table output" {
  create_credentials "test-token"
  run vector restore list --no-json
  assert_success
  assert_output_contains "ID"
  assert_output_contains "STATUS"
}

@test "restore list --json returns valid JSON" {
  create_credentials "test-token"
  run vector restore list --json
  assert_success
  is_valid_json
}


# --- restore show ---

@test "restore show returns valid JSON (default non-TTY)" {
  create_credentials "test-token"
  run vector restore show 01JTEST0000000000000RESTORE01
  assert_success
  is_valid_json
}

@test "restore show --no-json returns key-value output" {
  create_credentials "test-token"
  run vector restore show 01JTEST0000000000000RESTORE01 --no-json
  assert_success
  assert_output_contains "ID"
  assert_output_contains "Status"
}


# --- restore create ---

@test "restore create succeeds" {
  create_credentials "test-token"
  run vector restore create 01JTEST000000000000BACKUP01
  assert_success
}

@test "restore create --no-json returns text output" {
  create_credentials "test-token"
  run vector restore create 01JTEST000000000000BACKUP01 --no-json
  assert_success
  assert_output_contains "Restore initiated"
}

@test "restore create with flags succeeds" {
  create_credentials "test-token"
  run vector restore create 01JTEST000000000000BACKUP01 --drop-tables --disable-foreign-keys
  assert_success
}


# --- auth required ---

@test "restore list without auth fails with exit code 2" {
  run vector restore list
  assert_failure
  assert_exit_code 2
}

@test "restore show without auth fails with exit code 2" {
  run vector restore show 01JTEST0000000000000RESTORE01
  assert_failure
  assert_exit_code 2
}

@test "restore create without auth fails with exit code 2" {
  run vector restore create 01JTEST000000000000BACKUP01
  assert_failure
  assert_exit_code 2
}
