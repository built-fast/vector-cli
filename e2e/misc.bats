#!/usr/bin/env bats
# misc.bats - E2E tests for vector php-versions and mcp commands

load test_helper


# --- php-versions ---

@test "php-versions returns valid JSON (default non-TTY)" {
  create_credentials "test-token"
  run vector php-versions
  assert_success
  is_valid_json
}

@test "php-versions --no-json returns table output" {
  create_credentials "test-token"
  run vector php-versions --no-json
  assert_success
  assert_output_contains "VERSION"
}

@test "php-versions --json returns valid JSON" {
  create_credentials "test-token"
  run vector php-versions --json
  assert_success
  is_valid_json
}

@test "php-versions without auth fails with exit code 2" {
  run vector php-versions
  assert_failure
  assert_exit_code 2
}


# --- mcp setup help ---

@test "mcp setup --help shows usage" {
  run vector mcp setup --help
  assert_success
  assert_output_contains "Configure the Vector MCP server"
}

@test "mcp --help shows setup subcommand" {
  run vector mcp --help
  assert_success
  assert_output_contains "setup"
}

@test "mcp setup --help shows --target flag" {
  run vector mcp setup --help
  assert_success
  assert_output_contains "--target"
}

@test "mcp setup --help shows --force flag" {
  run vector mcp setup --help
  assert_success
  assert_output_contains "--force"
}

@test "mcp setup --help shows --global flag" {
  run vector mcp setup --help
  assert_success
  assert_output_contains "--global"
}

@test "mcp setup --global without --target code fails" {
  create_credentials "test-token"
  run vector mcp setup --global
  assert_failure
  assert_output_contains "only applies"
}
