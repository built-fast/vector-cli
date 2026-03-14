#!/usr/bin/env bats
# output.bats - E2E tests for output formatting, global flags, and error handling

load test_helper


# --- --json flag forces JSON output ---

@test "--json flag forces JSON output on site list" {
  create_credentials "test-token"
  run vector site list --json
  assert_success
  is_valid_json
}

@test "--json flag forces JSON output on auth status" {
  create_credentials "test-token"
  run vector auth status --json
  assert_success
  is_valid_json
}


# --- --no-json flag forces table output ---

@test "--no-json flag forces table output on site list" {
  create_credentials "test-token"
  run vector site list --no-json
  assert_success
  assert_output_contains "ID"
  assert_output_contains "STATUS"
  # Table output should not be valid JSON
  ! is_valid_json
}

@test "--no-json flag forces table output on auth status" {
  create_credentials "test-token"
  run vector auth status --no-json
  assert_success
  assert_output_contains "Token source"
  assert_output_contains "API URL"
  ! is_valid_json
}


# --- Piped output (non-TTY) defaults to JSON ---

@test "non-TTY output defaults to JSON for site list" {
  create_credentials "test-token"
  # BATS runs without a TTY, so default output should be JSON
  run vector site list
  assert_success
  is_valid_json
}

@test "non-TTY output defaults to JSON for account show" {
  create_credentials "test-token"
  run vector account show
  assert_success
  is_valid_json
}


# --- Error responses include structured error messages on stderr ---

@test "auth error includes structured error message" {
  # No credentials — should fail with auth error
  run vector site list
  assert_failure
  assert_output_contains "Error:"
  assert_output_contains "Authentication required"
}

@test "validation error includes structured error message" {
  create_credentials "test-token"
  # site create without --customer-id triggers client-side validation
  run vector site create
  assert_failure
  assert_output_contains "Error:"
  assert_output_contains "--customer-id is required"
}

@test "network error includes structured error message" {
  # Point at a non-existent server
  create_config "http://127.0.0.1:1"
  create_credentials "test-token"
  run vector auth login --token test-token
  assert_failure
  assert_output_contains "Error:"
  assert_output_contains "Network error"
}


# --- --version prints version string ---

@test "--version prints version string" {
  run vector --version
  assert_success
  assert_output_contains "vector v"
}

@test "--version output contains build info" {
  run vector --version
  assert_success
  # Format: "vector v<version> (<commit>) built <date>"
  assert_output_contains "built"
}


# --- --help prints help text ---

@test "--help prints help text" {
  run vector --help
  assert_success
  assert_output_contains "Usage:"
  assert_output_contains "vector"
  assert_output_contains "Available Commands:"
}

@test "--help shows global flags" {
  run vector --help
  assert_success
  assert_output_contains "--json"
  assert_output_contains "--no-json"
  assert_output_contains "--token"
  assert_output_contains "--version"
}

@test "--help shows available command groups" {
  run vector --help
  assert_success
  assert_output_contains "auth"
  assert_output_contains "site"
  assert_output_contains "env"
  assert_output_contains "deploy"
}

@test "subcommand --help shows command usage" {
  run vector site --help
  assert_success
  assert_output_contains "Usage:"
  assert_output_contains "site"
}


# --- Invalid commands show usage hint and exit non-zero ---

@test "invalid command exits non-zero" {
  run vector nonexistentcommand
  assert_failure
}

@test "invalid command shows error message" {
  run vector nonexistentcommand
  assert_failure
  assert_output_contains "Error:"
  assert_output_contains "unknown command"
}

@test "invalid subcommand shows help text" {
  # Cobra shows help for unknown subcommands of command groups (exit 0)
  run vector site nonexistentsubcmd
  assert_success
  assert_output_contains "Usage:"
  assert_output_contains "Available Commands:"
}


# --- Exit codes match expected values ---

# Exit code 1 = generic/config errors
@test "exit code 1 for unknown command" {
  run vector nonexistentcommand
  assert_exit_code 1
}

# Exit code 2 = auth errors
@test "exit code 2 for unauthenticated request" {
  # No credentials created
  run vector site list
  assert_exit_code 2
}

@test "exit code 2 for auth status without credentials" {
  run vector auth status
  assert_exit_code 2
}

# Exit code 3 = validation errors
@test "exit code 3 for missing required flag" {
  create_credentials "test-token"
  # site create requires --customer-id
  run vector site create
  assert_exit_code 3
}

@test "exit code 3 for conflicting flags" {
  create_credentials "test-token"
  # env update with both --custom-domain and --clear-custom-domain
  run vector env update 01JTEST00000000000000000AA --custom-domain foo.com --clear-custom-domain
  assert_exit_code 3
}

# Exit code 4 = not found (404 from API)
@test "exit code 4 for API 404 response" {
  create_credentials "test-token"
  # Point API URL at Prism with a prefix that won't match any spec route
  # Prism returns 404 for unmatched routes
  create_config "$PRISM_URL/nonexistent"
  run vector site list
  assert_exit_code 4
}

# Exit code 5 = server/network errors
@test "exit code 5 for network error on auth login" {
  # Point at a port with nothing listening
  create_config "http://127.0.0.1:1"
  create_credentials "test-token"
  run vector auth login --token test-token
  assert_exit_code 5
}

@test "exit code 5 for network error on auth status" {
  create_config "http://127.0.0.1:1"
  create_credentials "test-token"
  run vector auth status
  assert_exit_code 5
}
