#!/usr/bin/env bats
# backup.bats - E2E tests for vector backup commands

load test_helper


# --- backup list ---

@test "backup list returns valid JSON (default non-TTY)" {
  create_credentials "test-token"
  run vector backup list
  assert_success
  is_valid_json
}

@test "backup list --no-json returns table output" {
  create_credentials "test-token"
  run vector backup list --no-json
  assert_success
  assert_output_contains "ID"
  assert_output_contains "STATUS"
}

@test "backup list --json returns valid JSON" {
  create_credentials "test-token"
  run vector backup list --json
  assert_success
  is_valid_json
}


# --- backup show ---

@test "backup show returns valid JSON (default non-TTY)" {
  create_credentials "test-token"
  run vector backup show 01JTEST000000000000BACKUP01
  assert_success
  is_valid_json
}

@test "backup show --no-json returns key-value output" {
  create_credentials "test-token"
  run vector backup show 01JTEST000000000000BACKUP01 --no-json
  assert_success
  assert_output_contains "ID"
  assert_output_contains "Status"
}


# --- backup create ---

@test "backup create with --site-id succeeds" {
  create_credentials "test-token"
  run vector backup create --site-id 01JTEST00000000000000SITE01
  assert_success
}

@test "backup create with --environment-id succeeds" {
  create_credentials "test-token"
  run vector backup create --environment-id 01JTEST00000000000000000AA
  assert_success
}

@test "backup create --no-json returns text output" {
  create_credentials "test-token"
  run vector backup create --site-id 01JTEST00000000000000SITE01 --no-json
  assert_success
  assert_output_contains "Backup created"
}

@test "backup create with --scope and --description succeeds" {
  create_credentials "test-token"
  run vector backup create --site-id 01JTEST00000000000000SITE01 --scope database --description "Test backup"
  assert_success
}

@test "backup create without --site-id or --environment-id fails" {
  create_credentials "test-token"
  run vector backup create
  assert_failure
}


# --- backup download create ---

@test "backup download create returns valid JSON (default non-TTY)" {
  create_credentials "test-token"
  run vector backup download create 01JTEST000000000000BACKUP01
  assert_success
  is_valid_json
}

@test "backup download create --no-json returns key-value output" {
  create_credentials "test-token"
  run vector backup download create 01JTEST000000000000BACKUP01 --no-json
  assert_success
  assert_output_contains "ID"
  assert_output_contains "Status"
}


# --- backup download status ---

@test "backup download status returns valid JSON (default non-TTY)" {
  create_credentials "test-token"
  run vector backup download status 01JTEST000000000000BACKUP01 01JTEST00000000000DOWNLOAD01
  assert_success
  is_valid_json
}

@test "backup download status --no-json returns key-value output" {
  create_credentials "test-token"
  run vector backup download status 01JTEST000000000000BACKUP01 01JTEST00000000000DOWNLOAD01 --no-json
  assert_success
  assert_output_contains "ID"
  assert_output_contains "Status"
}


# --- auth required ---

@test "backup list without auth fails with exit code 2" {
  run vector backup list
  assert_failure
  assert_exit_code 2
}

@test "backup show without auth fails with exit code 2" {
  run vector backup show 01JTEST000000000000BACKUP01
  assert_failure
  assert_exit_code 2
}

@test "backup create without auth fails with exit code 2" {
  run vector backup create --site-id 01JTEST00000000000000SITE01
  assert_failure
  assert_exit_code 2
}

@test "backup download create without auth fails with exit code 2" {
  run vector backup download create 01JTEST000000000000BACKUP01
  assert_failure
  assert_exit_code 2
}

@test "backup download status without auth fails with exit code 2" {
  run vector backup download status 01JTEST000000000000BACKUP01 01JTEST00000000000DOWNLOAD01
  assert_failure
  assert_exit_code 2
}
