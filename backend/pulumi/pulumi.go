// Package pulumi implements the Pulumi IaC backend.
package pulumi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/grokify/mogo/net/http/healthz"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/lightsail"
	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optdestroy"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optpreview"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optrefresh"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optup"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/plexusone/omnideploy/backend"
	"github.com/plexusone/omnideploy/config"
	"github.com/plexusone/omnideploy/secrets"
	"github.com/plexusone/omnideploy/target"
)

func init() {
	backend.Register(New())
}

// Backend implements the Pulumi IaC backend.
type Backend struct {
	// WorkDir is the working directory for Pulumi state
	WorkDir string
}

// New creates a new Pulumi backend.
func New() *Backend {
	workDir := os.Getenv("OMNIDEPLOY_WORK_DIR")
	if workDir == "" {
		home, _ := os.UserHomeDir()
		workDir = filepath.Join(home, ".omnideploy", "pulumi")
	}
	return &Backend{WorkDir: workDir}
}

// Name returns the backend name.
func (b *Backend) Name() string {
	return "pulumi"
}

// Description returns a human-readable description.
func (b *Backend) Description() string {
	return "Pulumi - Infrastructure as Code using Go"
}

// Apply provisions resources using Pulumi.
func (b *Backend) Apply(ctx context.Context, spec *target.ResourceSpec, opts backend.ApplyOptions) (*backend.Result, error) {
	stackName := opts.StackName
	if stackName == "" {
		stackName = spec.StackName
	}

	// Resolve secret refs up front so a missing secret fails the deploy
	// before any infrastructure changes (RMI-OMNIAGENT-006).
	resolved, err := resolveSecrets(ctx, spec)
	if err != nil {
		return nil, err
	}

	// Create the Pulumi program
	program := b.createProgram(spec, resolved)

	// Create or select the stack
	stack, err := b.getOrCreateStack(ctx, stackName, spec.Config.Name, program)
	if err != nil {
		return nil, fmt.Errorf("creating stack: %w", err)
	}

	// Set AWS region
	if err := stack.SetConfig(ctx, "aws:region", auto.ConfigValue{Value: spec.Region}); err != nil {
		return nil, fmt.Errorf("setting region: %w", err)
	}

	// Run update
	var upOpts []optup.Option
	if opts.OnOutput != nil {
		upOpts = append(upOpts, optup.ProgressStreams(writerFunc(opts.OnOutput)))
	}

	result, err := stack.Up(ctx, upOpts...)
	if err != nil {
		return nil, fmt.Errorf("deploying: %w", err)
	}

	// Extract outputs
	outputs := make(map[string]string)
	for k, v := range result.Outputs {
		if s, ok := v.Value.(string); ok {
			outputs[k] = s
		}
	}

	// Post-deploy health verification (RMI-OMNIDEPLOY-001): the deployment
	// reaching ACTIVE proves the target's own internal health gate passed
	// (e.g. Lightsail already waited on its container health check before
	// returning), but not that the service is reachable from here — DNS
	// propagation, TLS cert readiness, or a network path issue can still
	// leave a "successfully deployed" service unreachable. Skip silently
	// when the config declares no health check or the target exposed no
	// URL output — there's nothing to verify against.
	if err := verifyHealth(ctx, spec.Config, outputs["url"], opts.OnOutput); err != nil {
		return nil, fmt.Errorf("deployment succeeded but health verification failed: %w", err)
	}

	// Extract resource changes
	var created, updated, deleted int
	if result.Summary.ResourceChanges != nil {
		changes := *result.Summary.ResourceChanges
		created = changes["create"]
		updated = changes["update"]
		deleted = changes["delete"]
	}

	return &backend.Result{
		StackName:        stackName,
		Outputs:          outputs,
		ResourcesCreated: created,
		ResourcesUpdated: updated,
		ResourcesDeleted: deleted,
	}, nil
}

