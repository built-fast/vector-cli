package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/built-fast/vector-cli/internal/api"
	"github.com/built-fast/vector-cli/internal/output"
)

const sitesBasePath = "/api/v1/vector/sites"

// NewSiteCmd creates the site command group.
func NewSiteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "site",
		Short: "Manage sites",
		Long:  "Manage Vector sites including creating, updating, deleting, and performing actions like suspend, clone, and cache purge.",
	}

	cmd.AddCommand(newSiteListCmd())
	cmd.AddCommand(newSiteShowCmd())
	cmd.AddCommand(newSiteCreateCmd())
	cmd.AddCommand(newSiteUpdateCmd())
	cmd.AddCommand(newSiteDeleteCmd())
	cmd.AddCommand(newSiteCloneCmd())
	cmd.AddCommand(newSiteSuspendCmd())
	cmd.AddCommand(newSiteUnsuspendCmd())
	cmd.AddCommand(newSiteResetSFTPPasswordCmd())
	cmd.AddCommand(newSiteResetDBPasswordCmd())
	cmd.AddCommand(newSitePurgeCacheCmd())
	cmd.AddCommand(newSiteLogsCmd())
	cmd.AddCommand(newSiteWPReconfigCmd())
	cmd.AddCommand(NewSiteSSHKeyCmd())

	return cmd
}

func newSiteListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all sites",
		Long: "Retrieve a paginated list of all sites for the authenticated account.",
		Example: `  # List all sites
  vector site list

  # List with pagination
  vector site list --page 2 --per-page 50`,
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			page, perPage := getPagination(cmd)
			query := buildPaginationQuery(page, perPage)

			resp, err := app.Client.Get(cmd.Context(), sitesBasePath, query)
			if err != nil {
				return fmt.Errorf("failed to list sites: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to list sites: %w", err)
			}

			if app.Output.Format() == output.JSON {
				data, err := parseResponseData(body)
				if err != nil {
					return fmt.Errorf("failed to list sites: %w", err)
				}
				return app.Output.JSON(json.RawMessage(data))
			}

			data, meta, err := parseResponseWithMeta(body)
			if err != nil {
				return fmt.Errorf("failed to list sites: %w", err)
			}

			var items []map[string]any
			if err := json.Unmarshal(data, &items); err != nil {
				return fmt.Errorf("failed to list sites: %w", err)
			}

			headers := []string{"ID", "CUSTOMER ID", "STATUS", "DEV DOMAIN", "TAGS"}
			var rows [][]string
			for _, item := range items {
				tags := tagsFromItem(item)
				rows = append(rows, []string{
					getString(item, "id"),
					formatString(getString(item, "your_customer_id")),
					getString(item, "status"),
					formatString(getString(item, "dev_domain")),
					formatTags(tags),
				})
			}

			app.Output.Table(headers, rows)
			if meta != nil && meta.LastPage > 1 {
				app.Output.Pagination(meta.CurrentPage, meta.LastPage, meta.Total)
			}
			return nil
		},
	}
	addPaginationFlags(cmd)
	return cmd
}

func newSiteShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <site-id>",
		Short: "Show site details",
		Long: "Retrieve details of a specific site including its environments.",
		Example: `  # Show site details
  vector site show site-abc123`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			resp, err := app.Client.Get(cmd.Context(), sitesBasePath+"/"+args[0], nil)
			if err != nil {
				return fmt.Errorf("failed to show site: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to show site: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to show site: %w", err)
			}

			if app.Output.Format() == output.JSON {
				return app.Output.JSON(json.RawMessage(data))
			}

			var item map[string]any
			if err := json.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to show site: %w", err)
			}

			tags := tagsFromItem(item)

			pairs := []output.KeyValue{
				{Key: "ID", Value: getString(item, "id")},
				{Key: "Customer ID", Value: formatString(getString(item, "your_customer_id"))},
				{Key: "Status", Value: getString(item, "status")},
				{Key: "Tags", Value: formatTags(tags)},
				{Key: "Dev Domain", Value: formatString(getString(item, "dev_domain"))},
				{Key: "Dev DB Host", Value: formatString(getString(item, "dev_db_host"))},
				{Key: "Dev DB Name", Value: formatString(getString(item, "dev_db_name"))},
				{Key: "Created", Value: getString(item, "created_at")},
				{Key: "Updated", Value: getString(item, "updated_at")},
			}
			app.Output.KeyValue(pairs)

			// Print environments table if present
			envs := getSlice(item, "environments")
			if len(envs) > 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout())
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Environments:")
				headers := []string{"ID", "NAME", "PRODUCTION", "STATUS", "PHP", "PLATFORM DOMAIN", "CUSTOM DOMAIN"}
				var rows [][]string
				for _, e := range envs {
					env, ok := e.(map[string]any)
					if !ok {
						continue
					}
					rows = append(rows, []string{
						getString(env, "id"),
						getString(env, "name"),
						formatBool(getBool(env, "is_production")),
						getString(env, "status"),
						getString(env, "php_version"),
						formatString(getString(env, "platform_domain")),
						formatString(getString(env, "custom_domain")),
					})
				}
				app.Output.Table(headers, rows)
			}

			return nil
		},
	}
}

func newSiteCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new site",
		Long: "Create a new site with a development container. Returns credentials that are only shown once.",
		Example: `  # Create a site
  vector site create --customer-id cust-001 --php-version 8.2

  # Create with WordPress auto-install
  vector site create --customer-id cust-001 --php-version 8.2 --wp-admin-email admin@example.com

  # Create and wait for site to become active
  vector site create --customer-id cust-001 --wait`,
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			waitEnabled, interval, timeout, err := getWaitConfig(cmd)
			if err != nil {
				return err
			}

			customerID, _ := cmd.Flags().GetString("customer-id")
			if customerID == "" {
				return &api.APIError{
					Message:  "--customer-id is required",
					ExitCode: 3,
				}
			}

			reqBody := map[string]any{
				"your_customer_id": customerID,
			}

			phpVersion, _ := cmd.Flags().GetString("php-version")
			if phpVersion != "" {
				reqBody["dev_php_version"] = phpVersion
			}

			if cmd.Flags().Changed("tags") {
				tagsStr, _ := cmd.Flags().GetString("tags")
				if tagsStr != "" {
					reqBody["tags"] = strings.Split(tagsStr, ",")
				} else {
					reqBody["tags"] = []string{}
				}
			}

			if cmd.Flags().Changed("production-domain") {
				v, _ := cmd.Flags().GetString("production-domain")
				reqBody["production_domain"] = v
			}
			if cmd.Flags().Changed("staging-domain") {
				v, _ := cmd.Flags().GetString("staging-domain")
				reqBody["staging_domain"] = v
			}
			if cmd.Flags().Changed("wp-admin-email") {
				v, _ := cmd.Flags().GetString("wp-admin-email")
				reqBody["wp_admin_email"] = v
			}
			if cmd.Flags().Changed("wp-admin-user") {
				v, _ := cmd.Flags().GetString("wp-admin-user")
				reqBody["wp_admin_user"] = v
			}
			if cmd.Flags().Changed("wp-site-title") {
				v, _ := cmd.Flags().GetString("wp-site-title")
				reqBody["wp_site_title"] = v
			}

			resp, err := app.Client.Post(cmd.Context(), sitesBasePath, reqBody)
			if err != nil {
				return fmt.Errorf("failed to create site: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to create site: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to create site: %w", err)
			}

			if !waitEnabled {
				if app.Output.Format() == output.JSON {
					return app.Output.JSON(json.RawMessage(data))
				}

				var item map[string]any
				if err := json.Unmarshal(data, &item); err != nil {
					return fmt.Errorf("failed to create site: %w", err)
				}

				app.Output.KeyValue(siteCreateCredentialPairs(item))
				return nil
			}

			var createItem map[string]any
			if err := json.Unmarshal(data, &createItem); err != nil {
				return fmt.Errorf("failed to create site: %w", err)
			}

			siteID := getString(createItem, "id")
			if siteID == "" {
				return fmt.Errorf("failed to create site: response missing site ID")
			}

			// TTY/table mode: print one-time credentials before entering alt screen
			if app.Output.Format() != output.JSON {
				app.Output.KeyValue(siteCreateCredentialPairs(createItem))
				_, _ = fmt.Fprintln(cmd.OutOrStdout())
			}

			cfg := &waitConfig{
				ResourceID:       siteID,
				PollPath:         sitesBasePath + "/" + siteID,
				Interval:         interval,
				Timeout:          timeout,
				TerminalStatuses: map[string]bool{"active": true},
				FailedStatuses:   map[string]bool{"failed": true},
				Noun:             "Site",
				FormatDisplay:    siteFormatDisplay,
			}

			result, err := waitForResource(cmd.Context(), app, cfg)
			if err != nil {
				return err
			}

			if app.Output.Format() == output.JSON {
				merged := siteCreateMergeCredentials(createItem, result.FinalData)
				return app.Output.JSON(merged)
			}

			var finalItem map[string]any
			if err := json.Unmarshal(result.FinalData, &finalItem); err != nil {
				return fmt.Errorf("failed to create site: %w", err)
			}

			app.Output.Message(fmt.Sprintf("Site %s %s in %s", siteID, result.Status, result.Elapsed.Truncate(time.Second)))
			app.Output.KeyValue([]output.KeyValue{
				{Key: "ID", Value: getString(finalItem, "id")},
				{Key: "Customer ID", Value: formatString(getString(finalItem, "your_customer_id"))},
				{Key: "Status", Value: getString(finalItem, "status")},
				{Key: "Dev Domain", Value: formatString(getString(finalItem, "dev_domain"))},
			})
			return nil
		},
	}

	cmd.Flags().String("customer-id", "", "Your internal customer identifier (required)")
	cmd.Flags().String("php-version", "", "PHP version for development container")
	cmd.Flags().String("tags", "", "Comma-separated tags")
	cmd.Flags().String("production-domain", "", "Custom domain for production environment")
	cmd.Flags().String("staging-domain", "", "Custom domain for staging environment")
	cmd.Flags().String("wp-admin-email", "", "WordPress admin email for auto-install")
	cmd.Flags().String("wp-admin-user", "", "WordPress admin username")
	cmd.Flags().String("wp-site-title", "", "WordPress site title")
	addWaitFlags(cmd)

	return cmd
}

// siteFormatDisplay formats site data for the alternate screen display.
func siteFormatDisplay(data map[string]any) []string {
	return []string{
		fmt.Sprintf("%16s: %s", "ID", getString(data, "id")),
		fmt.Sprintf("%16s: %s", "Customer ID", formatString(getString(data, "your_customer_id"))),
		fmt.Sprintf("%16s: %s", "Status", getString(data, "status")),
		fmt.Sprintf("%16s: %s", "Dev Domain", formatString(getString(data, "dev_domain"))),
	}
}

