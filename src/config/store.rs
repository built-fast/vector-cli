use serde::{Deserialize, Serialize};
use std::fs;
use std::path::Path;

use crate::api::ApiError;

use super::paths::{config_dir, config_file, credentials_file};

#[derive(Debug, Default, Serialize, Deserialize)]
pub struct Config {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub api_url: Option<String>,
}

#[derive(Debug, Default, Serialize, Deserialize)]
pub struct Credentials {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub api_key: Option<String>,
}

impl Config {
    pub fn load() -> Result<Self, ApiError> {
        let path = config_file()?;
        if !path.exists() {
            return Ok(Self::default());
        }
        let content = fs::read_to_string(&path)
            .map_err(|e| ApiError::ConfigError(format!("Failed to read config: {}", e)))?;
        serde_json::from_str(&content)
            .map_err(|e| ApiError::ConfigError(format!("Failed to parse config: {}", e)))
    }

    #[allow(dead_code)]
    pub fn save(&self) -> Result<(), ApiError> {
        ensure_config_dir()?;
        let path = config_file()?;
        let content = serde_json::to_string_pretty(self)
            .map_err(|e| ApiError::ConfigError(format!("Failed to serialize config: {}", e)))?;
        fs::write(&path, content)
            .map_err(|e| ApiError::ConfigError(format!("Failed to write config: {}", e)))?;
        Ok(())
    }
}

impl Credentials {
    pub fn load() -> Result<Self, ApiError> {
        let path = credentials_file()?;
        if !path.exists() {
            return Ok(Self::default());
        }
        let content = fs::read_to_string(&path)
            .map_err(|e| ApiError::ConfigError(format!("Failed to read credentials: {}", e)))?;
        serde_json::from_str(&content)
            .map_err(|e| ApiError::ConfigError(format!("Failed to parse credentials: {}", e)))
    }

    pub fn save(&self) -> Result<(), ApiError> {
        ensure_config_dir()?;
        let path = credentials_file()?;
        let content = serde_json::to_string_pretty(self).map_err(|e| {
            ApiError::ConfigError(format!("Failed to serialize credentials: {}", e))
        })?;
        fs::write(&path, &content)
            .map_err(|e| ApiError::ConfigError(format!("Failed to write credentials: {}", e)))?;

        #[cfg(unix)]
        set_permissions(&path)?;

        Ok(())
    }

    pub fn clear(&mut self) -> Result<(), ApiError> {
        self.api_key = None;
        self.save()
    }
}

fn ensure_config_dir() -> Result<(), ApiError> {
    let dir = config_dir()?;
    if !dir.exists() {
        fs::create_dir_all(&dir).map_err(|e| {
            ApiError::ConfigError(format!("Failed to create config directory: {}", e))
        })?;

        #[cfg(unix)]
        {
            use std::os::unix::fs::PermissionsExt;
            let permissions = fs::Permissions::from_mode(0o700);
            fs::set_permissions(&dir, permissions).map_err(|e| {
                ApiError::ConfigError(format!("Failed to set directory permissions: {}", e))
            })?;
        }
    }
    Ok(())
}

#[cfg(unix)]
fn set_permissions(path: &Path) -> Result<(), ApiError> {
    use std::os::unix::fs::PermissionsExt;
    let permissions = fs::Permissions::from_mode(0o600);
    fs::set_permissions(path, permissions)
        .map_err(|e| ApiError::ConfigError(format!("Failed to set file permissions: {}", e)))
}
