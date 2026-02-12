use clap::{Parser, Subcommand};
use std::path::PathBuf;

#[derive(Parser)]
#[command(name = "vector")]
#[command(about = "CLI for Vector Pro API", long_about = None)]
#[command(version)]
pub struct Cli {
    /// Output JSON instead of tables
    #[arg(long, global = true)]
    pub json: bool,

    /// Output tables instead of JSON (default when TTY)
    #[arg(long, global = true)]
    pub no_json: bool,

    #[command(subcommand)]
    pub command: Commands,
}

#[derive(Subcommand)]
pub enum Commands {
    /// Manage authentication
    Auth {
        #[command(subcommand)]
        command: AuthCommands,
    },
    /// Manage sites
    Site {
        #[command(subcommand)]
        command: SiteCommands,
    },
    /// Manage environments
    Env {
        #[command(subcommand)]
        command: EnvCommands,
    },
    /// Manage deployments
    Deploy {
        #[command(subcommand)]
        command: DeployCommands,
    },
    /// Manage SSL certificates
    Ssl {
        #[command(subcommand)]
        command: SslCommands,
    },
    /// Manage database import/export
    Db {
        #[command(subcommand)]
        command: DbCommands,
    },
    /// Manage WAF rules and blocklists
    Waf {
        #[command(subcommand)]
        command: WafCommands,
    },
    /// Manage account settings
    Account {
        #[command(subcommand)]
        command: AccountCommands,
    },
    /// View events
    Event {
        #[command(subcommand)]
        command: EventCommands,
    },
    /// Manage webhooks
    Webhook {
        #[command(subcommand)]
        command: WebhookCommands,
    },
    /// List available PHP versions
    PhpVersions,
    /// Configure MCP integration for Claude
    Mcp {
        #[command(subcommand)]
        command: McpCommands,
    },
}

#[derive(Subcommand)]
pub enum AuthCommands {
    /// Log in with an API token
    Login {
        /// API token (reads from stdin if not provided)
        #[arg(long, env = "VECTOR_API_KEY")]
        token: Option<String>,
    },
    /// Log out and clear credentials
    Logout,
    /// Check authentication status
    Status,
}

#[derive(Subcommand)]
pub enum SiteCommands {
    /// List all sites
    List {
        /// Page number
        #[arg(long, default_value = "1")]
        page: u32,
        /// Items per page
        #[arg(long, default_value = "15")]
        per_page: u32,
    },
    /// Show site details
    Show {
        /// Site ID
        id: String,
    },
    /// Create a new site
    Create {
        /// Customer ID for the site
        #[arg(long)]
        customer_id: String,
        /// PHP version for the dev environment
        #[arg(long)]
        dev_php_version: String,
        /// Tags for the site
        #[arg(long)]
        tags: Option<Vec<String>>,
    },
    /// Update a site
    Update {
        /// Site ID
        id: String,
        /// Customer ID
        #[arg(long)]
        customer_id: Option<String>,
        /// Tags
        #[arg(long)]
        tags: Option<Vec<String>>,
    },
    /// Delete a site
    Delete {
        /// Site ID
        id: String,
        /// Skip confirmation
        #[arg(long)]
        force: bool,
    },
    /// Clone a site
    Clone {
        /// Site ID to clone
        id: String,
        /// Customer ID for the new site
        #[arg(long)]
        customer_id: Option<String>,
        /// PHP version for the new dev environment
        #[arg(long)]
        dev_php_version: Option<String>,
        /// Tags for the new site
        #[arg(long)]
        tags: Option<Vec<String>>,
    },
    /// Suspend a site
    Suspend {
        /// Site ID
        id: String,
    },
    /// Unsuspend a site
    Unsuspend {
        /// Site ID
        id: String,
    },
    /// Reset SFTP password
    ResetSftpPassword {
        /// Site ID
        id: String,
    },
    /// Reset database password
    ResetDbPassword {
        /// Site ID
        id: String,
    },
    /// Purge site cache
    PurgeCache {
        /// Site ID
        id: String,
        /// Cache tag to purge
        #[arg(long)]
        cache_tag: Option<String>,
        /// URL to purge
        #[arg(long)]
        url: Option<String>,
    },
    /// View site logs
    Logs {
        /// Site ID
        id: String,
        /// Start time (ISO 8601 format)
        #[arg(long)]
        start_time: Option<String>,
        /// End time (ISO 8601 format)
        #[arg(long)]
        end_time: Option<String>,
        /// Number of log entries
        #[arg(long)]
        limit: Option<u32>,
        /// Environment name to filter
        #[arg(long)]
        environment: Option<String>,
        /// Deployment ID to filter
        #[arg(long)]
        deployment_id: Option<String>,
        /// Log level to filter (e.g., error, warning, info)
        #[arg(long)]
        level: Option<String>,
        /// Pagination cursor from previous response
        #[arg(long)]
        cursor: Option<String>,
    },
    /// Regenerate wp-config.php
    WpReconfig {
        /// Site ID
        id: String,
    },
    /// Manage site SSH keys
    SshKey {
        #[command(subcommand)]
        command: SiteSshKeyCommands,
    },
}

