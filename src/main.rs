mod api;
mod cli;
mod commands;
mod config;
mod output;

use clap::Parser;
use std::process;

use api::{ApiClient, ApiError, EXIT_SUCCESS};
use cli::{
    AuthCommands, Cli, Commands, DeployCommands, EnvCommands, McpCommands, SiteCommands,
    SslCommands,
};
use commands::{auth, deploy, env, mcp, site, ssl};
use config::{Config, Credentials};
use output::{print_error, OutputFormat};

fn main() {
    let cli = Cli::parse();
    let format = OutputFormat::detect(cli.json, cli.no_json);

    let result = run(cli.command, format);

    match result {
        Ok(()) => process::exit(EXIT_SUCCESS),
        Err(e) => {
            print_error(&e.to_string());
            process::exit(e.exit_code());
        }
    }
}

fn run(command: Commands, format: OutputFormat) -> Result<(), ApiError> {
    match command {
        Commands::Auth { command } => run_auth(command, format),
        Commands::Site { command } => run_site(command, format),
        Commands::Env { command } => run_env(command, format),
        Commands::Deploy { command } => run_deploy(command, format),
        Commands::Ssl { command } => run_ssl(command, format),
        Commands::Mcp { command } => run_mcp(command, format),
    }
}

fn run_auth(command: AuthCommands, format: OutputFormat) -> Result<(), ApiError> {
    match command {
        AuthCommands::Login { token } => auth::login(token, format),
        AuthCommands::Logout => auth::logout(format),
        AuthCommands::Status => auth::status(format),
    }
}

fn get_client() -> Result<ApiClient, ApiError> {
    let config = Config::load()?;
    let creds = Credentials::load()?;

    let token = auth::get_api_key(&creds).ok_or_else(|| {
        ApiError::Unauthorized(
            "Not logged in. Run 'vector auth login' to authenticate.".to_string(),
        )
    })?;

    ApiClient::new(config.api_url, Some(token))
}

fn run_site(command: SiteCommands, format: OutputFormat) -> Result<(), ApiError> {
    let client = get_client()?;

    match command {
        SiteCommands::List { page, per_page } => site::list(&client, page, per_page, format),
        SiteCommands::Show { id } => site::show(&client, &id, format),
        SiteCommands::Create {
            domain,
            php_version,
        } => site::create(&client, &domain, php_version, format),
        SiteCommands::Update { id, php_version } => site::update(&client, &id, php_version, format),
        SiteCommands::Delete { id, force } => site::delete(&client, &id, force, format),
        SiteCommands::Suspend { id } => site::suspend(&client, &id, format),
        SiteCommands::Unsuspend { id } => site::unsuspend(&client, &id, format),
        SiteCommands::ResetSftpPassword { id } => site::reset_sftp_password(&client, &id, format),
        SiteCommands::ResetDbPassword { id } => site::reset_db_password(&client, &id, format),
        SiteCommands::PurgeCache { id, path } => site::purge_cache(&client, &id, path, format),
        SiteCommands::Logs { id, r#type, lines } => {
            site::logs(&client, &id, &r#type, lines, format)
        }
    }
}

fn run_env(command: EnvCommands, format: OutputFormat) -> Result<(), ApiError> {
    let client = get_client()?;

    match command {
        EnvCommands::List {
            site_id,
            page,
            per_page,
        } => env::list(&client, &site_id, page, per_page, format),
        EnvCommands::Show { site_id, env_id } => env::show(&client, &site_id, &env_id, format),
        EnvCommands::Create {
            site_id,
            name,
            is_production,
            custom_domain,
            php_version,
        } => env::create(
            &client,
            &site_id,
            &name,
            is_production,
            custom_domain,
            php_version,
            format,
        ),
        EnvCommands::Update {
            site_id,
            env_id,
            custom_domain,
            php_version,
        } => env::update(
            &client,
            &site_id,
            &env_id,
            custom_domain,
            php_version,
            format,
        ),
        EnvCommands::Delete { site_id, env_id } => env::delete(&client, &site_id, &env_id, format),
        EnvCommands::Suspend { site_id, env_id } => {
            env::suspend(&client, &site_id, &env_id, format)
        }
        EnvCommands::Unsuspend { site_id, env_id } => {
            env::unsuspend(&client, &site_id, &env_id, format)
        }
    }
}

fn run_deploy(command: DeployCommands, format: OutputFormat) -> Result<(), ApiError> {
    let client = get_client()?;

    match command {
        DeployCommands::List {
            site_id,
            env_id,
            page,
            per_page,
        } => deploy::list(&client, &site_id, &env_id, page, per_page, format),
        DeployCommands::Show {
            site_id,
            env_id,
            deploy_id,
        } => deploy::show(&client, &site_id, &env_id, &deploy_id, format),
        DeployCommands::Create { site_id, env_id } => {
            deploy::create(&client, &site_id, &env_id, format)
        }
        DeployCommands::Rollback {
            site_id,
            env_id,
            deploy_id,
        } => deploy::rollback(&client, &site_id, &env_id, &deploy_id, format),
    }
}

fn run_ssl(command: SslCommands, format: OutputFormat) -> Result<(), ApiError> {
    let client = get_client()?;

    match command {
        SslCommands::Status { site_id, env_id } => ssl::status(&client, &site_id, &env_id, format),
        SslCommands::Nudge { site_id, env_id } => ssl::nudge(&client, &site_id, &env_id, format),
    }
}

fn run_mcp(command: McpCommands, format: OutputFormat) -> Result<(), ApiError> {
    match command {
        McpCommands::Setup { force } => mcp::setup(force, format),
    }
}
