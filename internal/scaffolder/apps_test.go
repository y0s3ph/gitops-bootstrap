package scaffolder

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScaffoldApp_CreatesApplicationPerEnvironment(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "repo")
	cfg := testConfig(repoPath)

	s := New(cfg)
	require.NoError(t, s.ScaffoldApp("my-api", 8080))

	for _, env := range []string{"dev", "staging", "production"} {
		appFile := filepath.Join(repoPath, "apps", "my-api-"+env+".yaml")
		_, err := os.Stat(appFile)
		require.NoError(t, err, "Application for %s should exist", env)
	}
}

func TestScaffoldApp_ApplicationPointsToOverlay(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "repo")
	cfg := testConfig(repoPath)

	s := New(cfg)
	require.NoError(t, s.ScaffoldApp("my-api", 8080))

	data, err := os.ReadFile(filepath.Join(repoPath, "apps/my-api-staging.yaml"))
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "path: environments/staging/my-api")
	assert.Contains(t, content, "namespace: staging")
}

func TestScaffoldApp_DevHasAutoSync(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "repo")
	cfg := testConfig(repoPath)

	s := New(cfg)
	require.NoError(t, s.ScaffoldApp("my-api", 8080))

	data, err := os.ReadFile(filepath.Join(repoPath, "apps/my-api-dev.yaml"))
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "automated:")
	assert.Contains(t, content, "selfHeal: true")
	assert.Contains(t, content, "prune: true")
}

func TestScaffoldApp_ProductionNoAutoSync(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "repo")
	cfg := testConfig(repoPath)

	s := New(cfg)
	require.NoError(t, s.ScaffoldApp("my-api", 8080))

	data, err := os.ReadFile(filepath.Join(repoPath, "apps/my-api-production.yaml"))
	require.NoError(t, err)

	content := string(data)
	assert.NotContains(t, content, "automated:")
	assert.Contains(t, content, "CreateNamespace=true")
}

func TestScaffold_ExampleAppWhenEnabled(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "repo")
	cfg := testConfig(repoPath)
	cfg.ScaffoldExample = true

	s := New(cfg)
	result, err := s.Scaffold()
	require.NoError(t, err)

	hasExampleApp := false
	for _, f := range result.Created {
		if filepath.Base(f) == "example-api-dev.yaml" {
			hasExampleApp = true
			break
		}
	}
	assert.True(t, hasExampleApp, "example-api should be scaffolded when ScaffoldExample is true")

	_, err = os.Stat(filepath.Join(repoPath, "environments/base/example-api/deployment.yaml"))
	require.NoError(t, err)
}

func TestScaffold_NoExampleAppWhenDisabled(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "repo")
	cfg := testConfig(repoPath)
	cfg.ScaffoldExample = false

	s := New(cfg)
	_, err := s.Scaffold()
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(repoPath, "environments/base/example-api"))
	assert.True(t, os.IsNotExist(err), "example-api should not exist when ScaffoldExample is false")
}

// --- Flux app definition tests ---

func TestScaffoldFluxApp_CreatesKustomizationPerEnvironment(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "repo")
	cfg := testFluxConfig(repoPath)

	s := New(cfg)
	require.NoError(t, s.ScaffoldApp("my-api", 8080))

	for _, env := range []string{"dev", "staging", "production"} {
		appFile := filepath.Join(repoPath, "apps", "my-api-"+env+".yaml")
		_, err := os.Stat(appFile)
		require.NoError(t, err, "Kustomization for %s should exist", env)
	}
}

func TestScaffoldFluxApp_KustomizationPointsToOverlay(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "repo")
	cfg := testFluxConfig(repoPath)

	s := New(cfg)
	require.NoError(t, s.ScaffoldApp("my-api", 8080))

	data, err := os.ReadFile(filepath.Join(repoPath, "apps/my-api-staging.yaml"))
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "kind: Kustomization")
	assert.Contains(t, content, "path: ./environments/staging/my-api")
	assert.Contains(t, content, "targetNamespace: staging")
	assert.Contains(t, content, "sourceRef:")
	assert.Contains(t, content, "name: gitops-repo")
}

func TestScaffoldFluxApp_DevHasAutoPrune(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "repo")
	cfg := testFluxConfig(repoPath)

	s := New(cfg)
	require.NoError(t, s.ScaffoldApp("my-api", 8080))

	data, err := os.ReadFile(filepath.Join(repoPath, "apps/my-api-dev.yaml"))
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "prune: true")
	assert.NotContains(t, content, "suspend: true")
}

func TestScaffoldFluxApp_ProductionSuspended(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "repo")
	cfg := testFluxConfig(repoPath)

	s := New(cfg)
	require.NoError(t, s.ScaffoldApp("my-api", 8080))

	data, err := os.ReadFile(filepath.Join(repoPath, "apps/my-api-production.yaml"))
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "suspend: true", "production should be suspended when AutoSync is false")
	assert.Contains(t, content, "prune: false")
}

func TestScaffoldFluxApp_NoArgoCDReferences(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "repo")
	cfg := testFluxConfig(repoPath)

	s := New(cfg)
	require.NoError(t, s.ScaffoldApp("my-api", 8080))

	for _, env := range []string{"dev", "staging", "production"} {
		data, err := os.ReadFile(filepath.Join(repoPath, "apps", "my-api-"+env+".yaml"))
		require.NoError(t, err)
		content := string(data)
		assert.NotContains(t, content, "argoproj.io", "Flux app should not contain ArgoCD references")
	}
}