#[derive(Subcommand)]
pub enum SiteSshKeyCommands {
    /// List SSH keys for a site
    List {
        /// Site ID
        site_id: String,
        /// Page number
        #[arg(long, default_value = "1")]
        page: u32,
        /// Items per page
        #[arg(long, default_value = "15")]
        per_page: u32,
    },
    /// Add an SSH key to a site
    Add {
        /// Site ID
        site_id: String,
        /// Key name
        #[arg(long)]
        name: String,
        /// Public key content
        #[arg(long)]
        public_key: String,
    },
    /// Remove an SSH key from a site
    Remove {
        /// Site ID
        site_id: String,
        /// SSH key ID
        key_id: String,
    },
}

#[derive(Subcommand)]
pub enum EnvCommands {
    /// List environments for a site
    List {
        /// Site ID
        site_id: String,
        /// Page number
        #[arg(long, default_value = "1")]
        page: u32,
        /// Items per page
        #[arg(long, default_value = "15")]
        per_page: u32,
    },
    /// Show environment details
    Show {
        /// Environment ID
        env_id: String,
    },
    /// Create a new environment
    Create {
        /// Site ID
        site_id: String,
        /// Environment name
        #[arg(long)]
        name: String,
        /// Custom domain
        #[arg(long)]
        custom_domain: String,
        /// PHP version
        #[arg(long)]
        php_version: String,
        /// Mark as production environment
        #[arg(long)]
        is_production: bool,
        /// Tags
        #[arg(long)]
        tags: Option<Vec<String>>,
    },
    /// Update an environment
    Update {
        /// Environment ID
        env_id: String,
        /// New environment name
        #[arg(long)]
        name: Option<String>,
        /// Custom domain
        #[arg(long)]
        custom_domain: Option<String>,
        /// Tags
        #[arg(long)]
        tags: Option<Vec<String>>,
    },
    /// Delete an environment
    Delete {
        /// Environment ID
        env_id: String,
    },
    /// Reset environment database password
    ResetDbPassword {
        /// Environment ID
        env_id: String,
    },
    /// Manage environment secrets
    Secret {
        #[command(subcommand)]
        command: EnvSecretCommands,
    },
    /// Manage environment database
    Db {
        #[command(subcommand)]
        command: EnvDbCommands,
    },
}

