use serde::Serialize;
use serde_json::Value;

use crate::api::{ApiClient, ApiError};
use crate::output::{
    format_option, print_json, print_key_value, print_message, print_table, OutputFormat,
};

#[derive(Debug, Serialize)]
struct CreateRateLimitRequest {
    name: String,
    request_count: u32,
    timeframe: u32,
    block_time: u32,
    #[serde(skip_serializing_if = "Option::is_none")]
    description: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    value: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    operator: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    variables: Option<Vec<String>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    transformations: Option<Vec<String>>,
}

#[derive(Debug, Serialize)]
struct UpdateRateLimitRequest {
    #[serde(skip_serializing_if = "Option::is_none")]
    name: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    description: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    request_count: Option<u32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    timeframe: Option<u32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    block_time: Option<u32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    value: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    operator: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    variables: Option<Vec<String>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    transformations: Option<Vec<String>>,
}

#[derive(Debug, Serialize)]
struct AddReferrerRequest {
    hostname: String,
}

// Rate Limit commands

pub fn rate_limit_list(
    client: &ApiClient,
    site_id: &str,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let response: Value =
        client.get(&format!("/api/v1/vector/sites/{}/waf/rate-limits", site_id))?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    let rules = response["data"]
        .as_array()
        .ok_or_else(|| ApiError::Other("Invalid response format".to_string()))?;

    if rules.is_empty() {
        print_message("No rate limit rules found.");
        return Ok(());
    }

    let rows: Vec<Vec<String>> = rules
        .iter()
        .map(|r| {
            let config = &r["configuration"];
            vec![
                r["id"]
                    .as_u64()
                    .map(|v| v.to_string())
                    .unwrap_or("-".to_string()),
                r["name"].as_str().unwrap_or("-").to_string(),
                format!(
                    "{}/{}s",
                    config["request_count"].as_u64().unwrap_or(0),
                    config["timeframe"].as_u64().unwrap_or(0)
                ),
                format!("{}s", config["block_time"].as_u64().unwrap_or(0)),
            ]
        })
        .collect();

    print_table(vec!["ID", "Name", "Requests/Time", "Block Time"], rows);

    Ok(())
}

pub fn rate_limit_show(
    client: &ApiClient,
    site_id: &str,
    rule_id: &str,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let response: Value = client.get(&format!(
        "/api/v1/vector/sites/{}/waf/rate-limits/{}",
        site_id, rule_id
    ))?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    let rule = &response["data"];
    let config = &rule["configuration"];

    print_key_value(vec![
        (
            "ID",
            rule["id"]
                .as_u64()
                .map(|v| v.to_string())
                .unwrap_or("-".to_string()),
        ),
        ("Name", rule["name"].as_str().unwrap_or("-").to_string()),
        (
            "Description",
            format_option(&rule["description"].as_str().map(String::from)),
        ),
        (
            "Request Count",
            config["request_count"]
                .as_u64()
                .map(|v| v.to_string())
                .unwrap_or("-".to_string()),
        ),
        (
            "Timeframe (s)",
            config["timeframe"]
                .as_u64()
                .map(|v| v.to_string())
                .unwrap_or("-".to_string()),
        ),
        (
            "Block Time (s)",
            config["block_time"]
                .as_u64()
                .map(|v| v.to_string())
                .unwrap_or("-".to_string()),
        ),
        (
            "Value",
            format_option(&config["value"].as_str().map(String::from)),
        ),
        (
            "Operator",
            format_option(&config["operator"].as_str().map(String::from)),
        ),
        ("Variables", format_array(&config["variables"])),
        ("Transformations", format_array(&config["transformations"])),
    ]);

    Ok(())
}

#[allow(clippy::too_many_arguments)]
pub fn rate_limit_create(
    client: &ApiClient,
    site_id: &str,
    name: &str,
    request_count: u32,
    timeframe: u32,
    block_time: u32,
    description: Option<String>,
    value: Option<String>,
    operator: Option<String>,
    variables: Option<Vec<String>>,
    transformations: Option<Vec<String>>,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let body = CreateRateLimitRequest {
        name: name.to_string(),
        request_count,
        timeframe,
        block_time,
        description,
        value,
        operator,
        variables,
        transformations,
    };

    let response: Value = client.post(
        &format!("/api/v1/vector/sites/{}/waf/rate-limits", site_id),
        &body,
    )?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    let rule = &response["data"];
    print_message(&format!(
        "Rate limit created: {} (ID: {})",
        rule["name"].as_str().unwrap_or("-"),
        rule["id"]
            .as_u64()
            .map(|v| v.to_string())
            .unwrap_or("-".to_string())
    ));

    Ok(())
}

#[allow(clippy::too_many_arguments)]
pub fn rate_limit_update(
    client: &ApiClient,
    site_id: &str,
    rule_id: &str,
    name: Option<String>,
    description: Option<String>,
    request_count: Option<u32>,
    timeframe: Option<u32>,
    block_time: Option<u32>,
    value: Option<String>,
    operator: Option<String>,
    variables: Option<Vec<String>>,
    transformations: Option<Vec<String>>,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let body = UpdateRateLimitRequest {
        name,
        description,
        request_count,
        timeframe,
        block_time,
        value,
        operator,
        variables,
        transformations,
    };

    let response: Value = client.put(
        &format!(
            "/api/v1/vector/sites/{}/waf/rate-limits/{}",
            site_id, rule_id
        ),
        &body,
    )?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    print_message("Rate limit updated successfully.");
    Ok(())
}

