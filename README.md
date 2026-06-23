# admiralctl

Command-line interface for the Admiral PaaS platform.

`admiralctl` communicates with `admirald` to manage nodes, applications, instances, backups, routes, and operations.

## Quick start

```bash
# Configure CLI
admiralctl init --server https://admirald.example.com:8080 --token your-token

# Check platform health
admiralctl status

# List worker nodes
admiralctl nodes list

# Manage applications
admiralctl apps list
admiralctl apps apply -f app.yaml
admiralctl apps validate -f app.yaml
admiralctl apps deactivate --name myapp
admiralctl apps activate --name myapp

# Manage instances
admiralctl instances list
admiralctl instances show INSTANCE_ID
admiralctl instances inspect INSTANCE_ID
admiralctl instances provision --app myapp --tier small --customer cust_001
admiralctl instances provision --app myapp --tier small --customer cust_001 --node worker-01 --logical-instance-id li_001
admiralctl instances migrate --target-node worker-02 INSTANCE_ID
admiralctl instances pause INSTANCE_ID
admiralctl instances resume INSTANCE_ID
admiralctl instances restart INSTANCE_ID
admiralctl instances resize --tier large INSTANCE_ID
admiralctl instances deprovision INSTANCE_ID

# Backup and restore
admiralctl instances backup --service SERVICE_NAME INSTANCE_ID
admiralctl backups list
admiralctl backups storage get
admiralctl backups restore --backup-id BK_ID --instance-id INST_ID --service SERVICE_NAME
admiralctl backups delete BK_ID
admiralctl backups prune

> **Note:** Restore requires the target instance to be in `paused` or `stopped` state.

# Routes
admiralctl routes list
admiralctl routes sync

# Operations
admiralctl operations list
admiralctl operations show OP_ID
admiralctl operations retry OP_ID

# Storage
admiralctl storage instances
admiralctl storage nodes

# Users
admiralctl user list
admiralctl user create username --type admin
```

## Documentation

See [docs/man.md](docs/man.md) for the full command reference.

## Output formats

All `list` subcommands support `--output table` (default) and `--output json`.

## Configuration

Configuration is stored in `~/.config/admiralctl/config.yaml` and can be initialized with `admiralctl init`.

Environment variables:
- `ADMIRAL_SERVER_URL`
- `ADMIRAL_ADMIN_TOKEN`
- `ADMIRAL_TLS_CA_FILE`
- `ADMIRAL_OPERATOR`

> **Security Note:** Avoid using the `--token` flag in shared environments, as it may expose your token in the process list. Use the `ADMIRAL_ADMIN_TOKEN` environment variable instead.