#[derive(Subcommand)]
pub enum EnvSecretCommands {
    /// List secrets for an environment
    List {
        /// Environment ID
        env_id: String,
        /// Page number
        #[arg(long, default_value = "1")]
        page: u32,
        /// Items per page
        #[arg(long, default_value = "15")]
        per_page: u32,
    },
    /// Show secret details
    Show {
        /// Secret ID
        secret_id: String,
    },
    /// Create a secret
    Create {
        /// Environment ID
        env_id: String,
        /// Secret key
        #[arg(long)]
        key: String,
        /// Secret value
        #[arg(long)]
        value: String,
        /// Store as a plain environment variable instead of a secret
        #[arg(long)]
        no_secret: bool,
    },
    /// Update a secret
    Update {
        /// Secret ID
        secret_id: String,
        /// Secret key
        #[arg(long)]
        key: Option<String>,
        /// Secret value
        #[arg(long)]
        value: Option<String>,
        /// Store as a plain environment variable instead of a secret
        #[arg(long)]
        no_secret: bool,
    },
    /// Delete a secret
    Delete {
        /// Secret ID
        secret_id: String,
    },
}

#[derive(Subcommand)]
pub enum EnvDbCommands {
    /// Import a SQL file directly (files under 50MB)
    Import {
        /// Environment ID
        env_id: String,
        /// Path to SQL file
        file: PathBuf,
        /// Drop all existing tables before import
        #[arg(long)]
        drop_tables: bool,
        /// Disable foreign key checks during import
        #[arg(long)]
        disable_foreign_keys: bool,
        /// Search string for search-and-replace during import
        #[arg(long)]
        search_replace_from: Option<String>,
        /// Replace string for search-and-replace during import
        #[arg(long)]
        search_replace_to: Option<String>,
    },
    /// Manage import sessions for large files
    ImportSession {
        #[command(subcommand)]
        command: EnvDbImportSessionCommands,
    },
    /// Promote dev database to this environment
    Promote {
        /// Environment ID
        env_id: String,
        /// Drop all existing tables before promote
        #[arg(long)]
        drop_tables: bool,
        /// Disable foreign key checks during promote
        #[arg(long)]
        disable_foreign_keys: bool,
    },
    /// Check promote status
    PromoteStatus {
        /// Environment ID
        env_id: String,
        /// Promote ID
        promote_id: String,
    },
}

#[derive(Subcommand)]
pub enum EnvDbImportSessionCommands {
    /// Create an import session
    Create {
        /// Environment ID
        env_id: String,
        /// Filename
        #[arg(long)]
        filename: Option<String>,
        /// Content length in bytes
        #[arg(long)]
        content_length: Option<u64>,
        /// Drop all existing tables before import
        #[arg(long)]
        drop_tables: bool,
        /// Disable foreign key checks during import
        #[arg(long)]
        disable_foreign_keys: bool,
        /// Search string for search-and-replace during import
        #[arg(long)]
        search_replace_from: Option<String>,
        /// Replace string for search-and-replace during import
        #[arg(long)]
        search_replace_to: Option<String>,
    },
    /// Run an import session
    Run {
        /// Environment ID
        env_id: String,
        /// Import ID
        import_id: String,
    },
    /// Check import session status
    Status {
        /// Environment ID
        env_id: String,
        /// Import ID
        import_id: String,
    },
}

#[derive(Subcommand)]
pub enum DeployCommands {
    /// List deployments for an environment
    List {
        /// Environment ID
        env_id: String,
        /// Page number
        #[arg(long, default_value = "1")]
        page: u32,
        /// Items per page
        #[arg(long, default_value = "15")]
        per_page: u32,
    },
    /// Show deployment details
    Show {
        /// Deployment ID
        deploy_id: String,
    },
    /// Trigger a new deployment
    Trigger {
        /// Environment ID
        env_id: String,
        /// Include wp-content/uploads in the deployment
        #[arg(long)]
        include_uploads: bool,
        /// Include database in the deployment
        #[arg(long)]
        include_database: bool,
    },
    /// Rollback to a previous deployment
    Rollback {
        /// Environment ID
        env_id: String,
        /// Target deployment ID to rollback to
        #[arg(long)]
        target_deployment_id: Option<String>,
    },
}