// siteCreateCredentialPairs builds the key-value pairs for the site create response,
// including one-time SFTP, DB, and WP admin credentials.
func siteCreateCredentialPairs(item map[string]any) []output.KeyValue {
	pairs := []output.KeyValue{
		{Key: "ID", Value: getString(item, "id")},
		{Key: "Customer ID", Value: formatString(getString(item, "your_customer_id"))},
		{Key: "Status", Value: getString(item, "status")},
		{Key: "Dev Domain", Value: formatString(getString(item, "dev_domain"))},
		{Key: "Dev DB Host", Value: formatString(getString(item, "dev_db_host"))},
		{Key: "Dev DB Name", Value: formatString(getString(item, "dev_db_name"))},
	}

	sftp := getMap(item, "dev_sftp")
	if sftp != nil {
		pairs = append(pairs,
			output.KeyValue{Key: "SFTP Host", Value: getString(sftp, "hostname")},
			output.KeyValue{Key: "SFTP Port", Value: fmt.Sprintf("%.0f", getFloat(sftp, "port"))},
			output.KeyValue{Key: "SFTP User", Value: getString(sftp, "username")},
			output.KeyValue{Key: "SFTP Password", Value: getString(sftp, "password")},
		)
	}

	dbUser := getString(item, "dev_db_username")
	dbPass := getString(item, "dev_db_password")
	if dbUser != "" {
		pairs = append(pairs, output.KeyValue{Key: "DB Username", Value: dbUser})
	}
	if dbPass != "" {
		pairs = append(pairs, output.KeyValue{Key: "DB Password", Value: dbPass})
	}

	wp := getMap(item, "wp_admin")
	if wp != nil {
		pairs = append(pairs,
			output.KeyValue{Key: "WP Admin User", Value: getString(wp, "user")},
			output.KeyValue{Key: "WP Admin Email", Value: getString(wp, "email")},
			output.KeyValue{Key: "WP Admin Password", Value: getString(wp, "password")},
			output.KeyValue{Key: "WP Site Title", Value: getString(wp, "site_title")},
		)
	}

	return pairs
}

// siteCreateMergeCredentials merges one-time credential fields from the initial
// create response into the final polled data for JSON output.
func siteCreateMergeCredentials(createItem map[string]any, finalData json.RawMessage) json.RawMessage {
	var finalItem map[string]any
	if err := json.Unmarshal(finalData, &finalItem); err != nil {
		return finalData
	}

	// Merge one-time credential fields that are not present in the polled response
	credentialKeys := []string{"dev_sftp", "dev_db_username", "dev_db_password", "wp_admin"}
	for _, key := range credentialKeys {
		if val, ok := createItem[key]; ok && val != nil {
			if _, exists := finalItem[key]; !exists || finalItem[key] == nil {
				finalItem[key] = val
			}
		}
	}

	merged, err := json.Marshal(finalItem)
	if err != nil {
		return finalData
	}
	return merged
}

func newSiteUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <site-id>",
		Short: "Update a site",
		Long: "Update a site's metadata such as customer ID and tags.",
		Example: `  # Update tags
  vector site update site-abc123 --tags production,primary`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			reqBody := map[string]any{}

			if cmd.Flags().Changed("customer-id") {
				v, _ := cmd.Flags().GetString("customer-id")
				reqBody["your_customer_id"] = v
			}
			if cmd.Flags().Changed("tags") {
				tagsStr, _ := cmd.Flags().GetString("tags")
				if tagsStr != "" {
					reqBody["tags"] = strings.Split(tagsStr, ",")
				} else {
					reqBody["tags"] = nil
				}
			}

			resp, err := app.Client.Put(cmd.Context(), sitesBasePath+"/"+args[0], reqBody)
			if err != nil {
				return fmt.Errorf("failed to update site: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to update site: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to update site: %w", err)
			}

			if app.Output.Format() == output.JSON {
				return app.Output.JSON(json.RawMessage(data))
			}

			var item map[string]any
			if err := json.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to update site: %w", err)
			}

			tags := tagsFromItem(item)
			app.Output.KeyValue([]output.KeyValue{
				{Key: "ID", Value: getString(item, "id")},
				{Key: "Customer ID", Value: formatString(getString(item, "your_customer_id"))},
				{Key: "Status", Value: getString(item, "status")},
				{Key: "Tags", Value: formatTags(tags)},
				{Key: "Dev Domain", Value: formatString(getString(item, "dev_domain"))},
			})
			return nil
		},
	}

	cmd.Flags().String("customer-id", "", "Your internal customer identifier")
	cmd.Flags().String("tags", "", "Comma-separated tags (empty string clears tags)")

	return cmd
}

func newSiteDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <site-id>",
		Short: "Delete a site",
		Long: "Initiate deletion of a site. All environments must be terminated first. This operation is irreversible.",
		Example: `  # Delete a site (prompts for confirmation)
  vector site delete site-abc123

  # Delete without confirmation
  vector site delete site-abc123 --force`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			force, _ := cmd.Flags().GetBool("force")
			if !force {
				if !confirmAction(cmd, fmt.Sprintf("Are you sure you want to delete site %s?", args[0])) {
					app.Output.Message("Aborted.")
					return nil
				}
			}

			resp, err := app.Client.Delete(cmd.Context(), sitesBasePath+"/"+args[0])
			if err != nil {
				return fmt.Errorf("failed to delete site: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to delete site: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to delete site: %w", err)
			}

			if app.Output.Format() == output.JSON {
				return app.Output.JSON(json.RawMessage(data))
			}

			var item map[string]any
			if err := json.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to delete site: %w", err)
			}

			app.Output.Message(fmt.Sprintf("Site %s deletion initiated.", getString(item, "id")))
			return nil
		},
	}

	cmd.Flags().Bool("force", false, "Skip confirmation prompt")

	return cmd
}

func newSiteCloneCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clone <site-id>",
		Short: "Clone a site",
		Long: "Create a new site by cloning an existing site's development container including files and database.",
		Example: `  # Clone a site
  vector site clone site-abc123

  # Clone with a different customer ID
  vector site clone site-abc123 --customer-id cust-002`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			reqBody := map[string]any{}

			if cmd.Flags().Changed("customer-id") {
				v, _ := cmd.Flags().GetString("customer-id")
				reqBody["your_customer_id"] = v
			}
			if cmd.Flags().Changed("php-version") {
				v, _ := cmd.Flags().GetString("php-version")
				reqBody["dev_php_version"] = v
			}
			if cmd.Flags().Changed("tags") {
				tagsStr, _ := cmd.Flags().GetString("tags")
				if tagsStr != "" {
					reqBody["tags"] = strings.Split(tagsStr, ",")
				} else {
					reqBody["tags"] = []string{}
				}
			}

			resp, err := app.Client.Post(cmd.Context(), sitesBasePath+"/"+args[0]+"/clone", reqBody)
			if err != nil {
				return fmt.Errorf("failed to clone site: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to clone site: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to clone site: %w", err)
			}

			if app.Output.Format() == output.JSON {
				return app.Output.JSON(json.RawMessage(data))
			}

			var item map[string]any
			if err := json.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to clone site: %w", err)
			}

			pairs := []output.KeyValue{
				{Key: "ID", Value: getString(item, "id")},
				{Key: "Customer ID", Value: formatString(getString(item, "your_customer_id"))},
				{Key: "Status", Value: getString(item, "status")},
				{Key: "Dev Domain", Value: formatString(getString(item, "dev_domain"))},
			}

			dbUser := getString(item, "dev_db_username")
			dbPass := getString(item, "dev_db_password")
			if dbUser != "" {
				pairs = append(pairs, output.KeyValue{Key: "DB Username", Value: dbUser})
			}
			if dbPass != "" {
				pairs = append(pairs, output.KeyValue{Key: "DB Password", Value: dbPass})
			}

			app.Output.KeyValue(pairs)
			return nil
		},
	}

	cmd.Flags().String("customer-id", "", "Customer identifier for cloned site")
	cmd.Flags().String("php-version", "", "PHP version for cloned site")
	cmd.Flags().String("tags", "", "Comma-separated tags for cloned site")

	return cmd
}

func newSiteSuspendCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "suspend <site-id>",
		Short: "Suspend a site",
		Long: "Suspend a site's development container. The site must be active.",
		Example: `  # Suspend a site
  vector site suspend site-abc123`,
		Args: cobra.ExactArgs(1),
		RunE:  siteActionRunE("suspend", "PUT"),
	}
}

func newSiteUnsuspendCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "unsuspend <site-id>",
		Short: "Unsuspend a site",
		Long: "Resume a previously suspended site's development container.",
		Example: `  # Unsuspend a site
  vector site unsuspend site-abc123`,
		Args: cobra.ExactArgs(1),
		RunE:  siteActionRunE("unsuspend", "PUT"),
	}
}

func newSiteResetSFTPPasswordCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reset-sftp-password <site-id>",
		Short: "Reset SFTP password",
		Long: "Generate a new SFTP password for the site's development container. The new password is only shown once.",
		Example: `  # Reset SFTP password
  vector site reset-sftp-password site-abc123`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			resp, err := app.Client.Post(cmd.Context(), sitesBasePath+"/"+args[0]+"/sftp/reset-password", nil)
			if err != nil {
				return fmt.Errorf("failed to reset SFTP password: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to reset SFTP password: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to reset SFTP password: %w", err)
			}

			if app.Output.Format() == output.JSON {
				return app.Output.JSON(json.RawMessage(data))
			}

			var item map[string]any
			if err := json.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to reset SFTP password: %w", err)
			}

			sftp := getMap(item, "dev_sftp")
			if sftp == nil {
				app.Output.Message("SFTP password reset successfully.")
				return nil
			}

			app.Output.KeyValue([]output.KeyValue{
				{Key: "Hostname", Value: getString(sftp, "hostname")},
				{Key: "Port", Value: fmt.Sprintf("%.0f", getFloat(sftp, "port"))},
				{Key: "Username", Value: getString(sftp, "username")},
				{Key: "Password", Value: getString(sftp, "password")},
			})
			return nil
		},
	}
}

func newSiteResetDBPasswordCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reset-db-password <site-id>",
		Short: "Reset database password",
		Long: "Generate a new database password for the site's development container. The new password is only shown once.",
		Example: `  # Reset database password
  vector site reset-db-password site-abc123`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			resp, err := app.Client.Post(cmd.Context(), sitesBasePath+"/"+args[0]+"/db/reset-password", nil)
			if err != nil {
				return fmt.Errorf("failed to reset database password: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to reset database password: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to reset database password: %w", err)
			}

			if app.Output.Format() == output.JSON {
				return app.Output.JSON(json.RawMessage(data))
			}

			var item map[string]any
			if err := json.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to reset database password: %w", err)
			}

			app.Output.KeyValue([]output.KeyValue{
				{Key: "DB Username", Value: getString(item, "dev_db_username")},
				{Key: "DB Password", Value: getString(item, "dev_db_password")},
			})
			return nil
		},
	}
}

func newSitePurgeCacheCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "purge-cache <site-id>",
		Short: "Purge CDN cache",
		Long: "Purge the CDN cache for a site. Can purge the entire cache, by cache tag, or a specific URL.",
		Example: `  # Purge entire cache
  vector site purge-cache site-abc123

  # Purge a specific URL
  vector site purge-cache site-abc123 --url https://example.com/page`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			reqBody := map[string]any{}
			if cmd.Flags().Changed("cache-tag") {
				v, _ := cmd.Flags().GetString("cache-tag")
				reqBody["cache_tag"] = v
			}
			if cmd.Flags().Changed("url") {
				v, _ := cmd.Flags().GetString("url")
				reqBody["url"] = v
			}

			resp, err := app.Client.Post(cmd.Context(), sitesBasePath+"/"+args[0]+"/purge-cache", reqBody)
			if err != nil {
				return fmt.Errorf("failed to purge cache: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to purge cache: %w", err)
			}

			if app.Output.Format() == output.JSON {
				data, err := parseResponseData(body)
				if err != nil {
					return fmt.Errorf("failed to purge cache: %w", err)
				}
				return app.Output.JSON(json.RawMessage(data))
			}

			// Extract message from response
			var envelope struct {
				Message string `json:"message"`
			}
			if err := json.Unmarshal(body, &envelope); err == nil && envelope.Message != "" {
				app.Output.Message(envelope.Message)
			} else {
				app.Output.Message("Cache purged successfully.")
			}
			return nil
		},
	}

	cmd.Flags().String("cache-tag", "", "Purge only content with this cache tag")
	cmd.Flags().String("url", "", "Purge a specific URL")

	return cmd
}

func newSiteLogsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs <site-id>",
		Short: "View site logs",
		Long: "Retrieve logs for a site. Logs are returned in reverse chronological order.",
		Example: `  # View recent logs
  vector site logs site-abc123

  # View error logs with a limit
  vector site logs site-abc123 --level error --limit 100`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			query := buildLogsQuery(cmd)

			resp, err := app.Client.Get(cmd.Context(), sitesBasePath+"/"+args[0]+"/logs", query)
			if err != nil {
				return fmt.Errorf("failed to get logs: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to get logs: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to get logs: %w", err)
			}

			if app.Output.Format() == output.JSON {
				return app.Output.JSON(json.RawMessage(data))
			}

			var logData map[string]any
			if err := json.Unmarshal(data, &logData); err != nil {
				return fmt.Errorf("failed to get logs: %w", err)
			}

			printLogEntries(cmd.OutOrStdout(), logData)
			return nil
		},
	}

	cmd.Flags().String("start-time", "", "Start time (RFC3339 or relative, e.g., now-1h)")
	cmd.Flags().String("end-time", "", "End time (RFC3339 or relative)")
	cmd.Flags().Int("limit", 0, "Maximum number of log entries (1-1000)")
	cmd.Flags().String("environment", "", "Filter by environment name")
	cmd.Flags().String("deployment-id", "", "Filter by deployment ID")
	cmd.Flags().String("level", "", "Filter by log level (error, warning, info)")
	cmd.Flags().String("cursor", "", "Pagination cursor from previous response")

	return cmd
}

func newSiteWPReconfigCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "wp-reconfig <site-id>",
		Short: "Regenerate wp-config.php",
		Long: "Regenerate the wp-config.php file for the site's development container.",
		Example: `  # Regenerate wp-config.php
  vector site wp-reconfig site-abc123`,
		Args: cobra.ExactArgs(1),
		RunE:  sitePostActionRunE("wp/reconfig", "wp-config regenerated"),
	}
}

// siteActionRunE returns a RunE function for simple site action endpoints (suspend/unsuspend).
func siteActionRunE(action, method string) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		app, err := requireApp(cmd)
		if err != nil {
			return err
		}

		path := sitesBasePath + "/" + args[0] + "/" + action

		var (
			resp *http.Response
			reqErr error
		)
		switch method {
		case "PUT":
			resp, reqErr = app.Client.Put(cmd.Context(), path, nil)
		default:
			resp, reqErr = app.Client.Post(cmd.Context(), path, nil)
		}
		if reqErr != nil {
			return fmt.Errorf("failed to %s site: %w", action, reqErr)
		}
		defer func() { _ = resp.Body.Close() }()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to %s site: %w", action, err)
		}

		data, parseErr := parseResponseData(body)
		if parseErr != nil {
			return fmt.Errorf("failed to %s site: %w", action, parseErr)
		}

		if app.Output.Format() == output.JSON {
			return app.Output.JSON(json.RawMessage(data))
		}

		var item map[string]any
		if err := json.Unmarshal(data, &item); err != nil {
			return fmt.Errorf("failed to %s site: %w", action, err)
		}

		app.Output.Message(fmt.Sprintf("Site %s %s initiated.", getString(item, "id"), action))
		return nil
	}
}