// Preview shows what would be provisioned.
func (b *Backend) Preview(ctx context.Context, spec *target.ResourceSpec) (*backend.PreviewResult, error) {
	resolved, err := resolveSecrets(ctx, spec)
	if err != nil {
		return nil, err
	}
	program := b.createProgram(spec, resolved)

	stack, err := b.getOrCreateStack(ctx, spec.StackName, spec.Config.Name, program)
	if err != nil {
		return nil, fmt.Errorf("creating stack: %w", err)
	}

	if err := stack.SetConfig(ctx, "aws:region", auto.ConfigValue{Value: spec.Region}); err != nil {
		return nil, fmt.Errorf("setting region: %w", err)
	}

	result, err := stack.Preview(ctx, optpreview.ProgressStreams(os.Stdout))
	if err != nil {
		return nil, fmt.Errorf("previewing: %w", err)
	}

	// Build summary from ChangeSummary
	var created, updated, deleted int
	if result.ChangeSummary != nil {
		created = result.ChangeSummary["create"]
		updated = result.ChangeSummary["update"]
		deleted = result.ChangeSummary["delete"]
	}

	return &backend.PreviewResult{
		StackName: spec.StackName,
		Changes:   []backend.Change{}, // Detailed steps not available in this API version
		Summary:   fmt.Sprintf("Changes: %d create, %d update, %d delete", created, updated, deleted),
	}, nil
}

// Destroy removes all resources.
func (b *Backend) Destroy(ctx context.Context, stackName string, opts backend.DestroyOptions) error {
	stack, err := auto.SelectStackLocalSource(ctx, stackName, b.WorkDir)
	if err != nil {
		return fmt.Errorf("selecting stack: %w", err)
	}

	var destroyOpts []optdestroy.Option
	if opts.OnOutput != nil {
		destroyOpts = append(destroyOpts, optdestroy.ProgressStreams(writerFunc(opts.OnOutput)))
	}

	_, err = stack.Destroy(ctx, destroyOpts...)
	if err != nil {
		return fmt.Errorf("destroying: %w", err)
	}

	return nil
}

// Refresh refreshes the state.
func (b *Backend) Refresh(ctx context.Context, stackName string) error {
	stack, err := auto.SelectStackLocalSource(ctx, stackName, b.WorkDir)
	if err != nil {
		return fmt.Errorf("selecting stack: %w", err)
	}

	_, err = stack.Refresh(ctx, optrefresh.ProgressStreams(os.Stdout))
	if err != nil {
		return fmt.Errorf("refreshing: %w", err)
	}

	return nil
}

// getOrCreateStack creates or selects a Pulumi stack.
func (b *Backend) getOrCreateStack(ctx context.Context, stackName, projectName string, program pulumi.RunFunc) (auto.Stack, error) {
	// Ensure work directory exists
	if err := os.MkdirAll(b.WorkDir, 0o755); err != nil {
		return auto.Stack{}, fmt.Errorf("creating work dir: %w", err)
	}

	// Try to create a new stack, or select existing
	stack, err := auto.UpsertStackInlineSource(ctx, stackName, projectName, program,
		auto.WorkDir(b.WorkDir),
	)
	if err != nil {
		return auto.Stack{}, err
	}

	return stack, nil
}

// createProgram creates the Pulumi program for the given spec.
func (b *Backend) createProgram(spec *target.ResourceSpec, resolvedSecrets map[string]string) pulumi.RunFunc {
	return func(ctx *pulumi.Context) error {
		switch spec.Target {
		case "lightsail":
			return b.deployLightsail(ctx, spec, resolvedSecrets)
		case "lightsail-instance":
			return b.deployLightsailInstance(ctx, spec, resolvedSecrets)
		default:
			return fmt.Errorf("unsupported target: %s", spec.Target)
		}
	}
}

