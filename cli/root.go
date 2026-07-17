// SPDX-FileCopyrightText: William Moreno Reyes CP | MBA
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"

	"github.com/admiral-project/admiral/admiralctl/internal/client"
	"github.com/admiral-project/admiral/admiralctl/internal/config"
	"github.com/admiral-project/admiral/admiralctl/internal/version"
	"github.com/spf13/cobra"
)

var (
	cfg           *config.Config
	currentClient *client.Client

	serverURLFlag string
	caCertFlag    string
	operatorFlag  string
)

var rootCmd = &cobra.Command{
	Use:   "admiralctl",
	Short: "Admiral official CLI",
	Long: `admiralctl is the official command-line interface for the Admiral PaaS platform.

It communicates with the admirald control plane to manage nodes, applications,
instances, backups, routes, secrets, and operations.`,
	PersistentPreRunE: loadClient,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&serverURLFlag, "server", "", "Control plane server endpoint URL")
	rootCmd.PersistentFlags().StringVar(&caCertFlag, "ca-cert", "", "CA certificate file for admirald HTTPS validation")
	rootCmd.PersistentFlags().StringVar(&operatorFlag, "operator", "", "Operator name for audit logs")

	rootCmd.Version = version.Version
	rootCmd.SetVersionTemplate("admiralctl {{.Version}}\n")

	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print the CLI version",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, _ []string) {
			cmd.Printf("admiralctl %s\n", version.Version)
		},
	})
}

// Execute runs the root command.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

// loadClient loads the CLI configuration, applies flag overrides, and creates
// the API client. It is skipped for commands that do not need API access.
func loadClient(cmd *cobra.Command, _ []string) error {
	if err := validateOutputFlag(cmd); err != nil {
		return err
	}
	if skipClientLoad(cmd) {
		return nil
	}

	var err error
	cfg, err = config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if serverURLFlag != "" {
		cfg.ServerURL = serverURLFlag
	}
	if caCertFlag != "" {
		cfg.CACertFile = caCertFlag
	}
	if operatorFlag != "" {
		cfg.Operator = operatorFlag
	}

	if cfg.Token == "" {
		return fmt.Errorf("no authentication token configured. Run 'admiralctl init --token <token>' or set ADMIRAL_ADMIN_TOKEN")
	}

	currentClient, err = client.New(cfg.ServerURL, cfg.Token, cfg.CACertFile, client.WithOperator(cfg.Operator))
	if err != nil {
		return fmt.Errorf("create TLS client: %w", err)
	}
	return nil
}

// validateOutputFlag rejects typos instead of silently falling back to table
// output. Commands without an --output flag are unaffected.
func validateOutputFlag(cmd *cobra.Command) error {
	flag := cmd.Flags().Lookup("output")
	if flag == nil {
		return nil
	}
	value, err := cmd.Flags().GetString("output")
	if err != nil {
		return fmt.Errorf("read --output: %w", err)
	}
	if value != "table" && value != "json" {
		return fmt.Errorf("invalid --output value %q: must be table or json", value)
	}
	return nil
}

// skipClientLoad returns true for commands that do not need an API client.
func skipClientLoad(cmd *cobra.Command) bool {
	switch cmd.Name() {
	case "init", "version":
		return true
	default:
		return false
	}
}

// clientOrNil returns the current API client. It panics if called before
// loadClient has run successfully. This should never happen because Cobra
// runs PersistentPreRunE before any command's RunE.
func clientOrNil() *client.Client {
	if currentClient == nil {
		cobra.CheckErr(fmt.Errorf("API client not initialized"))
	}
	return currentClient
}

// SetClient sets the API client for the CLI. Used for testing.
func SetClient(c *client.Client) {
	currentClient = c
}
