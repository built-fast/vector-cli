use std::process::Command;

fn vector_cmd() -> Command {
    Command::new(env!("CARGO_BIN_EXE_vector"))
}

#[test]
fn test_help() {
    let output = vector_cmd().arg("--help").output().expect("Failed to run");
    assert!(output.status.success());
    let stdout = String::from_utf8_lossy(&output.stdout);
    assert!(stdout.contains("CLI for Vector Pro API"));
    assert!(stdout.contains("auth"));
    assert!(stdout.contains("site"));
    assert!(stdout.contains("env"));
    assert!(stdout.contains("deploy"));
    assert!(stdout.contains("ssl"));
    assert!(stdout.contains("mcp"));
}

#[test]
fn test_version() {
    let output = vector_cmd()
        .arg("--version")
        .output()
        .expect("Failed to run");
    assert!(output.status.success());
    let stdout = String::from_utf8_lossy(&output.stdout);
    assert!(stdout.contains("vector"));
}

#[test]
fn test_auth_help() {
    let output = vector_cmd()
        .args(["auth", "--help"])
        .output()
        .expect("Failed to run");
    assert!(output.status.success());
    let stdout = String::from_utf8_lossy(&output.stdout);
    assert!(stdout.contains("login"));
    assert!(stdout.contains("logout"));
    assert!(stdout.contains("status"));
}

#[test]
fn test_site_help() {
    let output = vector_cmd()
        .args(["site", "--help"])
        .output()
        .expect("Failed to run");
    assert!(output.status.success());
    let stdout = String::from_utf8_lossy(&output.stdout);
    assert!(stdout.contains("list"));
    assert!(stdout.contains("show"));
    assert!(stdout.contains("create"));
    assert!(stdout.contains("delete"));
    assert!(stdout.contains("suspend"));
    assert!(stdout.contains("purge-cache"));
    assert!(stdout.contains("logs"));
}

#[test]
fn test_env_help() {
    let output = vector_cmd()
        .args(["env", "--help"])
        .output()
        .expect("Failed to run");
    assert!(output.status.success());
    let stdout = String::from_utf8_lossy(&output.stdout);
    assert!(stdout.contains("list"));
    assert!(stdout.contains("create"));
    assert!(stdout.contains("suspend"));
}

#[test]
fn test_deploy_help() {
    let output = vector_cmd()
        .args(["deploy", "--help"])
        .output()
        .expect("Failed to run");
    assert!(output.status.success());
    let stdout = String::from_utf8_lossy(&output.stdout);
    assert!(stdout.contains("list"));
    assert!(stdout.contains("create"));
    assert!(stdout.contains("rollback"));
}

#[test]
fn test_ssl_help() {
    let output = vector_cmd()
        .args(["ssl", "--help"])
        .output()
        .expect("Failed to run");
    assert!(output.status.success());
    let stdout = String::from_utf8_lossy(&output.stdout);
    assert!(stdout.contains("status"));
    assert!(stdout.contains("nudge"));
}

#[test]
fn test_mcp_help() {
    let output = vector_cmd()
        .args(["mcp", "--help"])
        .output()
        .expect("Failed to run");
    assert!(output.status.success());
    let stdout = String::from_utf8_lossy(&output.stdout);
    assert!(stdout.contains("setup"));
    assert!(stdout.contains("Claude"));
}

#[test]
fn test_mcp_setup_help() {
    let output = vector_cmd()
        .args(["mcp", "setup", "--help"])
        .output()
        .expect("Failed to run");
    assert!(output.status.success());
    let stdout = String::from_utf8_lossy(&output.stdout);
    assert!(stdout.contains("--force"));
}

#[test]
fn test_mcp_setup_requires_auth() {
    let output = vector_cmd()
        .args(["mcp", "setup"])
        .env("VECTOR_CONFIG_DIR", "/tmp/vector-test-nonexistent")
        .env_remove("VECTOR_API_KEY")
        .output()
        .expect("Failed to run");
    assert!(!output.status.success());
    assert_eq!(output.status.code(), Some(2)); // EXIT_AUTH_ERROR
}

#[test]
fn test_auth_status_not_logged_in() {
    let output = vector_cmd()
        .args(["auth", "status", "--json"])
        .env("VECTOR_CONFIG_DIR", "/tmp/vector-test-nonexistent")
        .output()
        .expect("Failed to run");
    assert!(output.status.success());
    let stdout = String::from_utf8_lossy(&output.stdout);
    assert!(stdout.contains("authenticated"));
    assert!(stdout.contains("false"));
}

#[test]
fn test_site_list_requires_auth() {
    let output = vector_cmd()
        .args(["site", "list"])
        .env("VECTOR_CONFIG_DIR", "/tmp/vector-test-nonexistent")
        .env_remove("VECTOR_API_KEY")
        .output()
        .expect("Failed to run");
    assert!(!output.status.success());
    assert_eq!(output.status.code(), Some(2)); // EXIT_AUTH_ERROR
}

#[test]
fn test_invalid_subcommand() {
    let output = vector_cmd()
        .args(["invalid"])
        .output()
        .expect("Failed to run");
    assert!(!output.status.success());
}

#[test]
fn test_json_flag() {
    let output = vector_cmd()
        .args(["--json", "auth", "status"])
        .env("VECTOR_CONFIG_DIR", "/tmp/vector-test-nonexistent")
        .output()
        .expect("Failed to run");
    let stdout = String::from_utf8_lossy(&output.stdout);
    // Should be valid JSON
    assert!(serde_json::from_str::<serde_json::Value>(&stdout).is_ok());
}
