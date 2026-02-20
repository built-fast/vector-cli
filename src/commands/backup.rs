use serde::Serialize;
use serde_json::Value;

use crate::api::{ApiClient, ApiError};
use crate::output::{
    OutputFormat, extract_pagination, format_option, print_json, print_key_value, print_message,
    print_pagination, print_table,
};

#[derive(Debug, Serialize)]
struct PaginationQuery {
    page: u32,
    per_page: u32,
}

#[derive(Debug, Serialize)]
struct CreateBackupRequest {
    r#type: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    description: Option<String>,
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
        &format!("/api/v1/vector/sites/{}/backups", site_id),
        &query,
    )?;

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
                b["type"].as_str().unwrap_or("-").to_string(),
                b["status"].as_str().unwrap_or("-").to_string(),
                format_option(&b["description"].as_str().map(String::from)),
                format_option(&b["created_at"].as_str().map(String::from)),
            ]
        })
        .collect();

    print_table(vec!["ID", "Type", "Status", "Description", "Created"], rows);

    if let Some((current, last, total)) = extract_pagination(&response) {
        print_pagination(current, last, total);
    }

    Ok(())
}

pub fn show(
    client: &ApiClient,
    site_id: &str,
    backup_id: &str,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let response: Value = client.get(&format!(
        "/api/v1/vector/sites/{}/backups/{}",
        site_id, backup_id
    ))?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    let backup = &response["data"];

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
            "Snapshot ID",
            format_option(&backup["snapshot_id"].as_str().map(String::from)),
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
    site_id: &str,
    description: Option<String>,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let body = CreateBackupRequest {
        r#type: "manual".to_string(),
        description,
    };

    let response: Value = client.post(
        &format!("/api/v1/vector/sites/{}/backups", site_id),
        &body,
    )?;

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
