use serde_json::Value;

use crate::api::{ApiClient, ApiError};
use crate::output::{format_option, print_json, print_key_value, print_message, OutputFormat};

pub fn status(
    client: &ApiClient,
    site_id: &str,
    env_id: &str,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let response: Value = client.get(&format!(
        "/api/v1/vector/sites/{}/environments/{}/ssl",
        site_id, env_id
    ))?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    let ssl = &response["data"];

    print_key_value(vec![
        ("Status", ssl["status"].as_str().unwrap_or("-").to_string()),
        (
            "Provisioning Step",
            format_option(&ssl["provisioning_step"].as_str().map(String::from)),
        ),
        (
            "Certificate",
            format_option(&ssl["certificate_status"].as_str().map(String::from)),
        ),
        (
            "Expires",
            format_option(&ssl["expires_at"].as_str().map(String::from)),
        ),
    ]);

    Ok(())
}

pub fn nudge(
    client: &ApiClient,
    site_id: &str,
    env_id: &str,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let response: Value = client.post_empty(&format!(
        "/api/v1/vector/sites/{}/environments/{}/ssl/nudge",
        site_id, env_id
    ))?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    print_message("SSL provisioning nudge sent.");
    Ok(())
}
