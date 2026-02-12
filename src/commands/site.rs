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
struct CreateSiteRequest {
    your_customer_id: String,
    dev_php_version: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    tags: Option<Vec<String>>,
}

#[derive(Debug, Serialize)]
struct UpdateSiteRequest {
    #[serde(skip_serializing_if = "Option::is_none")]
    your_customer_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    tags: Option<Vec<String>>,
}

#[derive(Debug, Serialize)]
struct CloneSiteRequest {
    #[serde(skip_serializing_if = "Option::is_none")]
    your_customer_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    dev_php_version: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    tags: Option<Vec<String>>,
}

#[derive(Debug, Serialize)]
struct PurgeCacheRequest {
    #[serde(skip_serializing_if = "Option::is_none")]
    cache_tag: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    url: Option<String>,
}

#[derive(Debug, Serialize)]
struct LogsQuery {
    #[serde(skip_serializing_if = "Option::is_none")]
    start_time: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    end_time: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    limit: Option<u32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    environment: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    deployment_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    level: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    cursor: Option<String>,
}

#[derive(Debug, Serialize)]
struct CreateSshKeyRequest {
    name: String,
    public_key: String,
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
                s["status"].as_str().unwrap_or("-").to_string(),
                format_option(&s["your_customer_id"].as_str().map(String::from)),
                format_option(&s["dev_domain"].as_str().map(String::from)),
            ]
        })
        .collect();

    print_table(vec!["ID", "Status", "Customer ID", "Dev Domain"], rows);

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
        ("Status", site["status"].as_str().unwrap_or("-").to_string()),
        (
            "Customer ID",
            format_option(&site["your_customer_id"].as_str().map(String::from)),
        ),
        (
            "Dev Domain",
            format_option(&site["dev_domain"].as_str().map(String::from)),
        ),
        (
            "Dev PHP Version",
            format_option(&site["dev_php_version"].as_str().map(String::from)),
        ),
        (
            "Dev DB Host",
            format_option(&site["dev_db_host"].as_str().map(String::from)),
        ),
        (
            "Dev DB Name",
            format_option(&site["dev_db_name"].as_str().map(String::from)),
        ),
        ("Tags", format_tags(&site["tags"])),
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
    customer_id: &str,
    dev_php_version: &str,
    tags: Option<Vec<String>>,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let body = CreateSiteRequest {
        your_customer_id: customer_id.to_string(),
        dev_php_version: dev_php_version.to_string(),
        tags,
    };

    let response: Value = client.post("/api/v1/vector/sites", &body)?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    let site = &response["data"];
    print_message(&format!(
        "Site created: {} ({})",
        site["id"].as_str().unwrap_or("-"),
        site["status"].as_str().unwrap_or("-")
    ));

    Ok(())
}

