use serde::Serialize;
use serde_json::Value;

use crate::api::{ApiClient, ApiError};
use crate::output::{
    OutputFormat, extract_pagination, format_bool, format_option, print_json, print_key_value,
    print_message, print_pagination, print_table,
};

#[derive(Debug, Serialize)]
struct PaginationQuery {
    page: u32,
    per_page: u32,
}

#[derive(Debug, Serialize)]
struct CreateSshKeyRequest {
    name: String,
    public_key: String,
}

#[derive(Debug, Serialize)]
struct CreateApiKeyRequest {
    name: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    abilities: Option<Vec<String>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    expires_at: Option<String>,
}

#[derive(Debug, Serialize)]
struct CreateSecretRequest {
    key: String,
    value: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    is_secret: Option<bool>,
}

#[derive(Debug, Serialize)]
struct UpdateSecretRequest {
    #[serde(skip_serializing_if = "Option::is_none")]
    key: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    value: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    is_secret: Option<bool>,
}

// Account summary

pub fn show(client: &ApiClient, format: OutputFormat) -> Result<(), ApiError> {
    let response: Value = client.get("/api/v1/vector/account")?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    let data = &response["data"];
    let owner = &data["owner"];
    let account = &data["account"];
    let sites = &data["sites"];
    let envs = &data["environments"];

    print_key_value(vec![
        (
            "Owner Name",
            format_option(&owner["name"].as_str().map(String::from)),
        ),
        (
            "Owner Email",
            format_option(&owner["email"].as_str().map(String::from)),
        ),
        (
            "Account Name",
            format_option(&account["name"].as_str().map(String::from)),
        ),
        (
            "Company",
            format_option(&account["company"].as_str().map(String::from)),
        ),
        (
            "Total Sites",
            sites["total"]
                .as_u64()
                .map(|v| v.to_string())
                .unwrap_or("-".to_string()),
        ),
        (
            "Active Sites",
            sites["by_status"]["active"]
                .as_u64()
                .map(|v| v.to_string())
                .unwrap_or("-".to_string()),
        ),
        (
            "Total Environments",
            envs["total"]
                .as_u64()
                .map(|v| v.to_string())
                .unwrap_or("-".to_string()),
        ),
        (
            "Active Environments",
            envs["by_status"]["active"]
                .as_u64()
                .map(|v| v.to_string())
                .unwrap_or("-".to_string()),
        ),
    ]);

    Ok(())
}

// SSH Key commands (account-level)

pub fn ssh_key_list(
    client: &ApiClient,
    page: u32,
    per_page: u32,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let query = PaginationQuery { page, per_page };
    let response: Value = client.get_with_query("/api/v1/vector/ssh-keys", &query)?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    let keys = response["data"]
        .as_array()
        .ok_or_else(|| ApiError::Other("Invalid response format".to_string()))?;

    if keys.is_empty() {
        print_message("No SSH keys found.");
        return Ok(());
    }

    let rows: Vec<Vec<String>> = keys
        .iter()
        .map(|k| {
            vec![
                k["id"].as_str().unwrap_or("-").to_string(),
                k["name"].as_str().unwrap_or("-").to_string(),
                format_option(&k["fingerprint"].as_str().map(String::from)),
                format_option(&k["created_at"].as_str().map(String::from)),
            ]
        })
        .collect();

    print_table(vec!["ID", "Name", "Fingerprint", "Created"], rows);

    if let Some((current, last, total)) = extract_pagination(&response) {
        print_pagination(current, last, total);
    }

    Ok(())
}

pub fn ssh_key_show(
    client: &ApiClient,
    key_id: &str,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let response: Value = client.get(&format!("/api/v1/vector/ssh-keys/{}", key_id))?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    let key = &response["data"];

    print_key_value(vec![
        ("ID", key["id"].as_str().unwrap_or("-").to_string()),
        ("Name", key["name"].as_str().unwrap_or("-").to_string()),
        (
            "Fingerprint",
            format_option(&key["fingerprint"].as_str().map(String::from)),
        ),
        (
            "Public Key Preview",
            format_option(&key["public_key_preview"].as_str().map(String::from)),
        ),
        (
            "Account Default",
            key["is_account_default"]
                .as_bool()
                .map(|v| v.to_string())
                .unwrap_or("-".to_string()),
        ),
        (
            "Created",
            format_option(&key["created_at"].as_str().map(String::from)),
        ),
    ]);

    Ok(())
}

pub fn ssh_key_create(
    client: &ApiClient,
    name: &str,
    public_key: &str,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let body = CreateSshKeyRequest {
        name: name.to_string(),
        public_key: public_key.to_string(),
    };

    let response: Value = client.post("/api/v1/vector/ssh-keys", &body)?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    let key = &response["data"];
    print_message(&format!(
        "SSH key created: {} ({})",
        key["name"].as_str().unwrap_or("-"),
        key["id"].as_str().unwrap_or("-")
    ));

    Ok(())
}

pub fn ssh_key_delete(
    client: &ApiClient,
    key_id: &str,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let response: Value = client.delete(&format!("/api/v1/vector/ssh-keys/{}", key_id))?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    print_message("SSH key deleted successfully.");
    Ok(())
}

// API Key commands

