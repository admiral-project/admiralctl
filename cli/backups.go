// SPDX-FileCopyrightText: William Moreno Reyes CP | MBA
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"strings"

	"github.com/admiral-project/admiral/admiralctl/internal/output"
	"github.com/admiral-project/admiral/admirald/pkg/admiral"
	"github.com/spf13/cobra"
)

var backupsCmd = &cobra.Command{
	Use:   "backups",
	Short: "List, show, restore, storage config, delete, and prune backups",
}

func init() {
	rootCmd.AddCommand(backupsCmd)
	backupsCmd.AddCommand(backupsListCmd)
	backupsCmd.AddCommand(backupsShowCmd)
	backupsCmd.AddCommand(backupsRestoreCmd)
	backupsCmd.AddCommand(backupsStorageCmd)
	backupsCmd.AddCommand(backupsDeleteCmd)
	backupsCmd.AddCommand(backupsPruneCmd)
}

var backupsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List backups",
	RunE:  runBackupsList,
}

var backupsShowCmd = &cobra.Command{
	Use:   "show <backup_id>",
	Short: "Show details for a specific backup",
	Args:  cobra.ExactArgs(1),
	RunE:  runBackupsShow,
}

var backupsRestoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore a backup to an instance",
	RunE:  runBackupsRestore,
}

var backupsStorageCmd = &cobra.Command{
	Use:   "storage",
	Short: "Manage backup storage configuration",
}

var backupsStorageGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Show current backup storage configuration",
	RunE:  runBackupsStorageGet,
}

var backupsStorageSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Update backup storage configuration",
	RunE:  runBackupsStorageSet,
}

var backupsStorageTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Test storage connectivity",
	RunE:  runBackupsStorageTest,
}

var backupsDeleteCmd = &cobra.Command{
	Use:   "delete <backup_id>",
	Short: "Delete a backup",
	Args:  cobra.ExactArgs(1),
	RunE:  runBackupsDelete,
}

var backupsPruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Prune old succeeded backups",
	RunE:  runBackupsPrune,
}

func init() {
	backupsListCmd.Flags().String("output", "table", "Output format: table or json")

	backupsRestoreCmd.Flags().String("backup-id", "", "Backup ID to restore (required)")
	_ = backupsRestoreCmd.MarkFlagRequired("backup-id")
	backupsRestoreCmd.Flags().String("instance-id", "", "Target instance ID (required)")
	_ = backupsRestoreCmd.MarkFlagRequired("instance-id")
	backupsRestoreCmd.Flags().String("service", "", "Service name matching the backup source (required)")
	_ = backupsRestoreCmd.MarkFlagRequired("service")
	backupsRestoreCmd.Flags().String("target-node", "", "Optional target node ID")
	backupsRestoreCmd.Flags().String("source-type", "", "Optional source type override")
	backupsRestoreCmd.Flags().String("source-uri", "", "Optional source URI override")
	backupsRestoreCmd.Flags().Bool("verify-checksum", true, "Verify checksum during restore")

	backupsStorageGetCmd.Flags().String("output", "table", "Output format: table or json")
	backupsStorageSetCmd.Flags().String("backend", "s3", "Storage backend: s3 or local")
	backupsStorageSetCmd.Flags().String("endpoint", "", "S3-compatible endpoint URL")
	backupsStorageSetCmd.Flags().String("region", "us-east-1", "S3 region")
	backupsStorageSetCmd.Flags().String("bucket", "", "S3 bucket name")
	backupsStorageSetCmd.Flags().String("prefix", "", "S3 key prefix")
	backupsStorageSetCmd.Flags().String("access-key-env", "ADMIRAL_AWS_ACCESS_KEY_ID", "Env var name for access key")
	backupsStorageSetCmd.Flags().String("secret-key-env", "ADMIRAL_AWS_SECRET_ACCESS_KEY", "Env var name for secret key")
}

func init() {
	backupsStorageCmd.AddCommand(backupsStorageGetCmd)
	backupsStorageCmd.AddCommand(backupsStorageSetCmd)
	backupsStorageCmd.AddCommand(backupsStorageTestCmd)
}

func runBackupsList(cmd *cobra.Command, _ []string) error {
	backups, err := clientOrNil().GetBackups()
	if err != nil {
		return err
	}

	outputFlag, _ := cmd.Flags().GetString("output")
	if outputFlag == "json" {
		output.PrintJSON(backups)
		return nil
	}

	headers := []string{"BACKUP ID", "INSTANCE ID", "TYPE", "STORAGE", "STATUS", "CREATED AT"}
	var rows [][]string
	for _, b := range backups {
		rows = append(rows, []string{
			fmt.Sprintf("%v", b["id"]),
			fmt.Sprintf("%v", b["instance_id"]),
			fmt.Sprintf("%v", b["backup_type"]),
			fmt.Sprintf("%v", b["storage_backend"]),
			fmt.Sprintf("%v", b["status"]),
			fmt.Sprintf("%v", b["created_at"]),
		})
	}
	output.PrintTable(headers, rows)
	return nil
}

