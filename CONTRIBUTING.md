# Contributing to gostrap

Thanks for your interest in contributing! This guide will help you get started.

## Getting Started

1. **Fork** the repository and clone your fork
2. **Create a branch** from `main` for your change
3. **Make your changes**, following the guidelines below
4. **Open a Pull Request** against `main`

### Prerequisites

- [Go 1.24+](https://go.dev/dl/)
- [Docker](https://docs.docker.com/get-docker/)
- [kind](https://kind.sigs.k8s.io/) — `go install sigs.k8s.io/kind@latest`
- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- [golangci-lint](https://golangci-lint.run/welcome/install/)

### Build and Test

```bash
# Build
go build -o gostrap ./cmd/gostrap/

# Run tests
go test -race ./...

# Run linter
golangci-lint run

# Or use the Makefile
make build
make test
make lint
```

### Local Kubernetes Cluster

Scripts in `hack/` manage a local kind cluster for testing:

```bash
./hack/setup-kind.sh       # Create cluster
./hack/teardown-kind.sh    # Delete cluster
```

## How to Contribute

### Report a Bug

Open an [issue](https://github.com/y0s3ph/gostrap/issues/new) with:
- What you expected to happen
- What actually happened
- Steps to reproduce
- Go version, OS, and Kubernetes version

### Suggest a Feature

Open an [issue](https://github.com/y0s3ph/gostrap/issues/new) to discuss your idea before writing code. This saves everyone time and ensures the feature fits the project direction.

### Submit a Pull Request

1. **One PR per feature/fix** — keep changes focused
2. **Write tests** — new features need tests, bug fixes need regression tests
3. **Update docs** — if your change affects CLI behavior, update the README
4. **All checks must pass** — `golangci-lint run` and `go test -race ./...`

## Code Guidelines

### Project Structure

```
internal/
├── cli/          # Cobra commands (init, add-app, add-env)
├── config/       # .gostrap.yaml persistence
├── wizard/       # Interactive prompts (Charmbracelet huh)
├── scaffolder/   # Repo structure generation
├── installer/    # Cluster installation (ArgoCD, Flux, secrets)
├── models/       # Configuration structs
└── templates/    # Go text/template files (embedded via embed.FS)
```

### Conventions

- **Commit messages**: follow [Conventional Commits](https://www.conventionalcommits.org/)
  - `feat:` new feature
  - `fix:` bug fix
  - `docs:` documentation only
  - `test:` adding or updating tests
  - `refactor:` code change that neither fixes a bug nor adds a feature
  - `chore:` maintenance (deps, CI, etc.)
  - `ci:` CI/CD changes

- **Branch naming**: `feat/short-description`, `fix/short-description`, `docs/short-description`

- **Go style**: follow standard Go conventions. The linter enforces most of them. Avoid unnecessary comments — code should be self-explanatory.

- **Templates**: all Kubernetes manifests are Go `text/template` files in `internal/templates/`. They are embedded in the binary via `embed.FS`.

- **Idempotency**: all scaffolding operations must be idempotent. Never overwrite existing files.

### Adding a New CLI Command

1. Create `internal/cli/<command>.go` with a Cobra command
2. Register it in the `init()` function with `rootCmd.AddCommand()`
3. Add interactive mode (Charmbracelet `huh`) and flag-based non-interactive mode
4. Reuse or extend the scaffolder for file generation
5. Add tests in `internal/cli/<command>_test.go`

### Adding a New Template

1. Create the `.tmpl` file under `internal/templates/`
2. Templates are automatically available via `templates.FS`
3. Use `renderTemplateWithData()` or `renderTemplate()` in the scaffolder
4. Add tests verifying the rendered output

## Pull Request Checklist

Before submitting, make sure:

- [ ] `go test -race ./...` passes
- [ ] `golangci-lint run` reports 0 issues
- [ ] New code has tests
- [ ] README is updated if CLI behavior changed
- [ ] Commit messages follow Conventional Commits
- [ ] PR description explains **what** and **why**

## Code of Conduct

Be respectful, constructive, and inclusive. We're all here to build something useful.

## Questions?

Open an issue or start a [discussion](https://github.com/y0s3ph/gostrap/discussions). No question is too small.