// deployLightsail deploys to AWS LightSail.
func (b *Backend) deployLightsail(ctx *pulumi.Context, spec *target.ResourceSpec, resolvedSecrets map[string]string) error {
	cfg := spec.Config

	// Create container service
	service, err := lightsail.NewContainerService(ctx, cfg.Name, &lightsail.ContainerServiceArgs{
		Name:       pulumi.String(cfg.Name),
		Power:      pulumi.String(sizeToPower(cfg.Resources.Size)),
		Scale:      pulumi.Int(cfg.Service.Replicas),
		IsDisabled: pulumi.Bool(false),
		Tags:       pulumi.ToStringMap(cfg.Tags),
	})
	if err != nil {
		return fmt.Errorf("creating container service: %w", err)
	}

	// Build container definition
	containers := lightsail.ContainerServiceDeploymentVersionContainerArray{
		&lightsail.ContainerServiceDeploymentVersionContainerArgs{
			ContainerName: pulumi.String(cfg.Name),
			Image:         pulumi.String(cfg.Container.Image),
			Commands:      pulumi.ToStringArray(cfg.Container.Args),
			Environment:   buildEnvironment(cfg.Environment, resolvedSecrets),
			Ports:         buildPortMap(cfg.Container.Ports),
		},
	}

	// Build public endpoint
	var publicEndpoint *lightsail.ContainerServiceDeploymentVersionPublicEndpointArgs
	if cfg.Service.Public && len(cfg.Container.Ports) > 0 {
		endpointArgs := &lightsail.ContainerServiceDeploymentVersionPublicEndpointArgs{
			ContainerName: pulumi.String(cfg.Name),
			ContainerPort: pulumi.Int(cfg.Container.Ports[0].ContainerPort),
		}

		if cfg.Container.HealthCheck != nil && cfg.Container.HealthCheck.Path != "" {
			endpointArgs.HealthCheck = &lightsail.ContainerServiceDeploymentVersionPublicEndpointHealthCheckArgs{
				Path:               pulumi.String(cfg.Container.HealthCheck.Path),
				IntervalSeconds:    pulumi.Int(int(cfg.Container.HealthCheck.Interval.Seconds())),
				TimeoutSeconds:     pulumi.Int(int(cfg.Container.HealthCheck.Timeout.Seconds())),
				HealthyThreshold:   pulumi.Int(cfg.Container.HealthCheck.HealthyThreshold),
				UnhealthyThreshold: pulumi.Int(cfg.Container.HealthCheck.UnhealthyThreshold),
			}
		}

		publicEndpoint = endpointArgs
	}

	// Create deployment
	deployment, err := lightsail.NewContainerServiceDeploymentVersion(ctx, cfg.Name+"-deployment", &lightsail.ContainerServiceDeploymentVersionArgs{
		ServiceName:    service.Name,
		Containers:     containers,
		PublicEndpoint: publicEndpoint,
	})
	if err != nil {
		return fmt.Errorf("creating deployment: %w", err)
	}

	// Export outputs
	ctx.Export("url", service.Url)
	ctx.Export("state", deployment.State)
	ctx.Export("service_name", service.Name)

	return nil
}

// instanceSSHUser is the SSH user for Lightsail instances created from
// Ubuntu blueprints — the blueprint grokify-omniagent's reference
// implementation uses, and the common case for this target.
const instanceSSHUser = "ubuntu"

