use std::io::{self, BufRead, IsTerminal};

use serde::Deserialize;
use serde_json::Value;

use crate::api::{ApiClient, ApiError};
use crate::config::{Config, Credentials};
use crate::output::{print_json, print_key_value, print_message, OutputFormat};

#[derive(Debug, Deserialize)]
struct UserResponse {
    data: UserData,
}

#[derive(Debug, Deserialize)]
struct UserData {
    id: u64,
    name: String,
    email: String,
}

pub fn login(token: Option<String>, format: OutputFormat) -> Result<(), ApiError> {
    let api_token = match token {
        Some(t) => t,
        None => read_token()?,
    };

    if api_token.is_empty() {
        return Err(ApiError::ConfigError("Token cannot be empty".to_string()));
    }

    let config = Config::load()?;
    let mut client = ApiClient::new(config.api_url, None)?;
    client.set_token(api_token.clone());

    let response: Value = client.get("/api/v1/vector/user")?;

    let mut creds = Credentials::load()?;
    creds.api_key = Some(api_token);
    creds.save()?;

    if format == OutputFormat::Json {
        print_json(&response);
    } else {
        print_message("Successfully authenticated.");
        if let Some(data) = response.get("data") {
            if let Some(email) = data.get("email").and_then(|v| v.as_str()) {
                print_message(&format!("Logged in as: {}", email));
            }
        }
    }

    Ok(())
}

pub fn logout(format: OutputFormat) -> Result<(), ApiError> {
    let mut creds = Credentials::load()?;

    if creds.api_key.is_none() {
        if format == OutputFormat::Json {
            print_json(&serde_json::json!({"message": "Not logged in"}));
        } else {
            print_message("Not logged in.");
        }
        return Ok(());
    }

    creds.clear()?;

    if format == OutputFormat::Json {
        print_json(&serde_json::json!({"message": "Logged out successfully"}));
    } else {
        print_message("Logged out successfully.");
    }

    Ok(())
}

pub fn status(format: OutputFormat) -> Result<(), ApiError> {
    let config = Config::load()?;
    let creds = Credentials::load()?;

    let token = match get_api_key(&creds) {
        Some(t) => t,
        None => {
            if format == OutputFormat::Json {
                print_json(&serde_json::json!({
                    "authenticated": false,
                    "message": "Not logged in"
                }));
            } else {
                print_message("Not logged in. Run 'vector auth login' to authenticate.");
            }
            return Ok(());
        }
    };

    let client = ApiClient::new(config.api_url, Some(token))?;
    let response: UserResponse = client.get("/api/v1/vector/user")?;

    if format == OutputFormat::Json {
        print_json(&serde_json::json!({
            "authenticated": true,
            "user": {
                "id": response.data.id,
                "name": response.data.name,
                "email": response.data.email
            }
        }));
    } else {
        print_key_value(vec![
            ("Status", "Authenticated".to_string()),
            ("Name", response.data.name),
            ("Email", response.data.email),
        ]);
    }

    Ok(())
}

fn read_token() -> Result<String, ApiError> {
    let stdin = io::stdin();

    if stdin.is_terminal() {
        eprint!("API Token: ");
        rpassword::read_password()
            .map_err(|e| ApiError::ConfigError(format!("Failed to read token: {}", e)))
    } else {
        let mut line = String::new();
        stdin
            .lock()
            .read_line(&mut line)
            .map_err(|e| ApiError::ConfigError(format!("Failed to read from stdin: {}", e)))?;
        Ok(line.trim().to_string())
    }
}

pub fn get_api_key(creds: &Credentials) -> Option<String> {
    std::env::var("VECTOR_API_KEY")
        .ok()
        .or_else(|| creds.api_key.clone())
}
