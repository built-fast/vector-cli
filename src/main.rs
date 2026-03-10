mod api;
mod cli;
mod commands;
mod config;
mod output;

use clap::Parser;
use serde_json::Value;
use std::process;

use api::{ApiClient, ApiError, EXIT_SUCCESS};
use cli::{
    AccountApiKeyCommands, AccountCommands, AccountSecretCommands, AccountSshKeyCommands,
    ArchiveCommands, AuthCommands, BackupCommands, BackupDownloadCommands, Cli, Commands,
    DbCommands, DbExportCommands, DbImportSessionCommands, DeployCommands, EnvCommands,
    EnvDbCommands, EnvDomainChangeCommands, EnvSecretCommands, EventCommands, McpCommands,
    RestoreCommands, SiteCommands, SiteSshKeyCommands, SslCommands, WafAllowedReferrerCommands,
    WafBlockedIpCommands, WafBlockedReferrerCommands, WafCommands, WafRateLimitCommands,
    WebhookCommands,
};
use commands::{
    account, archive, auth, backup, db, deploy, env, event, mcp, restore, site, ssl, waf, webhook,
};
use config::{Config, Credentials};
use output::{OutputFormat, print_error, print_json, print_message, print_table};

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
        Commands::Db { command } => run_db(command, format),
        Commands::Archive { command } => run_archive(command, format),
        Commands::Waf { command } => run_waf(command, format),
        Commands::Account { command } => run_account(command, format),
        Commands::Backup { command } => run_backup(command, format),
        Commands::Restore { command } => run_restore(command, format),
        Commands::Event { command } => run_event(command, format),
        Commands::Webhook { command } => run_webhook(command, format),
        Commands::PhpVersions => run_php_versions(format),
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
            customer_id,
            dev_php_version,
            tags,
        } => site::create(&client, &customer_id, &dev_php_version, tags, format),
        SiteCommands::Update {
            id,
            customer_id,
            tags,
        } => site::update(&client, &id, customer_id, tags, format),
        SiteCommands::Delete { id, force } => site::delete(&client, &id, force, format),
        SiteCommands::Clone {
            id,
            customer_id,
            dev_php_version,
            tags,
        } => site::clone(&client, &id, customer_id, dev_php_version, tags, format),
        SiteCommands::Suspend { id } => site::suspend(&client, &id, format),
        SiteCommands::Unsuspend { id } => site::unsuspend(&client, &id, format),
        SiteCommands::ResetSftpPassword { id } => site::reset_sftp_password(&client, &id, format),
        SiteCommands::ResetDbPassword { id } => site::reset_db_password(&client, &id, format),
        SiteCommands::PurgeCache { id, cache_tag, url } => {
            site::purge_cache(&client, &id, cache_tag, url, format)
        }
        SiteCommands::Logs {
            id,
            start_time,
            end_time,
            limit,
            environment,
            deployment_id,
            level,
            cursor,
        } => site::logs(
            &client,
            &id,
            start_time,
            end_time,
            limit,
            environment,
            deployment_id,
            level,
            cursor,
            format,
        ),
        SiteCommands::WpReconfig { id } => site::wp_reconfig(&client, &id, format),
        SiteCommands::SshKey { command } => run_site_ssh_key(&client, command, format),
    }
}

