package wizard

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/y0s3ph/gitops-bootstrap/internal/models"
)

func writeTestConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
	return path
}

func TestLoadConfig_FullConfig(t *testing.T) {
	path := writeTestConfig(t, `
controller:
  type: argocd
  version: "2.13.1"
secrets:
  type: sealed-secrets
environments:
  - name: dev
    auto_sync: true
    prune: true
  - name: production
    auto_sync: false
    require_pr: true
repo_path: ./my-repo
`)

	cfg, err := LoadConfig(path)
	require.NoError(t, err)

	assert.Equal(t, models.ControllerArgoCD, cfg.Controller.Type)
	assert.Equal(t, "2.13.1", cfg.Controller.Version)
	assert.Equal(t, models.SecretsSealedSecrets, cfg.Secrets.Type)
	assert.Len(t, cfg.Environments, 2)
	assert.Equal(t, "dev", cfg.Environments[0].Name)
	assert.Equal(t, "./my-repo", cfg.RepoPath)
}

func TestLoadConfig_FluxWithSOPS(t *testing.T) {
	path := writeTestConfig(t, `
controller:
  type: flux
  version: "2.4.0"
secrets:
  type: sops
environments:
  - name: staging
repo_path: ./flux-repo
`)

	cfg, err := LoadConfig(path)
	require.NoError(t, err)

	assert.Equal(t, models.ControllerFlux, cfg.Controller.Type)
	assert.Equal(t, models.SecretsSOPS, cfg.Secrets.Type)
}

func TestLoadConfig_DefaultsApplied(t *testing.T) {
	path := writeTestConfig(t, `
controller:
  type: argocd
secrets:
  type: sealed-secrets
`)

	cfg, err := LoadConfig(path)
	require.NoError(t, err)

	assert.Equal(t, models.DefaultArgoCDVersion, cfg.Controller.Version, "should default controller version")
	assert.Len(t, cfg.Environments, 3, "should default to dev/staging/production")
	assert.Equal(t, "./gitops-repo", cfg.RepoPath, "should default repo path")
}

func TestLoadConfig_InvalidController(t *testing.T) {
	path := writeTestConfig(t, `
controller:
  type: jenkins
secrets:
  type: sealed-secrets
repo_path: ./repo
`)

	_, err := LoadConfig(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid controller type")
}

func TestLoadConfig_MissingSecrets(t *testing.T) {
	path := writeTestConfig(t, `
controller:
  type: argocd
  version: "2.13.1"
repo_path: ./repo
`)

	_, err := LoadConfig(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "secrets type is required")
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	path := writeTestConfig(t, `{{{ not yaml`)

	_, err := LoadConfig(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing config file")
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := LoadConfig("/nonexistent/config.yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reading config file")
}
