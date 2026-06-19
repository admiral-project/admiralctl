# admiralctl — Command Reference

## SYNOPSIS

`admiralctl <command> [subcommand] [flags]`

## COMMANDS

### init

Initialize or update local CLI configuration.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--server` | string | `https://localhost:8080` | admirald API endpoint |
| `--token` | string | env `ADMIRAL_ADMIN_TOKEN` | Shared authentication token |
| `--ca-cert` | string | env `ADMIRAL_TLS_CA_FILE` | CA certificate for TLS |

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
| `--os` | string | no | Operating system (default: linux) |
| `--podman` | string | no | Podman version (default: 4.9.0) |

### nodes enable NODE_ID

Activate a node for provisioning.

### nodes disable NODE_ID

Deactivate a node.

### nodes remove NODE_ID

Remove a registered node from the platform.

This removes the node record, its routes, backups, and customer apps
from the database. The operation is refused if the node has active
instances unless `--force` is used.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--force` | bool | `false` | Remove even with active instances |

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
| `-f` | string | yes | Path to YAML file |

### apps validate

Validate an application definition YAML file locally.

| Flag | Type | Required | Description |
|------|------|----------|-------------|
| `-f` | string | yes | Path to YAML file |

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

### instances list

List customer application instances.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--output` | string | `table` | Output format: `table` or `json` |

### instances show INSTANCE_ID

Show details of a specific instance.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--output` | string | `table` | Output format: `table` or `json` |

### instances inspect INSTANCE_ID

Inspect containers and volumes of a running instance.

### instances provision

Provision a new customer application instance.

| Flag | Type | Required | Description |
|------|------|----------|-------------|
| `--app` | string | yes | Application definition name |
| `--tier` | string | yes | Tier name |
| `--customer` | string | yes | Customer ID |
| `--node` | string | no | Explicit node ID to target |
| `--logical-instance-id` | string | no | Preserve the logical instance identity across reprovisioning or migration |
| `--output` | string | no | Output format: `table` or `json` |

### instances migrate --target-node NODE_ID INSTANCE_ID

Start an offline migration to another worker node while preserving the logical instance identity.

| Flag | Type | Required | Description |
|------|------|----------|-------------|
| `--target-node` | string | yes | Target worker node ID |
| `--wait` | bool | no | Wait until the migration operation reaches a terminal state |

### instances pause INSTANCE_ID

Pause a running instance.

### instances resume INSTANCE_ID

Resume a paused instance.

### instances start INSTANCE_ID

Start a stopped instance.

### instances stop INSTANCE_ID

Stop a running instance.

### instances restart INSTANCE_ID

Restart an instance (stop then start).

### instances backup --service SERVICE_NAME INSTANCE_ID

Trigger a backup for a specific service.

Flags must appear before the instance ID.

| Flag | Type | Required | Description |
|------|------|----------|-------------|
| `--service` | string | yes | Service name declared in the app definition |

### instances deprovision INSTANCE_ID

Deprovision (delete) an instance.

### instances destroy INSTANCE_ID

Alias for deprovision.

### instances resize --tier TIER_NAME INSTANCE_ID

Change the tier of an instance.

| Flag | Type | Required | Description |
|------|------|----------|-------------|
| `--tier` | string | yes | Target tier name |

If the target tier would exceed the assigned node capacity policy, the action is rejected before dispatch and the CLI prints the policy rejection details.

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

> **Precondition:** The target instance must be in `paused` or `stopped` state.
> Restore is rejected with `HTTP 409` if the instance is `running`.

| Flag | Type | Required | Description |
|------|------|----------|-------------|
| `--backup-id` | string | yes | Backup ID |
| `--instance-id` | string | yes | Target instance ID |
| `--service` | string | yes | Service name matching the backup source |
| `--target-node` | string | no | Target node ID |
| `--source-type` | string | no | Source type override |
| `--source-uri` | string | no | Source URI override |
| `--verify-checksum` | bool | no | Verify checksum (default: true) |

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

### user create

Create a new administrative user.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--type` | string | `admin` | Role: superadmin, admin, platform, support, audit |

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

### help

Print general usage information.
