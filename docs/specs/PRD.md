# PRD: Lightsail Instance Target

## Problem

`omnideploy` has one AWS target today — `lightsail` — which deploys to
**Lightsail Container Service**. That target is right when a deployment
needs to be a signed, multi-arch, publicly-pullable artifact (this is
what `omniagent-discord` uses), but it has one structural cost problem
for a different, equally common case: **Lightsail Container Service has
no persistent disk.** Every local file is wiped on redeploy. Getting real
persistence there means adding an external Lightsail managed database
(+$15/mo minimum) on top of the container price.

Some deployments — a personal Discord bot with local SQLite memory, a
scheduled report generator, anything that's cheaper and simpler as a bare
binary than a container — need exactly that persistence, and need it at
minimal, capped cost. `grokify-omniagent` is the first concrete case: its
own initiative currently plans to get persistence via **hand-rolled bash
scripts** (`setup.sh` + `deploy.sh`) driving a bare Lightsail VM instance
by hand. That works, but it's a one-off, untested-by-CI, per-repo script
pair — exactly the kind of thing `omnideploy` exists to replace with a
declarative, reusable target.

## Product

A second AWS target, `lightsail-instance`, alongside the existing
`lightsail` (container) target — not specific to Discord, grokify, or any
one runtime. It deploys a pre-built binary to a Lightsail VM instance as
a systemd service, with the instance's included persistent disk as the
storage layer, SSH-only ingress by default, and the same declarative
`omnideploy preview`/`up` workflow the container target already has —
including post-deploy health verification (`RMI-OMNIDEPLOY-001`).

## Principles

1. **Reproduce the proven reference implementation, don't redesign it.**
   `grokify-omniagent/deploy/lightsail/{setup,deploy}.sh` is real,
   working behavior: cross-compile locally, ship the binary + config over
   SSH, install/restart via systemd, secrets via an `EnvironmentFile=`.
   The target automates exactly this, declaratively.
2. **Redeploys must be idempotent, not instance-recreating.** The
   persistent disk is the entire point; any design that risks recreating
   the instance on a routine binary update defeats the purpose.
3. **Container and instance targets share everything that can be
   shared** — secret resolution, region/tags/environment config, the
   post-deploy verification *pattern* — and diverge only where the
   underlying infrastructure genuinely differs (how the artifact is
   shipped, how health is checked).
4. **Minimal, capped cost is a first-class constraint**, not an
   afterthought: default to the smallest viable Lightsail bundle, no
   required add-on services (managed DB, load balancer) unless the
   caller opts in.

## Users

Any `omnideploy` caller that wants a cheap, persistent, single-instance
deployment: `grokify-omniagent` first, any future low-cost
OmniAgent-family or internal tool deployment after.

## Out of scope (v1)

- Multi-instance / autoscaling (a single persistent-disk VM is
  inherently one instance; horizontal scaling is what the container
  target is for).
- Non-Lightsail VM providers (EC2, DigitalOcean Droplets) — the `target`
  abstraction supports adding them later without touching this target.
- Automated OS patching/upgrades of the instance itself.
