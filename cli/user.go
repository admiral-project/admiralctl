// SPDX-FileCopyrightText: William Moreno Reyes CP | MBA
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"

	"github.com/admiral-project/admiral/admiralctl/internal/output"
	"github.com/spf13/cobra"
)

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "Manage admin users",
}

func init() {
	rootCmd.AddCommand(userCmd)
	userCmd.AddCommand(userListCmd)
	userCmd.AddCommand(userCreateCmd)
	userCmd.AddCommand(userSetPasswordCmd)
}

var userListCmd = &cobra.Command{
	Use:   "list",
	Short: "List admin users",
	RunE:  runUserList,
}

var userCreateCmd = &cobra.Command{
	Use:   "create <username>",
	Short: "Create a new admin user",
	Args:  cobra.ExactArgs(1),
	RunE:  runUserCreate,
}

var userSetPasswordCmd = &cobra.Command{
	Use:   "set-password <username>",
	Short: "Set a new password for an admin user",
	Args:  cobra.ExactArgs(1),
	RunE:  runUserSetPassword,
}

func init() {
	userListCmd.Flags().String("output", "table", "Output format: table or json")
	userCreateCmd.Flags().String("type", "admin", "Role: superadmin, admin, platform, support, audit")
}

func runUserList(cmd *cobra.Command, _ []string) error {
	users, err := clientOrNil().ListUsers()
	if err != nil {
		return err
	}

	outputFlag, _ := cmd.Flags().GetString("output")
	if outputFlag == "json" {
		output.PrintJSON(users)
		return nil
	}

	headers := []string{"USERNAME", "ROLE", "CREATED AT"}
	var rows [][]string
	for _, u := range users {
		rows = append(rows, []string{
			fmt.Sprintf("%v", u["username"]),
			fmt.Sprintf("%v", u["role"]),
			fmt.Sprintf("%v", u["created_at"]),
		})
	}
	output.PrintTable(headers, rows)
	return nil
}

func runUserCreate(cmd *cobra.Command, args []string) error {
	userType, _ := cmd.Flags().GetString("type")
	username := args[0]

	fmt.Fprintf(cmd.OutOrStdout(), "Enter password for user %q: ", username)
	password, err := readPassword()
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "\nFailed to read password: %v\n", err)
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout())

	res, err := clientOrNil().CreateUser(username, password, userType)
	if err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "User %q created successfully with role %q\n", res["username"], res["role"])
	return nil
}

func runUserSetPassword(cmd *cobra.Command, args []string) error {
	username := args[0]

	fmt.Fprintf(cmd.OutOrStdout(), "Enter new password for user %q: ", username)
	password, err := readPassword()
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "\nFailed to read password: %v\n", err)
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout())

	if err := clientOrNil().SetPassword(username, password); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Password for user %q updated successfully.\n", username)
	return nil
}