#[derive(Subcommand)]
pub enum SslCommands {
    /// Check SSL status
    Status {
        /// Environment ID
        env_id: String,
    },
    /// Nudge SSL provisioning
    Nudge {
        /// Environment ID
        env_id: String,
        /// Retry from failed state
        #[arg(long)]
        retry: bool,
    },
}

#[derive(Subcommand)]
pub enum DbCommands {
    /// Import a SQL file directly (files under 50MB)
    Import {
        /// Site ID
        site_id: String,
        /// Path to SQL file
        file: PathBuf,
        /// Drop all existing tables before import
        #[arg(long)]
        drop_tables: bool,
        /// Disable foreign key checks during import
        #[arg(long)]
        disable_foreign_keys: bool,
        /// Search string for search-and-replace during import
        #[arg(long)]
        search_replace_from: Option<String>,
        /// Replace string for search-and-replace during import
        #[arg(long)]
        search_replace_to: Option<String>,
    },
    /// Manage import sessions for large files
    ImportSession {
        #[command(subcommand)]
        command: DbImportSessionCommands,
    },
    /// Manage database exports
    Export {
        #[command(subcommand)]
        command: DbExportCommands,
    },
}

#[derive(Subcommand)]
pub enum DbImportSessionCommands {
    /// Create an import session
    Create {
        /// Site ID
        site_id: String,
        /// Filename
        #[arg(long)]
        filename: Option<String>,
        /// Content length in bytes
        #[arg(long)]
        content_length: Option<u64>,
        /// Drop all existing tables before import
        #[arg(long)]
        drop_tables: bool,
        /// Disable foreign key checks during import
        #[arg(long)]
        disable_foreign_keys: bool,
        /// Search string for search-and-replace during import
        #[arg(long)]
        search_replace_from: Option<String>,
        /// Replace string for search-and-replace during import
        #[arg(long)]
        search_replace_to: Option<String>,
    },
    /// Run an import session
    Run {
        /// Site ID
        site_id: String,
        /// Import ID
        import_id: String,
    },
    /// Check import session status
    Status {
        /// Site ID
        site_id: String,
        /// Import ID
        import_id: String,
    },
}

#[derive(Subcommand)]
pub enum DbExportCommands {
    /// Start a database export
    Create {
        /// Site ID
        site_id: String,
        /// Export format (currently only "sql" supported)
        #[arg(long)]
        format: Option<String>,
    },
    /// Check export status
    Status {
        /// Site ID
        site_id: String,
        /// Export ID
        export_id: String,
    },
}

#[derive(Subcommand)]
pub enum WafCommands {
    /// Manage rate limit rules
    RateLimit {
        #[command(subcommand)]
        command: WafRateLimitCommands,
    },
    /// Manage blocked IPs
    BlockedIp {
        #[command(subcommand)]
        command: WafBlockedIpCommands,
    },
    /// Manage blocked referrers
    BlockedReferrer {
        #[command(subcommand)]
        command: WafBlockedReferrerCommands,
    },
    /// Manage allowed referrers
    AllowedReferrer {
        #[command(subcommand)]
        command: WafAllowedReferrerCommands,
    },
}

