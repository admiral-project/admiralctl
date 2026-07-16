# admiralctl — Command Reference

## SYNOPSIS

`admiralctl <command> [subcommand] [flags]`

## GLOBAL OPTIONS

| Flag | Type | Description |
|------|------|-------------|
| `--server` | string | Control plane server endpoint URL |
| `--token` | string | Authentication token (prefer `ADMIRAL_ADMIN_TOKEN` env var) |
| `--ca-cert` | string | CA certificate file for TLS validation |
| `--operator` | string | Operator name for audit logs |

## COMMANDS

### init

Initialize or update local CLI configuration.

| Flag | Type | Description |
|------|------|-------------|
| `--server` | string | admirald API endpoint |
| `--token` | string | Shared authentication token |
| `--ca-cert` | string | CA certificate for TLS |
| `--generate-signing-key` | bool | Generate Ed25519 key pair; save the private seed under `~/.config/admiralctl/signing-key.seed` with mode 0600 |

### status

Check admirald health.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--output` | string | `table` | Output format: `table` or `json` |

### nodes list

List registered worker nodes.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--output` | string | `table` | Output format: `table` or `json` |

### nodes show NODE_ID

Show details of a specific node.

### nodes register

Register a new worker node.

| Flag | Type | Required | Description |
|------|------|----------|-------------|
| `--id` | string | yes | Unique node ID |
| `--hostname` | string | yes | Node hostname |
| `--ip` | string | yes | Node IP address |
| `--wireguard-ip` | string | no | WireGuard VPN IP address |
| `--role` | string | no | Node role: `worker`, `admin`, `portal` |
| `--public-ip` | string | no | Public IP address |
| `--os` | string | no | Operating system (default: `linux`) |
| `--podman` | string | no | Podman version (default: `4.9.0`) |
| `--token` | string | no | Pre-generated node token; prefer `ADMIRAL_NODE_TOKEN` or the secure prompt |

### nodes enable NODE_ID

Activate a node for provisioning.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--force` | bool | `false` | Skip confirmation prompt |

### nodes disable NODE_ID

Deactivate a node.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--force` | bool | `false` | Skip confirmation prompt |

### nodes remove NODE_ID

Remove a registered node from the platform.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--force` | bool | `false` | Remove even with active instances |

### nodes ready

Check if a worker node is ready and reachable.

| Flag | Type | Required | Description |
|------|------|----------|-------------|
| `--node` | string | yes | Node ID |

### apps list

List registered application definitions.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--output` | string | `table` | Output format: `table` or `json` |

### apps show APP_NAME

Show details of a specific application definition.

### apps apply

Register or update an application definition from a YAML file.

| Flag | Type | Required | Description |
|------|------|----------|-------------|
| `-f`, `--file` | string | yes | Path to YAML file |

### apps validate

Validate an application definition YAML file locally.

| Flag | Type | Required | Description |
|------|------|----------|-------------|
| `-f`, `--file` | string | yes | Path to YAML file |

### apps activate

Mark an application definition as provisionable.

| Flag | Type | Required | Description |
|------|------|----------|-------------|
| `--name` | string | yes | Application definition name |

### apps deactivate

Mark an application definition as non-provisionable.

| Flag | Type | Required | Description |
|------|------|----------|-------------|
| `--name` | string | yes | Application definition name |
| `--force` | bool | no | Skip confirmation prompt |

### instances list

List customer application instances.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--output` | string | `table` | Output format: `table` or `json` |

### instances show INSTANCE_ID

Show details of a specific instance.

### instances inspect INSTANCE_ID

Inspect containers and volumes of a running instance.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--result` | bool | `false` | Show the last result instead of triggering a new inspect |

### instances credentials INSTANCE_ID

