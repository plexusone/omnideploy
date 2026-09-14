# PLAN: Lightsail Instance Target

Tracked in VisionStudio as `INIT-OMNIDEPLOY-001`. RMI IDs are stable;
commits carry `Refs: RMI-OMNIDEPLOY-<NNN>`.

## Phase 0 — Specs

- [x] PRD/TRD/PLAN/ROADMAP written; `INIT-OMNIDEPLOY-001` transitioned
  `proposed` → `planned`; Phase 1/2 RMIs filed.

## Phase 1 — Core VM Target

- [ ] `RMI-OMNIDEPLOY-002` `InstanceConfig`/`VMHealthCheck` config schema
- [ ] `RMI-OMNIDEPLOY-003` `target/lightsailinstance` package
- [ ] `RMI-OMNIDEPLOY-004` `deployLightsailInstance`: instance, keypair,
  firewall, redeploy-safe binary shipping (new dependency:
  `pulumi-command`'s `command/remote`)
- [ ] `RMI-OMNIDEPLOY-005` CLI wiring + unit tests

## Phase 2 — Parity with the Container Target

- [ ] `RMI-OMNIDEPLOY-006` Secrets via remote `.env` +
  `EnvironmentFile=`
- [ ] `RMI-OMNIDEPLOY-007` HTTP + systemd health-check verification
  variants

## Phase 3 — Migration (no new omnideploy RMIs)

Re-point `INIT-GROKIFYOMNIAGENT-001`'s Phase 3 ("Lightsail Instance
Deployment": `RMI-GROKIFYOMNIAGENT-007/008/009`) and Phase 4
("Documentation": `RMI-GROKIFYOMNIAGENT-011/012`) descriptions/acceptance
criteria at `omnideploy --target lightsail-instance` instead of the bash
scripts. Done directly against that initiative's existing RMIs once
Phase 1-2 land — the bash scripts stay as the validated reference
behavior the target reproduces, not something deleted in this pass.

## Verification

`go build ./...` / `go vet ./...` / `golangci-lint run` after each RMI,
same discipline as every other RMI this session. Real end-to-end
verification against an actual Lightsail instance is
`RMI-GROKIFYOMNIAGENT-006`/`009` (already queued), once Phase 1-2 land —
not spun up speculatively mid-implementation.
