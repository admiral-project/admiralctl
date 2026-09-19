// SPDX-FileCopyrightText: William Moreno Reyes CP | MBA
// SPDX-License-Identifier: Apache-2.0
package cli

import (
	"fmt"
	"github.com/spf13/cobra"
)

var tokensCmd = &cobra.Command{Use: "tokens", Short: "Manage your operator tokens"}
var tokensListCmd = &cobra.Command{Use: "list", RunE: func(cmd *cobra.Command, _ []string) error {
	items, err := clientOrNil().ListOperatorTokens()
	if err != nil {
		return err
	}
	for _, v := range items {
		fmt.Fprintf(cmd.OutOrStdout(), "%v\t%v\t%v\t%v\n", v["id"], v["label"], v["scope"], v["prefix"])
	}
	return nil
}}
var tokensCreateCmd = &cobra.Command{Use: "create <label>", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
	scope, _ := cmd.Flags().GetString("scope")
	expires, _ := cmd.Flags().GetString("expires-at")
	v, err := clientOrNil().CreateOperatorToken(args[0], scope, expires)
	if err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Token (shown once): %v\n", v["token"])
	return nil
}}
var tokensRevokeCmd = &cobra.Command{Use: "revoke <id>", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error { return clientOrNil().RevokeOperatorToken(args[0]) }}

func init() {
	rootCmd.AddCommand(tokensCmd)
	tokensCmd.AddCommand(tokensListCmd, tokensCreateCmd, tokensRevokeCmd)
	tokensCreateCmd.Flags().String("scope", "admin", "Token scope: read, write, or admin")
	tokensCreateCmd.Flags().String("expires-at", "", "Optional RFC3339 expiration")
}
