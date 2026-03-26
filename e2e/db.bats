#!/usr/bin/env bats
# db.bats - E2E tests for vector db commands

load test_helper


# --- db export create ---

@test "db export create succeeds" {
  create_credentials "test-token"
  run vector db export create 01JTEST00000000000000SITE01
  assert_success
}

@test "db export create --no-json returns text output" {
  create_credentials "test-token"
  run vector db export create 01JTEST00000000000000SITE01 --no-json
  assert_success
  assert_output_contains "Export started"
}

@test "db export create with --format succeeds" {
  create_credentials "test-token"
  run vector db export create 01JTEST00000000000000SITE01 --format sql
  assert_success
}


# --- db export status ---

@test "db export status returns valid JSON (default non-TTY)" {
  create_credentials "test-token"
  run vector db export status 01JTEST00000000000000SITE01 01JTEST0000000000000EXPORT01
  assert_success
  is_valid_json
}

@test "db export status --no-json returns key-value output" {
  create_credentials "test-token"
  run vector db export status 01JTEST00000000000000SITE01 01JTEST0000000000000EXPORT01 --no-json
  assert_success
  assert_output_contains "Export ID"
  assert_output_contains "Status"
}


# --- db import-session create ---

@test "db import-session create returns valid JSON (default non-TTY)" {
  create_credentials "test-token"
  run vector db import-session create 01JTEST00000000000000SITE01 --content-length 52428800
  assert_success
  is_valid_json
}

@test "db import-session create --no-json returns key-value output" {
  create_credentials "test-token"
  run vector db import-session create 01JTEST00000000000000SITE01 --content-length 52428800 --no-json
  assert_success
  assert_output_contains "Import ID"
  assert_output_contains "Status"
}

@test "db import-session create with flags succeeds" {
  create_credentials "test-token"
  run vector db import-session create 01JTEST00000000000000SITE01 --content-length 1024 --filename test.sql --drop-tables
  assert_success
}


# --- db import-session run ---

@test "db import-session run returns valid JSON (default non-TTY)" {
  create_credentials "test-token"
  run vector db import-session run 01JTEST00000000000000SITE01 01JTEST000000000000IMPORT01
  assert_success
  is_valid_json
}

@test "db import-session run --no-json returns key-value output" {
  create_credentials "test-token"
  run vector db import-session run 01JTEST00000000000000SITE01 01JTEST000000000000IMPORT01 --no-json
  assert_success
  assert_output_contains "Import ID"
  assert_output_contains "Status"
}


# --- db import-session status ---

@test "db import-session status returns valid JSON (default non-TTY)" {
  create_credentials "test-token"
  run vector db import-session status 01JTEST00000000000000SITE01 01JTEST000000000000IMPORT01
  assert_success
  is_valid_json
}

@test "db import-session status --no-json returns key-value output" {
  create_credentials "test-token"
  run vector db import-session status 01JTEST00000000000000SITE01 01JTEST000000000000IMPORT01 --no-json
  assert_success
  assert_output_contains "Import ID"
  assert_output_contains "Status"
}


# --- auth required ---

@test "db export create without auth fails with exit code 2" {
  run vector db export create 01JTEST00000000000000SITE01
  assert_failure
  assert_exit_code 2
}

@test "db export status without auth fails with exit code 2" {
  run vector db export status 01JTEST00000000000000SITE01 01JTEST0000000000000EXPORT01
  assert_failure
  assert_exit_code 2
}

@test "db import-session create without auth fails with exit code 2" {
  run vector db import-session create 01JTEST00000000000000SITE01 --content-length 1024
  assert_failure
  assert_exit_code 2
}

@test "db import-session run without auth fails with exit code 2" {
  run vector db import-session run 01JTEST00000000000000SITE01 01JTEST000000000000IMPORT01
  assert_failure
  assert_exit_code 2
}

@test "db import-session status without auth fails with exit code 2" {
  run vector db import-session status 01JTEST00000000000000SITE01 01JTEST000000000000IMPORT01
  assert_failure
  assert_exit_code 2
}
