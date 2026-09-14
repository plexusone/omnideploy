# TRD: Lightsail Instance Target

## Architecture

```text
omnideploy up --target lightsail-instance --config deploy.yaml
        │
        ▼
target/lightsailinstance.Target
  ├── Validate(cfg)     — bundle/blueprint sanity, RemotePath/ServiceName set
  └── ResourceSpec(cfg) — target.ResourceSpec{Target: "lightsail-instance", Config: cfg}
        │
        ▼
backend/pulumi.Backend.Apply
  ├── resolveSecrets(spec)           — unchanged, RMI-OMNIAGENT-006's resolver
  ├── createProgram → deployLightsailInstance(ctx, spec, resolvedSecrets)
  │     ├── lightsail.NewKeyPair              (Pulumi-managed SSH keypair)
  │     ├── lightsail.NewInstance             (BlueprintId, BundleId, AvailabilityZone, KeyPairName)
  │     ├── lightsail.NewInstancePublicPorts   (SSH-only ingress by default)
  │     ├── remote.NewCopyToRemote             (binary + config + rendered .env)
  │     └── remote.NewCommand                  (systemd unit install; Update on redeploy;
  │                                              Triggers = binary content hash)
  └── verifyHealth(spec, outputs)     — generalized: "http" (healthz.Probe against
                                         PublicIP:Port+Path) or "systemd" (SSH
                                         `systemctl is-active`) per Config.Instance.HealthCheck.Kind
```

## Dependencies

- `github.com/pulumi/pulumi-command/sdk/go/command/remote` (v1.2.1,
  verified against pkg.go.dev — not yet an `omnideploy` dependency) for
  `CopyToRemote` and `Command` (SSH-based remote exec). `CopyToRemote` is
  used over the deprecated `CopyFile`.
- `github.com/pulumi/pulumi-aws/sdk/v6/go/aws/lightsail` (already a
  dependency) for `Instance`, `InstancePublicPorts`, `KeyPair`.

## Config schema additions (`config/types.go`)

```go
type DeployConfig struct {
    // ... existing fields ...
    Container ContainerConfig `yaml:"container,omitempty" json:"container,omitempty"` // was required; now omitempty
    Instance  *InstanceConfig `yaml:"instance,omitempty" json:"instance,omitempty"`    // new
}

type InstanceConfig struct {
    Blueprint   string         `yaml:"blueprint"`    // e.g. "ubuntu_22_04"
    Bundle      string         `yaml:"bundle"`        // Lightsail bundle id, e.g. "nano_3_0"
    BinaryPath  string         `yaml:"binary_path"`   // local path to the pre-built binary
    RemotePath  string         `yaml:"remote_path"`   // e.g. "/opt/omniagent"
    ServiceName string         `yaml:"service_name"`  // systemd unit name
    SystemdUnit string         `yaml:"systemd_unit,omitempty"` // optional override
    HealthCheck *VMHealthCheck `yaml:"health_check,omitempty"`
}

type VMHealthCheck struct {
    Kind string `yaml:"kind"` // "http" | "systemd"
    Path string `yaml:"path,omitempty"` // http only
    Port int    `yaml:"port,omitempty"` // http only
}
```

`Name`/`Region`/`Environment`/`Secrets`/`Tags` on `DeployConfig` are
reused unchanged. `Container` becomes optional (only one of
`Container`/`Instance` is expected to be set per deploy; `Validate`
rejects both-set or neither-set).

## Redeploy mechanics (the core correctness requirement)

`lightsail.Instance`'s `UserData` field is cloud-init and **only runs
once, at first boot** — it cannot be used for redeploys. The target
instead uses `remote.Command`'s `Create`/`Update` pair:

- `Create`: install systemd unit, `daemon-reload`, `enable`, `start`.
- `Update`: `daemon-reload && restart` (the binary/config were already
  refreshed by the preceding `CopyToRemote`, which itself re-runs when
  its `Source` content changes).
- `Triggers`: a hash of the binary's content, so Pulumi only re-runs the
  command when the binary actually changed — a no-op `up` with an
  unchanged binary does not restart the service.

This reproduces `deploy.sh`'s `scp` + `ssh systemctl restart` sequence
declaratively, gated by Pulumi's own change-detection instead of always
running unconditionally.

## Secrets

Reuses the `env:`/`ssm:`/`secretsmanager:` resolver unchanged (same
`resolveSecrets` call already in `Apply`). Injection differs: resolved
values are rendered into a `.env` file (`KEY=value` lines) shipped via
the same `CopyToRemote` step, and the systemd unit references it via
`EnvironmentFile=<RemotePath>/.env` — mirroring `setup.sh`'s existing
`EnvironmentFile=/opt/omniagent/.env` pattern exactly.

## Health verification

Generalizes `RMI-OMNIDEPLOY-001`'s `verifyHealth`/`verifyHealthRetry`:

- `Kind == "http"`: unchanged — `healthz.Probe` against
  `http://<PublicIpAddress>:<Port><Path>`.
- `Kind == "systemd"` (new): a `remote.Command` (or direct SSH exec via
  the same `Connection`) running `systemctl is-active --quiet
  <ServiceName>`, retried with the same backoff shape as
  `verifyHealthRetry`. Needed for tools with no HTTP surface at all
  (e.g. a scheduled report generator).
- `Config.Instance.HealthCheck == nil`: skip verification entirely
  (documented no-op, same convention as the container path when no
  health check is configured).

## Firewall

`lightsail.InstancePublicPorts` defaults to SSH (22) only. An optional
app port (for the `Kind == "http"` health-check case, or an app that
itself serves traffic) is opened only when `VMHealthCheck.Port` is set —
keeps the default posture minimal.
