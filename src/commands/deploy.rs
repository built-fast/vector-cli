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
struct RollbackRequest {
    #[serde(skip_serializing_if = "Option::is_none")]
    target_deployment_id: Option<String>,
}

pub fn list(
    client: &ApiClient,
    env_id: &str,
    page: u32,
    per_page: u32,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let query = PaginationQuery { page, per_page };
    let response: Value = client.get_with_query(
        &format!("/api/v1/vector/environments/{}/deployments", env_id),
        &query,
    )?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    let deploys = response["data"]
        .as_array()
        .ok_or_else(|| ApiError::Other("Invalid response format".to_string()))?;

    if deploys.is_empty() {
        print_message("No deployments found.");
        return Ok(());
    }

    let rows: Vec<Vec<String>> = deploys
        .iter()
        .map(|d| {
            vec![
                d["id"].as_str().unwrap_or("-").to_string(),
                d["status"].as_str().unwrap_or("-").to_string(),
                format_option(&d["actor"].as_str().map(String::from)),
                format_option(&d["created_at"].as_str().map(String::from)),
            ]
        })
        .collect();

    print_table(vec!["ID", "Status", "Actor", "Created"], rows);

    if let Some((current, last, total)) = extract_pagination(&response) {
        print_pagination(current, last, total);
    }

    Ok(())
}

pub fn show(
    client: &ApiClient,
    deploy_id: &str,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let response: Value = client.get(&format!("/api/v1/vector/deployments/{}", deploy_id))?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    let deploy = &response["data"];

    print_key_value(vec![
        ("ID", deploy["id"].as_str().unwrap_or("-").to_string()),
        (
            "Status",
            deploy["status"].as_str().unwrap_or("-").to_string(),
        ),
        (
            "Actor",
            format_option(&deploy["actor"].as_str().map(String::from)),
        ),
        (
            "Created",
            format_option(&deploy["created_at"].as_str().map(String::from)),
        ),
        (
            "Updated",
            format_option(&deploy["updated_at"].as_str().map(String::from)),
        ),
    ]);

    if let Some(stdout) = deploy["stdout"].as_str() {
        if !stdout.is_empty() {
            println!("\n--- stdout ---\n{}", stdout);
        }
    }

    if let Some(stderr) = deploy["stderr"].as_str() {
        if !stderr.is_empty() {
            println!("\n--- stderr ---\n{}", stderr);
        }
    }

    Ok(())
}

pub fn trigger(
    client: &ApiClient,
    env_id: &str,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let response: Value = client.post_empty(&format!(
        "/api/v1/vector/environments/{}/deployments",
        env_id
    ))?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    let deploy = &response["data"];
    print_message(&format!(
        "Deployment initiated: {} ({})",
        deploy["id"].as_str().unwrap_or("-"),
        deploy["status"].as_str().unwrap_or("-")
    ));

    Ok(())
}

pub fn rollback(
    client: &ApiClient,
    env_id: &str,
    target_deployment_id: Option<String>,
    format: OutputFormat,
) -> Result<(), ApiError> {
    let body = RollbackRequest {
        target_deployment_id,
    };

    let response: Value = client.post(
        &format!("/api/v1/vector/environments/{}/rollback", env_id),
        &body,
    )?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    let deploy = &response["data"];
    print_message(&format!(
        "Rollback initiated: {} ({})",
        deploy["id"].as_str().unwrap_or("-"),
        deploy["status"].as_str().unwrap_or("-")
    ));

    Ok(())
}
