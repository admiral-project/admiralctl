// SPDX-FileCopyrightText: William Moreno Reyes CP | MBA
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/admiral-project/admiral/admiralctl/internal/config"
	"github.com/admiral-project/admiral/admirald/pkg/admiral/tlsconfig"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize or update local CLI settings",
	Long: `Initialize admiralctl with the control plane endpoint and authentication token.

The token can be provided via --token, the ADMIRAL_ADMIN_TOKEN environment
variable, or interactively. Prefer environment variables over --token to avoid
exposing the secret in the process list.`,
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().String("server", "", "Control plane server endpoint URL")
	initCmd.Flags().String("token", "", "Authentication token (visible in process list; prefer ADMIRAL_ADMIN_TOKEN)")
	initCmd.Flags().String("ca-cert", "", "CA certificate file for admirald HTTPS validation")
	initCmd.Flags().Bool("generate-signing-key", false, "Generate Ed25519 signing key pair for task verification")
}

func runInit(cmd *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	generateSigningKey, _ := cmd.Flags().GetBool("generate-signing-key")
	if generateSigningKey {
		pub, priv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return fmt.Errorf("generate signing key: %w", err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), "private_key:"+hex.EncodeToString(priv.Seed()))
		fmt.Fprintln(cmd.OutOrStdout(), "public_key:"+hex.EncodeToString(pub))
		return nil
	}

	serverURL, _ := cmd.Flags().GetString("server")
	if serverURL == "" {
		serverURL = cfg.ServerURL
	}
	token, _ := cmd.Flags().GetString("token")
	caCert, _ := cmd.Flags().GetString("ca-cert")
	if caCert == "" {
		caCert = cfg.CACertFile
	}

	resolved := resolveToken(cmd, token, cfg.Token)
	if resolved == "" {
		return fmt.Errorf("authentication token is required. Use --token or export ADMIRAL_ADMIN_TOKEN")
	}
	if err := tlsconfig.ValidateURLScheme(serverURL, "https"); err != nil {
		return err
	}

	cfg.ServerURL = serverURL
	cfg.Token = resolved
	cfg.CACertFile = caCert

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Configuration initialized successfully!\nSaved to: %s\nTarget server: %s\n", config.GetConfigPath(), cfg.ServerURL)
	return nil
}
