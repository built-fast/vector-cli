package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/spf13/cobra"

	"github.com/built-fast/vector-cli/internal/api"
	"github.com/built-fast/vector-cli/internal/output"
)

const envsBasePath = "/api/v1/vector/environments"

// NewEnvCmd creates the env command group.
func NewEnvCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "env",
		Short: "Manage environments",
		Long:  "Manage Vector environments including creating, updating, deleting, and managing secrets and database promotes.",
	}

	cmd.AddCommand(newEnvListCmd())
	cmd.AddCommand(newEnvShowCmd())
	cmd.AddCommand(newEnvCreateCmd())
	cmd.AddCommand(newEnvUpdateCmd())
	cmd.AddCommand(newEnvDeleteCmd())
	cmd.AddCommand(NewEnvSecretCmd())
	cmd.AddCommand(NewEnvDBCmd())

	return cmd
}

func newEnvListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list <site-id>",
		Short: "List environments for a site",
		Long:  "Retrieve a paginated list of environments for a site.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			page, perPage := getPagination(cmd)
			query := buildPaginationQuery(page, perPage)
			query.Set("site", args[0])

			resp, err := app.Client.Get(cmd.Context(), envsBasePath, query)
			if err != nil {
				return fmt.Errorf("failed to list environments: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to list environments: %w", err)
			}

			if app.Output.Format() == output.JSON {
				data, err := parseResponseData(body)
				if err != nil {
					return fmt.Errorf("failed to list environments: %w", err)
				}
				return app.Output.JSON(json.RawMessage(data))
			}

			data, meta, err := parseResponseWithMeta(body)
			if err != nil {
				return fmt.Errorf("failed to list environments: %w", err)
			}

			var items []map[string]any
			if err := json.Unmarshal(data, &items); err != nil {
				return fmt.Errorf("failed to list environments: %w", err)
			}

			headers := []string{"ID", "NAME", "PRODUCTION", "STATUS", "PHP", "PLATFORM DOMAIN", "CUSTOM DOMAIN"}
			var rows [][]string
			for _, item := range items {
				rows = append(rows, []string{
					getString(item, "id"),
					getString(item, "name"),
					formatBool(getBool(item, "is_production")),
					getString(item, "status"),
					getString(item, "php_version"),
					formatString(getString(item, "platform_domain")),
					formatString(getString(item, "custom_domain")),
				})
			}

			app.Output.Table(headers, rows)
			if meta.LastPage > 1 {
				app.Output.Pagination(meta.CurrentPage, meta.LastPage, meta.Total)
			}
			return nil
		},
	}
	addPaginationFlags(cmd)
	return cmd
}

func newEnvShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <env-id>",
		Short: "Show environment details",
		Long:  "Retrieve details of a specific environment.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			resp, err := app.Client.Get(cmd.Context(), envsBasePath+"/"+args[0], nil)
			if err != nil {
				return fmt.Errorf("failed to show environment: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to show environment: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to show environment: %w", err)
			}

			if app.Output.Format() == output.JSON {
				return app.Output.JSON(json.RawMessage(data))
			}

			var item map[string]any
			if err := json.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to show environment: %w", err)
			}

			tags := tagsFromItem(item)
			pairs := []output.KeyValue{
				{Key: "ID", Value: getString(item, "id")},
				{Key: "Site ID", Value: getString(item, "vector_site_id")},
				{Key: "Name", Value: getString(item, "name")},
				{Key: "Production", Value: formatBool(getBool(item, "is_production"))},
				{Key: "Status", Value: getString(item, "status")},
				{Key: "PHP Version", Value: getString(item, "php_version")},
				{Key: "Tags", Value: formatTags(tags)},
				{Key: "Platform Domain", Value: formatString(getString(item, "platform_domain"))},
				{Key: "Custom Domain", Value: formatString(getString(item, "custom_domain"))},
				{Key: "DNS Target", Value: formatString(getString(item, "dns_target"))},
				{Key: "Database Host", Value: formatString(getString(item, "database_host"))},
				{Key: "Database Name", Value: formatString(getString(item, "database_name"))},
				{Key: "Created", Value: getString(item, "created_at")},
				{Key: "Updated", Value: getString(item, "updated_at")},
			}

			cert := getMap(item, "custom_domain_certificate")
			if cert != nil {
				pairs = append(pairs, output.KeyValue{Key: "Certificate Status", Value: formatString(getString(cert, "status"))})
			}

			app.Output.KeyValue(pairs)
			return nil
		},
	}
}

func newEnvCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <site-id>",
		Short: "Create an environment",
		Long:  "Create a new environment for a site.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			name, _ := cmd.Flags().GetString("name")
			if name == "" {
				return &api.APIError{
					Message:  "--name is required",
					ExitCode: 3,
				}
			}

			phpVersion, _ := cmd.Flags().GetString("php-version")
			if phpVersion == "" {
				return &api.APIError{
					Message:  "--php-version is required",
					ExitCode: 3,
				}
			}

			reqBody := map[string]any{
				"name":        name,
				"php_version": phpVersion,
			}

			if cmd.Flags().Changed("custom-domain") {
				v, _ := cmd.Flags().GetString("custom-domain")
				if v != "" {
					reqBody["custom_domain"] = v
				} else {
					reqBody["custom_domain"] = nil
				}
			}

			if cmd.Flags().Changed("production") {
				v, _ := cmd.Flags().GetBool("production")
				reqBody["is_production"] = v
			}

			if cmd.Flags().Changed("tags") {
				tagsStr, _ := cmd.Flags().GetString("tags")
				if tagsStr != "" {
					reqBody["tags"] = strings.Split(tagsStr, ",")
				} else {
					reqBody["tags"] = []string{}
				}
			}

			path := sitesBasePath + "/" + args[0] + "/environments"
			resp, err := app.Client.Post(cmd.Context(), path, reqBody)
			if err != nil {
				return fmt.Errorf("failed to create environment: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to create environment: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to create environment: %w", err)
			}

			if app.Output.Format() == output.JSON {
				return app.Output.JSON(json.RawMessage(data))
			}

			var item map[string]any
			if err := json.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to create environment: %w", err)
			}

			tags := tagsFromItem(item)
			app.Output.KeyValue([]output.KeyValue{
				{Key: "ID", Value: getString(item, "id")},
				{Key: "Site ID", Value: getString(item, "vector_site_id")},
				{Key: "Name", Value: getString(item, "name")},
				{Key: "Production", Value: formatBool(getBool(item, "is_production"))},
				{Key: "Status", Value: getString(item, "status")},
				{Key: "PHP Version", Value: getString(item, "php_version")},
				{Key: "Tags", Value: formatTags(tags)},
				{Key: "Platform Domain", Value: formatString(getString(item, "platform_domain"))},
				{Key: "Custom Domain", Value: formatString(getString(item, "custom_domain"))},
			})
			return nil
		},
	}

	cmd.Flags().String("name", "", "Environment name (slug format, required)")
	cmd.Flags().String("php-version", "", "PHP version (required)")
	cmd.Flags().String("custom-domain", "", "Custom domain for the environment")
	cmd.Flags().Bool("production", false, "Mark as production environment")
	cmd.Flags().String("tags", "", "Comma-separated tags")

	return cmd
}

func newEnvUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <env-id>",
		Short: "Update an environment",
		Long:  "Update an environment's custom domain or tags. Domain changes trigger async infrastructure updates.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			customDomainSet := cmd.Flags().Changed("custom-domain")
			clearDomainSet := cmd.Flags().Changed("clear-custom-domain")

			if customDomainSet && clearDomainSet {
				return &api.APIError{
					Message:  "--custom-domain and --clear-custom-domain cannot be used together",
					ExitCode: 3,
				}
			}

			reqBody := map[string]any{}

			if customDomainSet {
				v, _ := cmd.Flags().GetString("custom-domain")
				reqBody["custom_domain"] = v
			}
			if clearDomainSet {
				reqBody["custom_domain"] = nil
			}

			if cmd.Flags().Changed("tags") {
				tagsStr, _ := cmd.Flags().GetString("tags")
				if tagsStr != "" {
					reqBody["tags"] = strings.Split(tagsStr, ",")
				} else {
					reqBody["tags"] = nil
				}
			}

			resp, err := app.Client.Put(cmd.Context(), envsBasePath+"/"+args[0], reqBody)
			if err != nil {
				return fmt.Errorf("failed to update environment: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			isDomainChange := resp.StatusCode == http.StatusAccepted

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to update environment: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to update environment: %w", err)
			}

			if app.Output.Format() == output.JSON {
				return app.Output.JSON(json.RawMessage(data))
			}

			var item map[string]any
			if err := json.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to update environment: %w", err)
			}

			tags := tagsFromItem(item)
			app.Output.KeyValue([]output.KeyValue{
				{Key: "ID", Value: getString(item, "id")},
				{Key: "Name", Value: getString(item, "name")},
				{Key: "Status", Value: getString(item, "status")},
				{Key: "Tags", Value: formatTags(tags)},
				{Key: "Custom Domain", Value: formatString(getString(item, "custom_domain"))},
				{Key: "DNS Target", Value: formatString(getString(item, "dns_target"))},
			})

			if isDomainChange {
				_, _ = fmt.Fprintln(cmd.OutOrStdout())
				output.PrintMessage(cmd.OutOrStdout(), "Domain change initiated. DNS records must be configured for the new domain.")
				pdc := getMap(item, "pending_domain_change")
				if pdc != nil {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Old Domain: %s\n", formatString(getString(pdc, "old_domain")))
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  New Domain: %s\n", formatString(getString(pdc, "new_domain")))
				}
			}

			return nil
		},
	}

	cmd.Flags().String("custom-domain", "", "Set custom domain")
	cmd.Flags().Bool("clear-custom-domain", false, "Remove custom domain")
	cmd.Flags().String("tags", "", "Comma-separated tags (empty string clears tags)")

	return cmd
}

func newEnvDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <env-id>",
		Short: "Delete an environment",
		Long:  "Initiate deletion of an environment. This operation is irreversible.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			force, _ := cmd.Flags().GetBool("force")
			if !force {
				if !confirmAction(cmd, fmt.Sprintf("Are you sure you want to delete environment %s?", args[0])) {
					output.PrintMessage(cmd.OutOrStdout(), "Aborted.")
					return nil
				}
			}

			resp, err := app.Client.Delete(cmd.Context(), envsBasePath+"/"+args[0])
			if err != nil {
				return fmt.Errorf("failed to delete environment: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to delete environment: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to delete environment: %w", err)
			}

			if app.Output.Format() == output.JSON {
				return app.Output.JSON(json.RawMessage(data))
			}

			var item map[string]any
			if err := json.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to delete environment: %w", err)
			}

			app.Output.Message(fmt.Sprintf("Environment %s deletion initiated.", getString(item, "id")))
			return nil
		},
	}

	cmd.Flags().Bool("force", false, "Skip confirmation prompt")

	return cmd
}
