// SPDX-FileCopyrightText: William Moreno Reyes CP | MBA
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"path/filepath"

	"github.com/admiral-project/admiral/admiralctl/internal/output"
	"github.com/admiral-project/admiral/admirald/pkg/admiral"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

var appsCmd = &cobra.Command{
	Use:   "apps",
	Short: "Manage and apply application definition templates",
}

func init() {
	rootCmd.AddCommand(appsCmd)
	appsCmd.AddCommand(appsListCmd)
	appsCmd.AddCommand(appsShowCmd)
	appsCmd.AddCommand(appsApplyCmd)
	appsCmd.AddCommand(appsValidateCmd)
	appsCmd.AddCommand(appsActivateCmd)
	appsCmd.AddCommand(appsDeactivateCmd)
}

var appsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List application definitions",
	RunE:  runAppsList,
}

var appsShowCmd = &cobra.Command{
	Use:   "show <app_name>",
	Short: "Show details for an application definition",
	Args:  cobra.ExactArgs(1),
	RunE:  runAppsShow,
}

var appsApplyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply an application definition from a YAML file",
	RunE:  runAppsApply,
}

var appsValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate an application definition YAML file",
	RunE:  runAppsValidate,
}

var appsActivateCmd = &cobra.Command{
	Use:   "activate",
	Short: "Activate an application definition",
	RunE:  runAppsActivate,
}

var appsDeactivateCmd = &cobra.Command{
	Use:   "deactivate",
	Short: "Deactivate an application definition",
	RunE:  runAppsDeactivate,
}

func init() {
	appsListCmd.Flags().String("output", "table", "Output format: table or json")
	appsApplyCmd.Flags().StringP("file", "f", "", "Path to app definition YAML file (required)")
	_ = appsApplyCmd.MarkFlagRequired("file")
	appsValidateCmd.Flags().StringP("file", "f", "", "Path to app definition YAML file (required)")
	_ = appsValidateCmd.MarkFlagRequired("file")
	appsActivateCmd.Flags().String("name", "", "Application definition name (required)")
	_ = appsActivateCmd.MarkFlagRequired("name")
	appsDeactivateCmd.Flags().String("name", "", "Application definition name (required)")
	_ = appsDeactivateCmd.MarkFlagRequired("name")
	appsDeactivateCmd.Flags().Bool("force", false, "Skip confirmation prompt")
}

func runAppsList(cmd *cobra.Command, _ []string) error {
	apps, err := clientOrNil().GetApps()
	if err != nil {
		return err
	}

	outputFlag, _ := cmd.Flags().GetString("output")
	if outputFlag == "json" {
		output.PrintJSON(apps)
		return nil
	}

	headers := []string{"APP NAME", "DISPLAY NAME", "STATUS", "CREATED AT"}
	var rows [][]string
	for _, a := range apps {
		rows = append(rows, []string{
			fmt.Sprintf("%v", a["name"]),
			fmt.Sprintf("%v", a["display_name"]),
			fmt.Sprintf("%v", a["status"]),
			fmt.Sprintf("%v", a["created_at"]),
		})
	}
	output.PrintTable(headers, rows)
	return nil
}

func runAppsShow(cmd *cobra.Command, args []string) error {
	app, err := clientOrNil().GetApp(args[0])
	if err != nil {
		return err
	}
	output.PrintJSON(app)
	return nil
}

func runAppsApply(cmd *cobra.Command, _ []string) error {
	file, _ := cmd.Flags().GetString("file")
	data, payload, err := readAndValidateAppFile(cmd, file)
	if err != nil {
		return err
	}
	_ = payload

	name, err := clientOrNil().ApplyApp(string(data))
	if err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Application definition %q applied successfully!\n", name)
	return nil
}

func runAppsValidate(cmd *cobra.Command, _ []string) error {
	file, _ := cmd.Flags().GetString("file")
	_, payload, err := readAndValidateAppFile(cmd, file)
	if err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "YAML Validation: PASSED\nApp: %s\nServices: %d\nTiers: %d\n", payload.Name, len(payload.Services), len(payload.Tiers))
	return nil
}

func readAndValidateAppFile(cmd *cobra.Command, file string) ([]byte, admiral.AppDefinitionPayload, error) {
	var payload admiral.AppDefinitionPayload
	path, err := filepath.Abs(file)
	if err != nil {
		return nil, payload, fmt.Errorf("resolve file path %s: %w", file, err)
	}
	path, err = sanitizeInputFilePath(path)
	if err != nil {
		return nil, payload, fmt.Errorf("validate file path %s: %w", file, err)
	}
	data, err := readInputFile(path)
	if err != nil {
		return nil, payload, fmt.Errorf("read file %s: %w", file, err)
	}
	if err := yaml.Unmarshal(data, &payload); err != nil {
		return nil, payload, fmt.Errorf("YAML syntax validation failed: %w", err)
	}
	if err := admiral.ValidateAppDefinition(payload); err != nil {
		return nil, payload, fmt.Errorf("application definition validation failed: %w", err)
	}
	return data, payload, nil
}

func runAppsActivate(cmd *cobra.Command, _ []string) error {
	name, _ := cmd.Flags().GetString("name")
	if err := clientOrNil().UpdateAppStatus(name, "active"); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Application definition %q is now active.\n", name)
	return nil
}

func runAppsDeactivate(cmd *cobra.Command, _ []string) error {
	name, _ := cmd.Flags().GetString("name")
	if !confirmDestructive(cmd, "deactivate", fmt.Sprintf("app definition %q", name)) {
		fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
		return nil
	}
	if err := clientOrNil().UpdateAppStatus(name, "inactive"); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Application definition %q is now inactive.\n", name)
	return nil
}
