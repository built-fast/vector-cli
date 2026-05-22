#!/usr/bin/env bats
# account.bats - E2E tests for vector account commands

load test_helper


# --- account show ---

@test "account show returns valid JSON (default non-TTY)" {
  create_credentials "test-token"
  run vector account show
  assert_success
  is_valid_json
}

@test "account show --no-json returns key-value output" {
  create_credentials "test-token"
  run vector account show --no-json
  assert_success
  assert_output_contains "Owner Name"
  assert_output_contains "Account Name"
  assert_output_contains "Total Sites"
}


# --- account ssh-key list ---

@test "account ssh-key list returns valid JSON (default non-TTY)" {
  create_credentials "test-token"
  run vector account ssh-key list
  assert_success
  is_valid_json
}

@test "account ssh-key list --no-json returns table output" {
  create_credentials "test-token"
  run vector account ssh-key list --no-json
  assert_success
  assert_output_contains "ID"
  assert_output_contains "NAME"
  assert_output_contains "FINGERPRINT"
}

@test "account ssh-key list --json returns valid JSON" {
  create_credentials "test-token"
  run vector account ssh-key list --json
  assert_success
  is_valid_json
}


# --- account ssh-key show ---

@test "account ssh-key show returns valid JSON (default non-TTY)" {
  create_credentials "test-token"
  run vector account ssh-key show 01JTEST00000000000000000AA
  assert_success
  is_valid_json
}

@test "account ssh-key show --no-json returns key-value output" {
  create_credentials "test-token"
  run vector account ssh-key show 01JTEST00000000000000000AA --no-json
  assert_success
  assert_output_contains "ID"
  assert_output_contains "Name"
  assert_output_contains "Fingerprint"
}


# --- account ssh-key create ---

@test "account ssh-key create succeeds" {
  create_credentials "test-token"
  run vector account ssh-key create \
    --name "deploy key" \
    --public-key "ssh-rsa AAAAB3NzaC1yc2EA user@host"
  assert_success
  is_valid_json
}

@test "account ssh-key create --no-json returns key-value output" {
  create_credentials "test-token"
  run vector account ssh-key create \
    --name "deploy key" \
    --public-key "ssh-rsa AAAAB3NzaC1yc2EA user@host" \
    --no-json
  assert_success
  assert_output_contains "ID"
  assert_output_contains "Name"
}


# --- account ssh-key delete ---

@test "account ssh-key delete succeeds" {
  create_credentials "test-token"
  run vector account ssh-key delete 01JTEST00000000000000000AA
  assert_success
}


# --- account api-key list ---

@test "account api-key list returns valid JSON (default non-TTY)" {
  create_credentials "test-token"
  run vector account api-key list
  assert_success
  is_valid_json
}

@test "account api-key list --no-json returns table output" {
  create_credentials "test-token"
  run vector account api-key list --no-json
  assert_success
  assert_output_contains "NAME"
  assert_output_contains "ABILITIES"
}

@test "account api-key list --json returns valid JSON" {
  create_credentials "test-token"
  run vector account api-key list --json
  assert_success
  is_valid_json
}


# --- account api-key create ---

@test "account api-key create succeeds" {
  create_credentials "test-token"
  run vector account api-key create --name "Test API Key"
  assert_success
  is_valid_json
}

@test "account api-key create --no-json returns key-value output" {
  create_credentials "test-token"
  run vector account api-key create --name "Test API Key" --no-json
  assert_success
  assert_output_contains "Name"
  assert_output_contains "Token"
}


# --- account api-key delete ---

@test "account api-key delete succeeds" {
  create_credentials "test-token"
  run vector account api-key delete 12345
  assert_success
}


# --- account secret list ---

@test "account secret list returns valid JSON (default non-TTY)" {
  create_credentials "test-token"
  run vector account secret list
  assert_success
  is_valid_json
}

