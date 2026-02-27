use serde::Serialize;
use serde_json::Value;

use crate::api::{ApiClient, ApiError};
use crate::output::{
    OutputFormat, extract_pagination, format_archivable_type, format_option, print_json,
    print_key_value, print_message, print_pagination, print_table,
};

#[derive(Debug, Serialize)]
struct ListBackupsQuery {
    #[serde(skip_serializing_if = "Option::is_none")]
    r#type: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    site_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    environment_id: Option<String>,
    page: u32,
    per_page: u32,
}

#[derive(Debug, Serialize)]
struct CreateBackupRequest {
    #[serde(skip_serializing_if = "Option::is_none")]
    site_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    environment_id: Option<String>,
    scope: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    description: Option<String>,
}

pub fn list(
    client: &ApiClient,
    site_id: Option<String>,
    environment_id: Option<String>,
    backup_type: Option<String>,
    page: u32,
    per_page: u32,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let query = ListBackupsQuery {
        r#type: backup_type,
        site_id,
        environment_id,
        page,
        per_page,
    };
    let response: Value = client.get_with_query("/api/v1/vector/backups", &query)?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    let backups = response["data"]
        .as_array()
        .ok_or_else(|| ApiError::Other("Invalid response format".to_string()))?;

    if backups.is_empty() {
        print_message("No backups found.");
        return Ok(());
    }

    let rows: Vec<Vec<String>> = backups
        .iter()
        .map(|b| {
            vec![
                b["id"].as_str().unwrap_or("-").to_string(),
                b["archivable_type"]
                    .as_str()
                    .map(format_archivable_type)
                    .unwrap_or_else(|| "-".to_string()),
                b["type"].as_str().unwrap_or("-").to_string(),
                b["scope"].as_str().unwrap_or("-").to_string(),
                b["status"].as_str().unwrap_or("-").to_string(),
                format_option(&b["description"].as_str().map(String::from)),
                format_option(&b["created_at"].as_str().map(String::from)),
            ]
        })
        .collect();

    print_table(
        vec![
            "ID",
            "Model",
            "Type",
            "Scope",
            "Status",
            "Description",
            "Created",
        ],
        rows,
    );

    if let Some((current, last, total)) = extract_pagination(&response) {
        print_pagination(current, last, total);
    }

    Ok(())
}

pub fn show(client: &ApiClient, backup_id: &str, format: OutputFormat) -> Result<(), ApiError> {
    let response: Value = client.get(&format!("/api/v1/vector/backups/{}", backup_id))?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    let backup = &response["data"];

    print_key_value(vec![
        ("ID", backup["id"].as_str().unwrap_or("-").to_string()),
        (
            "Model",
            backup["archivable_type"]
                .as_str()
                .map(format_archivable_type)
                .unwrap_or_else(|| "-".to_string()),
        ),
        (
            "Model ID",
            backup["archivable_id"].as_str().unwrap_or("-").to_string(),
        ),
        ("Type", backup["type"].as_str().unwrap_or("-").to_string()),
        ("Scope", backup["scope"].as_str().unwrap_or("-").to_string()),
        (
            "Status",
            backup["status"].as_str().unwrap_or("-").to_string(),
        ),
        (
            "Description",
            format_option(&backup["description"].as_str().map(String::from)),
        ),
        (
            "File Snapshot ID",
            format_option(&backup["file_snapshot_id"].as_str().map(String::from)),
        ),
        (
            "Database Snapshot ID",
            format_option(&backup["database_snapshot_id"].as_str().map(String::from)),
        ),
        (
            "Started At",
            format_option(&backup["started_at"].as_str().map(String::from)),
        ),
        (
            "Completed At",
            format_option(&backup["completed_at"].as_str().map(String::from)),
        ),
        (
            "Created At",
            format_option(&backup["created_at"].as_str().map(String::from)),
        ),
        (
            "Updated At",
            format_option(&backup["updated_at"].as_str().map(String::from)),
        ),
    ]);

    Ok(())
}

pub fn create(
    client: &ApiClient,
    site_id: Option<String>,
    environment_id: Option<String>,
    scope: &str,
    description: Option<String>,
    format: OutputFormat,
) -> Result<(), ApiError> {
    if site_id.is_none() && environment_id.is_none() {
        return Err(ApiError::Other(
            "Either --site-id or --environment-id is required".to_string(),
        ));
    }

    let body = CreateBackupRequest {
        site_id,
        environment_id,
        scope: scope.to_string(),
        description,
    };

    let response: Value = client.post("/api/v1/vector/backups", &body)?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    let backup = &response["data"];
    print_message(&format!(
        "Backup created: {} ({})",
        backup["id"].as_str().unwrap_or("-"),
        backup["status"].as_str().unwrap_or("-")
    ));

    print_key_value(vec![
        ("ID", backup["id"].as_str().unwrap_or("-").to_string()),
        ("Type", backup["type"].as_str().unwrap_or("-").to_string()),
        (
            "Status",
            backup["status"].as_str().unwrap_or("-").to_string(),
        ),
        (
            "Description",
            format_option(&backup["description"].as_str().map(String::from)),
        ),
        (
            "Created At",
            format_option(&backup["created_at"].as_str().map(String::from)),
        ),
    ]);

    Ok(())
}

pub fn download_create(
    client: &ApiClient,
    backup_id: &str,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let response: Value =
        client.post_empty(&format!("/api/v1/vector/backups/{}/downloads", backup_id))?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    let data = &response["data"];
    print_message(&format!(
        "Download requested: {} ({})",
        data["id"].as_str().unwrap_or("-"),
        data["status"].as_str().unwrap_or("-")
    ));
    print_message("\nCheck status with:");
    print_message(&format!(
        "  vector backup download status {} {}",
        backup_id,
        data["id"].as_str().unwrap_or("DOWNLOAD_ID")
    ));

    Ok(())
}

pub fn download_status(
    client: &ApiClient,
    backup_id: &str,
    download_id: &str,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let response: Value = client.get(&format!(
        "/api/v1/vector/backups/{}/downloads/{}",
        backup_id, download_id
    ))?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    let data = &response["data"];
    print_key_value(vec![
        ("ID", data["id"].as_str().unwrap_or("-").to_string()),
        ("Status", data["status"].as_str().unwrap_or("-").to_string()),
        (
            "Size (bytes)",
            format_option(&data["size_bytes"].as_u64().map(|v| v.to_string())),
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
            "Download URL",
            format_option(&data["download_url"].as_str().map(String::from)),
        ),
        (
            "Download Expires",
            format_option(&data["download_expires_at"].as_str().map(String::from)),
        ),
        (
            "Started At",
            format_option(&data["started_at"].as_str().map(String::from)),
        ),
        (
            "Completed At",
            format_option(&data["completed_at"].as_str().map(String::from)),
        ),
        (
            "Created At",
            format_option(&data["created_at"].as_str().map(String::from)),
        ),
    ]);

    Ok(())
}
