# gitops-bootstrap

> From zero to GitOps in one command — opinionated CLI to bootstrap a production-ready GitOps workflow on any Kubernetes cluster.

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)
[![Go 1.23+](https://img.shields.io/badge/go-1.23%2B-00ADD8.svg)](https://go.dev/)

---

## The Problem

Adopting GitOps is widely accepted as a best practice, but getting started is surprisingly painful:

- **Too many choices**: ArgoCD vs. Flux, Helm vs. Kustomize, Sealed Secrets vs. SOPS vs. External Secrets, mono-repo vs. multi-repo…
- **Hours of glue work**: Installing the controller, structuring the repo, wiring environments, setting up secrets management, configuring RBAC, health checks, notifications…
- **Tribal knowledge**: Most teams figure it out through blog posts, trial and error, and copying from previous jobs. The "right" structure lives in someone's head, not in code.
- **Inconsistency**: Every team in the organization ends up with a slightly different GitOps setup, making platform support harder.

**gitops-bootstrap** solves this by encoding opinionated best practices into a single CLI that scaffolds a complete, production-ready GitOps workflow in minutes.

## Core Principles

| Principle | Description |
|---|---|
| **Opinionated defaults, escape hatches everywhere** | Sensible defaults for 90% of cases, with every choice overridable via flags or config. |
| **Convention over configuration** | Standard directory structure and naming so teams across the org speak the same language. |
| **Day-2 ready** | Not just initial setup — includes patterns for promotions, rollbacks, secrets rotation, and drift detection. |
| **Cluster-agnostic** | Works on EKS, GKE, AKS, k3s, kind, or any conformant Kubernetes cluster. |
| **Idempotent** | Safe to re-run. Applies only what's missing, never overwrites existing customizations. |

## What It Sets Up

Running `gitops-bootstrap init` on a cluster produces:

### In the Cluster

- GitOps controller installed and configured (ArgoCD by default, Flux as alternative)
- Namespace structure for the controller and managed environments
- RBAC for the GitOps controller (least privilege)
- Secrets management operator (Sealed Secrets by default, External Secrets Operator as alternative)
- Optional: Ingress for the ArgoCD UI with TLS

### In the Git Repository

```
gitops-repo/
├── bootstrap/                      # One-time cluster setup
│   ├── argocd/                     # ArgoCD installation manifests
│   │   ├── namespace.yaml
│   │   ├── install.yaml            # Pinned ArgoCD version
│   │   └── argocd-cm.yaml         # Custom configuration
│   └── sealed-secrets/             # Secrets management setup
│       ├── namespace.yaml
│       └── controller.yaml
│
├── apps/                           # Application definitions (App of Apps pattern)
│   ├── _root.yaml                  # Root Application pointing to this directory
│   ├── my-api.yaml                 # ArgoCD Application for my-api
│   └── my-frontend.yaml           # ArgoCD Application for my-frontend
│
├── environments/                   # Per-environment configuration
│   ├── base/                       # Shared base manifests
│   │   ├── my-api/
│   │   │   ├── kustomization.yaml
│   │   │   ├── deployment.yaml
│   │   │   ├── service.yaml
│   │   │   └── hpa.yaml
│   │   └── my-frontend/
│   │       ├── kustomization.yaml
│   │       ├── deployment.yaml
│   │       └── service.yaml
│   │
│   ├── dev/                        # Dev overrides
│   │   ├── my-api/
│   │   │   ├── kustomization.yaml  # patches: replicas=1, resources=small
│   │   │   └── sealed-secret.yaml
│   │   └── kustomization.yaml
│   │
│   ├── staging/                    # Staging overrides
│   │   ├── my-api/
│   │   │   ├── kustomization.yaml
│   │   │   └── sealed-secret.yaml
│   │   └── kustomization.yaml
│   │
│   └── production/                 # Production overrides
│       ├── my-api/
│       │   ├── kustomization.yaml  # patches: replicas=3, resources=large, PDB
│       │   └── sealed-secret.yaml
│       └── kustomization.yaml
│
├── platform/                       # Platform-level services (managed by platform team)
│   ├── cert-manager/
│   ├── external-dns/
│   ├── ingress-nginx/
│   └── monitoring/
│
├── policies/                       # OPA/Kyverno policies (optional)
│   ├── require-labels.yaml
│   ├── disallow-latest-tag.yaml
│   └── require-resource-limits.yaml
│
└── docs/
    ├── ARCHITECTURE.md             # How this repo is structured and why
    ├── ADDING-AN-APP.md            # Step-by-step guide for dev teams
    ├── SECRETS.md                  # How to manage secrets in this setup
    └── TROUBLESHOOTING.md          # Common issues and fixes
```

## Features (Planned)

### Phase 1 — Core Bootstrap

- [x] Interactive CLI wizard: choose GitOps controller, secrets manager, environments
- [x] Non-interactive mode via flags/config file for CI/automation
- [x] Generate repo structure following App of Apps pattern
- [ ] Kustomize-based environment management (base + overlays)
- [ ] Scaffold example application with full environment promotion path
- [ ] Generate RBAC manifests for the GitOps controller
- [ ] Generate documentation (ARCHITECTURE.md, ADDING-AN-APP.md, SECRETS.md)
- [ ] Install ArgoCD (Helm-based, pinned version) with opinionated defaults
- [ ] Sealed Secrets setup with key generation and backup instructions

### Phase 2 — Flux Support & Advanced Secrets

- [ ] Flux CD as alternative GitOps controller
- [ ] External Secrets Operator integration (AWS Secrets Manager, Vault)
- [ ] SOPS-based secrets as a third option
- [ ] Multi-cluster support (hub-spoke model)
- [ ] Helm chart support alongside Kustomize

### Phase 3 — Day-2 Operations

- [ ] `gitops-bootstrap add-app <name>` — scaffold a new application with all environments
- [ ] `gitops-bootstrap add-env <name>` — add a new environment to all applications
- [ ] `gitops-bootstrap validate` — lint the repo structure and check for common mistakes
- [ ] `gitops-bootstrap diff <env-a> <env-b>` — compare configuration between environments
- [ ] `gitops-bootstrap promote <app> --from dev --to staging` — generate promotion PR
- [ ] Pre-commit hooks for manifest validation (kubeval, kustomize build)

### Phase 4 — Platform Integration

- [ ] Notifications setup (Slack, Teams) for sync status
- [ ] ArgoCD Notifications integration
- [ ] Image Updater configuration for automated image promotions
- [ ] GitHub Actions / GitLab CI workflow templates for PR-based promotions
- [ ] Webhook configuration for automatic sync on push
- [ ] Dashboard: terminal-based overview of sync status across environments

## Architecture

```mermaid
graph TD
    subgraph CLI["gitops-bootstrap CLI"]
        Wizard["<b>Wizard</b><br/>Interactive prompts<br/>Config file<br/>Flags"]
        Scaffolder["<b>Scaffolder</b><br/>Repo structure<br/>Templates<br/>Docs"]
        Installer["<b>Installer</b><br/>Helm Go SDK<br/>client-go<br/>Health checks"]

        Wizard --> Engine
        Scaffolder --> Engine
        Installer --> Engine

        Engine["<b>Template Engine</b><br/>Go text/template<br/>Embedded via embed.FS"]
    end

    Engine --> Git["Git Repo"]
    Engine --> K8s["K8s API"]
    Engine --> Helm["Helm<br/>(ArgoCD / Flux)"]
```

### Component Responsibilities

- **Wizard**: Gathers user preferences through interactive prompts or config file/flags. Validates input and produces a normalized configuration object.
- **Scaffolder**: Generates the Git repository structure from Go `text/template` templates. Handles directory creation, manifest rendering, and documentation generation. Idempotent — detects existing files and skips them.
- **Installer**: Applies bootstrap manifests to the cluster. Installs ArgoCD/Flux via the Helm Go SDK, sets up secrets management, configures RBAC. Includes health checks to verify successful installation.
- **Template Engine**: Go `text/template`-based rendering layer used by both Scaffolder and Installer. Templates are embedded in the binary via `embed.FS`. All generated manifests are templates with sensible defaults that can be customized.

## Tech Stack

| Component | Technology | Rationale |
|---|---|---|
| Language | **Go 1.23+** | Native to the Kubernetes ecosystem; compiles to a single static binary with zero runtime dependencies |
| CLI framework | **Cobra** | De facto standard for Go CLIs — used by kubectl, helm, gh, and most CNCF tools |
| Terminal UI | **Bubble Tea + Lip Gloss** (Charmbracelet) | Rich interactive TUIs with progress indicators, selection menus, and styled output |
| Template engine | **text/template** (stdlib) | Go's built-in template engine — no external dependency, sufficient for YAML manifest generation |
| K8s client | **client-go** (official) | The reference Kubernetes client library, always up-to-date with the latest API |
| Helm integration | **Helm Go SDK** (`helm.sh/helm/v3`) | Native library integration — no subprocess calls, no dependency on the user having Helm installed |
| Config file | **YAML** (`gopkg.in/yaml.v3`) + **go-playground/validator** | Natural format for K8s engineers, with struct tag-based validation |
| Build & release | **GoReleaser** | Cross-compilation, GitHub releases, Homebrew tap, Docker images — all in one workflow |
| Testing | **testing** (stdlib) + **testify** | Standard Go testing with `t.TempDir()` for repo scaffolding tests |
| Linting | **golangci-lint** | Meta-linter aggregating 50+ linters in a single fast run |

## Planned CLI Interface

```bash
# Interactive wizard (recommended for first-time setup)
gitops-bootstrap init

# Non-interactive with flags
gitops-bootstrap init \
  --controller argocd \
  --controller-version 2.13.1 \
  --secrets sealed-secrets \
  --environments dev,staging,production \
  --repo-path ./gitops-repo \
  --cluster-context prod-eu-west-1

# From config file (for reproducibility / team standardization)
gitops-bootstrap init --config bootstrap-config.yaml

# Add a new application to the existing structure
gitops-bootstrap add-app my-new-service \
  --type deployment \
  --port 8080 \
  --has-ingress \
  --has-hpa

# Add a new environment
gitops-bootstrap add-env qa --base staging

# Validate repo structure
gitops-bootstrap validate ./gitops-repo

# Compare environments
gitops-bootstrap diff dev production --app my-api

# Promote an app between environments
gitops-bootstrap promote my-api --from staging --to production
```

### Example Config File

```yaml
# bootstrap-config.yaml
controller:
  type: argocd
  version: "2.13.1"
  ingress:
    enabled: true
    host: argocd.internal.company.com
    tls: true

secrets:
  type: sealed-secrets

environments:
  - name: dev
    auto_sync: true
    prune: true
  - name: staging
    auto_sync: true
    prune: false
  - name: production
    auto_sync: false      # manual sync for production
    prune: false
    require_pr: true

applications:
  - name: example-api
    type: deployment
    port: 8080
    replicas:
      dev: 1
      staging: 2
      production: 3
    has_ingress: true
    has_hpa: true
    hpa:
      min_replicas: 2
      max_replicas: 10
      target_cpu: 70

platform_services:
  cert_manager: true
  external_dns: false
  ingress_nginx: true
  monitoring: false

policies:
  enabled: true
  engine: kyverno
```

### Interactive Wizard Flow (Planned)

```
$ gitops-bootstrap init

  ╭─────────────────────────────────────────╮
  │        gitops-bootstrap v0.1.0          │
  │   From zero to GitOps in one command    │
  ╰─────────────────────────────────────────╯

  ? Select GitOps controller:
    ❯ ArgoCD (recommended)
      Flux CD

  ? ArgoCD version: (2.13.1)

  ? Select secrets management:
    ❯ Sealed Secrets (simple, self-contained)
      External Secrets Operator (AWS SM, Vault, etc.)
      SOPS (git-native encryption)

  ? Environments to create: (dev, staging, production)

  ? Scaffold an example application? (Y/n)

  ? Target repo path: (./gitops-repo)

  ? Cluster context: (current: prod-eu-west-1)

  ⠸ Installing ArgoCD v2.13.1...          ✓
  ⠸ Setting up Sealed Secrets...          ✓
  ⠸ Generating repo structure...          ✓
  ⠸ Creating example application...       ✓
  ⠸ Generating documentation...           ✓
  ⠸ Verifying cluster health...           ✓

  ✓ GitOps bootstrap complete!

  Next steps:
    1. cd ./gitops-repo && git init && git add -A && git commit -m "feat: initial gitops structure"
    2. Push to your Git provider
    3. ArgoCD UI: https://argocd.internal.company.com
    4. Read docs/ADDING-AN-APP.md to onboard your first real application
```

## Design Decisions

### Why App of Apps?

The [App of Apps pattern](https://argo-cd.readthedocs.io/en/stable/operator-manual/cluster-bootstrapping/#app-of-apps-pattern) is the most widely adopted approach for managing multiple applications with ArgoCD. A single root Application watches the `apps/` directory, and each file there defines an Application pointing to its environment-specific manifests. This provides:
- Single entry point for the entire cluster state.
- Self-service: dev teams add an Application YAML to onboard.
- Declarative: the list of applications is version-controlled.

### Why Kustomize over Helm for app manifests?

- Kustomize works with plain YAML — no templating language to learn.
- Overlays make environment differences explicit and auditable.
- Better for GitOps: `kustomize build` output is deterministic.
- Helm is used for installing third-party software (ArgoCD, cert-manager), not for application manifests.

### Why Sealed Secrets as default?

- Zero external dependencies (no Vault, no cloud provider secrets manager).
- Works on any cluster, including local development (kind, k3s).
- Encrypted secrets can live in Git safely.
- Simple mental model: `kubeseal` encrypts, controller decrypts.
- External Secrets Operator is offered as an alternative for teams already using AWS Secrets Manager, Vault, etc.

### Why Go?

- **Native to the Kubernetes ecosystem**: Go is the language behind Kubernetes, ArgoCD, Flux, Helm, and most CNCF tooling. Contributors from this ecosystem already write Go.
- **Single binary distribution**: `curl`, `chmod +x`, done. No runtime, no interpreter, no virtual environments. Works seamlessly in CI pipelines, air-gapped environments, and scratch containers.
- **First-class Kubernetes and Helm libraries**: `client-go` and the Helm Go SDK are the reference implementations — always up-to-date, fully featured, and well-documented. No subprocess wrappers needed.
- **Embedded templates**: Go's `embed.FS` allows shipping all manifest templates inside the binary itself, eliminating the need to manage template files on disk.
- **Cross-compilation**: A single `goreleaser` config produces binaries for Linux, macOS, and Windows (amd64/arm64), plus Homebrew formulas and Docker images.

### Why not just use a Helm chart for everything?

Helm charts are great for distributing reusable software, but they're a poor fit for representing an organization's unique GitOps repository structure. Each team's environments, naming conventions, and promotion workflows are different. A scaffolding tool that generates plain YAML with Kustomize overlays gives teams full ownership and visibility over their manifests, without the abstraction layer of `values.yaml`.

## Project Structure (Planned)

```
gitops-bootstrap/
├── cmd/
│   └── gitops-bootstrap/
│       └── main.go                 # Entrypoint
├── internal/
│   ├── wizard/
│   │   ├── wizard.go               # Interactive prompts (Bubble Tea)
│   │   └── config.go               # Config file parsing & validation
│   ├── scaffolder/
│   │   ├── repo.go                 # Repo structure generation
│   │   ├── apps.go                 # Application manifest generation
│   │   ├── environments.go         # Environment overlay generation
│   │   └── docs.go                 # Documentation generation
│   ├── installer/
│   │   ├── argocd.go               # ArgoCD installation via Helm Go SDK
│   │   ├── flux.go                 # Flux installation
│   │   ├── secrets.go              # Sealed Secrets / ESO setup
│   │   └── health.go               # Post-install health checks
│   ├── templates/                  # Go text/template files (embedded via embed.FS)
│   │   ├── argocd/
│   │   ├── flux/
│   │   ├── apps/
│   │   ├── environments/
│   │   ├── platform/
│   │   ├── policies/
│   │   ├── docs/
│   │   └── embed.go                # //go:embed directives
│   └── models/
│       └── config.go               # Configuration structs with validation tags
├── pkg/
│   └── kube/
│       └── client.go               # Kubernetes client helpers (client-go)
├── tests/
│   ├── wizard_test.go
│   ├── scaffolder_test.go
│   ├── installer_test.go
│   └── testdata/
│       └── sample-configs/
├── examples/
│   ├── bootstrap-config.yaml       # Example config file
│   └── github-actions-promote.yml  # Example promotion workflow
├── .goreleaser.yaml                # Cross-compilation & release config
├── go.mod
├── go.sum
├── LICENSE
└── README.md
```

## Related & Prior Art

| Tool | Comparison |
|---|---|
| [ArgoCD Autopilot](https://github.com/argoproj-labs/argocd-autopilot) | Go-based, ArgoCD-only, opinionated but less flexible. Inspiration for the App of Apps approach. |
| [Flux Bootstrap](https://fluxcd.io/flux/cmd/flux_bootstrap/) | Built into Flux CLI, Flux-only, focused on controller installation. |
| [Kubefirst](https://kubefirst.io/) | Full platform (CI/CD, secrets, IDP), heavier scope, includes cloud provisioning. |
| [Backstage](https://backstage.io/) | Developer portal with scaffolding capabilities, much larger scope. |

**gitops-bootstrap** differentiates by being lightweight, controller-agnostic, and focused exclusively on the GitOps repo structure + controller setup — without trying to be an entire platform.

## Development

### Prerequisites

- [Go 1.23+](https://go.dev/dl/)
- [Docker](https://docs.docker.com/get-docker/)
- [kind](https://kind.sigs.k8s.io/) — `go install sigs.k8s.io/kind@latest`
- [kubectl](https://kubernetes.io/docs/tasks/tools/)

### Build

```bash
go build -o gitops-bootstrap ./cmd/gitops-bootstrap/
```

### Test

```bash
go test ./...
```

### Local Kubernetes Cluster

Scripts in `hack/` manage a local [kind](https://kind.sigs.k8s.io/) cluster preconfigured with ingress port mappings (80/443) for testing ArgoCD UI access:

```bash
# Create cluster (idempotent — skips if already exists)
./hack/setup-kind.sh            # default name: gitops-dev
./hack/setup-kind.sh my-cluster # custom name

# Delete cluster
./hack/teardown-kind.sh
./hack/teardown-kind.sh my-cluster
```

### Quick Smoke Test

```bash
# Build and run the wizard in non-interactive mode
go run ./cmd/gitops-bootstrap/ init \
  --controller argocd \
  --secrets sealed-secrets \
  --environments dev,staging,production \
  --repo-path ./test-repo \
  --cluster-context kind-gitops-dev
```

## Contributing

Contributions are welcome. Please open an issue to discuss your idea before submitting a PR.

This project follows:
- [Conventional Commits](https://www.conventionalcommits.org/) for commit messages.
- Trunk-based development with short-lived feature branches.
- All code must pass `golangci-lint run` and `go test ./...` before merge.

## License

[Apache License 2.0](LICENSE)