@test "account secret list --no-json returns table output" {
  create_credentials "test-token"
  run vector account secret list --no-json
  assert_success
  assert_output_contains "ID"
  assert_output_contains "KEY"
  assert_output_contains "SECRET"
}

@test "account secret list --json returns valid JSON" {
  create_credentials "test-token"
  run vector account secret list --json
  assert_success
  is_valid_json
}


# --- account secret show ---

@test "account secret show returns valid JSON (default non-TTY)" {
  create_credentials "test-token"
  run vector account secret show 01JTEST00000000000000000AA
  assert_success
  is_valid_json
}

@test "account secret show --no-json returns key-value output" {
  create_credentials "test-token"
  run vector account secret show 01JTEST00000000000000000AA --no-json
  assert_success
  assert_output_contains "ID"
  assert_output_contains "Key"
  assert_output_contains "Secret"
}


# --- account secret create ---

@test "account secret create succeeds" {
  create_credentials "test-token"
  run vector account secret create --key "MY_SECRET" --value "secret123"
  assert_success
  is_valid_json
}

@test "account secret create --no-json returns key-value output" {
  create_credentials "test-token"
  run vector account secret create --key "MY_SECRET" --value "secret123" --no-json
  assert_success
  assert_output_contains "ID"
  assert_output_contains "Key"
}


# --- account secret update ---

@test "account secret update succeeds" {
  create_credentials "test-token"
  run vector account secret update 01JTEST00000000000000000AA --value "new-value"
  assert_success
  is_valid_json
}

@test "account secret update --no-json returns key-value output" {
  create_credentials "test-token"
  run vector account secret update 01JTEST00000000000000000AA --value "new-value" --no-json
  assert_success
  assert_output_contains "ID"
  assert_output_contains "Key"
}


# --- account secret delete ---

@test "account secret delete succeeds" {
  create_credentials "test-token"
  run vector account secret delete 01JTEST00000000000000000AA
  assert_success
}


# --- auth required ---

@test "account show without auth fails with exit code 2" {
  run vector account show
  assert_failure
  assert_exit_code 2
}

@test "account ssh-key list without auth fails with exit code 2" {
  run vector account ssh-key list
  assert_failure
  assert_exit_code 2
}

@test "account ssh-key show without auth fails with exit code 2" {
  run vector account ssh-key show 01JTEST00000000000000000AA
  assert_failure
  assert_exit_code 2
}

@test "account ssh-key create without auth fails with exit code 2" {
  run vector account ssh-key create --name "test" --public-key "ssh-rsa AAAA"
  assert_failure
  assert_exit_code 2
}

@test "account ssh-key delete without auth fails with exit code 2" {
  run vector account ssh-key delete 01JTEST00000000000000000AA
  assert_failure
  assert_exit_code 2
}

@test "account api-key list without auth fails with exit code 2" {
  run vector account api-key list
  assert_failure
  assert_exit_code 2
}

@test "account api-key create without auth fails with exit code 2" {
  run vector account api-key create --name "test"
  assert_failure
  assert_exit_code 2
}

@test "account api-key delete without auth fails with exit code 2" {
  run vector account api-key delete 12345
  assert_failure
  assert_exit_code 2
}

@test "account secret list without auth fails with exit code 2" {
  run vector account secret list
  assert_failure
  assert_exit_code 2
}

@test "account secret show without auth fails with exit code 2" {
  run vector account secret show 01JTEST00000000000000000AA
  assert_failure
  assert_exit_code 2
}

@test "account secret create without auth fails with exit code 2" {
  run vector account secret create --key "MY_SECRET" --value "secret123"
  assert_failure
  assert_exit_code 2
}

@test "account secret update without auth fails with exit code 2" {
  run vector account secret update 01JTEST00000000000000000AA --value "new-value"
  assert_failure
  assert_exit_code 2
}

@test "account secret delete without auth fails with exit code 2" {
  run vector account secret delete 01JTEST00000000000000000AA
  assert_failure
  assert_exit_code 2
}
