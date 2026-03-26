use std::io::Read;

use reqwest::blocking::{Body, Client, Response};
use reqwest::header::{
    ACCEPT, AUTHORIZATION, CONTENT_LENGTH, CONTENT_TYPE, HeaderMap, HeaderValue,
};
use serde::de::DeserializeOwned;
use serde::{Deserialize, Serialize};

use super::error::ApiError;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CompletedPart {
    pub part_number: u64,
    pub etag: String,
}

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

    pub fn put_file(
        &self,
        url: &str,
        file: std::fs::File,
        content_length: u64,
    ) -> Result<(), ApiError> {
        let response = self
            .client
            .put(url)
            .header(CONTENT_TYPE, "application/gzip")
            .header(CONTENT_LENGTH, content_length)
            .body(Body::from(file))
            .send()
            .map_err(ApiError::NetworkError)?;

        if response.status().is_success() {
            Ok(())
        } else {
            let status = response.status();
            let body = response.text().map_err(ApiError::NetworkError)?;
            Err(ApiError::Other(format!(
                "Upload failed ({}): {}",
                status, body
            )))
        }
    }

    pub fn put_file_part<R: Read + Send + 'static>(
        &self,
        url: &str,
        reader: R,
        content_length: u64,
    ) -> Result<String, ApiError> {
        let response = self
            .client
            .put(url)
            .header(CONTENT_TYPE, "application/gzip")
            .body(Body::sized(reader, content_length))
            .send()
            .map_err(ApiError::NetworkError)?;

        if response.status().is_success() {
            let etag = response
                .headers()
                .get("etag")
                .and_then(|v| v.to_str().ok())
                .map(|s| s.to_string())
                .ok_or_else(|| ApiError::Other("S3 response missing ETag header".to_string()))?;
            Ok(etag)
        } else {
            let status = response.status();
            let body = response.text().map_err(ApiError::NetworkError)?;
            Err(ApiError::Other(format!(
                "Part upload failed ({}): {}",
                status, body
            )))
        }
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
}
