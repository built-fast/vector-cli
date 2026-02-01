use serde::Serialize;
use serde_json::Value;

use crate::api::{ApiClient, ApiError};
use crate::output::{
    extract_pagination, format_option, print_json, print_pagination, print_table, OutputFormat,
};

#[derive(Debug, Serialize)]
struct EventsQuery {
    #[serde(skip_serializing_if = "Option::is_none")]
    from: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    to: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    event: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    page: Option<u32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    per_page: Option<u32>,
}

pub fn list(
    client: &ApiClient,
    from: Option<String>,
    to: Option<String>,
    event: Option<String>,
    page: Option<u32>,
    per_page: Option<u32>,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let query = EventsQuery {
        from,
        to,
        event,
        page,
        per_page,
    };

    let response: Value = client.get_with_query("/api/v1/vector/events", &query)?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    let events = response["data"]
        .as_array()
        .ok_or_else(|| ApiError::Other("Invalid response format".to_string()))?;

    if events.is_empty() {
        println!("No events found.");
        return Ok(());
    }

    let rows: Vec<Vec<String>> = events
        .iter()
        .map(|e| {
            vec![
                e["id"].as_str().unwrap_or("-").to_string(),
                e["event"].as_str().unwrap_or("-").to_string(),
                format_actor(&e["actor"]),
                format_resource(&e["resource"]),
                format_option(&e["created_at"].as_str().map(String::from)),
            ]
        })
        .collect();

    print_table(vec!["ID", "Event", "Actor", "Resource", "Created"], rows);

    if let Some((current, last, total)) = extract_pagination(&response) {
        print_pagination(current, last, total);
    }

    Ok(())
}

fn format_actor(value: &Value) -> String {
    if value.is_null() {
        return "-".to_string();
    }
    if let Some(token_name) = value["token_name"].as_str() {
        return token_name.to_string();
    }
    if let Some(ip) = value["ip"].as_str() {
        return ip.to_string();
    }
    "-".to_string()
}

fn format_resource(value: &Value) -> String {
    if let Some(resource_type) = value["type"].as_str() {
        if let Some(resource_id) = value["id"].as_str() {
            return format!("{}:{}", resource_type, resource_id);
        }
        return resource_type.to_string();
    }
    "-".to_string()
}