pub fn update(
    client: &ApiClient,
    id: &str,
    customer_id: Option<String>,
    tags: Option<Vec<String>>,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let body = UpdateSiteRequest {
        your_customer_id: customer_id,
        tags,
    };
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

pub fn clone(
    client: &ApiClient,
    id: &str,
    customer_id: Option<String>,
    dev_php_version: Option<String>,
    tags: Option<Vec<String>>,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let body = CloneSiteRequest {
        your_customer_id: customer_id,
        dev_php_version,
        tags,
    };

    let response: Value = client.post(&format!("/api/v1/vector/sites/{}/clone", id), &body)?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    let site = &response["data"];
    print_message(&format!(
        "Site clone initiated: {} ({})",
        site["id"].as_str().unwrap_or("-"),
        site["status"].as_str().unwrap_or("-")
    ));

    Ok(())
}

pub fn suspend(client: &ApiClient, id: &str, format: OutputFormat) -> Result<(), ApiError> {
    let response: Value = client.put_empty(&format!("/api/v1/vector/sites/{}/suspend", id))?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    print_message("Site suspension initiated.");
    Ok(())
}

pub fn unsuspend(client: &ApiClient, id: &str, format: OutputFormat) -> Result<(), ApiError> {
    let response: Value = client.put_empty(&format!("/api/v1/vector/sites/{}/unsuspend", id))?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    print_message("Site unsuspension initiated.");
    Ok(())
}

pub fn reset_sftp_password(
    client: &ApiClient,
    id: &str,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let response: Value =
        client.post_empty(&format!("/api/v1/vector/sites/{}/sftp/reset-password", id))?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    if let Some(sftp) = response["data"]["dev_sftp"].as_object() {
        print_key_value(vec![
            (
                "Hostname",
                format_option(
                    &sftp
                        .get("hostname")
                        .and_then(|v| v.as_str())
                        .map(String::from),
                ),
            ),
            (
                "Port",
                format_option(
                    &sftp
                        .get("port")
                        .and_then(|v| v.as_u64())
                        .map(|v| v.to_string()),
                ),
            ),
            (
                "Username",
                format_option(
                    &sftp
                        .get("username")
                        .and_then(|v| v.as_str())
                        .map(String::from),
                ),
            ),
            (
                "Password",
                format_option(
                    &sftp
                        .get("password")
                        .and_then(|v| v.as_str())
                        .map(String::from),
                ),
            ),
        ]);
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
    let response: Value =
        client.post_empty(&format!("/api/v1/vector/sites/{}/db/reset-password", id))?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    let data = &response["data"];
    print_key_value(vec![
        (
            "Username",
            format_option(&data["dev_db_username"].as_str().map(String::from)),
        ),
        (
            "Password",
            format_option(&data["dev_db_password"].as_str().map(String::from)),
        ),
    ]);

    Ok(())
}

pub fn purge_cache(
    client: &ApiClient,
    id: &str,
    cache_tag: Option<String>,
    url: Option<String>,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let body = PurgeCacheRequest { cache_tag, url };
    let response: Value =
        client.post(&format!("/api/v1/vector/sites/{}/purge-cache", id), &body)?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    print_message("Cache purged successfully.");
    Ok(())
}

#[allow(clippy::too_many_arguments)]
pub fn logs(
    client: &ApiClient,
    id: &str,
    start_time: Option<String>,
    end_time: Option<String>,
    limit: Option<u32>,
    environment: Option<String>,
    deployment_id: Option<String>,
    level: Option<String>,
    cursor: Option<String>,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let query = LogsQuery {
        start_time,
        end_time,
        limit,
        environment,
        deployment_id,
        level,
        cursor,
    };
    let response: Value =
        client.get_with_query(&format!("/api/v1/vector/sites/{}/logs", id), &query)?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    // Parse the Axiom-style log format
    if let Some(tables) = response["data"]["logs"]["tables"].as_array() {
        for table in tables {
            if let Some(rows) = table["rows"].as_array() {
                for row in rows {
                    if let Some(row_arr) = row.as_array() {
                        // Typically: [timestamp, message, level]
                        let parts: Vec<String> = row_arr
                            .iter()
                            .filter_map(|v| v.as_str().map(String::from))
                            .collect();
                        if !parts.is_empty() {
                            println!("{}", parts.join(" | "));
                        }
                    }
                }
            }
        }

        // Show pagination info if there are more results
        if response["data"]["has_more"].as_bool().unwrap_or(false)
            && let Some(next_cursor) = response["data"]["cursor"].as_str() {
                eprintln!();
                eprintln!(
                    "More results available. Use --cursor {} to continue.",
                    next_cursor
                );
            }
    } else {
        print_message("No logs available.");
    }

    Ok(())
}

pub fn wp_reconfig(client: &ApiClient, id: &str, format: OutputFormat) -> Result<(), ApiError> {
    let response: Value = client.post_empty(&format!("/api/v1/vector/sites/{}/wp/reconfig", id))?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    print_message("wp-config.php regenerated successfully.");
    Ok(())
}

// SSH Key subcommands

pub fn ssh_key_list(
    client: &ApiClient,
    site_id: &str,
    page: u32,
    per_page: u32,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let query = PaginationQuery { page, per_page };
    let response: Value = client.get_with_query(
        &format!("/api/v1/vector/sites/{}/ssh-keys", site_id),
        &query,
    )?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    let keys = response["data"]
        .as_array()
        .ok_or_else(|| ApiError::Other("Invalid response format".to_string()))?;

    if keys.is_empty() {
        print_message("No SSH keys found.");
        return Ok(());
    }

    let rows: Vec<Vec<String>> = keys
        .iter()
        .map(|k| {
            vec![
                k["id"].as_str().unwrap_or("-").to_string(),
                k["name"].as_str().unwrap_or("-").to_string(),
                format_option(&k["fingerprint"].as_str().map(String::from)),
                format_option(&k["created_at"].as_str().map(String::from)),
            ]
        })
        .collect();

    print_table(vec!["ID", "Name", "Fingerprint", "Created"], rows);

    if let Some((current, last, total)) = extract_pagination(&response) {
        print_pagination(current, last, total);
    }

    Ok(())
}

pub fn ssh_key_add(
    client: &ApiClient,
    site_id: &str,
    name: &str,
    public_key: &str,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let body = CreateSshKeyRequest {
        name: name.to_string(),
        public_key: public_key.to_string(),
    };

    let response: Value =
        client.post(&format!("/api/v1/vector/sites/{}/ssh-keys", site_id), &body)?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    let key = &response["data"];
    print_message(&format!(
        "SSH key added: {} ({})",
        key["name"].as_str().unwrap_or("-"),
        key["id"].as_str().unwrap_or("-")
    ));

    Ok(())
}

pub fn ssh_key_remove(
    client: &ApiClient,
    site_id: &str,
    key_id: &str,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let response: Value = client.delete(&format!(
        "/api/v1/vector/sites/{}/ssh-keys/{}",
        site_id, key_id
    ))?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    print_message("SSH key removed successfully.");
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
