package scaffolder

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/y0s3ph/gostrap/internal/models"
)

func TestScaffold_GeneratesAllDocs(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "repo")
	cfg := testConfig(repoPath)

	s := New(cfg)
	_, err := s.Scaffold()
	require.NoError(t, err)

	expectedDocs := []string{
		"docs/ARCHITECTURE.md",
		"docs/ADDING-AN-APP.md",
		"docs/SECRETS.md",
		"docs/TROUBLESHOOTING.md",
	}

	for _, doc := range expectedDocs {
		_, err := os.Stat(filepath.Join(repoPath, doc))
		require.NoError(t, err, "%s should exist", doc)
	}
}

func TestScaffold_ArchitectureContainsControllerInfo(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "repo")
	cfg := testConfig(repoPath)

	s := New(cfg)
	_, err := s.Scaffold()
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(repoPath, "docs/ARCHITECTURE.md"))
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "ArgoCD")
	assert.Contains(t, content, "2.13.1")
	assert.Contains(t, content, "dev")
	assert.Contains(t, content, "staging")
	assert.Contains(t, content, "production")
}

func TestScaffold_SecretsDocAdaptsToSealedSecrets(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "repo")
	cfg := testConfig(repoPath)

	s := New(cfg)
	_, err := s.Scaffold()
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(repoPath, "docs/SECRETS.md"))
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "Sealed Secrets")
	assert.Contains(t, content, "kubeseal")
	assert.NotContains(t, content, "External Secrets Operator")
	assert.NotContains(t, content, "SOPS")
}

func TestScaffold_SecretsDocAdaptsToESO(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "repo")
	cfg := testConfig(repoPath)
	cfg.Secrets.Type = models.SecretsESO

	s := New(cfg)
	_, err := s.Scaffold()
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(repoPath, "docs/SECRETS.md"))
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "External Secrets Operator")
	assert.Contains(t, content, "ExternalSecret")
	assert.NotContains(t, content, "kubeseal")
}

func TestScaffold_SecretsDocAdaptsToSOPS(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "repo")
	cfg := testConfig(repoPath)
	cfg.Secrets.Type = models.SecretsSOPS

	s := New(cfg)
	_, err := s.Scaffold()
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(repoPath, "docs/SECRETS.md"))
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "SOPS")
	assert.Contains(t, content, "sops --encrypt")
	assert.NotContains(t, content, "kubeseal")
}

func TestScaffold_AddingAppDocListsEnvironments(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "repo")
	cfg := testConfig(repoPath)

	s := New(cfg)
	_, err := s.Scaffold()
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(repoPath, "docs/ADDING-AN-APP.md"))
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "environments/dev/")
	assert.Contains(t, content, "environments/staging/")
	assert.Contains(t, content, "environments/production/")
}

func TestScaffold_TroubleshootingExists(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "repo")
	cfg := testConfig(repoPath)

	s := New(cfg)
	_, err := s.Scaffold()
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(repoPath, "docs/TROUBLESHOOTING.md"))
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "OutOfSync")
	assert.Contains(t, content, "argocd-server")
}

// --- Flux docs tests ---

func TestScaffoldFlux_ArchitectureContainsFluxInfo(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "repo")
	cfg := testFluxConfig(repoPath)

	s := New(cfg)
	_, err := s.Scaffold()
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(repoPath, "docs/ARCHITECTURE.md"))
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "Flux CD")
	assert.Contains(t, content, "2.8.1")
	assert.Contains(t, content, "Flux Kustomization")
	assert.Contains(t, content, "GitRepository")
	assert.NotContains(t, content, "App of Apps")
}

func TestScaffoldFlux_TroubleshootingContainsFluxInfo(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "repo")
	cfg := testFluxConfig(repoPath)

	s := New(cfg)
	_, err := s.Scaffold()
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(repoPath, "docs/TROUBLESHOOTING.md"))
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "source-controller")
	assert.Contains(t, content, "kustomize-controller")
	assert.NotContains(t, content, "argocd-server")
	assert.NotContains(t, content, "OutOfSync")
}

func TestScaffoldFlux_AddingAppDocListsFluxKustomization(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "repo")
	cfg := testFluxConfig(repoPath)

	s := New(cfg)
	_, err := s.Scaffold()
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(repoPath, "docs/ADDING-AN-APP.md"))
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "Flux Kustomization")
	assert.Contains(t, content, "kustomize.toolkit.fluxcd.io")
	assert.NotContains(t, content, "argoproj.io")
}