#[derive(Subcommand)]
pub enum WafRateLimitCommands {
    /// List rate limit rules
    List {
        /// Site ID
        site_id: String,
    },
    /// Show rate limit rule details
    Show {
        /// Site ID
        site_id: String,
        /// Rule ID
        rule_id: String,
    },
    /// Create a rate limit rule
    Create {
        /// Site ID
        site_id: String,
        /// Rule name
        #[arg(long)]
        name: String,
        /// Number of requests allowed
        #[arg(long)]
        request_count: u32,
        /// Time window in seconds (1 or 10)
        #[arg(long)]
        timeframe: u32,
        /// Block duration in seconds (30, 60, 300, 900, 1800, 3600)
        #[arg(long)]
        block_time: u32,
        /// Rule description
        #[arg(long)]
        description: Option<String>,
        /// URL pattern to match
        #[arg(long)]
        value: Option<String>,
        /// Match operator
        #[arg(long)]
        operator: Option<String>,
        /// Request variables to inspect
        #[arg(long)]
        variables: Option<Vec<String>>,
        /// Transformations to apply
        #[arg(long)]
        transformations: Option<Vec<String>>,
    },
    /// Update a rate limit rule
    Update {
        /// Site ID
        site_id: String,
        /// Rule ID
        rule_id: String,
        /// Rule name
        #[arg(long)]
        name: Option<String>,
        /// Rule description
        #[arg(long)]
        description: Option<String>,
        /// Number of requests allowed
        #[arg(long)]
        request_count: Option<u32>,
        /// Time window in seconds
        #[arg(long)]
        timeframe: Option<u32>,
        /// Block duration in seconds
        #[arg(long)]
        block_time: Option<u32>,
        /// URL pattern to match
        #[arg(long)]
        value: Option<String>,
        /// Match operator
        #[arg(long)]
        operator: Option<String>,
        /// Request variables to inspect
        #[arg(long)]
        variables: Option<Vec<String>>,
        /// Transformations to apply
        #[arg(long)]
        transformations: Option<Vec<String>>,
    },
    /// Delete a rate limit rule
    Delete {
        /// Site ID
        site_id: String,
        /// Rule ID
        rule_id: String,
    },
}

#[derive(Subcommand)]
pub enum WafBlockedIpCommands {
    /// List blocked IPs
    List {
        /// Site ID
        site_id: String,
    },
    /// Add an IP to the blocklist
    Add {
        /// Site ID
        site_id: String,
        /// IP address
        ip: String,
    },
    /// Remove an IP from the blocklist
    Remove {
        /// Site ID
        site_id: String,
        /// IP address
        ip: String,
    },
}

#[derive(Subcommand)]
pub enum WafBlockedReferrerCommands {
    /// List blocked referrers
    List {
        /// Site ID
        site_id: String,
    },
    /// Add a hostname to the blocked referrers
    Add {
        /// Site ID
        site_id: String,
        /// Hostname
        hostname: String,
    },
    /// Remove a hostname from the blocked referrers
    Remove {
        /// Site ID
        site_id: String,
        /// Hostname
        hostname: String,
    },
}

#[derive(Subcommand)]
pub enum WafAllowedReferrerCommands {
    /// List allowed referrers
    List {
        /// Site ID
        site_id: String,
    },
    /// Add a hostname to the allowed referrers
    Add {
        /// Site ID
        site_id: String,
        /// Hostname
        hostname: String,
    },
    /// Remove a hostname from the allowed referrers
    Remove {
        /// Site ID
        site_id: String,
        /// Hostname
        hostname: String,
    },
}

#[derive(Subcommand)]
pub enum AccountCommands {
    /// Show account summary
    Show,
    /// Manage account SSH keys
    SshKey {
        #[command(subcommand)]
        command: AccountSshKeyCommands,
    },
    /// Manage API keys
    ApiKey {
        #[command(subcommand)]
        command: AccountApiKeyCommands,
    },
    /// Manage global secrets
    Secret {
        #[command(subcommand)]
        command: AccountSecretCommands,
    },
}

#[derive(Subcommand)]
pub enum AccountSshKeyCommands {
    /// List account SSH keys
    List {
        /// Page number
        #[arg(long, default_value = "1")]
        page: u32,
        /// Items per page
        #[arg(long, default_value = "15")]
        per_page: u32,
    },
    /// Show SSH key details
    Show {
        /// SSH key ID
        key_id: String,
    },
    /// Create an SSH key
    Create {
        /// Key name
        #[arg(long)]
        name: String,
        /// Public key content
        #[arg(long)]
        public_key: String,
    },
    /// Delete an SSH key
    Delete {
        /// SSH key ID
        key_id: String,
    },
}

