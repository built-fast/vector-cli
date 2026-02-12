use std::fs;
use std::path::PathBuf;

use serde::{Deserialize, Serialize};
use serde_json::{Map, Value, json};

use crate::api::ApiError;
use crate::commands::auth::get_api_key;
use crate::config::Credentials;
use crate::output::{OutputFormat, print_json, print_message};

#[derive(Debug, Serialize, Deserialize, Default)]
#[serde(rename_all = "camelCase")]
struct ClaudeConfig {
    #[serde(default)]
    mcp_servers: Map<String, Value>,
    #[serde(flatten)]
    other: Map<String, Value>,
}

fn get_claude_config_path() -> Result<PathBuf, ApiError> {
    #[cfg(target_os = "macos")]
    {
        let home = dirs::home_dir()
            .ok_or_else(|| ApiError::ConfigError("Could not determine home directory".into()))?;
        Ok(home.join("Library/Application Support/Claude/claude_desktop_config.json"))
    }

    #[cfg(target_os = "windows")]
    {
        let appdata = dirs::config_dir()
            .ok_or_else(|| ApiError::ConfigError("Could not determine AppData directory".into()))?;
        Ok(appdata.join("Claude/claude_desktop_config.json"))
    }

    #[cfg(target_os = "linux")]
    {
        let config = dirs::config_dir()
            .ok_or_else(|| ApiError::ConfigError("Could not determine config directory".into()))?;
        Ok(config.join("Claude/claude_desktop_config.json"))
    }

    #[cfg(not(any(target_os = "macos", target_os = "windows", target_os = "linux")))]
    {
        Err(ApiError::ConfigError("Unsupported platform".into()))
    }
}

pub fn setup(force: bool, format: OutputFormat) -> Result<(), ApiError> {
    let creds = Credentials::load()?;
    let token = get_api_key(&creds).ok_or_else(|| {
        ApiError::Unauthorized(
            "Not logged in. Run 'vector auth login' to authenticate.".to_string(),
        )
    })?;

    let config_path = get_claude_config_path()?;

    // Load existing config or create new one
    let mut config: ClaudeConfig = if config_path.exists() {
        let content = fs::read_to_string(&config_path)
            .map_err(|e| ApiError::ConfigError(format!("Failed to read Claude config: {}", e)))?;
        serde_json::from_str(&content)
            .map_err(|e| ApiError::ConfigError(format!("Failed to parse Claude config: {}", e)))?
    } else {
        ClaudeConfig::default()
    };

    // Check if vector is already configured
    if config.mcp_servers.contains_key("vector") && !force {
        return Err(ApiError::ConfigError(
            "Vector MCP server already configured. Use --force to overwrite.".to_string(),
        ));
    }

    // Create the Vector MCP server configuration
    let vector_config = json!({
        "command": "npx",
        "args": [
            "-y",
            "mcp-remote",
            "https://api.builtfast.com/mcp/vector",
            "--header",
            format!("Authorization: Bearer {}", token)
        ]
    });

    let was_updated = config.mcp_servers.contains_key("vector");

    // Add or update the vector server
    config
        .mcp_servers
        .insert("vector".to_string(), vector_config);

    // Ensure parent directory exists
    if let Some(parent) = config_path.parent()
        && !parent.exists()
    {
        fs::create_dir_all(parent).map_err(|e| {
            ApiError::ConfigError(format!("Failed to create Claude config directory: {}", e))
        })?;
    }

    // Write the config
    let content = serde_json::to_string_pretty(&config)
        .map_err(|e| ApiError::ConfigError(format!("Failed to serialize config: {}", e)))?;
    fs::write(&config_path, content)
        .map_err(|e| ApiError::ConfigError(format!("Failed to write Claude config: {}", e)))?;

    let action = if was_updated { "updated" } else { "added" };

    if format == OutputFormat::Json {
        print_json(&json!({
            "success": true,
            "action": action,
            "config_path": config_path.to_string_lossy(),
            "message": format!("Vector MCP server {} in Claude Desktop config", action)
        }));
    } else {
        print_message(&format!(
            "Vector MCP server {} in Claude Desktop config.",
            action
        ));
        print_message(&format!("Config written to: {}", config_path.display()));
        print_message("\nRestart Claude Desktop to apply changes.");
    }

    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_claude_config_empty() {
        let config: ClaudeConfig = serde_json::from_str("{}").unwrap();
        assert!(config.mcp_servers.is_empty());
        assert!(config.other.is_empty());
    }

    #[test]
    fn test_claude_config_preserves_other_mcp_servers() {
        let json = r#"{
            "mcpServers": {
                "other-server": {
                    "command": "node",
                    "args": ["server.js"]
                }
            }
        }"#;

        let mut config: ClaudeConfig = serde_json::from_str(json).unwrap();
        assert!(config.mcp_servers.contains_key("other-server"));

        // Add vector
        config
            .mcp_servers
            .insert("vector".to_string(), json!({"command": "npx"}));

        // Serialize and deserialize
        let serialized = serde_json::to_string(&config).unwrap();
        let restored: ClaudeConfig = serde_json::from_str(&serialized).unwrap();

        assert!(restored.mcp_servers.contains_key("other-server"));
        assert!(restored.mcp_servers.contains_key("vector"));
    }

    #[test]
    fn test_claude_config_preserves_other_fields() {
        let json = r#"{
            "mcpServers": {},
            "theme": "dark",
            "someOtherSetting": true
        }"#;

        let config: ClaudeConfig = serde_json::from_str(json).unwrap();
        assert!(config.other.contains_key("theme"));
        assert!(config.other.contains_key("someOtherSetting"));

        // Serialize back
        let serialized = serde_json::to_string(&config).unwrap();
        assert!(serialized.contains("theme"));
        assert!(serialized.contains("someOtherSetting"));
    }

    #[test]
    fn test_vector_config_structure() {
        let token = "test-token-123";
        let vector_config = json!({
            "command": "npx",
            "args": [
                "-y",
                "mcp-remote",
                "https://api.builtfast.com/mcp/vector",
                "--header",
                format!("Authorization: Bearer {}", token)
            ]
        });

        assert_eq!(vector_config["command"], "npx");
        let args = vector_config["args"].as_array().unwrap();
        assert_eq!(args[0], "-y");
        assert_eq!(args[1], "mcp-remote");
        assert_eq!(args[2], "https://api.builtfast.com/mcp/vector");
        assert_eq!(args[3], "--header");
        assert_eq!(args[4], "Authorization: Bearer test-token-123");
    }

    #[test]
    fn test_claude_config_roundtrip() {
        let original = r#"{
            "mcpServers": {
                "existing": {"command": "test"}
            },
            "customField": "value"
        }"#;

        let mut config: ClaudeConfig = serde_json::from_str(original).unwrap();
        config
            .mcp_servers
            .insert("vector".to_string(), json!({"command": "npx"}));

        let serialized = serde_json::to_string_pretty(&config).unwrap();
        let restored: ClaudeConfig = serde_json::from_str(&serialized).unwrap();

        assert_eq!(restored.mcp_servers.len(), 2);
        assert!(restored.mcp_servers.contains_key("existing"));
        assert!(restored.mcp_servers.contains_key("vector"));
        assert!(restored.other.contains_key("customField"));
    }

    #[test]
    fn test_get_claude_config_path() {
        let path = get_claude_config_path().unwrap();
        assert!(path.ends_with("claude_desktop_config.json"));
        assert!(path.to_string_lossy().contains("Claude"));
    }
}
