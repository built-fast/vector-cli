use serde::Serialize;
use serde_json::Value;

use crate::api::{ApiClient, ApiError};
use crate::output::{
    extract_pagination, format_bool, format_option, print_json, print_key_value, print_message,
    print_pagination, print_table, OutputFormat,
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
}

#[derive(Debug, Serialize)]
struct UpdateSecretRequest {
    #[serde(skip_serializing_if = "Option::is_none")]
    key: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    value: Option<String>,
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
                format_option(&e["fqdn"].as_str().map(String::from)),
            ]
        })
        .collect();

    print_table(vec!["ID", "Name", "Status", "Production", "FQDN"], rows);

    if let Some((current, last, total)) = extract_pagination(&response) {
        print_pagination(current, last, total);
    }

    Ok(())
}

pub fn show(
    client: &ApiClient,
    env_id: &str,
    format: OutputFormat,
) -> Result<(), ApiError> {
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
            "FQDN",
            format_option(&env["fqdn"].as_str().map(String::from)),
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

    let response: Value =
        client.put(&format!("/api/v1/vector/environments/{}", env_id), &body)?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    print_message("Environment updated successfully.");
    Ok(())
}

pub fn delete(
    client: &ApiClient,
    env_id: &str,
    format: OutputFormat,
) -> Result<(), ApiError> {
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
        "/api/v1/vector/environments/{}/database/reset-password",
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
                format_option(&s["created_at"].as_str().map(String::from)),
            ]
        })
        .collect();

    print_table(vec!["ID", "Key", "Created"], rows);

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
    format: OutputFormat,
) -> Result<(), ApiError> {
    let body = CreateSecretRequest {
        key: key.to_string(),
        value: value.to_string(),
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
    format: OutputFormat,
) -> Result<(), ApiError> {
    let body = UpdateSecretRequest { key, value };

    let response: Value =
        client.put(&format!("/api/v1/vector/secrets/{}", secret_id), &body)?;

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
