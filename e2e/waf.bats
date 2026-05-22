#!/usr/bin/env bats
# waf.bats - E2E tests for vector waf commands

load test_helper


# --- waf rate-limit list ---

@test "waf rate-limit list returns valid JSON (default non-TTY)" {
  create_credentials "test-token"
  run vector waf rate-limit list 01JTEST00000000000000000AA
  assert_success
  is_valid_json
}

@test "waf rate-limit list --no-json returns table output" {
  create_credentials "test-token"
  run vector waf rate-limit list 01JTEST00000000000000000AA --no-json
  assert_success
  assert_output_contains "ID"
  assert_output_contains "NAME"
}

@test "waf rate-limit list --json returns valid JSON" {
  create_credentials "test-token"
  run vector waf rate-limit list 01JTEST00000000000000000AA --json
  assert_success
  is_valid_json
}


# --- waf rate-limit show ---

@test "waf rate-limit show returns valid JSON (default non-TTY)" {
  create_credentials "test-token"
  run vector waf rate-limit show 01JTEST00000000000000000AA 12345
  assert_success
  is_valid_json
}

@test "waf rate-limit show --no-json returns key-value output" {
  create_credentials "test-token"
  run vector waf rate-limit show 01JTEST00000000000000000AA 12345 --no-json
  assert_success
  assert_output_contains "ID"
  assert_output_contains "Name"
}


# --- waf rate-limit create ---

@test "waf rate-limit create succeeds" {
  create_credentials "test-token"
  run vector waf rate-limit create 01JTEST00000000000000000AA \
    --name "Test Rate Limit" \
    --request-count 100 \
    --timeframe 1 \
    --block-time 60
  assert_success
  is_valid_json
}

@test "waf rate-limit create --no-json returns key-value output" {
  create_credentials "test-token"
  run vector waf rate-limit create 01JTEST00000000000000000AA \
    --name "Test Rate Limit" \
    --request-count 100 \
    --timeframe 1 \
    --block-time 60 \
    --no-json
  assert_success
  assert_output_contains "ID"
  assert_output_contains "Name"
}


# --- waf rate-limit update ---

@test "waf rate-limit update succeeds" {
  create_credentials "test-token"
  run vector waf rate-limit update 01JTEST00000000000000000AA 12345 \
    --name "Updated Rate Limit"
  assert_success
  is_valid_json
}

@test "waf rate-limit update --no-json returns key-value output" {
  create_credentials "test-token"
  run vector waf rate-limit update 01JTEST00000000000000000AA 12345 \
    --name "Updated Rate Limit" \
    --no-json
  assert_success
  assert_output_contains "ID"
  assert_output_contains "Name"
}


# --- waf rate-limit delete ---

@test "waf rate-limit delete succeeds" {
  create_credentials "test-token"
  run vector waf rate-limit delete 01JTEST00000000000000000AA 12345
  assert_success
}


# --- waf blocked-ip list ---

@test "waf blocked-ip list returns valid JSON (default non-TTY)" {
  create_credentials "test-token"
  run vector waf blocked-ip list 01JTEST00000000000000000AA
  assert_success
  is_valid_json
}

@test "waf blocked-ip list --no-json returns table output" {
  create_credentials "test-token"
  run vector waf blocked-ip list 01JTEST00000000000000000AA --no-json
  assert_success
  assert_output_contains "IP"
}

@test "waf blocked-ip list --json returns valid JSON" {
  create_credentials "test-token"
  run vector waf blocked-ip list 01JTEST00000000000000000AA --json
  assert_success
  is_valid_json
}


# --- waf blocked-ip add ---

@test "waf blocked-ip add succeeds" {
  create_credentials "test-token"
  run vector waf blocked-ip add 01JTEST00000000000000000AA 192.0.2.1
  assert_success
}


# --- waf blocked-ip remove ---

@test "waf blocked-ip remove succeeds" {
  create_credentials "test-token"
  run vector waf blocked-ip remove 01JTEST00000000000000000AA 192.0.2.1
  assert_success
}


# --- waf blocked-referrer list ---

@test "waf blocked-referrer list returns valid JSON (default non-TTY)" {
  create_credentials "test-token"
  run vector waf blocked-referrer list 01JTEST00000000000000000AA
  assert_success
  is_valid_json
}

@test "waf blocked-referrer list --no-json returns table output" {
  create_credentials "test-token"
  run vector waf blocked-referrer list 01JTEST00000000000000000AA --no-json
  assert_success
  assert_output_contains "HOSTNAME"
}