Show exposed credentials and setup notices for an instance.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--output` | string | `table` | Output format: `table` or `json` |

### instances provision

Provision a new customer application instance.

| Flag | Type | Required | Description |
|------|------|----------|-------------|
| `--app` | string | yes | Application definition name |
| `--tier` | string | yes | Tier name |
| `--customer` | string | yes | Customer ID |
| `--node` | string | no | Explicit node ID to target |
| `--logical-instance-id` | string | no | Preserve the logical instance identity |
| `--output` | string | no | Output format: `table` or `json` |
| `--wait` | bool | no | Wait until the operation reaches a terminal state |
| `--quiet` | bool | no | Suppress credential output |

### instances migrate INSTANCE_ID

Start an offline migration to another worker node.

| Flag | Type | Required | Description |
|------|------|----------|-------------|
| `--target-node` | string | yes | Target worker node ID |
| `--wait` | bool | no | Wait until the migration operation completes |

### instances pause|resume|reactivate|start|stop INSTANCE_ID

Change runtime state of an instance.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--wait` | bool | `false` | Wait until the operation reaches a terminal state |
| `--force` | bool | `false` | Skip confirmation prompt (for stop) |

### instances restart INSTANCE_ID

Restart an instance (stop then start).

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--force` | bool | `false` | Skip confirmation prompt |

### instances backup INSTANCE_ID

Trigger a backup for a specific service.

| Flag | Type | Required | Description |
|------|------|----------|-------------|
| `--service` | string | yes | Service name declared in the app definition |
| `--wait` | bool | no | Wait until the operation reaches a terminal state |

### instances deprovision INSTANCE_ID

Deprovision (delete) an instance.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--wait` | bool | `false` | Wait until the operation reaches a terminal state |
| `--force` | bool | `false` | Skip confirmation prompt |

### instances destroy INSTANCE_ID

Alias for deprovision.

### instances resize INSTANCE_ID

Change the tier of an instance.

| Flag | Type | Required | Description |
|------|------|----------|-------------|
| `--tier` | string | yes | Target tier name |
| `--wait` | bool | no | Wait until the operation reaches a terminal state |

### operations list

List platform operations.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--output` | string | `table` | Output format: `table` or `json` |

### operations show OPERATION_ID

Show details of a specific operation.

### operations retry OPERATION_ID

Re-queue a failed operation for execution.

### backups list

List backups.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--output` | string | `table` | Output format: `table` or `json` |

### backups show BACKUP_ID

Show details of a specific backup.

### backups restore

Restore a backup to an instance.

| Flag | Type | Required | Description |
|------|------|----------|-------------|
| `--backup-id` | string | yes | Backup ID |
| `--instance-id` | string | yes | Target instance ID |
| `--service` | string | yes | Service name matching the backup source |
| `--target-node` | string | no | Target node ID |
| `--source-type` | string | no | Source type override |
| `--source-uri` | string | no | Source URI override |
| `--verify-checksum` | bool | no | Verify checksum (default: `true`) |

### backups storage get

Show current backup storage configuration.

### backups storage set

Update backup storage configuration.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--backend` | string | `s3` | Storage backend: `s3` or `local` |
| `--endpoint` | string | | S3-compatible endpoint URL |
| `--region` | string | `us-east-1` | S3 region |
| `--bucket` | string | | S3 bucket name |
| `--prefix` | string | | S3 key prefix |
| `--access-key-env` | string | `ADMIRAL_AWS_ACCESS_KEY_ID` | Env var name for access key |
| `--secret-key-env` | string | `ADMIRAL_AWS_SECRET_ACCESS_KEY` | Env var name for secret key |

### backups storage test

Test storage connectivity.

### backups delete BACKUP_ID

Delete a backup.

### backups prune

Prune old succeeded backups.

### routes list

List public routes.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--output` | string | `table` | Output format: `table` or `json` |

### routes show HOSTNAME

Show details of a specific route.

### routes sync

Trigger route synchronization.

### routes enable HOSTNAME

Enable a route.

### routes disable HOSTNAME

Disable a route.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--force` | bool | `false` | Skip confirmation prompt |

### user create

Create a new administrative user.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--type` | string | `admin` | Role: `superadmin`, `admin`, `platform`, `support`, `audit` |

The password is read interactively from stdin.

### user list

List administrative users.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--output` | string | `table` | Output format: `table` or `json` |

### user set-password

Change the password of an administrative user.

The new password is read interactively from stdin.

### storage instances

List per-instance storage usage.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--output` | string | `table` | Output format: `table` or `json` |

### storage nodes

List per-node storage and resource usage.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--output` | string | `table` | Output format: `table` or `json` |

### version

Print the CLI version.

### help

Print general usage information.

## SEE ALSO

* [admiralctl-admin(8)](admiralctl-admin.8) — Administration workflows and examples.