func runBackupsShow(cmd *cobra.Command, args []string) error {
	backup, err := clientOrNil().GetBackup(args[0])
	if err != nil {
		return err
	}
	output.PrintJSON(backup)
	return nil
}

func runBackupsRestore(cmd *cobra.Command, _ []string) error {
	backupID, _ := cmd.Flags().GetString("backup-id")
	instanceID, _ := cmd.Flags().GetString("instance-id")
	service, _ := cmd.Flags().GetString("service")
	targetNode, _ := cmd.Flags().GetString("target-node")
	sourceType, _ := cmd.Flags().GetString("source-type")
	sourceURI, _ := cmd.Flags().GetString("source-uri")
	verifyChecksum, _ := cmd.Flags().GetBool("verify-checksum")

	req := admiral.RestoreBackupRequest{
		BackupID:       backupID,
		TargetAppID:    instanceID,
		Service:        service,
		TargetNodeID:   targetNode,
		RestoreMode:    "replace",
		VerifyChecksum: verifyChecksum,
	}
	if sourceType != "" || sourceURI != "" {
		req.Source.Type = sourceType
		req.Source.URI = sourceURI
	}

	res, err := clientOrNil().RestoreBackup(req)
	if err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Restore queued successfully!\nOperation ID: %s\n", res.OperationID)
	return nil
}

func runBackupsStorageGet(cmd *cobra.Command, _ []string) error {
	cfg, err := clientOrNil().GetBackupStorageConfig()
	if err != nil {
		return err
	}

	outputFlag, _ := cmd.Flags().GetString("output")
	if outputFlag == "json" {
		output.PrintJSON(cfg)
		return nil
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Backend:  %s\n", cfg.Backend)
	fmt.Fprintf(cmd.OutOrStdout(), "Enabled:  %v\n", cfg.Enabled)
	fmt.Fprintf(cmd.OutOrStdout(), "Endpoint: %s\n", cfg.Endpoint)
	fmt.Fprintf(cmd.OutOrStdout(), "Region:   %s\n", cfg.Region)
	fmt.Fprintf(cmd.OutOrStdout(), "Bucket:   %s\n", cfg.Bucket)
	fmt.Fprintf(cmd.OutOrStdout(), "Prefix:   %s\n", cfg.Prefix)
	return nil
}

func runBackupsStorageSet(cmd *cobra.Command, _ []string) error {
	backend, _ := cmd.Flags().GetString("backend")
	endpoint, _ := cmd.Flags().GetString("endpoint")
	region, _ := cmd.Flags().GetString("region")
	bucket, _ := cmd.Flags().GetString("bucket")
	prefix, _ := cmd.Flags().GetString("prefix")
	accessKeyEnv, _ := cmd.Flags().GetString("access-key-env")
	secretKeyEnv, _ := cmd.Flags().GetString("secret-key-env")

	cfg := admiral.BackupStorageConfig{
		Backend:      backend,
		Enabled:      true,
		Endpoint:     endpoint,
		Region:       region,
		Bucket:       bucket,
		Prefix:       prefix,
		AccessKeyEnv: accessKeyEnv,
		SecretKeyEnv: secretKeyEnv,
	}
	if err := clientOrNil().SetBackupStorageConfig(cfg); err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), "Backup storage configuration updated.")
	return nil
}

func runBackupsStorageTest(cmd *cobra.Command, _ []string) error {
	if err := clientOrNil().TestBackupStorageConfig(); err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), "Backup storage test passed.")
	return nil
}

func runBackupsDelete(cmd *cobra.Command, args []string) error {
	fmt.Fprintf(cmd.OutOrStdout(), "Are you sure you want to delete backup %s? (y/N): ", args[0])
	var confirm string
	_, _ = fmt.Fscanln(cmd.InOrStdin(), &confirm)
	if strings.ToLower(strings.TrimSpace(confirm)) != "y" {
		fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
		return nil
	}
	if err := clientOrNil().DeleteBackup(args[0]); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Backup %s deleted.\n", args[0])
	return nil
}

func runBackupsPrune(cmd *cobra.Command, _ []string) error {
	fmt.Fprint(cmd.OutOrStdout(), "Are you sure you want to prune old succeeded backups? (y/N): ")
	var confirm string
	_, _ = fmt.Fscanln(cmd.InOrStdin(), &confirm)
	if strings.ToLower(strings.TrimSpace(confirm)) != "y" {
		fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
		return nil
	}
	if err := clientOrNil().PruneBackups(); err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), "Backups pruned successfully.")
	return nil
}
