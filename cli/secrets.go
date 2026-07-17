package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var secretsCmd = &cobra.Command{
	Use:   "secrets",
	Short: "Manage encrypted platform secrets",
}

var secretsRotateCmd = &cobra.Command{
	Use:   "rotate",
	Short: "Re-encrypt stored secrets with the current Admiral key",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		result, err := clientOrNil().RotateSecrets()
		if err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Secret rotation complete: migrated=%d already_current=%d total=%d\n", result["migrated"], result["already_current"], result["total"])
		return nil
	},
}

func init() {
	rootCmd.AddCommand(secretsCmd)
	secretsCmd.AddCommand(secretsRotateCmd)
}
