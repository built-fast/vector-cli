use serde::Serialize;
use serde_json::Value;

use crate::api::{ApiClient, ApiError};
use crate::output::{
    extract_pagination, format_option, print_json, print_key_value, print_message,
    print_pagination, print_table, OutputFormat,
};

#[derive(Debug, Serialize)]
struct PaginationQuery {
    page: u32,
    per_page: u32,
}

#[derive(Debug, Serialize)]
struct CreateSiteRequest {
    domain: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    php_version: Option<String>,
}

#[derive(Debug, Serialize)]
struct UpdateSiteRequest {
    #[serde(skip_serializing_if = "Option::is_none")]
    php_version: Option<String>,
}

#[derive(Debug, Serialize)]
struct PurgeCacheRequest {
    #[serde(skip_serializing_if = "Option::is_none")]
    path: Option<String>,
}

#[derive(Debug, Serialize)]
struct LogsQuery {
    r#type: String,
    lines: u32,
}

pub fn list(
    client: &ApiClient,
    page: u32,
    per_page: u32,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let query = PaginationQuery { page, per_page };
    let response: Value = client.get_with_query("/api/v1/vector/sites", &query)?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    let sites = response["data"]
        .as_array()
        .ok_or_else(|| ApiError::Other("Invalid response format".to_string()))?;

    if sites.is_empty() {
        print_message("No sites found.");
        return Ok(());
    }

    let rows: Vec<Vec<String>> = sites
        .iter()
        .map(|s| {
            vec![
                s["id"].as_str().unwrap_or("-").to_string(),
                s["domain"].as_str().unwrap_or("-").to_string(),
                s["status"].as_str().unwrap_or("-").to_string(),
                format_option(&s["php_version"].as_str().map(String::from)),
            ]
        })
        .collect();

    print_table(vec!["ID", "Domain", "Status", "PHP"], rows);

    if let Some((current, last, total)) = extract_pagination(&response) {
        print_pagination(current, last, total);
    }

    Ok(())
}

pub fn show(client: &ApiClient, id: &str, format: OutputFormat) -> Result<(), ApiError> {
    let response: Value = client.get(&format!("/api/v1/vector/sites/{}", id))?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    let site = &response["data"];

    print_key_value(vec![
        ("ID", site["id"].as_str().unwrap_or("-").to_string()),
        ("Domain", site["domain"].as_str().unwrap_or("-").to_string()),
        ("Status", site["status"].as_str().unwrap_or("-").to_string()),
        (
            "PHP Version",
            format_option(&site["php_version"].as_str().map(String::from)),
        ),
        (
            "Created",
            format_option(&site["created_at"].as_str().map(String::from)),
        ),
        (
            "Updated",
            format_option(&site["updated_at"].as_str().map(String::from)),
        ),
    ]);

    Ok(())
}

pub fn create(
    client: &ApiClient,
    domain: &str,
    php_version: Option<String>,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let body = CreateSiteRequest {
        domain: domain.to_string(),
        php_version,
    };

    let response: Value = client.post("/api/v1/vector/sites", &body)?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    let site = &response["data"];
    print_message(&format!(
        "Site created: {} ({})",
        site["domain"].as_str().unwrap_or("-"),
        site["id"].as_str().unwrap_or("-")
    ));

    Ok(())
}

pub fn update(
    client: &ApiClient,
    id: &str,
    php_version: Option<String>,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let body = UpdateSiteRequest { php_version };
    let response: Value = client.put(&format!("/api/v1/vector/sites/{}", id), &body)?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    print_message("Site updated successfully.");
    Ok(())
}

pub fn delete(
    client: &ApiClient,
    id: &str,
    force: bool,
    format: OutputFormat,
) -> Result<(), ApiError> {
    if !force {
        eprint!("Are you sure you want to delete site {}? [y/N] ", id);
        let mut input = String::new();
        std::io::stdin().read_line(&mut input).ok();
        if !input.trim().eq_ignore_ascii_case("y") {
            print_message("Aborted.");
            return Ok(());
        }
    }

    let response: Value = client.delete(&format!("/api/v1/vector/sites/{}", id))?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    print_message("Site deleted successfully.");
    Ok(())
}

pub fn suspend(client: &ApiClient, id: &str, format: OutputFormat) -> Result<(), ApiError> {
    let response: Value = client.post_empty(&format!("/api/v1/vector/sites/{}/suspend", id))?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    print_message("Site suspended successfully.");
    Ok(())
}

pub fn unsuspend(client: &ApiClient, id: &str, format: OutputFormat) -> Result<(), ApiError> {
    let response: Value = client.post_empty(&format!("/api/v1/vector/sites/{}/unsuspend", id))?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    print_message("Site unsuspended successfully.");
    Ok(())
}

pub fn reset_sftp_password(
    client: &ApiClient,
    id: &str,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let response: Value =
        client.post_empty(&format!("/api/v1/vector/sites/{}/reset-sftp-password", id))?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    if let Some(password) = response["data"]["sftp_password"].as_str() {
        print_key_value(vec![("New SFTP Password", password.to_string())]);
    } else {
        print_message("SFTP password reset successfully.");
    }

    Ok(())
}

pub fn reset_db_password(
    client: &ApiClient,
    id: &str,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let response: Value = client.post_empty(&format!(
        "/api/v1/vector/sites/{}/reset-database-password",
        id
    ))?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    if let Some(password) = response["data"]["database_password"].as_str() {
        print_key_value(vec![("New DB Password", password.to_string())]);
    } else {
        print_message("Database password reset successfully.");
    }

    Ok(())
}

pub fn purge_cache(
    client: &ApiClient,
    id: &str,
    path: Option<String>,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let body = PurgeCacheRequest { path };
    let response: Value =
        client.post(&format!("/api/v1/vector/sites/{}/purge-cache", id), &body)?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    print_message("Cache purged successfully.");
    Ok(())
}

pub fn logs(
    client: &ApiClient,
    id: &str,
    log_type: &str,
    lines: u32,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let query = LogsQuery {
        r#type: log_type.to_string(),
        lines,
    };
    let response: Value =
        client.get_with_query(&format!("/api/v1/vector/sites/{}/logs", id), &query)?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    if let Some(logs) = response["data"]["logs"].as_str() {
        println!("{}", logs);
    } else if let Some(logs) = response["data"].as_str() {
        println!("{}", logs);
    } else {
        print_message("No logs available.");
    }

    Ok(())
}