pub fn api_key_list(
    client: &ApiClient,
    page: u32,
    per_page: u32,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let query = PaginationQuery { page, per_page };
    let response: Value = client.get_with_query("/api/v1/vector/api-keys", &query)?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    let keys = response["data"]
        .as_array()
        .ok_or_else(|| ApiError::Other("Invalid response format".to_string()))?;

    if keys.is_empty() {
        print_message("No API keys found.");
        return Ok(());
    }

    let rows: Vec<Vec<String>> = keys
        .iter()
        .map(|k| {
            vec![
                k["id"]
                    .as_u64()
                    .map(|v| v.to_string())
                    .unwrap_or("-".to_string()),
                k["name"].as_str().unwrap_or("-").to_string(),
                format_abilities(&k["abilities"]),
                format_option(&k["last_used_at"].as_str().map(String::from)),
                format_option(&k["expires_at"].as_str().map(String::from)),
            ]
        })
        .collect();

    print_table(
        vec!["ID", "Name", "Abilities", "Last Used", "Expires"],
        rows,
    );

    if let Some((current, last, total)) = extract_pagination(&response) {
        print_pagination(current, last, total);
    }

    Ok(())
}

pub fn api_key_create(
    client: &ApiClient,
    name: &str,
    abilities: Option<Vec<String>>,
    expires_at: Option<String>,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let body = CreateApiKeyRequest {
        name: name.to_string(),
        abilities,
        expires_at,
    };

    let response: Value = client.post("/api/v1/vector/api-keys", &body)?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    let data = &response["data"];
    print_key_value(vec![
        ("Name", data["name"].as_str().unwrap_or("-").to_string()),
        ("Token", data["token"].as_str().unwrap_or("-").to_string()),
        ("Abilities", format_abilities(&data["abilities"])),
        (
            "Expires",
            format_option(&data["expires_at"].as_str().map(String::from)),
        ),
    ]);

    print_message("\nSave this token - it won't be shown again!");

    Ok(())
}

pub fn api_key_delete(
    client: &ApiClient,
    token_id: &str,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let response: Value = client.delete(&format!("/api/v1/vector/api-keys/{}", token_id))?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    print_message("API key deleted successfully.");
    Ok(())
}

// Global Secret commands

pub fn secret_list(
    client: &ApiClient,
    page: u32,
    per_page: u32,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let query = PaginationQuery { page, per_page };
    let response: Value = client.get_with_query("/api/v1/vector/global-secrets", &query)?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    let secrets = response["data"]
        .as_array()
        .ok_or_else(|| ApiError::Other("Invalid response format".to_string()))?;

    if secrets.is_empty() {
        print_message("No global secrets found.");
        return Ok(());
    }

    let rows: Vec<Vec<String>> = secrets
        .iter()
        .map(|s| {
            vec![
                s["id"].as_str().unwrap_or("-").to_string(),
                s["key"].as_str().unwrap_or("-").to_string(),
                format_bool(s["is_secret"].as_bool().unwrap_or(true)),
                format_option(&s["value"].as_str().map(String::from)),
                format_option(&s["created_at"].as_str().map(String::from)),
            ]
        })
        .collect();

    print_table(vec!["ID", "Key", "Secret", "Value", "Created"], rows);

    if let Some((current, last, total)) = extract_pagination(&response) {
        print_pagination(current, last, total);
    }

    Ok(())
}

pub fn secret_show(
    client: &ApiClient,
    secret_id: &str,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let response: Value = client.get(&format!("/api/v1/vector/global-secrets/{}", secret_id))?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    let secret = &response["data"];

    print_key_value(vec![
        ("ID", secret["id"].as_str().unwrap_or("-").to_string()),
        ("Key", secret["key"].as_str().unwrap_or("-").to_string()),
        (
            "Secret",
            format_bool(secret["is_secret"].as_bool().unwrap_or(true)),
        ),
        (
            "Value",
            format_option(&secret["value"].as_str().map(String::from)),
        ),
        (
            "Created",
            format_option(&secret["created_at"].as_str().map(String::from)),
        ),
        (
            "Updated",
            format_option(&secret["updated_at"].as_str().map(String::from)),
        ),
    ]);

    Ok(())
}

pub fn secret_create(
    client: &ApiClient,
    key: &str,
    value: &str,
    no_secret: bool,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let body = CreateSecretRequest {
        key: key.to_string(),
        value: value.to_string(),
        is_secret: if no_secret { Some(false) } else { None },
    };

    let response: Value = client.post("/api/v1/vector/global-secrets", &body)?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    let secret = &response["data"];
    print_message(&format!(
        "Secret created: {} ({})",
        secret["key"].as_str().unwrap_or("-"),
        secret["id"].as_str().unwrap_or("-")
    ));

    Ok(())
}

pub fn secret_update(
    client: &ApiClient,
    secret_id: &str,
    key: Option<String>,
    value: Option<String>,
    no_secret: bool,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let body = UpdateSecretRequest {
        key,
        value,
        is_secret: if no_secret { Some(false) } else { None },
    };

    let response: Value = client.put(
        &format!("/api/v1/vector/global-secrets/{}", secret_id),
        &body,
    )?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    print_message("Secret updated successfully.");
    Ok(())
}

pub fn secret_delete(
    client: &ApiClient,
    secret_id: &str,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let response: Value = client.delete(&format!("/api/v1/vector/global-secrets/{}", secret_id))?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    print_message("Secret deleted successfully.");
    Ok(())
}

// Helper function to format abilities array
fn format_abilities(value: &Value) -> String {
    if let Some(arr) = value.as_array() {
        if arr.is_empty() {
            return "-".to_string();
        }
        arr.iter()
            .filter_map(|v| v.as_str())
            .collect::<Vec<_>>()
            .join(", ")
    } else {
        "-".to_string()
    }
}