@test "waf blocked-referrer list --json returns valid JSON" {
  create_credentials "test-token"
  run vector waf blocked-referrer list 01JTEST00000000000000000AA --json
  assert_success
  is_valid_json
}


# --- waf blocked-referrer add ---

@test "waf blocked-referrer add succeeds" {
  create_credentials "test-token"
  run vector waf blocked-referrer add 01JTEST00000000000000000AA spam.example.com
  assert_success
}


# --- waf blocked-referrer remove ---

@test "waf blocked-referrer remove succeeds" {
  create_credentials "test-token"
  run vector waf blocked-referrer remove 01JTEST00000000000000000AA spam.example.com
  assert_success
}


# --- waf allowed-referrer list ---

@test "waf allowed-referrer list returns valid JSON (default non-TTY)" {
  create_credentials "test-token"
  run vector waf allowed-referrer list 01JTEST00000000000000000AA
  assert_success
  is_valid_json
}

@test "waf allowed-referrer list --no-json returns table output" {
  create_credentials "test-token"
  run vector waf allowed-referrer list 01JTEST00000000000000000AA --no-json
  assert_success
  assert_output_contains "HOSTNAME"
}

@test "waf allowed-referrer list --json returns valid JSON" {
  create_credentials "test-token"
  run vector waf allowed-referrer list 01JTEST00000000000000000AA --json
  assert_success
  is_valid_json
}


# --- waf allowed-referrer add ---

@test "waf allowed-referrer add succeeds" {
  create_credentials "test-token"
  run vector waf allowed-referrer add 01JTEST00000000000000000AA example.com
  assert_success
}


# --- waf allowed-referrer remove ---

@test "waf allowed-referrer remove succeeds" {
  create_credentials "test-token"
  run vector waf allowed-referrer remove 01JTEST00000000000000000AA example.com
  assert_success
}


# --- auth required ---

@test "waf rate-limit list without auth fails with exit code 2" {
  run vector waf rate-limit list 01JTEST00000000000000000AA
  assert_failure
  assert_exit_code 2
}

@test "waf rate-limit show without auth fails with exit code 2" {
  run vector waf rate-limit show 01JTEST00000000000000000AA 12345
  assert_failure
  assert_exit_code 2
}

@test "waf rate-limit create without auth fails with exit code 2" {
  run vector waf rate-limit create 01JTEST00000000000000000AA \
    --name "Test" --request-count 100 --timeframe 1 --block-time 60
  assert_failure
  assert_exit_code 2
}

@test "waf rate-limit update without auth fails with exit code 2" {
  run vector waf rate-limit update 01JTEST00000000000000000AA 12345 --name "Test"
  assert_failure
  assert_exit_code 2
}

@test "waf rate-limit delete without auth fails with exit code 2" {
  run vector waf rate-limit delete 01JTEST00000000000000000AA 12345
  assert_failure
  assert_exit_code 2
}

@test "waf blocked-ip list without auth fails with exit code 2" {
  run vector waf blocked-ip list 01JTEST00000000000000000AA
  assert_failure
  assert_exit_code 2
}

@test "waf blocked-ip add without auth fails with exit code 2" {
  run vector waf blocked-ip add 01JTEST00000000000000000AA 192.0.2.1
  assert_failure
  assert_exit_code 2
}

@test "waf blocked-ip remove without auth fails with exit code 2" {
  run vector waf blocked-ip remove 01JTEST00000000000000000AA 192.0.2.1
  assert_failure
  assert_exit_code 2
}

@test "waf blocked-referrer list without auth fails with exit code 2" {
  run vector waf blocked-referrer list 01JTEST00000000000000000AA
  assert_failure
  assert_exit_code 2
}

@test "waf blocked-referrer add without auth fails with exit code 2" {
  run vector waf blocked-referrer add 01JTEST00000000000000000AA spam.example.com
  assert_failure
  assert_exit_code 2
}

@test "waf blocked-referrer remove without auth fails with exit code 2" {
  run vector waf blocked-referrer remove 01JTEST00000000000000000AA spam.example.com
  assert_failure
  assert_exit_code 2
}

@test "waf allowed-referrer list without auth fails with exit code 2" {
  run vector waf allowed-referrer list 01JTEST00000000000000000AA
  assert_failure
  assert_exit_code 2
}

@test "waf allowed-referrer add without auth fails with exit code 2" {
  run vector waf allowed-referrer add 01JTEST00000000000000000AA example.com
  assert_failure
  assert_exit_code 2
}

@test "waf allowed-referrer remove without auth fails with exit code 2" {
  run vector waf allowed-referrer remove 01JTEST00000000000000000AA example.com
  assert_failure
  assert_exit_code 2
}