#[derive(Subcommand)]
pub enum AccountApiKeyCommands {
    /// List API keys
    List {
        /// Page number
        #[arg(long, default_value = "1")]
        page: u32,
        /// Items per page
        #[arg(long, default_value = "15")]
        per_page: u32,
    },
    /// Create an API key
    Create {
        /// Key name
        #[arg(long)]
        name: String,
        /// Abilities
        #[arg(long)]
        abilities: Option<Vec<String>>,
        /// Expiration date (ISO 8601 format)
        #[arg(long)]
        expires_at: Option<String>,
    },
    /// Delete an API key
    Delete {
        /// Token ID
        token_id: String,
    },
}

#[derive(Subcommand)]
pub enum AccountSecretCommands {
    /// List global secrets
    List {
        /// Page number
        #[arg(long, default_value = "1")]
        page: u32,
        /// Items per page
        #[arg(long, default_value = "15")]
        per_page: u32,
    },
    /// Show secret details
    Show {
        /// Secret ID
        secret_id: String,
    },
    /// Create a secret
    Create {
        /// Secret key
        #[arg(long)]
        key: String,
        /// Secret value
        #[arg(long)]
        value: String,
        /// Store as a plain environment variable instead of a secret
        #[arg(long)]
        no_secret: bool,
    },
    /// Update a secret
    Update {
        /// Secret ID
        secret_id: String,
        /// Secret key
        #[arg(long)]
        key: Option<String>,
        /// Secret value
        #[arg(long)]
        value: Option<String>,
        /// Store as a plain environment variable instead of a secret
        #[arg(long)]
        no_secret: bool,
    },
    /// Delete a secret
    Delete {
        /// Secret ID
        secret_id: String,
    },
}

#[derive(Subcommand)]
pub enum EventCommands {
    /// List events
    List {
        /// Start date (ISO 8601 format)
        #[arg(long)]
        from: Option<String>,
        /// End date (ISO 8601 format)
        #[arg(long)]
        to: Option<String>,
        /// Event type filter
        #[arg(long)]
        event: Option<String>,
        /// Page number
        #[arg(long)]
        page: Option<u32>,
        /// Items per page
        #[arg(long)]
        per_page: Option<u32>,
    },
}

#[derive(Subcommand)]
pub enum WebhookCommands {
    /// List webhooks
    List {
        /// Page number
        #[arg(long, default_value = "1")]
        page: u32,
        /// Items per page
        #[arg(long, default_value = "15")]
        per_page: u32,
    },
    /// Show webhook details
    Show {
        /// Webhook ID
        webhook_id: String,
    },
    /// Create a webhook
    Create {
        /// Webhook name
        #[arg(long)]
        name: String,
        /// Webhook URL
        #[arg(long)]
        url: String,
        /// Events to subscribe to
        #[arg(long, required = true)]
        events: Vec<String>,
        /// Webhook secret for signature verification
        #[arg(long)]
        secret: Option<String>,
    },
    /// Update a webhook
    Update {
        /// Webhook ID
        webhook_id: String,
        /// Webhook name
        #[arg(long)]
        name: Option<String>,
        /// Webhook URL
        #[arg(long)]
        url: Option<String>,
        /// Events to subscribe to
        #[arg(long)]
        events: Option<Vec<String>>,
        /// Webhook secret
        #[arg(long)]
        secret: Option<String>,
        /// Enable/disable webhook
        #[arg(long)]
        enabled: Option<bool>,
    },
    /// Delete a webhook
    Delete {
        /// Webhook ID
        webhook_id: String,
    },
}

#[derive(Subcommand)]
pub enum McpCommands {
    /// Set up Claude Desktop with Vector MCP server
    Setup {
        /// Overwrite existing Vector MCP configuration
        #[arg(long)]
        force: bool,
    },
}
