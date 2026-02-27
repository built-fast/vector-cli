use serde::Serialize;
use serde_json::Value;

use crate::api::{ApiClient, ApiError};
use crate::output::{
    OutputFormat, extract_pagination, format_archivable_type, format_option, print_json,
    print_key_value, print_message, print_pagination, print_table,
};

#[derive(Debug, Serialize)]
struct ListRestoresQuery {
    #[serde(skip_serializing_if = "Option::is_none")]
    r#type: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    site_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    environment_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    backup_id: Option<String>,
    page: u32,
    per_page: u32,
}

#[derive(Debug, Serialize)]
struct CreateRestoreRequest {
    backup_id: String,
    scope: String,
}

pub fn list(
    client: &ApiClient,
    site_id: Option<String>,
    environment_id: Option<String>,
    restore_type: Option<String>,
    backup_id: Option<String>,
    page: u32,
    per_page: u32,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let query = ListRestoresQuery {
        r#type: restore_type,
        site_id,
        environment_id,
        backup_id,
        page,
        per_page,
    };
    let response: Value = client.get_with_query("/api/v1/vector/restores", &query)?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    let restores = response["data"]
        .as_array()
        .ok_or_else(|| ApiError::Other("Invalid response format".to_string()))?;

    if restores.is_empty() {
        print_message("No restores found.");
        return Ok(());
    }

    let rows: Vec<Vec<String>> = restores
        .iter()
        .map(|r| {
            vec![
                r["id"].as_str().unwrap_or("-").to_string(),
                r["archivable_type"]
                    .as_str()
                    .map(format_archivable_type)
                    .unwrap_or_else(|| "-".to_string()),
                r["vector_backup_id"].as_str().unwrap_or("-").to_string(),
                r["scope"].as_str().unwrap_or("-").to_string(),
                r["status"].as_str().unwrap_or("-").to_string(),
                format_option(&r["created_at"].as_str().map(String::from)),
            ]
        })
        .collect();

    print_table(
        vec!["ID", "Model", "Backup ID", "Scope", "Status", "Created"],
        rows,
    );

    if let Some((current, last, total)) = extract_pagination(&response) {
        print_pagination(current, last, total);
    }

    Ok(())
}

pub fn show(
    client: &ApiClient,
    restore_id: &str,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let response: Value =
        client.get(&format!("/api/v1/vector/restores/{}", restore_id))?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    let restore = &response["data"];

    print_key_value(vec![
        ("ID", restore["id"].as_str().unwrap_or("-").to_string()),
        (
            "Model",
            restore["archivable_type"]
                .as_str()
                .map(format_archivable_type)
                .unwrap_or_else(|| "-".to_string()),
        ),
        (
            "Model ID",
            restore["archivable_id"]
                .as_str()
                .unwrap_or("-")
                .to_string(),
        ),
        (
            "Backup ID",
            restore["vector_backup_id"]
                .as_str()
                .unwrap_or("-")
                .to_string(),
        ),
        (
            "Scope",
            restore["scope"].as_str().unwrap_or("-").to_string(),
        ),
        (
            "Trigger",
            restore["trigger"].as_str().unwrap_or("-").to_string(),
        ),
        (
            "Status",
            restore["status"].as_str().unwrap_or("-").to_string(),
        ),
        (
            "Error Message",
            format_option(&restore["error_message"].as_str().map(String::from)),
        ),
        (
            "Duration (ms)",
            format_option(&restore["duration_ms"].as_u64().map(|d| d.to_string())),
        ),
        (
            "Started At",
            format_option(&restore["started_at"].as_str().map(String::from)),
        ),
        (
            "Completed At",
            format_option(&restore["completed_at"].as_str().map(String::from)),
        ),
        (
            "Created At",
            format_option(&restore["created_at"].as_str().map(String::from)),
        ),
        (
            "Updated At",
            format_option(&restore["updated_at"].as_str().map(String::from)),
        ),
    ]);

    Ok(())
}

pub fn create(
    client: &ApiClient,
    backup_id: &str,
    scope: &str,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let body = CreateRestoreRequest {
        backup_id: backup_id.to_string(),
        scope: scope.to_string(),
    };

    let response: Value = client.post("/api/v1/vector/restores", &body)?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    let restore = &response["data"];
    let restore_id = restore["id"].as_str().unwrap_or("-");

    print_message(&format!(
        "Restore initiated. Use `vector restore show {}` to check progress.",
        restore_id
    ));

    print_key_value(vec![
        ("ID", restore_id.to_string()),
        (
            "Backup ID",
            restore["vector_backup_id"]
                .as_str()
                .unwrap_or("-")
                .to_string(),
        ),
        (
            "Scope",
            restore["scope"].as_str().unwrap_or("-").to_string(),
        ),
        (
            "Status",
            restore["status"].as_str().unwrap_or("-").to_string(),
        ),
        (
            "Created At",
            format_option(&restore["created_at"].as_str().map(String::from)),
        ),
    ]);

    Ok(())
}
