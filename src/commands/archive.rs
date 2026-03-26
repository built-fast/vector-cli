use std::io::{Read, Seek, SeekFrom};
use std::path::Path;
use std::thread;
use std::time::Duration;

use serde::Serialize;
use serde_json::Value;

use crate::api::{ApiClient, ApiError, CompletedPart};
use crate::output::{OutputFormat, format_option, print_json, print_key_value, print_message};

struct UploadPart {
    part_number: u64,
    url: String,
}

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
    let is_multipart = data["is_multipart"].as_bool().unwrap_or(false);

    if format == OutputFormat::Table {
        print_message(&format!("Import ID: {}", import_id));
    }

    // Step 2: Upload file
    let size_mb = content_length as f64 / 1_048_576.0;
    let completed_parts = if is_multipart {
        let upload_parts = parse_upload_parts(data)?;
        let part_count = upload_parts.len();

        if format == OutputFormat::Table {
            print_message(&format!(
                "Uploading {} ({:.1} MB) in {} parts...",
                filename, size_mb, part_count
            ));
        }

        Some(upload_multipart(
            client,
            path,
            content_length,
            &upload_parts,
            format,
        )?)
    } else {
        let upload_url = data["upload_url"]
            .as_str()
            .ok_or_else(|| ApiError::Other("Missing upload URL in response".to_string()))?;

        if format == OutputFormat::Table {
            print_message(&format!("Uploading {} ({:.1} MB)...", filename, size_mb));
        }

        let file_handle = std::fs::File::open(path)
            .map_err(|e| ApiError::Other(format!("Cannot open file: {}", e)))?;

        client.put_file(upload_url, file_handle, content_length)?;
        None
    };

    if format == OutputFormat::Table {
        print_message("Upload complete.");
    }

    // Step 3: Trigger import
    if format == OutputFormat::Table {
        print_message("Starting import...");
    }

    let run_path = format!("/api/v1/vector/sites/{}/imports/{}/run", site_id, import_id);

    let run_response: Value = if let Some(ref parts) = completed_parts {
        let body = serde_json::json!({ "parts": parts });
        client.post(&run_path, &body)?
    } else {
        client.post_empty(&run_path)?
    };

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

fn parse_upload_parts(data: &Value) -> Result<Vec<UploadPart>, ApiError> {
    let parts_array = data["upload_parts"]
        .as_array()
        .ok_or_else(|| ApiError::Other("Missing upload_parts in response".to_string()))?;

    parts_array
        .iter()
        .map(|p| {
            let part_number = p["part_number"]
                .as_u64()
                .ok_or_else(|| ApiError::Other("Missing part_number".to_string()))?;
            let url = p["url"]
                .as_str()
                .ok_or_else(|| ApiError::Other("Missing part url".to_string()))?
                .to_string();
            Ok(UploadPart { part_number, url })
        })
        .collect()
}

fn upload_multipart(
    client: &ApiClient,
    file_path: &Path,
    content_length: u64,
    parts: &[UploadPart],
    format: OutputFormat,
) -> Result<Vec<CompletedPart>, ApiError> {
    let part_count = parts.len() as u64;
    let base_size = content_length / part_count;
    let last_size = content_length - base_size * (part_count - 1);

    let mut completed = Vec::with_capacity(parts.len());

    for (i, part) in parts.iter().enumerate() {
        let chunk_size = if (i as u64) < part_count - 1 {
            base_size
        } else {
            last_size
        };
        let offset = base_size * i as u64;

        if format == OutputFormat::Table {
            print_message(&format!("Uploading part {}/{}...", i + 1, part_count));
        }

        let mut file = std::fs::File::open(file_path)
            .map_err(|e| ApiError::Other(format!("Cannot open file: {}", e)))?;
        file.seek(SeekFrom::Start(offset))
            .map_err(|e| ApiError::Other(format!("Cannot seek file: {}", e)))?;
        let reader = file.take(chunk_size);

        let etag = client.put_file_part(&part.url, reader, chunk_size)?;

        completed.push(CompletedPart {
            part_number: part.part_number,
            etag,
        });
    }

    Ok(completed)
}