fn run_site_ssh_key(
    client: &ApiClient,
    command: SiteSshKeyCommands,
    format: OutputFormat,
) -> Result<(), ApiError> {
    match command {
        SiteSshKeyCommands::List {
            site_id,
            page,
            per_page,
        } => site::ssh_key_list(client, &site_id, page, per_page, format),
        SiteSshKeyCommands::Add {
            site_id,
            name,
            public_key,
        } => site::ssh_key_add(client, &site_id, &name, &public_key, format),
        SiteSshKeyCommands::Remove { site_id, key_id } => {
            site::ssh_key_remove(client, &site_id, &key_id, format)
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
        EnvCommands::Show { env_id } => env::show(&client, &env_id, format),
        EnvCommands::Create {
            site_id,
            name,
            custom_domain,
            php_version,
            is_production,
            tags,
        } => env::create(
            &client,
            &site_id,
            &name,
            &custom_domain,
            &php_version,
            is_production,
            tags,
            format,
        ),
        EnvCommands::Update {
            env_id,
            name,
            custom_domain,
            clear_custom_domain,
            tags,
        } => {
            let custom_domain = if clear_custom_domain {
                Some(None)
            } else {
                custom_domain.map(Some)
            };
            env::update(&client, &env_id, name, custom_domain, tags, format)
        }
        EnvCommands::Delete { env_id } => env::delete(&client, &env_id, format),
        EnvCommands::ResetDbPassword { env_id } => env::reset_db_password(&client, &env_id, format),
        EnvCommands::Secret { command } => run_env_secret(&client, command, format),
        EnvCommands::Db { command } => run_env_db(&client, command, format),
    }
}

fn run_env_secret(
    client: &ApiClient,
    command: EnvSecretCommands,
    format: OutputFormat,
) -> Result<(), ApiError> {
    match command {
        EnvSecretCommands::List {
            env_id,
            page,
            per_page,
        } => env::secret_list(client, &env_id, page, per_page, format),
        EnvSecretCommands::Show { secret_id } => env::secret_show(client, &secret_id, format),
        EnvSecretCommands::Create {
            env_id,
            key,
            value,
            no_secret,
        } => env::secret_create(client, &env_id, &key, &value, no_secret, format),
        EnvSecretCommands::Update {
            secret_id,
            key,
            value,
            no_secret,
        } => env::secret_update(client, &secret_id, key, value, no_secret, format),
        EnvSecretCommands::Delete { secret_id } => env::secret_delete(client, &secret_id, format),
    }
}

fn run_env_db(
    client: &ApiClient,
    command: EnvDbCommands,
    format: OutputFormat,
) -> Result<(), ApiError> {
    match command {
        EnvDbCommands::Promote {
            env_id,
            drop_tables,
            disable_foreign_keys,
        } => env::db_promote(client, &env_id, drop_tables, disable_foreign_keys, format),
        EnvDbCommands::PromoteStatus { env_id, promote_id } => {
            env::db_promote_status(client, &env_id, &promote_id, format)
        }
        EnvDbCommands::DomainChange { command } => run_env_domain_change(client, command, format),
    }
}

fn run_env_domain_change(
    client: &ApiClient,
    command: EnvDomainChangeCommands,
    format: OutputFormat,
) -> Result<(), ApiError> {
    match command {
        EnvDomainChangeCommands::Create {
            env_id,
            custom_domain,
            clear_custom_domain,
        } => {
            let custom_domain = if clear_custom_domain {
                None
            } else {
                custom_domain
            };
            env::domain_change_create(client, &env_id, custom_domain, format)
        }
        EnvDomainChangeCommands::Status {
            env_id,
            domain_change_id,
        } => env::domain_change_status(client, &env_id, &domain_change_id, format),
    }
}

fn run_deploy(command: DeployCommands, format: OutputFormat) -> Result<(), ApiError> {
    let client = get_client()?;

    match command {
        DeployCommands::List {
            env_id,
            page,
            per_page,
        } => deploy::list(&client, &env_id, page, per_page, format),
        DeployCommands::Show { deploy_id } => deploy::show(&client, &deploy_id, format),
        DeployCommands::Trigger {
            env_id,
            include_uploads,
            include_database,
        } => deploy::trigger(&client, &env_id, include_uploads, include_database, format),
        DeployCommands::Rollback {
            env_id,
            target_deployment_id,
        } => deploy::rollback(&client, &env_id, target_deployment_id, format),
    }
}

fn run_ssl(command: SslCommands, format: OutputFormat) -> Result<(), ApiError> {
    let client = get_client()?;

    match command {
        SslCommands::Status { env_id } => ssl::status(&client, &env_id, format),
        SslCommands::Nudge { env_id, retry } => ssl::nudge(&client, &env_id, retry, format),
    }
}

fn run_db(command: DbCommands, format: OutputFormat) -> Result<(), ApiError> {
    let client = get_client()?;

    match command {
        DbCommands::ImportSession { command } => run_db_import_session(&client, command, format),
        DbCommands::Export { command } => run_db_export(&client, command, format),
    }
}

fn run_db_import_session(
    client: &ApiClient,
    command: DbImportSessionCommands,
    format: OutputFormat,
) -> Result<(), ApiError> {
    match command {
        DbImportSessionCommands::Create {
            site_id,
            filename,
            content_length,
            drop_tables,
            disable_foreign_keys,
            search_replace_from,
            search_replace_to,
        } => db::import_session_create(
            client,
            &site_id,
            filename,
            content_length,
            drop_tables,
            disable_foreign_keys,
            search_replace_from,
            search_replace_to,
            format,
        ),
        DbImportSessionCommands::Run { site_id, import_id } => {
            db::import_session_run(client, &site_id, &import_id, format)
        }
        DbImportSessionCommands::Status { site_id, import_id } => {
            db::import_session_status(client, &site_id, &import_id, format)
        }
    }
}

fn run_db_export(
    client: &ApiClient,
    command: DbExportCommands,
    format: OutputFormat,
) -> Result<(), ApiError> {
    match command {
        DbExportCommands::Create {
            site_id,
            format: export_format,
        } => db::export_create(client, &site_id, export_format, format),
        DbExportCommands::Status { site_id, export_id } => {
            db::export_status(client, &site_id, &export_id, format)
        }
    }
}

fn run_archive(command: ArchiveCommands, format: OutputFormat) -> Result<(), ApiError> {
    let client = get_client()?;

    match command {
        ArchiveCommands::Import {
            site_id,
            file,
            drop_tables,
            disable_foreign_keys,
            search_replace_from,
            search_replace_to,
            wait,
            poll_interval,
        } => archive::import(
            &client,
            &site_id,
            &file,
            drop_tables,
            disable_foreign_keys,
            search_replace_from,
            search_replace_to,
            wait,
            poll_interval,
            format,
        ),
    }
}

fn run_waf(command: WafCommands, format: OutputFormat) -> Result<(), ApiError> {
    let client = get_client()?;

    match command {
        WafCommands::RateLimit { command } => run_waf_rate_limit(&client, command, format),
        WafCommands::BlockedIp { command } => run_waf_blocked_ip(&client, command, format),
        WafCommands::BlockedReferrer { command } => {
            run_waf_blocked_referrer(&client, command, format)
        }
        WafCommands::AllowedReferrer { command } => {
            run_waf_allowed_referrer(&client, command, format)
        }
    }
}

fn run_waf_rate_limit(
    client: &ApiClient,
    command: WafRateLimitCommands,
    format: OutputFormat,
) -> Result<(), ApiError> {
    match command {
        WafRateLimitCommands::List { site_id } => waf::rate_limit_list(client, &site_id, format),
        WafRateLimitCommands::Show { site_id, rule_id } => {
            waf::rate_limit_show(client, &site_id, &rule_id, format)
        }
        WafRateLimitCommands::Create {
            site_id,
            name,
            request_count,
            timeframe,
            block_time,
            description,
            value,
            operator,
            variables,
            transformations,
        } => waf::rate_limit_create(
            client,
            &site_id,
            &name,
            request_count,
            timeframe,
            block_time,
            description,
            value,
            operator,
            variables,
            transformations,
            format,
        ),
        WafRateLimitCommands::Update {
            site_id,
            rule_id,
            name,
            description,
            request_count,
            timeframe,
            block_time,
            value,
            operator,
            variables,
            transformations,
        } => waf::rate_limit_update(
            client,
            &site_id,
            &rule_id,
            name,
            description,
            request_count,
            timeframe,
            block_time,
            value,
            operator,
            variables,
            transformations,
            format,
        ),
        WafRateLimitCommands::Delete { site_id, rule_id } => {
            waf::rate_limit_delete(client, &site_id, &rule_id, format)
        }
    }
}

fn run_waf_blocked_ip(
    client: &ApiClient,
    command: WafBlockedIpCommands,
    format: OutputFormat,
) -> Result<(), ApiError> {
    match command {
        WafBlockedIpCommands::List { site_id } => waf::blocked_ip_list(client, &site_id, format),
        WafBlockedIpCommands::Add { site_id, ip } => {
            waf::blocked_ip_add(client, &site_id, &ip, format)
        }
        WafBlockedIpCommands::Remove { site_id, ip } => {
            waf::blocked_ip_remove(client, &site_id, &ip, format)
        }
    }
}

fn run_waf_blocked_referrer(
    client: &ApiClient,
    command: WafBlockedReferrerCommands,
    format: OutputFormat,
) -> Result<(), ApiError> {
    match command {
        WafBlockedReferrerCommands::List { site_id } => {
            waf::blocked_referrer_list(client, &site_id, format)
        }
        WafBlockedReferrerCommands::Add { site_id, hostname } => {
            waf::blocked_referrer_add(client, &site_id, &hostname, format)
        }
        WafBlockedReferrerCommands::Remove { site_id, hostname } => {
            waf::blocked_referrer_remove(client, &site_id, &hostname, format)
        }
    }
}

fn run_waf_allowed_referrer(
    client: &ApiClient,
    command: WafAllowedReferrerCommands,
    format: OutputFormat,
) -> Result<(), ApiError> {
    match command {
        WafAllowedReferrerCommands::List { site_id } => {
            waf::allowed_referrer_list(client, &site_id, format)
        }
        WafAllowedReferrerCommands::Add { site_id, hostname } => {
            waf::allowed_referrer_add(client, &site_id, &hostname, format)
        }
        WafAllowedReferrerCommands::Remove { site_id, hostname } => {
            waf::allowed_referrer_remove(client, &site_id, &hostname, format)
        }
    }
}

fn run_account(command: AccountCommands, format: OutputFormat) -> Result<(), ApiError> {
    let client = get_client()?;

    match command {
        AccountCommands::Show => account::show(&client, format),
        AccountCommands::SshKey { command } => run_account_ssh_key(&client, command, format),
        AccountCommands::ApiKey { command } => run_account_api_key(&client, command, format),
        AccountCommands::Secret { command } => run_account_secret(&client, command, format),
    }
}

fn run_account_ssh_key(
    client: &ApiClient,
    command: AccountSshKeyCommands,
    format: OutputFormat,
) -> Result<(), ApiError> {
    match command {
        AccountSshKeyCommands::List { page, per_page } => {
            account::ssh_key_list(client, page, per_page, format)
        }
        AccountSshKeyCommands::Show { key_id } => account::ssh_key_show(client, &key_id, format),
        AccountSshKeyCommands::Create { name, public_key } => {
            account::ssh_key_create(client, &name, &public_key, format)
        }
        AccountSshKeyCommands::Delete { key_id } => {
            account::ssh_key_delete(client, &key_id, format)
        }
    }
}

fn run_account_api_key(
    client: &ApiClient,
    command: AccountApiKeyCommands,
    format: OutputFormat,
) -> Result<(), ApiError> {
    match command {
        AccountApiKeyCommands::List { page, per_page } => {
            account::api_key_list(client, page, per_page, format)
        }
        AccountApiKeyCommands::Create {
            name,
            abilities,
            expires_at,
        } => account::api_key_create(client, &name, abilities, expires_at, format),
        AccountApiKeyCommands::Delete { token_id } => {
            account::api_key_delete(client, &token_id, format)
        }
    }
}

fn run_account_secret(
    client: &ApiClient,
    command: AccountSecretCommands,
    format: OutputFormat,
) -> Result<(), ApiError> {
    match command {
        AccountSecretCommands::List { page, per_page } => {
            account::secret_list(client, page, per_page, format)
        }
        AccountSecretCommands::Show { secret_id } => {
            account::secret_show(client, &secret_id, format)
        }
        AccountSecretCommands::Create {
            key,
            value,
            no_secret,
        } => account::secret_create(client, &key, &value, no_secret, format),
        AccountSecretCommands::Update {
            secret_id,
            key,
            value,
            no_secret,
        } => account::secret_update(client, &secret_id, key, value, no_secret, format),
        AccountSecretCommands::Delete { secret_id } => {
            account::secret_delete(client, &secret_id, format)
        }
    }
}

fn run_backup(command: BackupCommands, format: OutputFormat) -> Result<(), ApiError> {
    let client = get_client()?;

    match command {
        BackupCommands::List {
            site_id,
            environment_id,
            backup_type,
            page,
            per_page,
        } => backup::list(
            &client,
            site_id,
            environment_id,
            backup_type,
            page,
            per_page,
            format,
        ),
        BackupCommands::Show { backup_id } => backup::show(&client, &backup_id, format),
        BackupCommands::Create {
            site_id,
            environment_id,
            scope,
            description,
        } => backup::create(
            &client,
            site_id,
            environment_id,
            &scope,
            description,
            format,
        ),
        BackupCommands::Download { command } => run_backup_download(&client, command, format),
    }
}

fn run_backup_download(
    client: &ApiClient,
    command: BackupDownloadCommands,
    format: OutputFormat,
) -> Result<(), ApiError> {
    match command {
        BackupDownloadCommands::Create { backup_id } => {
            backup::download_create(client, &backup_id, format)
        }
        BackupDownloadCommands::Status {
            backup_id,
            download_id,
        } => backup::download_status(client, &backup_id, &download_id, format),
    }
}

fn run_restore(command: RestoreCommands, format: OutputFormat) -> Result<(), ApiError> {
    let client = get_client()?;

    match command {
        RestoreCommands::List {
            site_id,
            environment_id,
            restore_type,
            backup_id,
            page,
            per_page,
        } => restore::list(
            &client,
            restore::ListRestoresQuery {
                r#type: restore_type,
                site_id,
                environment_id,
                backup_id,
                page,
                per_page,
            },
            format,
        ),
        RestoreCommands::Show { restore_id } => restore::show(&client, &restore_id, format),
        RestoreCommands::Create {
            backup_id,
            scope,
            drop_tables,
            disable_foreign_keys,
            search_replace_from,
            search_replace_to,
        } => restore::create(
            &client,
            &backup_id,
            &scope,
            drop_tables,
            disable_foreign_keys,
            search_replace_from,
            search_replace_to,
            format,
        ),
    }
}

fn run_event(command: EventCommands, format: OutputFormat) -> Result<(), ApiError> {
    let client = get_client()?;

    match command {
        EventCommands::List {
            from,
            to,
            event: event_type,
            page,
            per_page,
        } => event::list(&client, from, to, event_type, page, per_page, format),
    }
}

fn run_webhook(command: WebhookCommands, format: OutputFormat) -> Result<(), ApiError> {
    let client = get_client()?;

    match command {
        WebhookCommands::List { page, per_page } => webhook::list(&client, page, per_page, format),
        WebhookCommands::Show { webhook_id } => webhook::show(&client, &webhook_id, format),
        WebhookCommands::Create {
            name,
            url,
            events,
            secret,
        } => webhook::create(&client, &name, &url, events, secret, format),
        WebhookCommands::Update {
            webhook_id,
            name,
            url,
            events,
            secret,
            enabled,
        } => webhook::update(
            &client,
            &webhook_id,
            name,
            url,
            events,
            secret,
            enabled,
            format,
        ),
        WebhookCommands::Delete { webhook_id } => webhook::delete(&client, &webhook_id, format),
    }
}

fn run_php_versions(format: OutputFormat) -> Result<(), ApiError> {
    let client = get_client()?;
    let response: Value = client.get("/api/v1/vector/php-versions")?;

    if format == OutputFormat::Json {
        print_json(&response);
        return Ok(());
    }

    let versions = response["data"]
        .as_array()
        .ok_or_else(|| ApiError::Other("Invalid response format".to_string()))?;

    if versions.is_empty() {
        print_message("No PHP versions available.");
        return Ok(());
    }

    let rows: Vec<Vec<String>> = versions
        .iter()
        .map(|v| vec![v.as_str().unwrap_or("-").to_string()])
        .collect();

    print_table(vec!["Version"], rows);

    Ok(())
}

fn run_mcp(command: McpCommands, format: OutputFormat) -> Result<(), ApiError> {
    match command {
        McpCommands::Setup { force } => mcp::setup(force, format),
    }
}
