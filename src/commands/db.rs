use serde::Serialize;
use serde_json::Value;
use std::path::Path;

use crate::api::{ApiClient, ApiError};
use crate::output::{format_option, print_json, print_key_value, print_message, OutputFormat};

#[derive(Debug, Serialize)]
struct CreateImportSessionRequest {
    #[serde(skip_serializing_if = "Option::is_none")]
    filename: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    content_length: Option<u64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    content_md5: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    options: Option<ImportOptions>,
}

#[derive(Debug, Serialize)]
struct ImportOptions {
    #[serde(skip_serializing_if = "std::ops::Not::not")]
    drop_tables: bool,
    #[serde(skip_serializing_if = "std::ops::Not::not")]
    disable_foreign_keys: bool,
}

#[derive(Debug, Serialize)]
struct CreateExportRequest {
    #[serde(skip_serializing_if = "Option::is_none")]
    format: Option<String>,
}

pub fn import_direct(
    client: &ApiClient,
    site_id: &str,
    file_path: &Path,
    drop_tables: bool,
    disable_foreign_keys: bool,
    format: OutputFormat,
) -> Result<(), ApiError> {
    // Check file size - direct import only supports files under 50MB
    let metadata = std::fs::metadata(file_path)
        .map_err(|e| ApiError::Other(format!("Failed to read file: {}", e)))?;

    if metadata.len() > 50 * 1024 * 1024 {
        return Err(ApiError::Other(
            "File too large for direct import. Use 'import-session' for files over 50MB."
                .to_string(),
        ));
    }

    let mut path = format!("/api/v1/vector/sites/{}/db/import", site_id);
    let mut params = vec![];
    if drop_tables {
        params.push("drop_tables=true");
    }
    if disable_foreign_keys {
        params.push("disable_foreign_keys=true");
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

pub fn import_session_create(
    client: &ApiClient,
    site_id: &str,
    filename: Option<String>,
    content_length: Option<u64>,
    drop_tables: bool,
    disable_foreign_keys: bool,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let options = if drop_tables || disable_foreign_keys {
        Some(ImportOptions {
            drop_tables,
            disable_foreign_keys,
        })
    } else {
        None
    };

    let body = CreateImportSessionRequest {
        filename,
        content_length,
        content_md5: None,
        options,
    };

    let response: Value = client.post(
        &format!("/api/v1/vector/sites/{}/db/imports", site_id),
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
        "  vector db import-session run {} {}",
        site_id,
        data["id"].as_str().unwrap_or("IMPORT_ID")
    ));

    Ok(())
}

pub fn import_session_run(
    client: &ApiClient,
    site_id: &str,
    import_id: &str,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let response: Value = client.post_empty(&format!(
        "/api/v1/vector/sites/{}/db/imports/{}/run",
        site_id, import_id
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

pub fn import_session_status(
    client: &ApiClient,
    site_id: &str,
    import_id: &str,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let response: Value = client.get(&format!(
        "/api/v1/vector/sites/{}/db/imports/{}",
        site_id, import_id
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

pub fn export_create(
    client: &ApiClient,
    site_id: &str,
    export_format: Option<String>,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let body = CreateExportRequest {
        format: export_format,
    };

    let response: Value = client.post(
        &format!("/api/v1/vector/sites/{}/db/export", site_id),
        &body,
    )?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    let data = &response["data"];
    print_message(&format!(
        "Export started: {} ({})",
        data["id"].as_str().unwrap_or("-"),
        data["status"].as_str().unwrap_or("-")
    ));
    print_message("\nCheck status with:");
    print_message(&format!(
        "  vector db export status {} {}",
        site_id,
        data["id"].as_str().unwrap_or("EXPORT_ID")
    ));

    Ok(())
}

pub fn export_status(
    client: &ApiClient,
    site_id: &str,
    export_id: &str,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let response: Value = client.get(&format!(
        "/api/v1/vector/sites/{}/db/exports/{}",
        site_id, export_id
    ))?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    let data = &response["data"];
    print_key_value(vec![
        ("Export ID", data["id"].as_str().unwrap_or("-").to_string()),
        ("Status", data["status"].as_str().unwrap_or("-").to_string()),
        (
            "Format",
            format_option(&data["format"].as_str().map(String::from)),
        ),
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