pub fn rate_limit_delete(
    client: &ApiClient,
    site_id: &str,
    rule_id: &str,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let response: Value = client.delete(&format!(
        "/api/v1/vector/sites/{}/waf/rate-limits/{}",
        site_id, rule_id
    ))?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    print_message("Rate limit deleted successfully.");
    Ok(())
}

// Blocked IP commands

pub fn blocked_ip_list(
    client: &ApiClient,
    site_id: &str,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let response: Value =
        client.get(&format!("/api/v1/vector/sites/{}/waf/blocked-ips", site_id))?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    let ips = response["data"]
        .as_array()
        .ok_or_else(|| ApiError::Other("Invalid response format".to_string()))?;

    if ips.is_empty() {
        print_message("No blocked IPs found.");
        return Ok(());
    }

    let rows: Vec<Vec<String>> = ips
        .iter()
        .map(|ip| vec![ip["ip"].as_str().unwrap_or("-").to_string()])
        .collect();

    print_table(vec!["IP"], rows);

    Ok(())
}

pub fn blocked_ip_add(
    client: &ApiClient,
    site_id: &str,
    ip: &str,
    format: OutputFormat,
) -> Result<(), ApiError> {
    #[derive(Serialize)]
    struct AddIpRequest {
        ip: String,
    }

    let body = AddIpRequest { ip: ip.to_string() };

    let response: Value = client.post(
        &format!("/api/v1/vector/sites/{}/waf/blocked-ips", site_id),
        &body,
    )?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    print_message(&format!("IP {} added to blocklist.", ip));
    Ok(())
}

pub fn blocked_ip_remove(
    client: &ApiClient,
    site_id: &str,
    ip: &str,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let response: Value = client.delete(&format!(
        "/api/v1/vector/sites/{}/waf/blocked-ips/{}",
        site_id, ip
    ))?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    print_message(&format!("IP {} removed from blocklist.", ip));
    Ok(())
}

// Blocked Referrer commands

pub fn blocked_referrer_list(
    client: &ApiClient,
    site_id: &str,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let response: Value = client.get(&format!(
        "/api/v1/vector/sites/{}/waf/blocked-referrers",
        site_id
    ))?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    let referrers = response["data"]
        .as_array()
        .ok_or_else(|| ApiError::Other("Invalid response format".to_string()))?;

    if referrers.is_empty() {
        print_message("No blocked referrers found.");
        return Ok(());
    }

    let rows: Vec<Vec<String>> = referrers
        .iter()
        .map(|r| vec![r["hostname"].as_str().unwrap_or("-").to_string()])
        .collect();

    print_table(vec!["Hostname"], rows);

    Ok(())
}

pub fn blocked_referrer_add(
    client: &ApiClient,
    site_id: &str,
    hostname: &str,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let body = AddReferrerRequest {
        hostname: hostname.to_string(),
    };

    let response: Value = client.post(
        &format!("/api/v1/vector/sites/{}/waf/blocked-referrers", site_id),
        &body,
    )?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    print_message(&format!("Referrer {} added to blocklist.", hostname));
    Ok(())
}

pub fn blocked_referrer_remove(
    client: &ApiClient,
    site_id: &str,
    hostname: &str,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let response: Value = client.delete(&format!(
        "/api/v1/vector/sites/{}/waf/blocked-referrers/{}",
        site_id, hostname
    ))?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    print_message(&format!("Referrer {} removed from blocklist.", hostname));
    Ok(())
}

// Allowed Referrer commands

pub fn allowed_referrer_list(
    client: &ApiClient,
    site_id: &str,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let response: Value = client.get(&format!(
        "/api/v1/vector/sites/{}/waf/allowed-referrers",
        site_id
    ))?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    let referrers = response["data"]
        .as_array()
        .ok_or_else(|| ApiError::Other("Invalid response format".to_string()))?;

    if referrers.is_empty() {
        print_message("No allowed referrers found.");
        return Ok(());
    }

    let rows: Vec<Vec<String>> = referrers
        .iter()
        .map(|r| vec![r["hostname"].as_str().unwrap_or("-").to_string()])
        .collect();

    print_table(vec!["Hostname"], rows);

    Ok(())
}

pub fn allowed_referrer_add(
    client: &ApiClient,
    site_id: &str,
    hostname: &str,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let body = AddReferrerRequest {
        hostname: hostname.to_string(),
    };

    let response: Value = client.post(
        &format!("/api/v1/vector/sites/{}/waf/allowed-referrers", site_id),
        &body,
    )?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    print_message(&format!("Referrer {} added to allowlist.", hostname));
    Ok(())
}

pub fn allowed_referrer_remove(
    client: &ApiClient,
    site_id: &str,
    hostname: &str,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let response: Value = client.delete(&format!(
        "/api/v1/vector/sites/{}/waf/allowed-referrers/{}",
        site_id, hostname
    ))?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    print_message(&format!("Referrer {} removed from allowlist.", hostname));
    Ok(())
}

// Helper function to format arrays
fn format_array(value: &Value) -> String {
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
