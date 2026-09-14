# ROADMAP: Lightsail Instance Target

**Initiative:** `INIT-OMNIDEPLOY-001`
**Status:** Planned

> RMI IDs are stable and permanent. Commits implementing an item carry
> the trailer `Refs: RMI-OMNIDEPLOY-<NNN>`. Phase status is derived from
> member RMIs.

## Phase 1 — Core VM Target

**Theme:** Config schema, target registration, and the Pulumi resources
that make a first deploy and a redeploy both work.

- [ ] `RMI-OMNIDEPLOY-002` `InstanceConfig`/`VMHealthCheck` config schema
- [ ] `RMI-OMNIDEPLOY-003` `target/lightsailinstance` package
- [ ] `RMI-OMNIDEPLOY-004` `deployLightsailInstance` (instance, keypair,
  firewall, redeploy-safe binary shipping)
- [ ] `RMI-OMNIDEPLOY-005` CLI wiring and unit tests

## Phase 2 — Parity with the Container Target

**Theme:** Secrets injection and post-deploy health verification, matching
what the container target already has.

- [ ] `RMI-OMNIDEPLOY-006` Secrets via remote `.env` + systemd
  `EnvironmentFile=`
- [ ] `RMI-OMNIDEPLOY-007` HTTP and systemd health-check verification
  variants

## Phase 3 — Migration

Not new `omnideploy` RMIs — updates `INIT-GROKIFYOMNIAGENT-001`'s
existing Phase 3/4 RMIs to reference this target instead of hand-rolled
bash scripts, once Phase 1-2 above are complete.
