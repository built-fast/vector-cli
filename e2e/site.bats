#!/usr/bin/env bats
# site.bats - E2E tests for vector site commands

load test_helper


# --- site list ---

@test "site list returns valid JSON (default non-TTY)" {
  create_credentials "test-token"
  run vector site list
  assert_success
  is_valid_json
}

@test "site list --no-json returns table output with site data" {
  create_credentials "test-token"
  run vector site list --no-json
  assert_success
  assert_output_contains "ID"
  assert_output_contains "STATUS"
}

@test "site list --json returns valid JSON" {
  create_credentials "test-token"
  run vector site list --json
  assert_success
  is_valid_json
}


# --- site show ---

@test "site show returns valid JSON (default non-TTY)" {
  create_credentials "test-token"
  run vector site show 01JTEST00000000000000000AA
  assert_success
  is_valid_json
}

@test "site show --no-json returns key-value output" {
  create_credentials "test-token"
  run vector site show 01JTEST00000000000000000AA --no-json
  assert_success
  assert_output_contains "ID"
  assert_output_contains "Status"
}


# --- site create ---

@test "site create with required flags succeeds" {
  create_credentials "test-token"
  run vector site create --customer-id cust-123 --php-version 8.3
  assert_success
  is_valid_json
}

@test "site create --no-json returns key-value output" {
  create_credentials "test-token"
  run vector site create --customer-id cust-123 --php-version 8.3 --no-json
  assert_success
  assert_output_contains "ID"
  assert_output_contains "Status"
}

@test "site create without --customer-id fails with exit code 3" {
  create_credentials "test-token"
  run vector site create
  assert_failure
  assert_exit_code 3
}


# --- site update ---

@test "site update with flags succeeds" {
  create_credentials "test-token"
  run vector site update 01JTEST00000000000000000AA --customer-id new-cust
  assert_success
  is_valid_json
}

@test "site update --no-json returns key-value output" {
  create_credentials "test-token"
  run vector site update 01JTEST00000000000000000AA --customer-id new-cust --no-json
  assert_success
  assert_output_contains "ID"
  assert_output_contains "Customer ID"
}


# --- site delete ---

@test "site delete with --force succeeds" {
  create_credentials "test-token"
  run vector site delete 01JTEST00000000000000000AA --force
  assert_success
}

@test "site delete without --force aborts in non-TTY" {
  create_credentials "test-token"
  run vector site delete 01JTEST00000000000000000AA < /dev/null
  assert_success
  assert_output_contains "Aborted"
}


# --- site clone ---

@test "site clone succeeds" {
  create_credentials "test-token"
  run vector site clone 01JTEST00000000000000000AA
  assert_success
  is_valid_json
}

@test "site clone --no-json returns key-value output" {
  create_credentials "test-token"
  run vector site clone 01JTEST00000000000000000AA --no-json
  assert_success
  assert_output_contains "ID"
  assert_output_contains "Status"
}


# --- site suspend / unsuspend ---

@test "site suspend succeeds" {
  create_credentials "test-token"
  run vector site suspend 01JTEST00000000000000000AA
  assert_success
}

@test "site unsuspend succeeds" {
  create_credentials "test-token"
  run vector site unsuspend 01JTEST00000000000000000AA
  assert_success
}


# --- site reset-sftp-password / reset-db-password ---

@test "site reset-sftp-password succeeds" {
  create_credentials "test-token"
  run vector site reset-sftp-password 01JTEST00000000000000000AA
  assert_success
}

@test "site reset-db-password succeeds" {
  create_credentials "test-token"
  run vector site reset-db-password 01JTEST00000000000000000AA
  assert_success
}


# --- site purge-cache ---

@test "site purge-cache succeeds" {
  create_credentials "test-token"
  run vector site purge-cache 01JTEST00000000000000000AA
  assert_success
}


# --- site logs ---

@test "site logs succeeds" {
  create_credentials "test-token"
  run vector site logs 01JTEST00000000000000000AA --environment prod
  assert_success
}

@test "site logs requires --environment" {
  create_credentials "test-token"
  run vector site logs 01JTEST00000000000000000AA
  assert_failure
}


# --- site ssh-key ---

@test "site ssh-key list succeeds" {
  create_credentials "test-token"
  run vector site ssh-key list 01JTEST00000000000000000AA
  assert_success
}

@test "site ssh-key list --json returns valid JSON" {
  create_credentials "test-token"
  run vector site ssh-key list 01JTEST00000000000000000AA --json
  assert_success
  is_valid_json
}

@test "site ssh-key add succeeds" {
  create_credentials "test-token"
  run vector site ssh-key add 01JTEST00000000000000000AA \
    --name "test-key" \
    --public-key "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITestKeyData test@example.com"
  assert_success
}

@test "site ssh-key remove succeeds" {
  create_credentials "test-token"
  run vector site ssh-key remove 01JTEST00000000000000000AA 01JTEST00000000000000KEY01
  assert_success
}


# --- auth required ---

@test "site list without auth fails with exit code 2" {
  run vector site list
  assert_failure
  assert_exit_code 2
}

@test "site show without auth fails with exit code 2" {
  run vector site show 01JTEST00000000000000000AA
  assert_failure
  assert_exit_code 2
}

@test "site create without auth fails with exit code 2" {
  run vector site create --customer-id cust-123
  assert_failure
  assert_exit_code 2
}


# --- help ---

@test "site --help renders correctly" {
  run vector site --help
  assert_success
  assert_output_contains "Manage Vector sites"
  assert_output_contains "list"
  assert_output_contains "show"
  assert_output_contains "create"
  assert_output_contains "update"
  assert_output_contains "delete"
  assert_output_contains "clone"
  assert_output_contains "suspend"
  assert_output_contains "unsuspend"
  assert_output_contains "ssh-key"
}
