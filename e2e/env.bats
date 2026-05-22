#!/usr/bin/env bats
# env.bats - E2E tests for vector env commands (environments, secrets, db promote)

load test_helper


# --- env list ---

@test "env list returns valid JSON (default non-TTY)" {
  create_credentials "test-token"
  run vector env list 01JTEST00000000000000000AA
  assert_success
  is_valid_json
}

@test "env list --no-json returns table output" {
  create_credentials "test-token"
  run vector env list 01JTEST00000000000000000AA --no-json
  assert_success
  assert_output_contains "ID"
  assert_output_contains "STATUS"
}

@test "env list --json returns valid JSON" {
  create_credentials "test-token"
  run vector env list 01JTEST00000000000000000AA --json
  assert_success
  is_valid_json
}


# --- env show ---

@test "env show returns valid JSON (default non-TTY)" {
  create_credentials "test-token"
  run vector env show 01JTEST00000000000000000AA
  assert_success
  is_valid_json
}

@test "env show --no-json returns key-value output" {
  create_credentials "test-token"
  run vector env show 01JTEST00000000000000000AA --no-json
  assert_success
  assert_output_contains "ID"
  assert_output_contains "Status"
}


# --- env create ---

@test "env create with required flags succeeds" {
  create_credentials "test-token"
  run vector env create 01JTEST00000000000000000AA --name staging --php-version 8.3 --custom-domain example.com
  assert_success
  is_valid_json
}

@test "env create --no-json returns key-value output" {
  create_credentials "test-token"
  run vector env create 01JTEST00000000000000000AA --name staging --php-version 8.3 --custom-domain example.com --no-json
  assert_success
  assert_output_contains "ID"
  assert_output_contains "Status"
}

@test "env create without --name fails with exit code 3" {
  create_credentials "test-token"
  run vector env create 01JTEST00000000000000000AA --php-version 8.3
  assert_failure
  assert_exit_code 3
}

@test "env create without --php-version fails with exit code 3" {
  create_credentials "test-token"
  run vector env create 01JTEST00000000000000000AA --name staging
  assert_failure
  assert_exit_code 3
}


# --- env update ---

@test "env update with flags succeeds" {
  create_credentials "test-token"
  run vector env update 01JTEST00000000000000000AA --tags "live,primary"
  assert_success
  is_valid_json
}

@test "env update --no-json returns key-value output" {
  create_credentials "test-token"
  run vector env update 01JTEST00000000000000000AA --tags "live,primary" --no-json
  assert_success
  assert_output_contains "ID"
  assert_output_contains "Name"
}


# --- env delete ---

@test "env delete with --force succeeds" {
  create_credentials "test-token"
  run vector env delete 01JTEST00000000000000000AA --force
  assert_success
}

@test "env delete without --force aborts in non-TTY" {
  create_credentials "test-token"
  run vector env delete 01JTEST00000000000000000AA < /dev/null
  assert_success
  assert_output_contains "Aborted"
}


# --- env secret list ---

@test "env secret list returns valid JSON (default non-TTY)" {
  create_credentials "test-token"
  run vector env secret list 01JTEST00000000000000000AA
  assert_success
  is_valid_json
}

@test "env secret list --no-json returns table output" {
  create_credentials "test-token"
  run vector env secret list 01JTEST00000000000000000AA --no-json
  assert_success
  assert_output_contains "ID"
  assert_output_contains "KEY"
}

@test "env secret list --json returns valid JSON" {
  create_credentials "test-token"
  run vector env secret list 01JTEST00000000000000000AA --json
  assert_success
  is_valid_json
}


# --- env secret show ---

@test "env secret show returns valid JSON (default non-TTY)" {
  create_credentials "test-token"
  run vector env secret show 01JTEST00000000000000SEC01
  assert_success
  is_valid_json
}

@test "env secret show --no-json returns key-value output" {
  create_credentials "test-token"
  run vector env secret show 01JTEST00000000000000SEC01 --no-json
  assert_success
  assert_output_contains "ID"
  assert_output_contains "Key"
}


# --- env secret create ---

@test "env secret create with required flags succeeds" {
  create_credentials "test-token"
  run vector env secret create 01JTEST00000000000000000AA \
    --key MY_SECRET_KEY --value my-secret-value
  assert_success
  is_valid_json
}

@test "env secret create --no-json returns key-value output" {
  create_credentials "test-token"
  run vector env secret create 01JTEST00000000000000000AA \
    --key MY_SECRET_KEY --value my-secret-value --no-json
  assert_success
  assert_output_contains "ID"
  assert_output_contains "Key"
}


# --- env secret update ---

@test "env secret update with flags succeeds" {
  create_credentials "test-token"
  run vector env secret update 01JTEST00000000000000SEC01 --value new-value
  assert_success
  is_valid_json
}

@test "env secret update --no-json returns key-value output" {
  create_credentials "test-token"
  run vector env secret update 01JTEST00000000000000SEC01 --value new-value --no-json
  assert_success
  assert_output_contains "ID"
  assert_output_contains "Key"
}


# --- env secret delete ---

@test "env secret delete with --force succeeds" {
  create_credentials "test-token"
  run vector env secret delete 01JTEST00000000000000SEC01 --force
  assert_success
}

@test "env secret delete without --force aborts in non-TTY" {
  create_credentials "test-token"
  run vector env secret delete 01JTEST00000000000000SEC01 < /dev/null
  assert_success
  assert_output_contains "Aborted"
}


# --- env db promote ---

@test "env db promote succeeds" {
  create_credentials "test-token"
  run vector env db promote 01JTEST00000000000000000AA
  assert_success
  is_valid_json
}

@test "env db promote --no-json returns key-value output" {
  create_credentials "test-token"
  run vector env db promote 01JTEST00000000000000000AA --no-json
  assert_success
  assert_output_contains "ID"
  assert_output_contains "Status"
}


# --- env db promote-status ---

@test "env db promote-status succeeds" {
  create_credentials "test-token"
  run vector env db promote-status 01JTEST00000000000000000AA 01JTEST0000000000000PROM01
  assert_success
  is_valid_json
}

@test "env db promote-status --no-json returns key-value output" {
  create_credentials "test-token"
  run vector env db promote-status 01JTEST00000000000000000AA 01JTEST0000000000000PROM01 --no-json
  assert_success
  assert_output_contains "ID"
  assert_output_contains "Status"
}


# --- auth required ---

@test "env list without auth fails with exit code 2" {
  run vector env list 01JTEST00000000000000000AA
  assert_failure
  assert_exit_code 2
}

@test "env show without auth fails with exit code 2" {
  run vector env show 01JTEST00000000000000000AA
  assert_failure
  assert_exit_code 2
}

@test "env create without auth fails with exit code 2" {
  run vector env create 01JTEST00000000000000000AA --name staging --php-version 8.3
  assert_failure
  assert_exit_code 2
}

@test "env secret list without auth fails with exit code 2" {
  run vector env secret list 01JTEST00000000000000000AA
  assert_failure
  assert_exit_code 2
}

@test "env db promote without auth fails with exit code 2" {
  run vector env db promote 01JTEST00000000000000000AA
  assert_failure
  assert_exit_code 2
}