// deployLightsailInstance deploys to an AWS Lightsail VM instance running
// a pre-built binary as a systemd service, reproducing
// grokify-omniagent's reference deploy/lightsail/{setup,deploy}.sh
// declaratively (docs/specs/TRD.md). Unlike deployLightsail's Container
// Service target, the instance's disk is persistent across redeploys at
// no extra cost. remote.Command's Create/Update split (rather than
// lightsail.Instance's UserData, which is cloud-init and only runs once
// at first boot) is what makes redeploys idempotent: Create installs the
// systemd unit and starts the service; Update only restarts it; Triggers
// is keyed on the binary's content hash plus the unit content so either
// changing forces a re-run.
func (b *Backend) deployLightsailInstance(ctx *pulumi.Context, spec *target.ResourceSpec, resolvedSecrets map[string]string) error {
	cfg := spec.Config
	inst := cfg.Instance

	keyPair, err := lightsail.NewKeyPair(ctx, cfg.Name+"-keypair", &lightsail.KeyPairArgs{
		Name: pulumi.String(cfg.Name),
	})
	if err != nil {
		return fmt.Errorf("creating key pair: %w", err)
	}

	instance, err := lightsail.NewInstance(ctx, cfg.Name, &lightsail.InstanceArgs{
		Name:             pulumi.String(cfg.Name),
		AvailabilityZone: pulumi.String(availabilityZone(spec.Region)),
		BlueprintId:      pulumi.String(inst.Blueprint),
		BundleId:         pulumi.String(inst.Bundle),
		KeyPairName:      keyPair.Name,
		Tags:             pulumi.ToStringMap(cfg.Tags),
	})
	if err != nil {
		return fmt.Errorf("creating instance: %w", err)
	}

	portInfos := lightsail.InstancePublicPortsPortInfoArray{
		&lightsail.InstancePublicPortsPortInfoArgs{
			Protocol: pulumi.String("tcp"),
			FromPort: pulumi.Int(22),
			ToPort:   pulumi.Int(22),
		},
	}
	if inst.HealthCheck != nil && inst.HealthCheck.Kind == "http" {
		portInfos = append(portInfos, &lightsail.InstancePublicPortsPortInfoArgs{
			Protocol: pulumi.String("tcp"),
			FromPort: pulumi.Int(inst.HealthCheck.Port),
			ToPort:   pulumi.Int(inst.HealthCheck.Port),
		})
	}
	firewall, err := lightsail.NewInstancePublicPorts(ctx, cfg.Name+"-firewall", &lightsail.InstancePublicPortsArgs{
		InstanceName: instance.Name,
		PortInfos:    portInfos,
	})
	if err != nil {
		return fmt.Errorf("configuring firewall: %w", err)
	}

	conn := remote.ConnectionArgs{
		Host:       instance.PublicIpAddress,
		User:       pulumi.String(instanceSSHUser),
		PrivateKey: keyPair.PrivateKey,
	}

	binaryHash, err := binaryContentHash(inst.BinaryPath)
	if err != nil {
		return fmt.Errorf("hashing binary at %s: %w", inst.BinaryPath, err)
	}

	// The firewall must exist before SSH can reach the instance.
	artifact, err := remote.NewCopyToRemote(ctx, cfg.Name+"-artifact", &remote.CopyToRemoteArgs{
		Connection: conn,
		RemotePath: pulumi.String(inst.RemotePath),
		Source:     pulumi.NewFileAsset(inst.BinaryPath),
		Triggers:   pulumi.Array{pulumi.String(binaryHash)},
	}, pulumi.DependsOn([]pulumi.Resource{firewall}))
	if err != nil {
		return fmt.Errorf("copying artifact: %w", err)
	}

	unit := inst.SystemdUnit
	if unit == "" {
		unit = defaultSystemdUnit(inst)
	}

	// envContent carries both plain environment vars and resolved secrets
	// (RMI-OMNIDEPLOY-006) — it's written to <remote_path>.env on the
	// instance and consumed by the systemd unit's EnvironmentFile=. Only
	// its hash (not the content itself) goes into Triggers, and the
	// scripts that embed it are wrapped in pulumi.ToSecret below, so
	// secret values never appear in Pulumi state or CLI output.
	envContent := buildRemoteEnvFile(cfg.Environment, resolvedSecrets)
	envHash := hashString(envContent)

	createScript := pulumi.ToSecret(pulumi.String(installServiceScript(inst, unit, envContent))).(pulumi.StringOutput)
	updateScript := pulumi.ToSecret(pulumi.String(restartServiceScript(inst, envContent))).(pulumi.StringOutput)

	serviceCmd, err := remote.NewCommand(ctx, cfg.Name+"-service", &remote.CommandArgs{
		Connection: conn,
		Create:     createScript,
		Update:     updateScript,
		Triggers:   pulumi.Array{pulumi.String(binaryHash), pulumi.String(unit), pulumi.String(envHash)},
	}, pulumi.DependsOn([]pulumi.Resource{artifact}))
	if err != nil {
		return fmt.Errorf("installing service: %w", err)
	}

	// systemd health verification (RMI-OMNIDEPLOY-007) runs inline as
	// part of the same SSH session, retrying with the same
	// attempts/delay tuning as the HTTP path's post-Apply verifyHealth —
	// there's no HTTP surface to probe from Go for tools like
	// grokify-omniagent, so the retry loop lives in the remote script
	// itself instead. A non-zero exit fails this resource, which fails
	// stack.Up() and surfaces as a deployment error the same way a
	// failed HTTP verifyHealth does.
	if inst.HealthCheck != nil && inst.HealthCheck.Kind == "systemd" {
		if _, err := remote.NewCommand(ctx, cfg.Name+"-healthcheck", &remote.CommandArgs{
			Connection: conn,
			Create:     pulumi.String(systemdHealthCheckScript(inst.ServiceName)),
			Triggers:   pulumi.Array{pulumi.String(binaryHash), pulumi.String(unit), pulumi.String(envHash)},
		}, pulumi.DependsOn([]pulumi.Resource{serviceCmd})); err != nil {
			return fmt.Errorf("verifying service health: %w", err)
		}
	}

	// The HTTP health-check case is verified post-Apply by the existing
	// verifyHealth (Backend.Apply), reused via healthCheckPathFor —
	// export a "url" output so it has something to probe against, same
	// as deployLightsail's Container Service "url" output.
	if inst.HealthCheck != nil && inst.HealthCheck.Kind == "http" {
		ctx.Export("url", pulumi.Sprintf("http://%s:%d", instance.PublicIpAddress, inst.HealthCheck.Port))
	}

	ctx.Export("public_ip", instance.PublicIpAddress)
	ctx.Export("service_name", pulumi.String(inst.ServiceName))

	return nil
}

