# gostrap

[![CI](https://github.com/y0s3ph/gostrap/actions/workflows/ci.yml/badge.svg)](https://github.com/y0s3ph/gostrap/actions/workflows/ci.yml)
[![Go 1.24+](https://img.shields.io/badge/go-1.24%2B-00ADD8.svg)](https://go.dev/)
[![LinkedIn](https://img.shields.io/badge/LinkedIn-jph91-0A66C2.svg?logo=linkedin)](https://www.linkedin.com/in/jph91/)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)

<p align="center">
  <img src="gostrap.png" alt="gostrap logo" width="480">
</p>

From zero to GitOps in one command — opinionated CLI to bootstrap a production-ready GitOps workflow on any Kubernetes cluster.

<p align="center">
  <img src="demo.gif" alt="gostrap demo" width="700">
</p>

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [The Problem](#the-problem)
- [Core Principles](#core-principles)
- [What It Sets Up](#what-it-sets-up)
  - [In the Cluster](#in-the-cluster)
  - [In the Git Repository](#in-the-git-repository)
- [How It Fits In Your Workflow](#how-it-fits-in-your-workflow)
- [Roadmap](#roadmap)
- [Architecture](#architecture)
  - [Component Responsibilities](#component-responsibilities)
- [Tech Stack](#tech-stack)
- [CLI Interface](#cli-interface)
- [Design Decisions](#design-decisions)
  - [Why App of Apps (ArgoCD) / Kustomization chain (Flux)?](#why-app-of-apps-argocd--kustomization-chain-flux)
  - [ArgoCD vs Flux: which one should I choose?](#argocd-vs-flux-which-one-should-i-choose)
  - [Why Kustomize as default (with Helm as option)?](#why-kustomize-as-default-with-helm-as-option)
  - [Why Sealed Secrets as default?](#why-sealed-secrets-as-default)
  - [Secrets management: scalability and limitations](#secrets-management-scalability-and-limitations)
  - [Why Go?](#why-go)
  - [Why not just use a Helm chart for everything?](#why-not-just-use-a-helm-chart-for-everything)
- [Project Structure (Planned)](#project-structure-planned)
- [Related & Prior Art](#related--prior-art)
- [Development](#development)
  - [Prerequisites](#prerequisites)
  - [Build](#build)
  - [Test](#test)
  - [Local Kubernetes Cluster](#local-kubernetes-cluster)
  - [Quick Smoke Test](#quick-smoke-test)
- [Contributing](#contributing)
- [License](#license)

---

## Installation

### From source (recommended for now)

```bash
go install github.com/y0s3ph/gostrap/cmd/gostrap@latest
```

Or clone and build:

```bash
git clone https://github.com/y0s3ph/gostrap.git
cd gostrap
go build -o gostrap ./cmd/gostrap/
sudo mv gostrap /usr/local/bin/
```

### Prerequisites

- [Go 1.24+](https://go.dev/dl/) (for building from source)
- [kubectl](https://kubernetes.io/docs/tasks/tools/) (for cluster installation)
- A Kubernetes cluster (or [kind](https://kind.sigs.k8s.io/) for local testing)

## Quick Start

```bash
# 1. Bootstrap a GitOps repo + install the controller on your cluster
gostrap init

# 2. Add applications to the repo
gostrap add-app payments --port 3000 --repo-path ./gitops-repo
gostrap add-app frontend --port 3000 --repo-path ./gitops-repo

# 3. Add a new environment (creates overlays for all existing apps)
gostrap add-env qa --auto-sync --prune --repo-path ./gitops-repo

# 4. Commit and push — the GitOps controller picks up the rest
cd gitops-repo
git init && git add -A && git commit -m "feat: initial gitops structure"
git remote add origin <your-repo-url>
git push -u origin main
```

The interactive wizard guides you through every choice:

```
$ gostrap init

  gostrap   From zero to GitOps in one command

  ? Select GitOps controller:    ArgoCD / Flux CD
  ? Select secrets management:   Sealed Secrets / ESO / SOPS
  ? Application manifest format: Kustomize / Helm
  ? Environments to create:      dev, staging, production
  ? Scaffold an example app?     Yes
  ? Target repo path:            ./gitops-repo
  ? Cluster context:             kind-gitops-dev

  ✓ ArgoCD installed and ready
  ✓ Sealed Secrets ready
  ✓ Repository structure generated
```

For CI/automation, skip the wizard entirely:

```bash
gostrap init \
  --controller argocd \
  --secrets sealed-secrets \
  --manifest-type kustomize \
  --environments dev,staging,production \
  --repo-path ./gitops-repo \
  --cluster-context prod-eu-west-1
```

## The Problem

Adopting GitOps is widely accepted as a best practice, but getting started is surprisingly painful:

- **Too many choices**: ArgoCD vs. Flux, Helm vs. Kustomize, Sealed Secrets vs. SOPS vs. External Secrets, mono-repo vs. multi-repo…
- **Hours of glue work**: Installing the controller, structuring the repo, wiring environments, setting up secrets management, configuring RBAC, health checks, notifications…
- **Tribal knowledge**: Most teams figure it out through blog posts, trial and error, and copying from previous jobs. The "right" structure lives in someone's head, not in code.
- **Inconsistency**: Every team in the organization ends up with a slightly different GitOps setup, making platform support harder.

**gostrap** solves this by encoding opinionated best practices into a single CLI that scaffolds a complete, production-ready GitOps workflow in minutes.

## Core Principles

| Principle | Description |
|---|---|
| **Opinionated defaults, escape hatches everywhere** | Sensible defaults for 90% of cases, with every choice overridable via flags or config. |
| **Convention over configuration** | Standard directory structure and naming so teams across the org speak the same language. |
| **Day-2 ready** | Not just initial setup — includes patterns for promotions, rollbacks, secrets rotation, and drift detection. |
| **Cluster-agnostic** | Works on EKS, GKE, AKS, k3s, kind, or any conformant Kubernetes cluster. |
| **Idempotent** | Safe to re-run. Applies only what's missing, never overwrites existing customizations. |

## What It Sets Up

Running `gostrap init` on a cluster produces:

### In the Cluster

- GitOps controller installed and configured (ArgoCD or Flux CD — your choice)
- Namespace structure for the controller and managed environments
- RBAC for the GitOps controller (least privilege)
- Secrets management (Sealed Secrets, External Secrets Operator, or SOPS with age)
- Optional: Ingress for the controller UI with TLS (ArgoCD)

### In the Git Repository

```
gitops-repo/
├── bootstrap/                      # One-time cluster setup
│   ├── argocd/                     # ArgoCD installation manifests (if ArgoCD selected)
│   │   ├── namespace.yaml
│   │   ├── kustomization.yaml     # Pinned ArgoCD version + patches
│   │   └── appproject-default.yaml
│   ├── flux-system/                # Flux installation manifests (if Flux selected)
│   │   ├── namespace.yaml
│   │   ├── kustomization.yaml     # Pinned Flux version
│   │   └── gotk-sync.yaml         # GitRepository + root Kustomization
│   ├── sealed-secrets/             # Sealed Secrets (if selected)
│   │   ├── kustomization.yaml
│   │   └── sealedsecret-example.yaml
│   ├── external-secrets/           # ESO (if selected)
│   │   ├── kustomization.yaml
│   │   └── clustersecretstore-example.yaml
│   └── sops/                       # SOPS (if selected)
│       ├── kustomization.yaml
│       └── secret-example.yaml
│
├── apps/                           # Controller-specific app definitions
│   ├── _root.yaml                  # Root Application (ArgoCD) or GitRepository+Kustomization (Flux)
│   ├── my-api-dev.yaml            # Per-app per-env definition
│   └── my-api-staging.yaml
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
├── docs/
│   ├── ARCHITECTURE.md             # How this repo is structured and why
│   ├── ADDING-AN-APP.md            # Step-by-step guide for dev teams
│   ├── SECRETS.md                  # How to manage secrets in this setup
│   └── TROUBLESHOOTING.md          # Common issues and fixes
│
├── .gostrap.yaml                   # Repo config (used by add-app, add-env)
└── .pre-commit-config.yaml         # Pre-commit hooks (YAML lint, kubeconform, gostrap validate)
```

## How It Fits In Your Workflow

gostrap generates a **configuration-only** repository. Your application source code lives in separate repos — this is the standard GitOps separation of concerns:

```mermaid
graph LR
    subgraph Source["Source Code Repos"]
        API["my-api<br/><i>src/, Dockerfile</i>"]
        FE["my-frontend<br/><i>src/, Dockerfile</i>"]
    end

    subgraph CI["CI Pipeline"]
        Build["Build & Push<br/>container image"]
    end

    Registry["Container<br/>Registry"]

    subgraph GitOps["GitOps Repo <i>(generated by gostrap)</i>"]
        Manifests["K8s Manifests<br/><i>environments/, apps/</i>"]
    end

    K8s["Kubernetes<br/>Cluster"]

    API --> Build
    FE --> Build
    Build --> Registry
    Registry -- "update image tag<br/>(PR or automation)" --> Manifests
    Manifests -- "ArgoCD / Flux sync" --> K8s
```

| Repository | Contains | Who owns it |
|---|---|---|
| `my-api`, `my-frontend` | Application source code, Dockerfile, tests, CI pipeline | Development teams |
| `gitops-repo` (generated by gostrap) | Kubernetes manifests, environment overlays, controller config | Platform / DevOps team |
| `gostrap` (this repo) | The CLI tool itself | Open source |

The deployment flow is: developers push code → CI builds and pushes a container image → the image tag is updated in the gitops-repo (via PR or automation) → the GitOps controller (ArgoCD or Flux) detects the change and syncs to the cluster.

This separation gives you auditable deployments (every cluster change is a Git commit), rollback via `git revert`, and clear ownership boundaries between dev and ops.

## Roadmap

> Track progress and detailed issues on the [gostrap roadmap](https://github.com/users/y0s3ph/projects/1) board.

| Phase | Milestone | Status | Summary |
|---|---|---|---|
| **1 — Core Bootstrap** | [v0.1.0](https://github.com/y0s3ph/gostrap/milestone/1?closed=1) | Done | Interactive wizard, repo scaffolding, ArgoCD installer, Sealed Secrets, documentation generation |
| **2 — Flux & Advanced Secrets** | [v0.2.0](https://github.com/y0s3ph/gostrap/milestone/2?closed=1) | Done | Flux CD, External Secrets Operator, SOPS, Helm chart support |
| **3 — Day-2 Operations** | [v0.3.0](https://github.com/y0s3ph/gostrap/milestone/3?closed=1) | Done | `add-app`, `add-env`, `validate`, `diff`, `promote` commands, pre-commit hooks |
| **4 — Platform Integration** | [v0.4.0](https://github.com/y0s3ph/gostrap/milestone/4) | Planned | Multi-cluster hub-spoke, Notifications, Image Updater, CI workflow templates, webhooks, terminal dashboard |

## Architecture

```mermaid
graph TD
    subgraph CLI["gostrap CLI"]
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
| Language | **Go 1.24+** | Native to the Kubernetes ecosystem; compiles to a single static binary with zero runtime dependencies |
| CLI framework | **Cobra** | De facto standard for Go CLIs — used by kubectl, helm, gh, and most CNCF tools |
| Terminal UI | **Bubble Tea + Lip Gloss** (Charmbracelet) | Rich interactive TUIs with progress indicators, selection menus, and styled output |
| Template engine | **text/template** (stdlib) | Go's built-in template engine — no external dependency, sufficient for YAML manifest generation |
| K8s client | **client-go** (official) | The reference Kubernetes client library, always up-to-date with the latest API |
| Helm integration | **Helm Go SDK** (`helm.sh/helm/v3`) | Native library integration — no subprocess calls, no dependency on the user having Helm installed |
| Config file | **YAML** (`gopkg.in/yaml.v3`) + **go-playground/validator** | Natural format for K8s engineers, with struct tag-based validation |
| Build & release | **GoReleaser** | Cross-compilation, GitHub releases, Homebrew tap, Docker images — all in one workflow |
| Testing | **testing** (stdlib) + **testify** | Standard Go testing with `t.TempDir()` for repo scaffolding tests |
| Linting | **golangci-lint** | Meta-linter aggregating 50+ linters in a single fast run |

## CLI Interface

```bash
# Interactive wizard (recommended for first-time setup)
gostrap init

# Non-interactive with flags
gostrap init \
  --controller argocd \
  --controller-version 2.13.1 \
  --secrets sealed-secrets \
  --manifest-type kustomize \
  --environments dev,staging,production \
  --repo-path ./gitops-repo \
  --cluster-context prod-eu-west-1

# From config file (for reproducibility / team standardization)
gostrap init --config bootstrap-config.yaml

# Add a new application (interactive — prompts for name and port)
gostrap add-app --repo-path ./gitops-repo

# Add a new application (non-interactive)
gostrap add-app payments --port 3000 --repo-path ./gitops-repo

# Add a new environment (interactive — prompts for name and settings)
gostrap add-env --repo-path ./gitops-repo

# Add a new environment (non-interactive)
gostrap add-env qa --auto-sync --prune --repo-path ./gitops-repo

# Validate repo structure
gostrap validate ./gitops-repo

# Compare environments
gostrap diff dev staging
gostrap diff dev production --app my-api
gostrap diff staging production --repo-path ./gitops-repo

# Promote an app between environments
gostrap promote my-api --from dev --to staging
gostrap promote my-api --from staging --to production --dry-run
gostrap promote --from dev --to staging --yes             # all apps, skip confirm
```

### Pre-commit Hooks

Every repository scaffolded by gostrap includes a `.pre-commit-config.yaml` with three layers of validation:

| Hook | What it does |
|------|-------------|
| **check-yaml** | YAML syntax validation (skips Go/Helm templates) |
| **kubeconform** | Kubernetes manifest schema validation (skips CRDs) |
| **gostrap validate** | Full structural integrity check (config, overlays, app definitions) |

Plus `trailing-whitespace`, `end-of-file-fixer`, and `check-merge-conflict` for general hygiene.

**Setup** (in the generated GitOps repo):

```bash
pip install pre-commit
pre-commit install

# Run manually on all files
pre-commit run --all-files
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

manifest_type: kustomize   # or "helm"

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

## Design Decisions

### Why App of Apps (ArgoCD) / Kustomization chain (Flux)?

For **ArgoCD**, gostrap uses the [App of Apps pattern](https://argo-cd.readthedocs.io/en/stable/operator-manual/cluster-bootstrapping/#app-of-apps-pattern): a root Application watches `apps/`, and each file defines an Application pointing to environment overlays. For **Flux**, the equivalent is a root `Kustomization` CRD that watches `apps/`, where each file is a Flux `Kustomization` pointing to an overlay. Both approaches provide:
- Single entry point for the entire cluster state.
- Self-service: dev teams add a YAML to `apps/` to onboard.
- Declarative: the list of applications is version-controlled.

### ArgoCD vs Flux: which one should I choose?

gostrap supports both controllers as first-class options. ArgoCD is marked as "recommended" in the wizard because it offers a gentler onboarding experience, but Flux is equally well supported.

|  | **ArgoCD** | **Flux CD** |
|---|---|---|
| **CNCF status** | Graduated | Graduated |
| **Web UI** | Built-in dashboard with sync status, diff viewer, and rollback | No native UI (add [Weave GitOps](https://github.com/weaveworks/weave-gitops) or similar) |
| **Mental model** | One `Application` CRD = one deployed app, visual feedback | Modular controllers (source, kustomize, helm, notification) composed via CRDs |
| **RBAC** | Granular: SSO/OIDC, projects, per-repo/per-cluster policies | Delegates to Kubernetes RBAC; multi-tenancy via namespaced `Kustomization` |
| **Helm support** | Renders charts server-side; supports `values.yaml` overlays | `HelmRelease` CRD with dependency management and automated upgrades |
| **Multi-cluster** | Centralized hub managing remote clusters from a single UI | Agent-per-cluster (decentralized); each cluster reconciles independently |
| **Notifications** | Built-in notification engine (Slack, webhook, GitHub) | Separate `notification-controller` with provider CRDs |
| **Image automation** | Separate [Image Updater](https://argocd-image-updater.readthedocs.io/) project | Built-in `image-reflector-controller` + `image-automation-controller` |
| **Best for** | Teams wanting visual operations, onboarding newcomers to GitOps | Teams preferring pure Git workflows, no UI dependency, or advanced automation |

**TL;DR**: Choose **ArgoCD** if you value a web UI and visual feedback. Choose **Flux** if you prefer everything-as-code with no UI dependency and want tighter integration with Helm and image automation.

### Why Kustomize as default (with Helm as option)?

gostrap supports both **Kustomize** (default) and **Helm** for application manifests. Kustomize is the default because:

- Works with plain YAML — no templating language to learn.
- Overlays make environment differences explicit and auditable.
- Better for GitOps: `kustomize build` output is deterministic.

However, teams already invested in Helm can choose `--manifest-type helm` to generate a standard chart structure with per-environment `values.yaml` files. ArgoCD uses `source.helm` with `valueFiles`; Flux uses `HelmRelease` CRDs.

### Why Sealed Secrets as default?

- Zero external dependencies (no Vault, no cloud provider secrets manager).
- Works on any cluster, including local development (kind, k3s).
- Encrypted secrets can live in Git safely.
- Simple mental model: `kubeseal` encrypts, controller decrypts.
- External Secrets Operator is offered as an alternative for teams already using AWS Secrets Manager, Vault, etc.

### Secrets management: scalability and limitations

Each option fits different team sizes and needs:

| Option | Small team (1–5) | Growing team (5–20) | Large team (20+) |
|--------|------------------|---------------------|------------------|
| **Sealed Secrets** | Ideal | Good (key lives in cluster; no manual distribution) | Good, but no per-secret audit trail |
| **SOPS (age)** | Ideal | Usable; key rotation and onboarding are manual | **Not recommended** — shared key, no access control |
| **SOPS (KMS)** | Optional | **Ideal** — IAM controls access, no key distribution | **Ideal** — revoke access when people leave |
| **External Secrets Operator** | Optional | **Ideal** — central vault, dynamic secrets | **Best** — audit, leases, fine-grained policies |

**SOPS with a single age key** does not scale well: everyone shares the same private key, so when someone leaves you must rotate the key and re-encrypt all secrets; there is no per-environment or per-secret access control. For growing teams, prefer **External Secrets Operator** (Vault, AWS Secrets Manager, etc.) or **SOPS with KMS** (AWS KMS, GCP KMS, Azure Key Vault), where access is managed via IAM and key distribution is not needed.

Planned improvement: [SOPS with KMS support (#28)](https://github.com/y0s3ph/gostrap/issues/28) — use cloud KMS instead of (or in addition to) age for scalable, IAM-based access.

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
gostrap/
├── cmd/
│   └── gostrap/
│       └── main.go                 # Entrypoint
├── internal/
│   ├── cli/
│   │   ├── root.go                 # Root command registration
│   │   ├── init.go                 # gostrap init command
│   │   ├── add_app.go              # gostrap add-app command
│   │   ├── add_env.go              # gostrap add-env command
│   │   ├── validate.go             # gostrap validate command
│   │   ├── diff.go                 # gostrap diff command
│   │   └── promote.go              # gostrap promote command
│   ├── validator/
│   │   └── validator.go            # Repo structure validation engine
│   ├── differ/
│   │   └── differ.go               # Environment diff engine (LCS-based)
│   ├── promoter/
│   │   └── promoter.go             # Environment promotion engine
│   ├── config/
│   │   └── config.go               # .gostrap.yaml persistence (save/load repo config)
│   ├── wizard/
│   │   ├── wizard.go               # Interactive prompts (Bubble Tea)
│   │   └── config.go               # Config file parsing & validation
│   ├── scaffolder/
│   │   ├── repo.go                 # Repo structure generation
│   │   ├── apps.go                 # Application manifest generation
│   │   ├── environments.go         # Environment overlay generation
│   │   ├── env.go                  # Single-environment scaffolding (add-env)
│   │   ├── docs.go                 # Documentation generation
│   │   └── hooks.go                # Pre-commit config generation
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
│   │   ├── hooks/                  # Pre-commit hook templates
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

**gostrap** differentiates by being lightweight, controller-agnostic, and focused exclusively on the GitOps repo structure + controller setup — without trying to be an entire platform.

## Development

> For contributors and local development. If you just want to use gostrap, see [Installation](#installation).

### Prerequisites

- [Go 1.24+](https://go.dev/dl/)
- [Docker](https://docs.docker.com/get-docker/)
- [kind](https://kind.sigs.k8s.io/) — `go install sigs.k8s.io/kind@latest`
- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- [golangci-lint](https://golangci-lint.run/welcome/install/) — `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`

### Build

```bash
go build -o gostrap ./cmd/gostrap/
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
go run ./cmd/gostrap/ init \
  --controller argocd \
  --secrets sealed-secrets \
  --manifest-type kustomize \
  --environments dev,staging,production \
  --repo-path ./test-repo \
  --cluster-context kind-gitops-dev

# Add a new application to the repo
go run ./cmd/gostrap/ add-app payments --port 3000 --repo-path ./test-repo
```

## Contributing

Contributions are welcome! Please read the [Contributing Guide](CONTRIBUTING.md) for setup instructions, code conventions, and PR guidelines.

Looking for a place to start? Check the [good first issues](https://github.com/y0s3ph/gostrap/contribute).

**TL;DR**: fork → branch → code → `make test && make lint` → PR. Follow [Conventional Commits](https://www.conventionalcommits.org/).

## License

[Apache License 2.0](LICENSE)
