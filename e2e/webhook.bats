#!/usr/bin/env bats
# webhook.bats - E2E tests for vector webhook commands

load test_helper


# --- webhook list ---

@test "webhook list returns valid JSON (default non-TTY)" {
  create_credentials "test-token"
  run vector webhook list
  assert_success
  is_valid_json
}

@test "webhook list --no-json returns table output" {
  create_credentials "test-token"
  run vector webhook list --no-json
  assert_success
  assert_output_contains "ID"
  assert_output_contains "URL"
}

@test "webhook list --json returns valid JSON" {
  create_credentials "test-token"
  run vector webhook list --json
  assert_success
  is_valid_json
}


# --- webhook show ---

@test "webhook show returns valid JSON (default non-TTY)" {
  create_credentials "test-token"
  run vector webhook show 01JTEST0000000000000WEBHOOK01
  assert_success
  is_valid_json
}

@test "webhook show --no-json returns key-value output" {
  create_credentials "test-token"
  run vector webhook show 01JTEST0000000000000WEBHOOK01 --no-json
  assert_success
  assert_output_contains "ID"
  assert_output_contains "URL"
  assert_output_contains "Enabled"
}


# --- webhook create ---

@test "webhook create succeeds" {
  create_credentials "test-token"
  run vector webhook create --url https://example.com/webhook --events site.created,deployment.completed
  assert_success
}

@test "webhook create --no-json returns key-value output" {
  create_credentials "test-token"
  run vector webhook create --url https://example.com/webhook --events site.created --no-json
  assert_success
  assert_output_contains "ID"
  assert_output_contains "URL"
}

@test "webhook create with --type slack succeeds" {
  create_credentials "test-token"
  run vector webhook create --url https://hooks.slack.com/services/test --events site.created --type slack
  assert_success
}


# --- webhook update ---

@test "webhook update succeeds" {
  create_credentials "test-token"
  run vector webhook update 01JTEST0000000000000WEBHOOK01 --url https://example.com/new-webhook
  assert_success
}

@test "webhook update --no-json returns key-value output" {
  create_credentials "test-token"
  run vector webhook update 01JTEST0000000000000WEBHOOK01 --url https://example.com/new-webhook --no-json
  assert_success
  assert_output_contains "ID"
  assert_output_contains "URL"
}

@test "webhook update with --enabled succeeds" {
  create_credentials "test-token"
  run vector webhook update 01JTEST0000000000000WEBHOOK01 --enabled=false
  assert_success
}


# --- webhook delete ---

@test "webhook delete succeeds" {
  create_credentials "test-token"
  run vector webhook delete 01JTEST0000000000000WEBHOOK01
  assert_success
}

@test "webhook delete --no-json returns success message" {
  create_credentials "test-token"
  run vector webhook delete 01JTEST0000000000000WEBHOOK01 --no-json
  assert_success
  assert_output_contains "Webhook deleted successfully"
}


# --- auth required ---

@test "webhook list without auth fails with exit code 2" {
  run vector webhook list
  assert_failure
  assert_exit_code 2
}

@test "webhook show without auth fails with exit code 2" {
  run vector webhook show 01JTEST0000000000000WEBHOOK01
  assert_failure
  assert_exit_code 2
}

@test "webhook create without auth fails with exit code 2" {
  run vector webhook create --url https://example.com/webhook --events site.created
  assert_failure
  assert_exit_code 2
}

@test "webhook update without auth fails with exit code 2" {
  run vector webhook update 01JTEST0000000000000WEBHOOK01 --url https://example.com/new
  assert_failure
  assert_exit_code 2
}

@test "webhook delete without auth fails with exit code 2" {
  run vector webhook delete 01JTEST0000000000000WEBHOOK01
  assert_failure
  assert_exit_code 2
}