// sitePostActionRunE returns a RunE function for simple POST site action endpoints.
func sitePostActionRunE(subPath, successMsg string) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		app, err := requireApp(cmd)
		if err != nil {
			return err
		}

		resp, err := app.Client.Post(cmd.Context(), sitesBasePath+"/"+args[0]+"/"+subPath, nil)
		if err != nil {
			var apiErr *api.APIError
			if errors.As(err, &apiErr) {
				return apiErr
			}
			return fmt.Errorf("failed to %s: %w", successMsg, err)
		}
		defer func() { _ = resp.Body.Close() }()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("reading response: %w", err)
		}

		data, parseErr := parseResponseData(body)
		if parseErr != nil {
			return fmt.Errorf("parsing response: %w", parseErr)
		}

		if app.Output.Format() == output.JSON {
			return app.Output.JSON(json.RawMessage(data))
		}

		// Extract message from full response
		var envelope struct {
			Message string `json:"message"`
		}
		if err := json.Unmarshal(body, &envelope); err == nil && envelope.Message != "" {
			app.Output.Message(envelope.Message)
		} else {
			app.Output.Message("Operation completed successfully.")
		}
		return nil
	}
}

// buildLogsQuery builds query parameters for the logs endpoint.
func buildLogsQuery(cmd *cobra.Command) map[string][]string {
	q := make(map[string][]string)

	flagMap := map[string]string{
		"start-time":    "start_time",
		"end-time":      "end_time",
		"environment":   "environment",
		"deployment-id": "deployment_id",
		"level":         "level",
		"cursor":        "cursor",
	}

	for flag, param := range flagMap {
		if cmd.Flags().Changed(flag) {
			v, _ := cmd.Flags().GetString(flag)
			if v != "" {
				q[param] = []string{v}
			}
		}
	}

	if cmd.Flags().Changed("limit") {
		v, _ := cmd.Flags().GetInt("limit")
		if v > 0 {
			q["limit"] = []string{fmt.Sprintf("%d", v)}
		}
	}

	return q
}

// printLogEntries prints log entries from the logs API response.
func printLogEntries(w io.Writer, logData map[string]any) {
	logs := getMap(logData, "logs")
	if logs == nil {
		output.PrintMessage(w, "No logs found.")
		return
	}

	tables := getSlice(logs, "tables")
	if len(tables) == 0 {
		output.PrintMessage(w, "No logs found.")
		return
	}

	for _, t := range tables {
		table, ok := t.(map[string]any)
		if !ok {
			continue
		}

		columns := getSlice(table, "columns")
		rows := getSlice(table, "rows")

		if len(columns) == 0 || len(rows) == 0 {
			continue
		}

		// Build header names
		var headers []string
		for _, c := range columns {
			col, ok := c.(map[string]any)
			if !ok {
				continue
			}
			headers = append(headers, strings.ToUpper(getString(col, "name")))
		}

		// Build row data
		var tableRows [][]string
		for _, r := range rows {
			row, ok := r.([]any)
			if !ok {
				continue
			}
			var cells []string
			for _, cell := range row {
				cells = append(cells, fmt.Sprintf("%v", cell))
			}
			tableRows = append(tableRows, cells)
		}

		output.PrintTable(w, headers, tableRows)
	}

	// Show cursor info
	cursor := getString(logData, "cursor")
	hasMore := getBool(logData, "has_more")
	if hasMore && cursor != "" {
		_, _ = fmt.Fprintf(w, "\nMore logs available. Use --cursor %s to continue.\n", cursor)
	}
}

// tagsFromItem extracts tags as []string from a map item.
func tagsFromItem(item map[string]any) []string {
	rawTags := getSlice(item, "tags")
	var tags []string
	for _, t := range rawTags {
		if s, ok := t.(string); ok {
			tags = append(tags, s)
		}
	}
	return tags
}
