#!/usr/bin/env bats
# archive.bats - E2E tests for vector archive commands

load test_helper


# --- archive import help ---

@test "archive import --help shows usage" {
  run vector archive import --help
  assert_success
  assert_output_contains "Import a site archive from a local file"
}

@test "archive --help shows import subcommand" {
  run vector archive --help
  assert_success
  assert_output_contains "import"
}


# --- auth required ---

@test "archive import without auth fails with exit code 2" {
  # Create a dummy file to pass argument validation
  local tmpfile="$TEST_TEMP_DIR/test-archive.tar.gz"
  echo "dummy" > "$tmpfile"
  run vector archive import 01JTEST00000000000000SITE01 "$tmpfile"
  assert_failure
  assert_exit_code 2
}
