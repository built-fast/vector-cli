use serde::Serialize;
use serde_json::Value;
use std::path::Path;

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
struct ListEnvQuery {
    site: String,
    page: u32,
    per_page: u32,
}

#[derive(Debug, Serialize)]
struct CreateEnvRequest {
    name: String,
    custom_domain: String,
    php_version: String,
    #[serde(skip_serializing_if = "std::ops::Not::not")]
    is_production: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    tags: Option<Vec<String>>,
}

#[derive(Debug, Serialize)]
struct UpdateEnvRequest {
    #[serde(skip_serializing_if = "Option::is_none")]
    name: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    custom_domain: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    tags: Option<Vec<String>>,
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

pub fn list(
    client: &ApiClient,
    site_id: &str,
    page: u32,
    per_page: u32,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let query = ListEnvQuery {
        site: site_id.to_string(),
        page,
        per_page,
    };
    let response: Value = client.get_with_query("/api/v1/vector/environments", &query)?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    let envs = response["data"]
        .as_array()
        .ok_or_else(|| ApiError::Other("Invalid response format".to_string()))?;

    if envs.is_empty() {
        print_message("No environments found.");
        return Ok(());
    }

    let rows: Vec<Vec<String>> = envs
        .iter()
        .map(|e| {
            vec![
                e["id"].as_str().unwrap_or("-").to_string(),
                e["name"].as_str().unwrap_or("-").to_string(),
                e["status"].as_str().unwrap_or("-").to_string(),
                format_bool(e["is_production"].as_bool().unwrap_or(false)),
                format_option(&e["platform_domain"].as_str().map(String::from)),
            ]
        })
        .collect();

    print_table(
        vec!["ID", "Name", "Status", "Production", "Platform Domain"],
        rows,
    );

    if let Some((current, last, total)) = extract_pagination(&response) {
        print_pagination(current, last, total);
    }

    Ok(())
}

pub fn show(client: &ApiClient, env_id: &str, format: OutputFormat) -> Result<(), ApiError> {
    let response: Value = client.get(&format!("/api/v1/vector/environments/{}", env_id))?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    let env = &response["data"];

    print_key_value(vec![
        ("ID", env["id"].as_str().unwrap_or("-").to_string()),
        ("Name", env["name"].as_str().unwrap_or("-").to_string()),
        ("Status", env["status"].as_str().unwrap_or("-").to_string()),
        (
            "Production",
            format_bool(env["is_production"].as_bool().unwrap_or(false)),
        ),
        (
            "PHP Version",
            format_option(&env["php_version"].as_str().map(String::from)),
        ),
        (
            "Platform Domain",
            format_option(&env["platform_domain"].as_str().map(String::from)),
        ),
        (
            "Custom Domain",
            format_option(&env["custom_domain"].as_str().map(String::from)),
        ),
        (
            "Subdomain",
            format_option(&env["subdomain"].as_str().map(String::from)),
        ),
        (
            "Database Host",
            format_option(&env["database_host"].as_str().map(String::from)),
        ),
        (
            "Database Name",
            format_option(&env["database_name"].as_str().map(String::from)),
        ),
        (
            "Provisioning Step",
            format_option(&env["provisioning_step"].as_str().map(String::from)),
        ),
        ("Tags", format_tags(&env["tags"])),
        (
            "Created",
            format_option(&env["created_at"].as_str().map(String::from)),
        ),
        (
            "Updated",
            format_option(&env["updated_at"].as_str().map(String::from)),
        ),
    ]);

    Ok(())
}

#[allow(clippy::too_many_arguments)]
pub fn create(
    client: &ApiClient,
    site_id: &str,
    name: &str,
    custom_domain: &str,
    php_version: &str,
    is_production: bool,
    tags: Option<Vec<String>>,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let body = CreateEnvRequest {
        name: name.to_string(),
        custom_domain: custom_domain.to_string(),
        php_version: php_version.to_string(),
        is_production,
        tags,
    };

    let response: Value = client.post(
        &format!("/api/v1/vector/sites/{}/environments", site_id),
        &body,
    )?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    let env = &response["data"];
    print_message(&format!(
        "Environment created: {} ({})",
        env["name"].as_str().unwrap_or("-"),
        env["id"].as_str().unwrap_or("-")
    ));

    Ok(())
}

pub fn update(
    client: &ApiClient,
    env_id: &str,
    name: Option<String>,
    custom_domain: Option<String>,
    tags: Option<Vec<String>>,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let body = UpdateEnvRequest {
        name,
        custom_domain,
        tags,
    };

    let response: Value = client.put(&format!("/api/v1/vector/environments/{}", env_id), &body)?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    print_message("Environment updated successfully.");
    Ok(())
}

pub fn delete(client: &ApiClient, env_id: &str, format: OutputFormat) -> Result<(), ApiError> {
    let response: Value = client.delete(&format!("/api/v1/vector/environments/{}", env_id))?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    print_message("Environment deleted successfully.");
    Ok(())
}

pub fn reset_db_password(
    client: &ApiClient,
    env_id: &str,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let response: Value = client.post_empty(&format!(
        "/api/v1/vector/environments/{}/db/reset-password",
        env_id
    ))?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    print_message("Database password reset successfully.");
    Ok(())
}

// Secret subcommands

pub fn secret_list(
    client: &ApiClient,
    env_id: &str,
    page: u32,
    per_page: u32,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let query = PaginationQuery { page, per_page };
    let response: Value = client.get_with_query(
        &format!("/api/v1/vector/environments/{}/secrets", env_id),
        &query,
    )?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    let secrets = response["data"]
        .as_array()
        .ok_or_else(|| ApiError::Other("Invalid response format".to_string()))?;

    if secrets.is_empty() {
        print_message("No secrets found.");
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
    let response: Value = client.get(&format!("/api/v1/vector/secrets/{}", secret_id))?;

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
    env_id: &str,
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

    let response: Value = client.post(
        &format!("/api/v1/vector/environments/{}/secrets", env_id),
        &body,
    )?;

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

    let response: Value = client.put(&format!("/api/v1/vector/secrets/{}", secret_id), &body)?;

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
    let response: Value = client.delete(&format!("/api/v1/vector/secrets/{}", secret_id))?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    print_message("Secret deleted successfully.");
    Ok(())
}

// Environment DB commands

#[derive(Debug, Serialize)]
struct EnvImportOptions {
    #[serde(skip_serializing_if = "std::ops::Not::not")]
    drop_tables: bool,
    #[serde(skip_serializing_if = "std::ops::Not::not")]
    disable_foreign_keys: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    search_replace: Option<EnvSearchReplace>,
}

#[derive(Debug, Serialize)]
struct EnvSearchReplace {
    from: String,
    to: String,
}

#[derive(Debug, Serialize)]
struct EnvCreateImportSessionRequest {
    #[serde(skip_serializing_if = "Option::is_none")]
    filename: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    content_length: Option<u64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    content_md5: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    options: Option<EnvImportOptions>,
}

#[derive(Debug, Serialize)]
struct PromoteRequest {
    #[serde(skip_serializing_if = "std::ops::Not::not")]
    drop_tables: bool,
    #[serde(skip_serializing_if = "std::ops::Not::not")]
    disable_foreign_keys: bool,
}

#[allow(clippy::too_many_arguments)]
pub fn db_import(
    client: &ApiClient,
    env_id: &str,
    file_path: &Path,
    drop_tables: bool,
    disable_foreign_keys: bool,
    search_replace_from: Option<String>,
    search_replace_to: Option<String>,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let metadata = std::fs::metadata(file_path)
        .map_err(|e| ApiError::Other(format!("Failed to read file: {}", e)))?;

    if metadata.len() > 50 * 1024 * 1024 {
        return Err(ApiError::Other(
            "File too large for direct import. Use 'env db import-session' for files over 50MB."
                .to_string(),
        ));
    }

    let mut path = format!("/api/v1/vector/environments/{}/db/import", env_id);
    let mut params = vec![];
    if drop_tables {
        params.push("drop_tables=true".to_string());
    }
    if disable_foreign_keys {
        params.push("disable_foreign_keys=true".to_string());
    }
    if let Some(ref from) = search_replace_from {
        params.push(format!("search_replace_from={}", from));
    }
    if let Some(ref to) = search_replace_to {
        params.push(format!("search_replace_to={}", to));
    }
    if !params.is_empty() {
        path = format!("{}?{}", path, params.join("&"));
    }

    let response: Value = client.post_file(&path, file_path)?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    let data = &response["data"];
    if data["success"].as_bool().unwrap_or(false) {
        print_message(&format!(
            "Database imported successfully ({}ms).",
            data["duration_ms"].as_u64().unwrap_or(0)
        ));
    } else {
        return Err(ApiError::Other(
            data["error"]
                .as_str()
                .unwrap_or("Import failed")
                .to_string(),
        ));
    }

    Ok(())
}

#[allow(clippy::too_many_arguments)]
pub fn db_import_session_create(
    client: &ApiClient,
    env_id: &str,
    filename: Option<String>,
    content_length: Option<u64>,
    drop_tables: bool,
    disable_foreign_keys: bool,
    search_replace_from: Option<String>,
    search_replace_to: Option<String>,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let search_replace = match (search_replace_from, search_replace_to) {
        (Some(from), Some(to)) => Some(EnvSearchReplace { from, to }),
        _ => None,
    };

    let options = if drop_tables || disable_foreign_keys || search_replace.is_some() {
        Some(EnvImportOptions {
            drop_tables,
            disable_foreign_keys,
            search_replace,
        })
    } else {
        None
    };

    let body = EnvCreateImportSessionRequest {
        filename,
        content_length,
        content_md5: None,
        options,
    };

    let response: Value = client.post(
        &format!("/api/v1/vector/environments/{}/db/imports", env_id),
        &body,
    )?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    let data = &response["data"];
    print_key_value(vec![
        ("Import ID", data["id"].as_str().unwrap_or("-").to_string()),
        ("Status", data["status"].as_str().unwrap_or("-").to_string()),
        (
            "Upload URL",
            format_option(&data["upload_url"].as_str().map(String::from)),
        ),
        (
            "Expires",
            format_option(&data["upload_expires_at"].as_str().map(String::from)),
        ),
    ]);

    print_message("\nUpload your SQL file to the URL above, then run:");
    print_message(&format!(
        "  vector env db import-session run {} {}",
        env_id,
        data["id"].as_str().unwrap_or("IMPORT_ID")
    ));

    Ok(())
}

pub fn db_import_session_run(
    client: &ApiClient,
    env_id: &str,
    import_id: &str,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let response: Value = client.post_empty(&format!(
        "/api/v1/vector/environments/{}/db/imports/{}/run",
        env_id, import_id
    ))?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    let data = &response["data"];
    print_message(&format!(
        "Import started: {} ({})",
        import_id,
        data["status"].as_str().unwrap_or("-")
    ));

    Ok(())
}

pub fn db_import_session_status(
    client: &ApiClient,
    env_id: &str,
    import_id: &str,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let response: Value = client.get(&format!(
        "/api/v1/vector/environments/{}/db/imports/{}",
        env_id, import_id
    ))?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    let data = &response["data"];
    print_key_value(vec![
        ("Import ID", data["id"].as_str().unwrap_or("-").to_string()),
        ("Status", data["status"].as_str().unwrap_or("-").to_string()),
        (
            "Filename",
            format_option(&data["filename"].as_str().map(String::from)),
        ),
        (
            "Duration (ms)",
            format_option(&data["duration_ms"].as_u64().map(|v| v.to_string())),
        ),
        (
            "Error",
            format_option(&data["error_message"].as_str().map(String::from)),
        ),
        (
            "Created",
            format_option(&data["created_at"].as_str().map(String::from)),
        ),
        (
            "Completed",
            format_option(&data["completed_at"].as_str().map(String::from)),
        ),
    ]);

    Ok(())
}

pub fn db_promote(
    client: &ApiClient,
    env_id: &str,
    drop_tables: bool,
    disable_foreign_keys: bool,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let body = PromoteRequest {
        drop_tables,
        disable_foreign_keys,
    };

    let response: Value = client.post(
        &format!("/api/v1/vector/environments/{}/db/promote", env_id),
        &body,
    )?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    let data = &response["data"];
    print_message(&format!(
        "Promote started: {} ({})",
        data["id"].as_str().unwrap_or("-"),
        data["status"].as_str().unwrap_or("-")
    ));

    Ok(())
}

pub fn db_promote_status(
    client: &ApiClient,
    env_id: &str,
    promote_id: &str,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let response: Value = client.get(&format!(
        "/api/v1/vector/environments/{}/db/promotes/{}",
        env_id, promote_id
    ))?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    let data = &response["data"];
    print_key_value(vec![
        ("Promote ID", data["id"].as_str().unwrap_or("-").to_string()),
        ("Status", data["status"].as_str().unwrap_or("-").to_string()),
        (
            "Duration (ms)",
            format_option(&data["duration_ms"].as_u64().map(|v| v.to_string())),
        ),
        (
            "Error",
            format_option(&data["error_message"].as_str().map(String::from)),
        ),
        (
            "Created",
            format_option(&data["created_at"].as_str().map(String::from)),
        ),
        (
            "Completed",
            format_option(&data["completed_at"].as_str().map(String::from)),
        ),
    ]);

    Ok(())
}

// Helper function to format tags
fn format_tags(value: &Value) -> String {
    if let Some(tags) = value.as_array() {
        if tags.is_empty() {
            return "-".to_string();
        }
        tags.iter()
            .filter_map(|t| t.as_str())
            .collect::<Vec<_>>()
            .join(", ")
    } else {
        "-".to_string()
    }
}
