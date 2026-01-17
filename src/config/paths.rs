use std::env;
use std::path::PathBuf;

use crate::api::ApiError;

const APP_NAME: &str = "vector";
const CONFIG_FILE: &str = "config.json";
const CREDENTIALS_FILE: &str = "credentials.json";

pub fn config_dir() -> Result<PathBuf, ApiError> {
    if let Ok(dir) = env::var("VECTOR_CONFIG_DIR") {
        return Ok(PathBuf::from(dir));
    }

    if let Ok(xdg_config) = env::var("XDG_CONFIG_HOME") {
        return Ok(PathBuf::from(xdg_config).join(APP_NAME));
    }

    dirs::config_dir()
        .map(|p| p.join(APP_NAME))
        .ok_or_else(|| ApiError::ConfigError("Could not determine config directory".to_string()))
}

pub fn config_file() -> Result<PathBuf, ApiError> {
    Ok(config_dir()?.join(CONFIG_FILE))
}

pub fn credentials_file() -> Result<PathBuf, ApiError> {
    Ok(config_dir()?.join(CREDENTIALS_FILE))
}
