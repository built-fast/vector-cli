use clap::{Parser, Subcommand};

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
        /// Site domain
        #[arg(long)]
        domain: String,
        /// PHP version
        #[arg(long)]
        php_version: Option<String>,
    },
    /// Update a site
    Update {
        /// Site ID
        id: String,
        /// PHP version
        #[arg(long)]
        php_version: Option<String>,
    },
    /// Delete a site
    Delete {
        /// Site ID
        id: String,
        /// Skip confirmation
        #[arg(long)]
        force: bool,
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
        /// Specific path to purge
        #[arg(long)]
        path: Option<String>,
    },
    /// View site logs
    Logs {
        /// Site ID
        id: String,
        /// Log type (access, error, php)
        #[arg(long, default_value = "error")]
        r#type: String,
        /// Number of lines
        #[arg(long, default_value = "100")]
        lines: u32,
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
        /// Site ID
        site_id: String,
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
        /// Mark as production environment
        #[arg(long)]
        is_production: bool,
        /// Custom domain
        #[arg(long)]
        custom_domain: Option<String>,
        /// PHP version
        #[arg(long)]
        php_version: Option<String>,
    },
    /// Update an environment
    Update {
        /// Site ID
        site_id: String,
        /// Environment ID
        env_id: String,
        /// Custom domain
        #[arg(long)]
        custom_domain: Option<String>,
        /// PHP version
        #[arg(long)]
        php_version: Option<String>,
    },
    /// Delete an environment
    Delete {
        /// Site ID
        site_id: String,
        /// Environment ID
        env_id: String,
    },
    /// Suspend an environment
    Suspend {
        /// Site ID
        site_id: String,
        /// Environment ID
        env_id: String,
    },
    /// Unsuspend an environment
    Unsuspend {
        /// Site ID
        site_id: String,
        /// Environment ID
        env_id: String,
    },
}

#[derive(Subcommand)]
pub enum DeployCommands {
    /// List deployments for an environment
    List {
        /// Site ID
        site_id: String,
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
        /// Site ID
        site_id: String,
        /// Environment ID
        env_id: String,
        /// Deployment ID
        deploy_id: String,
    },
    /// Create a new deployment
    Create {
        /// Site ID
        site_id: String,
        /// Environment ID
        env_id: String,
    },
    /// Rollback to a previous deployment
    Rollback {
        /// Site ID
        site_id: String,
        /// Environment ID
        env_id: String,
        /// Deployment ID to rollback to
        deploy_id: String,
    },
}

#[derive(Subcommand)]
pub enum SslCommands {
    /// Check SSL status
    Status {
        /// Site ID
        site_id: String,
        /// Environment ID
        env_id: String,
    },
    /// Trigger SSL provisioning retry
    Nudge {
        /// Site ID
        site_id: String,
        /// Environment ID
        env_id: String,
    },
}