// availabilityZone derives a default AZ from the deploy region — Lightsail
// requires one, and a single, predictable AZ per region is the right
// default for a single-instance, capped-cost deployment (no multi-AZ
// redundancy to configure).
func availabilityZone(region string) string {
	return region + "a"
}

// binaryContentHash hashes the local binary so remote.Command's Triggers
// can detect "nothing changed" and skip re-running Create/Update — the
// idempotent-redeploy behavior documented on deployLightsailInstance.
func binaryContentHash(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return hashString(string(data)), nil
}

// hashString hashes s for use in remote.Command's Triggers — a one-way
// digest so a Triggers entry can detect "the .env content changed"
// without putting secret values themselves into Pulumi state.
func hashString(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// buildRemoteEnvFile renders systemd EnvironmentFile= syntax (KEY="value"
// per line, values double-quoted with internal backslashes/quotes
// escaped so whitespace and special characters survive) from cfg's plain
// environment plus resolved secrets. Secret values win on a key
// collision, matching buildEnvironment's precedence for the container
// target. Keys are sorted for deterministic output — otherwise Pulumi
// would see a spurious diff (and Triggers hash change) on every deploy
// from Go's randomized map iteration order alone.
func buildRemoteEnvFile(plain map[string]string, resolvedSecrets map[string]string) string {
	merged := make(map[string]string, len(plain)+len(resolvedSecrets))
	for k, v := range plain {
		merged[k] = v
	}
	for k, v := range resolvedSecrets {
		merged[k] = v
	}

	keys := make([]string, 0, len(merged))
	for k := range merged {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&b, "%s=%s\n", k, quoteEnvValue(merged[k]))
	}
	return b.String()
}

// quoteEnvValue double-quotes v for a systemd EnvironmentFile, escaping
// backslashes and double quotes so a value round-trips exactly.
func quoteEnvValue(v string) string {
	escaped := strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(v)
	return `"` + escaped + `"`
}

// defaultSystemdUnit renders a systemd unit for inst when the config
// doesn't supply its own (config.InstanceConfig.SystemdUnit), reproducing
// the reference setup.sh's hardening: no new privileges, a read-only
// system/home with one writable path (the binary's own directory, for
// e.g. a local SQLite file dropped next to it). EnvironmentFile uses the
// "-" prefix so a missing .env doesn't stop the service — Phase 2
// (RMI-OMNIDEPLOY-006) starts writing that file; this unit already
// expects it at that path.
func defaultSystemdUnit(inst *config.InstanceConfig) string {
	return fmt.Sprintf(`[Unit]
Description=%s
After=network.target

[Service]
Type=simple
ExecStart=%s
Restart=on-failure
RestartSec=5
EnvironmentFile=-%s.env
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=%s

[Install]
WantedBy=multi-user.target
`, inst.ServiceName, inst.RemotePath, inst.RemotePath, path.Dir(inst.RemotePath))
}

