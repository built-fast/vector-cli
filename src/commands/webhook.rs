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
struct CreateWebhookRequest {
    name: String,
    url: String,
    events: Vec<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    secret: Option<String>,
}

#[derive(Debug, Serialize)]
struct UpdateWebhookRequest {
    #[serde(skip_serializing_if = "Option::is_none")]
    name: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    url: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    events: Option<Vec<String>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    secret: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    enabled: Option<bool>,
}

pub fn list(
    client: &ApiClient,
    page: u32,
    per_page: u32,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let query = PaginationQuery { page, per_page };
    let response: Value = client.get_with_query("/api/v1/vector/webhooks", &query)?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    let webhooks = response["data"]
        .as_array()
        .ok_or_else(|| ApiError::Other("Invalid response format".to_string()))?;

    if webhooks.is_empty() {
        print_message("No webhooks found.");
        return Ok(());
    }

    let rows: Vec<Vec<String>> = webhooks
        .iter()
        .map(|w| {
            vec![
                w["id"].as_str().unwrap_or("-").to_string(),
                w["name"].as_str().unwrap_or("-").to_string(),
                w["url"].as_str().unwrap_or("-").to_string(),
                format_enabled(w["enabled"].as_bool()),
            ]
        })
        .collect();

    print_table(vec!["ID", "Name", "URL", "Enabled"], rows);

    if let Some((current, last, total)) = extract_pagination(&response) {
        print_pagination(current, last, total);
    }

    Ok(())
}

pub fn show(client: &ApiClient, webhook_id: &str, format: OutputFormat) -> Result<(), ApiError> {
    let response: Value = client.get(&format!("/api/v1/vector/webhooks/{}", webhook_id))?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    let webhook = &response["data"];

    print_key_value(vec![
        ("ID", webhook["id"].as_str().unwrap_or("-").to_string()),
        ("Name", webhook["name"].as_str().unwrap_or("-").to_string()),
        ("URL", webhook["url"].as_str().unwrap_or("-").to_string()),
        ("Enabled", format_enabled(webhook["enabled"].as_bool())),
        ("Events", format_events(&webhook["events"])),
        (
            "Has Secret",
            webhook["has_secret"]
                .as_bool()
                .map(|v| v.to_string())
                .unwrap_or("-".to_string()),
        ),
        (
            "Created",
            format_option(&webhook["created_at"].as_str().map(String::from)),
        ),
        (
            "Updated",
            format_option(&webhook["updated_at"].as_str().map(String::from)),
        ),
    ]);

    Ok(())
}

pub fn create(
    client: &ApiClient,
    name: &str,
    url: &str,
    events: Vec<String>,
    secret: Option<String>,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let body = CreateWebhookRequest {
        name: name.to_string(),
        url: url.to_string(),
        events,
        secret,
    };

    let response: Value = client.post("/api/v1/vector/webhooks", &body)?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    let webhook = &response["data"];
    print_message(&format!(
        "Webhook created: {} ({})",
        webhook["name"].as_str().unwrap_or("-"),
        webhook["id"].as_str().unwrap_or("-")
    ));

    Ok(())
}

#[allow(clippy::too_many_arguments)]
pub fn update(
    client: &ApiClient,
    webhook_id: &str,
    name: Option<String>,
    url: Option<String>,
    events: Option<Vec<String>>,
    secret: Option<String>,
    enabled: Option<bool>,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let body = UpdateWebhookRequest {
        name,
        url,
        events,
        secret,
        enabled,
    };

    let response: Value = client.put(&format!("/api/v1/vector/webhooks/{}", webhook_id), &body)?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    print_message("Webhook updated successfully.");
    Ok(())
}

pub fn delete(client: &ApiClient, webhook_id: &str, format: OutputFormat) -> Result<(), ApiError> {
    let response: Value = client.delete(&format!("/api/v1/vector/webhooks/{}", webhook_id))?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    print_message("Webhook deleted successfully.");
    Ok(())
}

fn format_enabled(value: Option<bool>) -> String {
    match value {
        Some(true) => "Yes".to_string(),
        Some(false) => "No".to_string(),
        None => "-".to_string(),
    }
}

fn format_events(value: &Value) -> String {
    if let Some(arr) = value.as_array() {
        if arr.is_empty() {
            return "-".to_string();
        }
        arr.iter()
            .filter_map(|v| v.as_str())
            .collect::<Vec<_>>()
            .join(", ")
    } else {
        "-".to_string()
    }
}
