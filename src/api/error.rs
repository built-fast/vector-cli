use serde::Deserialize;
use std::collections::HashMap;
use thiserror::Error;

pub const EXIT_SUCCESS: i32 = 0;
pub const EXIT_GENERAL_ERROR: i32 = 1;
pub const EXIT_AUTH_ERROR: i32 = 2;
pub const EXIT_VALIDATION_ERROR: i32 = 3;
pub const EXIT_NOT_FOUND: i32 = 4;
pub const EXIT_NETWORK_ERROR: i32 = 5;

#[derive(Debug, Error)]
pub enum ApiError {
    #[error("Authentication failed: {0}")]
    Unauthorized(String),

    #[error("Access denied: {0}")]
    Forbidden(String),

    #[error("Not found: {0}")]
    NotFound(String),

    #[error("Validation failed: {0}")]
    ValidationError(String),

    #[error("Server error: {0}")]
    ServerError(String),

    #[error("Network error: {0}")]
    NetworkError(#[from] reqwest::Error),

    #[error("Configuration error: {0}")]
    ConfigError(String),

    #[error("{0}")]
    Other(String),
}

impl ApiError {
    pub fn exit_code(&self) -> i32 {
        match self {
            ApiError::Unauthorized(_) | ApiError::Forbidden(_) => EXIT_AUTH_ERROR,
            ApiError::NotFound(_) => EXIT_NOT_FOUND,
            ApiError::ValidationError(_) => EXIT_VALIDATION_ERROR,
            ApiError::ServerError(_) | ApiError::NetworkError(_) => EXIT_NETWORK_ERROR,
            ApiError::ConfigError(_) | ApiError::Other(_) => EXIT_GENERAL_ERROR,
        }
    }

    pub fn from_response(status: u16, body: &str) -> Self {
        let message = parse_error_message(body);

        match status {
            401 => ApiError::Unauthorized(message),
            403 => ApiError::Forbidden(message),
            404 => ApiError::NotFound(message),
            422 => ApiError::ValidationError(message),
            500..=599 => ApiError::ServerError(message),
            _ => ApiError::Other(message),
        }
    }
}

#[derive(Debug, Deserialize)]
struct ErrorResponse {
    message: Option<String>,
    errors: Option<HashMap<String, Vec<String>>>,
}

fn parse_error_message(body: &str) -> String {
    if let Ok(response) = serde_json::from_str::<ErrorResponse>(body) {
        if let Some(errors) = response.errors {
            let error_messages: Vec<String> = errors
                .into_iter()
                .flat_map(|(field, messages)| {
                    messages
                        .into_iter()
                        .map(move |msg| format!("{}: {}", field, msg))
                })
                .collect();
            if !error_messages.is_empty() {
                return error_messages.join("; ");
            }
        }
        if let Some(message) = response.message {
            return message;
        }
    }
    body.to_string()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_exit_codes() {
        assert_eq!(ApiError::Unauthorized("".into()).exit_code(), EXIT_AUTH_ERROR);
        assert_eq!(ApiError::Forbidden("".into()).exit_code(), EXIT_AUTH_ERROR);
        assert_eq!(ApiError::NotFound("".into()).exit_code(), EXIT_NOT_FOUND);
        assert_eq!(ApiError::ValidationError("".into()).exit_code(), EXIT_VALIDATION_ERROR);
        assert_eq!(ApiError::ServerError("".into()).exit_code(), EXIT_NETWORK_ERROR);
        assert_eq!(ApiError::ConfigError("".into()).exit_code(), EXIT_GENERAL_ERROR);
        assert_eq!(ApiError::Other("".into()).exit_code(), EXIT_GENERAL_ERROR);
    }

    #[test]
    fn test_from_response_status_codes() {
        assert!(matches!(ApiError::from_response(401, "{}"), ApiError::Unauthorized(_)));
        assert!(matches!(ApiError::from_response(403, "{}"), ApiError::Forbidden(_)));
        assert!(matches!(ApiError::from_response(404, "{}"), ApiError::NotFound(_)));
        assert!(matches!(ApiError::from_response(422, "{}"), ApiError::ValidationError(_)));
        assert!(matches!(ApiError::from_response(500, "{}"), ApiError::ServerError(_)));
        assert!(matches!(ApiError::from_response(503, "{}"), ApiError::ServerError(_)));
        assert!(matches!(ApiError::from_response(400, "{}"), ApiError::Other(_)));
    }

    #[test]
    fn test_parse_error_message_with_message() {
        let body = r#"{"message": "Site not found", "http_status": 404}"#;
        assert_eq!(parse_error_message(body), "Site not found");
    }

    #[test]
    fn test_parse_error_message_with_validation_errors() {
        let body = r#"{"errors": {"domain": ["The domain field is required."]}}"#;
        assert_eq!(parse_error_message(body), "domain: The domain field is required.");
    }

    #[test]
    fn test_parse_error_message_plain_text() {
        let body = "Internal Server Error";
        assert_eq!(parse_error_message(body), "Internal Server Error");
    }

    #[test]
    fn test_error_display() {
        let err = ApiError::Unauthorized("Invalid token".into());
        assert_eq!(err.to_string(), "Authentication failed: Invalid token");

        let err = ApiError::NotFound("Site not found".into());
        assert_eq!(err.to_string(), "Not found: Site not found");
    }
}