// writeEnvFileScript renders the shell snippet that (re)writes
// <remote_path>.env from envContent and locks it down to the SSH user —
// systemd itself still reads it fine via EnvironmentFile= since the
// service also runs as that user in this target's current single-user
// model. Shared by installServiceScript and restartServiceScript so a
// secrets-only change (envContent changes, binary/unit don't) still
// lands on every redeploy.
func writeEnvFileScript(inst *config.InstanceConfig, envContent string) string {
	return fmt.Sprintf(`sudo tee %s.env > /dev/null <<'OMNIDEPLOY_ENV'
%s
OMNIDEPLOY_ENV
sudo chmod 600 %s.env
`, inst.RemotePath, envContent, inst.RemotePath)
}

// installServiceScript is remote.Command's Create step: write the
// systemd unit and the .env file, make the binary executable, and
// enable+start the service. Runs once — redeploys use
// restartServiceScript instead.
func installServiceScript(inst *config.InstanceConfig, unit, envContent string) string {
	return fmt.Sprintf(`set -eu
sudo mkdir -p %s
sudo chmod +x %s
sudo tee /etc/systemd/system/%s.service > /dev/null <<'OMNIDEPLOY_UNIT'
%s
OMNIDEPLOY_UNIT
%ssudo systemctl daemon-reload
sudo systemctl enable %s
sudo systemctl restart %s
`, path.Dir(inst.RemotePath), inst.RemotePath, inst.ServiceName, unit, writeEnvFileScript(inst, envContent), inst.ServiceName, inst.ServiceName)
}

// restartServiceScript is remote.Command's Update step: the binary was
// already copied by remote.CopyToRemote by the time this runs, so this
// only needs to refresh the .env file, make the binary executable again,
// and restart the service — no systemd unit reinstall, keeping redeploys
// fast and idempotent.
func restartServiceScript(inst *config.InstanceConfig, envContent string) string {
	return fmt.Sprintf(`set -eu
%ssudo chmod +x %s
sudo systemctl daemon-reload
sudo systemctl restart %s
`, writeEnvFileScript(inst, envContent), inst.RemotePath, inst.ServiceName)
}

// systemdHealthCheckScript polls `systemctl is-active <serviceName>` with
// the same attempts/delay tuning as the HTTP path's verifyHealthRetry,
// so both health-check kinds absorb the same amount of post-restart
// settling time before being declared unhealthy.
func systemdHealthCheckScript(serviceName string) string {
	return fmt.Sprintf(`set -eu
for i in $(seq 1 %d); do
  if sudo systemctl is-active --quiet %s; then
    echo "%s is active"
    exit 0
  fi
  sleep %d
done
echo "%s did not become active" >&2
exit 1
`, healthVerifyAttempts, serviceName, serviceName, int(healthVerifyDelay.Seconds()), serviceName)
}

// sizeToPower maps size to LightSail power.
func sizeToPower(size string) string {
	switch size {
	case "nano":
		return "nano"
	case "micro", "":
		return "micro"
	case "small":
		return "small"
	case "medium":
		return "medium"
	case "large":
		return "large"
	case "xlarge":
		return "xlarge"
	default:
		return "micro"
	}
}

// buildPortMap converts ports to Pulumi string map (port -> protocol).
func buildPortMap(ports []config.PortMapping) pulumi.StringMap {
	result := pulumi.StringMap{}
	for _, p := range ports {
		key := fmt.Sprintf("%d", p.ContainerPort)
		result[key] = pulumi.String(p.Protocol)
	}
	return result
}

// writerFunc adapts a callback to io.Writer.
type writerFunc func(string)

func (f writerFunc) Write(p []byte) (n int, err error) {
	f(string(p))
	return len(p), nil
}

