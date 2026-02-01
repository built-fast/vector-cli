use serde::Serialize;
use serde_json::Value;

use crate::api::{ApiClient, ApiError};
use crate::output::{
    format_bool, format_option, print_json, print_key_value, print_message, OutputFormat,
};

#[derive(Debug, Serialize)]
struct NudgeRequest {
    #[serde(skip_serializing_if = "std::ops::Not::not")]
    retry: bool,
}

pub fn status(
    client: &ApiClient,
    site_id: &str,
    env_name: &str,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let response: Value = client.get(&format!(
        "/api/v1/vector/sites/{}/environments/{}/ssl",
        site_id, env_name
    ))?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    let env = &response["data"];

    print_key_value(vec![
        ("Status", env["status"].as_str().unwrap_or("-").to_string()),
        (
            "Provisioning Step",
            format_option(&env["provisioning_step"].as_str().map(String::from)),
        ),
        (
            "Failure Reason",
            format_option(&env["failure_reason"].as_str().map(String::from)),
        ),
        (
            "Production",
            format_bool(env["is_production"].as_bool().unwrap_or(false)),
        ),
        (
            "Custom Domain",
            format_option(&env["custom_domain"].as_str().map(String::from)),
        ),
        (
            "FQDN",
            format_option(&env["fqdn"].as_str().map(String::from)),
        ),
    ]);

    Ok(())
}

pub fn nudge(
    client: &ApiClient,
    site_id: &str,
    env_name: &str,
    retry: bool,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let body = NudgeRequest { retry };

    let response: Value = client.post(
        &format!(
            "/api/v1/vector/sites/{}/environments/{}/ssl/nudge",
            site_id, env_name
        ),
        &body,
    )?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    if let Some(message) = response["message"].as_str() {
        print_message(message);
    } else {
        print_message("SSL provisioning nudge sent.");
    }

    Ok(())
}
