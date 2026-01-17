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
struct CreateEnvRequest {
    name: String,
    #[serde(skip_serializing_if = "std::ops::Not::not")]
    is_production: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    custom_domain: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    php_version: Option<String>,
}

#[derive(Debug, Serialize)]
struct UpdateEnvRequest {
    #[serde(skip_serializing_if = "Option::is_none")]
    custom_domain: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    php_version: Option<String>,
}

pub fn list(
    client: &ApiClient,
    site_id: &str,
    page: u32,
    per_page: u32,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let query = PaginationQuery { page, per_page };
    let response: Value = client.get_with_query(
        &format!("/api/v1/vector/sites/{}/environments", site_id),
        &query,
    )?;

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
    site_id: &str,
    env_id: &str,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let response: Value = client.get(&format!(
        "/api/v1/vector/sites/{}/environments/{}",
        site_id, env_id
    ))?;

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

pub fn create(
    client: &ApiClient,
    site_id: &str,
    name: &str,
    is_production: bool,
    custom_domain: Option<String>,
    php_version: Option<String>,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let body = CreateEnvRequest {
        name: name.to_string(),
        is_production,
        custom_domain,
        php_version,
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
    site_id: &str,
    env_id: &str,
    custom_domain: Option<String>,
    php_version: Option<String>,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let body = UpdateEnvRequest {
        custom_domain,
        php_version,
    };

    let response: Value = client.put(
        &format!("/api/v1/vector/sites/{}/environments/{}", site_id, env_id),
        &body,
    )?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    print_message("Environment updated successfully.");
    Ok(())
}

pub fn delete(
    client: &ApiClient,
    site_id: &str,
    env_id: &str,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let response: Value = client.delete(&format!(
        "/api/v1/vector/sites/{}/environments/{}",
        site_id, env_id
    ))?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    print_message("Environment deleted successfully.");
    Ok(())
}

pub fn suspend(
    client: &ApiClient,
    site_id: &str,
    env_id: &str,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let response: Value = client.post_empty(&format!(
        "/api/v1/vector/sites/{}/environments/{}/suspend",
        site_id, env_id
    ))?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    print_message("Environment suspended successfully.");
    Ok(())
}

pub fn unsuspend(
    client: &ApiClient,
    site_id: &str,
    env_id: &str,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let response: Value = client.post_empty(&format!(
        "/api/v1/vector/sites/{}/environments/{}/unsuspend",
        site_id, env_id
    ))?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    print_message("Environment unsuspended successfully.");
    Ok(())
}
