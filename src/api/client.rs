use reqwest::blocking::{Client, Response};
use reqwest::header::{ACCEPT, AUTHORIZATION, CONTENT_TYPE, HeaderMap, HeaderValue};
use serde::Serialize;
use serde::de::DeserializeOwned;

use super::error::ApiError;

const DEFAULT_BASE_URL: &str = "https://api.builtfast.com";
const USER_AGENT: &str = concat!("vector-cli/", env!("CARGO_PKG_VERSION"));

pub struct ApiClient {
    client: Client,
    base_url: String,
    token: Option<String>,
}

impl ApiClient {
    pub fn new(base_url: Option<String>, token: Option<String>) -> Result<Self, ApiError> {
        let client = Client::builder()
            .user_agent(USER_AGENT)
            .build()
            .map_err(ApiError::NetworkError)?;

        Ok(Self {
            client,
            base_url: base_url.unwrap_or_else(|| DEFAULT_BASE_URL.to_string()),
            token,
        })
    }

    pub fn set_token(&mut self, token: String) {
        self.token = Some(token);
    }

    fn headers(&self) -> Result<HeaderMap, ApiError> {
        let mut headers = HeaderMap::new();
        headers.insert(ACCEPT, HeaderValue::from_static("application/json"));
        headers.insert(CONTENT_TYPE, HeaderValue::from_static("application/json"));

        if let Some(ref token) = self.token {
            let auth_value = format!("Bearer {}", token);
            headers.insert(
                AUTHORIZATION,
                HeaderValue::from_str(&auth_value)
                    .map_err(|e| ApiError::ConfigError(e.to_string()))?,
            );
        }

        Ok(headers)
    }

    fn handle_response<T: DeserializeOwned>(&self, response: Response) -> Result<T, ApiError> {
        let status = response.status();
        let body = response.text().map_err(ApiError::NetworkError)?;

        if status.is_success() {
            serde_json::from_str(&body)
                .map_err(|e| ApiError::Other(format!("JSON parse error: {}", e)))
        } else {
            Err(ApiError::from_response(status.as_u16(), &body))
        }
    }

    pub fn get<T: DeserializeOwned>(&self, path: &str) -> Result<T, ApiError> {
        let url = format!("{}{}", self.base_url, path);
        let response = self
            .client
            .get(&url)
            .headers(self.headers()?)
            .send()
            .map_err(ApiError::NetworkError)?;

        self.handle_response(response)
    }

    pub fn get_with_query<T: DeserializeOwned, Q: Serialize>(
        &self,
        path: &str,
        query: &Q,
    ) -> Result<T, ApiError> {
        let url = format!("{}{}", self.base_url, path);
        let response = self
            .client
            .get(&url)
            .headers(self.headers()?)
            .query(query)
            .send()
            .map_err(ApiError::NetworkError)?;

        self.handle_response(response)
    }

    pub fn post<T: DeserializeOwned, B: Serialize>(
        &self,
        path: &str,
        body: &B,
    ) -> Result<T, ApiError> {
        let url = format!("{}{}", self.base_url, path);
        let response = self
            .client
            .post(&url)
            .headers(self.headers()?)
            .json(body)
            .send()
            .map_err(ApiError::NetworkError)?;

        self.handle_response(response)
    }

    pub fn post_empty<T: DeserializeOwned>(&self, path: &str) -> Result<T, ApiError> {
        let url = format!("{}{}", self.base_url, path);
        let response = self
            .client
            .post(&url)
            .headers(self.headers()?)
            .send()
            .map_err(ApiError::NetworkError)?;

        self.handle_response(response)
    }

    pub fn put<T: DeserializeOwned, B: Serialize>(
        &self,
        path: &str,
        body: &B,
    ) -> Result<T, ApiError> {
        let url = format!("{}{}", self.base_url, path);
        let response = self
            .client
            .put(&url)
            .headers(self.headers()?)
            .json(body)
            .send()
            .map_err(ApiError::NetworkError)?;

        self.handle_response(response)
    }

    pub fn put_empty<T: DeserializeOwned>(&self, path: &str) -> Result<T, ApiError> {
        let url = format!("{}{}", self.base_url, path);
        let response = self
            .client
            .put(&url)
            .headers(self.headers()?)
            .send()
            .map_err(ApiError::NetworkError)?;

        self.handle_response(response)
    }

    pub fn delete<T: DeserializeOwned>(&self, path: &str) -> Result<T, ApiError> {
        let url = format!("{}{}", self.base_url, path);
        let response = self
            .client
            .delete(&url)
            .headers(self.headers()?)
            .send()
            .map_err(ApiError::NetworkError)?;

        self.handle_response(response)
    }

    pub fn post_file<T: DeserializeOwned>(
        &self,
        path: &str,
        file_path: &std::path::Path,
    ) -> Result<T, ApiError> {
        use reqwest::blocking::multipart::{Form, Part};
        use std::fs::File;
        use std::io::Read;

        let url = format!("{}{}", self.base_url, path);

        let mut file = File::open(file_path)
            .map_err(|e| ApiError::Other(format!("Failed to open file: {}", e)))?;
        let mut buffer = Vec::new();
        file.read_to_end(&mut buffer)
            .map_err(|e| ApiError::Other(format!("Failed to read file: {}", e)))?;

        let file_name = file_path
            .file_name()
            .and_then(|n| n.to_str())
            .unwrap_or("file.sql")
            .to_string();

        let part = Part::bytes(buffer)
            .file_name(file_name)
            .mime_str("application/octet-stream")
            .map_err(|e| ApiError::Other(format!("Failed to set mime type: {}", e)))?;

        let form = Form::new().part("file", part);

        let mut headers = self.headers()?;
        headers.remove(CONTENT_TYPE);

        let response = self
            .client
            .post(&url)
            .headers(headers)
            .multipart(form)
            .send()
            .map_err(ApiError::NetworkError)?;

        self.handle_response(response)
    }
}
