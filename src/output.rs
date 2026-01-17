use comfy_table::{ContentArrangement, Table};
use serde::Serialize;
use serde_json::Value;

#[derive(Debug, Clone, Copy, PartialEq)]
pub enum OutputFormat {
    Json,
    Table,
}

impl OutputFormat {
    pub fn detect(json_flag: bool, no_json_flag: bool) -> Self {
        if json_flag {
            return OutputFormat::Json;
        }
        if no_json_flag {
            return OutputFormat::Table;
        }
        if atty::is(atty::Stream::Stdout) {
            OutputFormat::Table
        } else {
            OutputFormat::Json
        }
    }
}

pub fn print_json<T: Serialize>(data: &T) {
    match serde_json::to_string_pretty(data) {
        Ok(json) => println!("{}", json),
        Err(e) => eprintln!("Error serializing JSON: {}", e),
    }
}

pub fn print_message(message: &str) {
    println!("{}", message);
}

pub fn print_error(message: &str) {
    eprintln!("Error: {}", message);
}

pub fn print_table(headers: Vec<&str>, rows: Vec<Vec<String>>) {
    let mut table = Table::new();
    table.set_content_arrangement(ContentArrangement::Dynamic);
    table.load_preset(comfy_table::presets::UTF8_FULL_CONDENSED);
    table.set_header(headers);

    for row in rows {
        table.add_row(row);
    }

    println!("{}", table);
}

pub fn print_key_value(pairs: Vec<(&str, String)>) {
    let max_key_len = pairs.iter().map(|(k, _)| k.len()).max().unwrap_or(0);

    for (key, value) in pairs {
        println!("{:width$}  {}", key, value, width = max_key_len);
    }
}

pub fn format_option<T: std::fmt::Display>(opt: &Option<T>) -> String {
    match opt {
        Some(v) => v.to_string(),
        None => "-".to_string(),
    }
}

pub fn format_bool(b: bool) -> String {
    if b {
        "Yes".to_string()
    } else {
        "No".to_string()
    }
}

pub fn extract_pagination(value: &Value) -> Option<(u64, u64, u64)> {
    let meta = value.get("meta")?;
    let current_page = meta.get("current_page")?.as_u64()?;
    let last_page = meta.get("last_page")?.as_u64()?;
    let total = meta.get("total")?.as_u64()?;
    Some((current_page, last_page, total))
}

pub fn print_pagination(current_page: u64, last_page: u64, total: u64) {
    if last_page > 1 {
        println!("\nPage {} of {} ({} total)", current_page, last_page, total);
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::json;

    #[test]
    fn test_output_format_json_flag() {
        assert_eq!(OutputFormat::detect(true, false), OutputFormat::Json);
        assert_eq!(OutputFormat::detect(true, true), OutputFormat::Json); // json takes precedence
    }

    #[test]
    fn test_output_format_no_json_flag() {
        assert_eq!(OutputFormat::detect(false, true), OutputFormat::Table);
    }

    #[test]
    fn test_format_option_some() {
        assert_eq!(format_option(&Some("value")), "value");
        assert_eq!(format_option(&Some(42)), "42");
    }

    #[test]
    fn test_format_option_none() {
        assert_eq!(format_option::<String>(&None), "-");
    }

    #[test]
    fn test_format_bool() {
        assert_eq!(format_bool(true), "Yes");
        assert_eq!(format_bool(false), "No");
    }

    #[test]
    fn test_extract_pagination_valid() {
        let value = json!({
            "data": [],
            "meta": {
                "current_page": 1,
                "last_page": 5,
                "total": 50
            }
        });
        assert_eq!(extract_pagination(&value), Some((1, 5, 50)));
    }

    #[test]
    fn test_extract_pagination_missing_meta() {
        let value = json!({"data": []});
        assert_eq!(extract_pagination(&value), None);
    }

    #[test]
    fn test_extract_pagination_partial_meta() {
        let value = json!({
            "meta": {
                "current_page": 1
            }
        });
        assert_eq!(extract_pagination(&value), None);
    }
}
