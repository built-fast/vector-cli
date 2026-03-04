use std::path::Path;
use std::thread;
use std::time::Duration;

use serde::Serialize;
use serde_json::Value;

use crate::api::{ApiClient, ApiError};
use crate::output::{OutputFormat, format_option, print_json, print_key_value, print_message};

#[derive(Debug, Serialize)]
struct CreateImportSessionRequest {
    #[serde(skip_serializing_if = "Option::is_none")]
    filename: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    content_length: Option<u64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    options: Option<ImportOptions>,
}

#[derive(Debug, Serialize)]
struct ImportOptions {
    #[serde(skip_serializing_if = "std::ops::Not::not")]
    drop_tables: bool,
    #[serde(skip_serializing_if = "std::ops::Not::not")]
    disable_foreign_keys: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    search_replace: Option<SearchReplace>,
}

#[derive(Debug, Serialize)]
struct SearchReplace {
    from: String,
    to: String,
}

#[allow(clippy::too_many_arguments)]
pub fn import(
    client: &ApiClient,
    site_id: &str,
    file: &str,
    drop_tables: bool,
    disable_foreign_keys: bool,
    search_replace_from: Option<String>,
    search_replace_to: Option<String>,
    wait: bool,
    poll_interval: u64,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let path = Path::new(file);

    if !path.exists() {
        return Err(ApiError::Other(format!("File not found: {}", file)));
    }

    let metadata =
        std::fs::metadata(path).map_err(|e| ApiError::Other(format!("Cannot read file: {}", e)))?;

    let content_length = metadata.len();
    let filename = path
        .file_name()
        .and_then(|n| n.to_str())
        .unwrap_or(file)
        .to_string();

    // Build import options
    let search_replace = match (search_replace_from, search_replace_to) {
        (Some(from), Some(to)) => Some(SearchReplace { from, to }),
        _ => None,
    };

    let options = if drop_tables || disable_foreign_keys || search_replace.is_some() {
        Some(ImportOptions {
            drop_tables,
            disable_foreign_keys,
            search_replace,
        })
    } else {
        None
    };

    let body = CreateImportSessionRequest {
        filename: Some(filename.clone()),
        content_length: Some(content_length),
        options,
    };

    // Step 1: Create import session
    if format == OutputFormat::Table {
        print_message("Creating import session...");
    }

    let response: Value =
        client.post(&format!("/api/v1/vector/sites/{}/imports", site_id), &body)?;

    let data = &response["data"];
    let import_id = data["id"]
        .as_str()
        .ok_or_else(|| ApiError::Other("Missing import ID in response".to_string()))?;
    let upload_url = data["upload_url"]
        .as_str()
        .ok_or_else(|| ApiError::Other("Missing upload URL in response".to_string()))?;

    if format == OutputFormat::Table {
        print_message(&format!("Import ID: {}", import_id));
    }

    // Step 2: Upload file
    let size_mb = content_length as f64 / 1_048_576.0;
    if format == OutputFormat::Table {
        print_message(&format!("Uploading {} ({:.1} MB)...", filename, size_mb));
    }

    let file_handle = std::fs::File::open(path)
        .map_err(|e| ApiError::Other(format!("Cannot open file: {}", e)))?;

    client.put_file(upload_url, file_handle, content_length)?;

    if format == OutputFormat::Table {
        print_message("Upload complete.");
    }

    // Step 3: Trigger import
    if format == OutputFormat::Table {
        print_message("Starting import...");
    }

    let run_response: Value = client.post_empty(&format!(
        "/api/v1/vector/sites/{}/imports/{}/run",
        site_id, import_id
    ))?;

    if format == OutputFormat::Table {
        print_message("Import started.");
    }

    // Step 4: Poll if --wait
    if wait {
        if format == OutputFormat::Table {
            print_message("\nWaiting for import to complete...");
        }

        loop {
            thread::sleep(Duration::from_secs(poll_interval));

            let status_response: Value = client.get(&format!(
                "/api/v1/vector/sites/{}/imports/{}",
                site_id, import_id
            ))?;

            let status_data = &status_response["data"];
            let status = status_data["status"].as_str().unwrap_or("unknown");

            match status {
                "completed" => {
                    if format == OutputFormat::Json {
                        print_json(&status_response);
                    } else {
                        let duration = format_option(
                            &status_data["duration_ms"].as_u64().map(|v| v.to_string()),
                        );
                        print_message(&format!("Status: completed (duration: {}ms)", duration));
                    }
                    return Ok(());
                }
                "failed" => {
                    if format == OutputFormat::Json {
                        print_json(&status_response);
                        return Ok(());
                    }
                    let error_msg =
                        format_option(&status_data["error_message"].as_str().map(String::from));
                    return Err(ApiError::Other(format!("Import failed: {}", error_msg)));
                }
                _ => {
                    if format == OutputFormat::Table {
                        print_message(&format!("Status: {}", status));
                    }
                }
            }
        }
    }

    // Final output
    if format == OutputFormat::Json {
        print_json(&run_response);
    } else {
        print_key_value(vec![
            ("Import ID", import_id.to_string()),
            (
                "Status",
                run_response["data"]["status"]
                    .as_str()
                    .unwrap_or("-")
                    .to_string(),
            ),
        ]);
        print_message("\nCheck status with:");
        print_message(&format!(
            "  vector db import-session status {} {}",
            site_id, import_id
        ));
    }

    Ok(())
}
