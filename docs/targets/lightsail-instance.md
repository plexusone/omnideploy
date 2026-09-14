# AWS Lightsail Instance Target

A persistent Lightsail VM running your pre-built binary as a systemd service — for deployments that need cheap, capped-cost **local disk persistence** that [the LightSail (Container Service) target](lightsail.md) can't provide.

## Overview

Lightsail Container Service has no persistent disk: any local file is wiped on every redeploy, and getting real persistence there means adding an external Lightsail managed database (+$15/mo minimum). A Lightsail **instance** is a plain VM — its disk is included in the flat monthly price and survives both restarts and redeploys for free.

The instance target provisions:

- A Lightsail VM (`blueprint` + `bundle` you choose)
- A Pulumi-managed SSH keypair
- A firewall — **SSH-only by default**, opening one additional port only if you configure an HTTP health check
- The binary, shipped over SSH and installed as a systemd service
- Secrets and environment variables, written to a `systemd EnvironmentFile=` on the instance

Redeploys are idempotent: the binary is only re-shipped and the service only reinstalled/restarted when its content, the systemd unit, or the environment actually changed (tracked via content hashes), not on every `omnideploy up`.

## When to Use

**Good for:**

- Anything that needs a local file to survive a redeploy — SQLite, session state, uploaded files, local caches
- Discord/Slack-style bots and other outbound-only workloads (no inbound HTTP surface needed, so the firewall stays SSH-only)
- Cost-sensitive, single-instance workloads where a managed database is overkill

**Not ideal for:**

- Anything that needs auto-scaling or multiple replicas — this target manages exactly one instance
- Workloads that are already fully stateless — [the container target](lightsail.md) has less to manage (no SSH keys, no systemd unit)

## Configuration

### Basic Example

```yaml
name: my-bot
region: us-west-2

instance:
  blueprint: ubuntu_22_04
  bundle: nano_3_0
  binary_path: ./bin/my-bot        # local path, built beforehand (CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build)
  remote_path: /opt/my-bot/my-bot  # remote path on the instance
  service_name: my-bot
```

### Full Example

```yaml
name: my-bot
region: us-west-2

instance:
  blueprint: ubuntu_22_04
  bundle: nano_3_0
  binary_path: ./bin/my-bot
  remote_path: /opt/my-bot/my-bot
  service_name: my-bot

  # Optional: override the generated systemd unit entirely — use this to
  # grant ReadWritePaths beyond the binary's own directory (e.g. a
  # separate /data volume for a database file).
  systemd_unit: |
    [Unit]
    Description=my-bot
    After=network.target

    [Service]
    Type=simple
    ExecStart=/opt/my-bot/my-bot
    Restart=on-failure
    RestartSec=5
    EnvironmentFile=-/opt/my-bot/my-bot.env
    NoNewPrivileges=true
    ProtectSystem=strict
    ProtectHome=true
    ReadWritePaths=/data /opt/my-bot

    [Install]
    WantedBy=multi-user.target

  # Optional: post-deploy health verification (see below). Omit for
  # workloads with no HTTP surface — the firewall then stays SSH-only.
  health_check:
    kind: http
    path: /health
    port: 8080

environment:
  LOG_LEVEL: info

secrets:
  - name: DISCORD_BOT_TOKEN
    source: ssm:/my-bot/discord-token

tags:
  environment: production
```

## Deployment

```bash
# Build for the target first — CGO_ENABLED=0 keeps the binary
# dependency-free for a bare Ubuntu blueprint.
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/my-bot ./cmd/my-bot

# Deploy
omnideploy up --config deploy.yaml --target lightsail-instance

# Preview changes
omnideploy preview --config deploy.yaml --target lightsail-instance

# Destroy
omnideploy destroy --stack my-bot
```

Redeploying after a code change is just rebuilding the binary and running `omnideploy up` again — the binary's content hash changing is what triggers a re-ship and service restart; an unchanged binary/unit/environment is a no-op.

## Outputs

| Output | Description |
|--------|-------------|
| `public_ip` | Instance public IP address |
| `service_name` | systemd service name |
| `url` | Only present when `instance.health_check.kind` is `http` — `http://<public_ip>:<port>` |

## Health Checks

Two kinds, matched to whether the workload has an HTTP surface at all:

### `http`

Verified the same way as the container target: `omnideploy up` probes the URL after the deployment completes, retrying briefly to absorb DNS/settling time. Configuring this also opens the given port in the instance firewall (SSH stays open too).

```yaml
instance:
  health_check:
    kind: http
    path: /health
    port: 8080
```

### `systemd`

For workloads with no HTTP surface (a Discord/Slack bot, a worker process). Verified over the same SSH session used to install the service — `systemctl is-active` is polled with the same retry/backoff as the HTTP path. A service that never becomes active fails the deployment. **Opens no extra firewall port** — the instance stays SSH-only.

```yaml
instance:
  health_check:
    kind: systemd
```

Omit `health_check` entirely to skip post-deploy verification (deployment success is still gated on the systemd install/restart command itself succeeding).

## SSH Access

The target provisions its own Pulumi-managed keypair and connects as the `ubuntu` user — the default account on Ubuntu Lightsail blueprints. Non-Ubuntu blueprints aren't currently supported by this target.

## Limitations

### One Instance, No Auto-Scaling

This target manages exactly one VM. There's no replica count and no scaling policy — if you need that, use [ECS](ecs.md) or [Kubernetes](kubernetes.md) instead.

### Ubuntu Blueprints Only

The target connects as the `ubuntu` SSH user, so non-Ubuntu blueprints aren't currently supported.

### Bundle/Blueprint Values Aren't Validated Locally

Unlike the container target's fixed `resources.size` enum, Lightsail's instance bundle catalog is large, versioned, and changes over time — `instance.blueprint`/`instance.bundle` aren't checked against a hardcoded list. An invalid combination surfaces as an AWS API error during `omnideploy up`. Run `aws lightsail get-blueprints` / `aws lightsail get-bundles` to see current valid values.

## Next Steps

- [LightSail Container Service](lightsail.md) — for fully stateless workloads
- [Configuration Schema](../configuration/schema.md) — full configuration reference
