package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/rilldata/rill/cli/cmd/admin"
	"github.com/rilldata/rill/cli/cmd/auth"
	"github.com/rilldata/rill/cli/cmd/build"
	"github.com/rilldata/rill/cli/cmd/deploy"
	"github.com/rilldata/rill/cli/cmd/docs"
	"github.com/rilldata/rill/cli/cmd/env"
	"github.com/rilldata/rill/cli/cmd/initialize"
	"github.com/rilldata/rill/cli/cmd/org"
	"github.com/rilldata/rill/cli/cmd/project"
	"github.com/rilldata/rill/cli/cmd/runtime"
	"github.com/rilldata/rill/cli/cmd/source"
	"github.com/rilldata/rill/cli/cmd/start"
	"github.com/rilldata/rill/cli/cmd/user"
	versioncmd "github.com/rilldata/rill/cli/cmd/version"
	"github.com/rilldata/rill/cli/pkg/config"
	"github.com/rilldata/rill/cli/pkg/dotrill"
	"github.com/rilldata/rill/cli/pkg/update"
	"github.com/spf13/cobra"
)

func init() {
	cobra.EnableCommandSorting = false
}

// defaultAdminURL is the default admin server URL.
// Users can override it with the "--api-url" flag or by setting "api_url" in ~/.rill/config.yaml.
const defaultAdminURL = "https://admin.rilldata.io"

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "rill <command>",
	Short: "Rill CLI",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(ctx context.Context, ver config.Version) {
	err := runCmd(ctx, ver)
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
		os.Exit(1)
	}
}

func runCmd(ctx context.Context, ver config.Version) error {
	// Cobra config
	rootCmd.Version = ver.String()
	rootCmd.PersistentFlags().BoolP("help", "h", false, "Print usage") // Overrides message for help
	rootCmd.Flags().BoolP("version", "v", false, "Show rill version")  // Adds option to get version by passing --version or -v

	// Build CLI config
	cfg := &config.Config{
		Version: ver,
	}

	// Check version
	err := update.CheckVersion(ctx, cfg.Version.Number)
	if err != nil {
		return err
	}
	// Load admin token from .rill (may later be overridden by flag --api-token)
	token, err := dotrill.GetAccessToken()
	if err != nil {
		return fmt.Errorf("could not parse access token from ~/.rill: %w", err)
	}
	cfg.AdminTokenDefault = token

	// Load default org from .rill
	defaultOrg, err := dotrill.GetDefaultOrg()
	if err != nil {
		return fmt.Errorf("could not parse default org from ~/.rill: %w", err)
	}
	cfg.Org = defaultOrg

	// Load admin URL from .rill (override with --api-url)
	url, err := dotrill.GetDefaultAdminURL()
	if err != nil {
		return fmt.Errorf("could not parse default api URL from ~/.rill: %w", err)
	}
	if url == "" {
		url = defaultAdminURL
	}
	cfg.AdminURL = url

	// Add sub-commands
	rootCmd.AddCommand(initialize.InitCmd(cfg))
	rootCmd.AddCommand(start.StartCmd(cfg))
	rootCmd.AddCommand(build.BuildCmd(cfg))
	rootCmd.AddCommand(source.SourceCmd(cfg))
	rootCmd.AddCommand(admin.AdminCmd(cfg))
	rootCmd.AddCommand(runtime.RuntimeCmd(cfg))
	rootCmd.AddCommand(docs.DocsCmd())
	rootCmd.AddCommand(completionCmd)
	rootCmd.AddCommand(versioncmd.VersionCmd())

	// Set prompt for missing required parameters in config
	rootCmd.PersistentFlags().BoolVar(&cfg.Interactive, "interactive", true, "Prompt for missing required parameters")

	// Add sub-commands for admin
	// (This allows us to add persistent flags that apply only to the admin-related commands.)
	adminCmds := []*cobra.Command{
		org.OrgCmd(cfg),
		project.ProjectCmd(cfg),
		deploy.DeployCmd(cfg),
		user.UserCmd(cfg),
		env.EnvCmd(cfg),
		auth.LoginCmd(cfg),
		auth.LogoutCmd(cfg),
	}
	for _, cmd := range adminCmds {
		cmd.PersistentFlags().StringVar(&cfg.AdminURL, "api-url", cfg.AdminURL, "Base URL for the admin API")
		cmd.PersistentFlags().StringVar(&cfg.AdminTokenOverride, "api-token", "", "Token for authenticating with the admin API")
		rootCmd.AddCommand(cmd)
	}

	return rootCmd.ExecuteContext(ctx)
}