// resolveSecrets resolves spec.Config.Secrets to values (RMI-OMNIAGENT-006).
// Returns nil when the config declares no secrets so no AWS client is
// constructed for secret-free deploys.
func resolveSecrets(ctx context.Context, spec *target.ResourceSpec) (map[string]string, error) {
	if len(spec.Config.Secrets) == 0 {
		return nil, nil
	}
	resolver, err := secrets.NewResolver(ctx, spec.Region)
	if err != nil {
		return nil, fmt.Errorf("building secret resolver: %w", err)
	}
	resolved, err := resolver.Resolve(ctx, spec.Config.Secrets)
	if err != nil {
		return nil, fmt.Errorf("resolving secrets: %w", err)
	}
	return resolved, nil
}

// buildEnvironment merges plain environment variables with resolved
// secrets into one container environment map. Secret values are wrapped
// with pulumi.ToSecret so they are encrypted in Pulumi state rather than
// stored plaintext; on a name collision the secret wins over the plain
// entry, so promoting a variable from environment to secrets needs no
// removal of the old key.
func buildEnvironment(plain map[string]string, resolvedSecrets map[string]string) pulumi.StringMap {
	env := make(pulumi.StringMap, len(plain)+len(resolvedSecrets))
	for k, v := range plain {
		env[k] = pulumi.String(v)
	}
	for k, v := range resolvedSecrets {
		env[k] = pulumi.ToSecret(pulumi.String(v)).(pulumi.StringOutput)
	}
	return env
}

// healthVerifyAttempts and healthVerifyDelay bound post-deploy health
// verification: a deployment that just reached ACTIVE may still need a
// few seconds for DNS/TLS to settle from this machine's vantage point.
const (
	healthVerifyAttempts = 5
	healthVerifyDelay    = 3 * time.Second
	healthVerifyTimeout  = 5 * time.Second
)

// verifyHealth probes cfg's configured HTTP health-check path against
// serviceURL, retrying briefly to absorb DNS/TLS settling immediately
// after a deployment reaches ACTIVE. No-ops (returns nil) when the config
// declares no HTTP health check or serviceURL is empty — there's nothing
// to verify against for that target/config combination. Covers both the
// container target (Container.HealthCheck) and the instance target's
// HTTP-kind check (Instance.HealthCheck) via healthCheckPathFor; the
// instance target's systemd-kind check is verified inline by the Pulumi
// program itself (see deployLightsailInstance) since it has no HTTP
// surface to probe.
func verifyHealth(ctx context.Context, cfg *config.DeployConfig, serviceURL string, onOutput func(string)) error {
	path := healthCheckPathFor(cfg)
	if path == "" || serviceURL == "" {
		return nil
	}
	url := strings.TrimRight(serviceURL, "/") + path
	return verifyHealthRetry(ctx, url, healthVerifyAttempts, healthVerifyDelay, healthVerifyTimeout, onOutput)
}

// healthCheckPathFor returns the HTTP health-check path to verify
// against for cfg, or "" if cfg declares no HTTP health check.
func healthCheckPathFor(cfg *config.DeployConfig) string {
	if cfg.Container.HealthCheck != nil && cfg.Container.HealthCheck.Path != "" {
		return cfg.Container.HealthCheck.Path
	}
	if cfg.Instance != nil && cfg.Instance.HealthCheck != nil && cfg.Instance.HealthCheck.Kind == "http" {
		return cfg.Instance.HealthCheck.Path
	}
	return ""
}

// verifyHealthRetry is verifyHealth's retry loop, parameterized so tests
// can use short delays instead of the real (deployment-appropriate)
// defaults.
func verifyHealthRetry(ctx context.Context, url string, attempts int, delay, timeout time.Duration, onOutput func(string)) error {
	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		if onOutput != nil {
			onOutput(fmt.Sprintf("verifying health: %s (attempt %d/%d)", url, attempt, attempts))
		}
		if err := healthz.Probe(ctx, url, timeout); err != nil {
			lastErr = err
			if attempt < attempts {
				select {
				case <-time.After(delay):
				case <-ctx.Done():
					return fmt.Errorf("%s: %w", url, ctx.Err())
				}
			}
			continue
		}
		if onOutput != nil {
			onOutput(fmt.Sprintf("health verified: %s", url))
		}
		return nil
	}
	return fmt.Errorf("%s did not become healthy after %d attempts: %w", url, attempts, lastErr)
}
