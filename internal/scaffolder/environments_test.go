package scaffolder

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/y0s3ph/gostrap/internal/models"
)

func TestScaffoldApp_CreatesBaseManifests(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "repo")
	cfg := testConfig(repoPath)

	s := New(cfg)
	require.NoError(t, s.ScaffoldApp("my-api", 3000))

	expectedFiles := []string{
		"environments/base/my-api/kustomization.yaml",
		"environments/base/my-api/deployment.yaml",
		"environments/base/my-api/service.yaml",
	}

	for _, f := range expectedFiles {
		_, err := os.Stat(filepath.Join(repoPath, f))
		require.NoError(t, err, "%s should exist", f)
	}
}

func TestScaffoldApp_BaseDeploymentContent(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "repo")
	cfg := testConfig(repoPath)

	s := New(cfg)
	require.NoError(t, s.ScaffoldApp("my-api", 3000))

	data, err := os.ReadFile(filepath.Join(repoPath, "environments/base/my-api/deployment.yaml"))
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "name: my-api")
	assert.Contains(t, content, "containerPort: 3000")
	assert.Contains(t, content, "image: my-api:latest")
	assert.Contains(t, content, "/healthz")
	assert.Contains(t, content, "/readyz")
}

func TestScaffoldApp_BaseServiceContent(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "repo")
	cfg := testConfig(repoPath)

	s := New(cfg)
	require.NoError(t, s.ScaffoldApp("my-api", 9090))

	data, err := os.ReadFile(filepath.Join(repoPath, "environments/base/my-api/service.yaml"))
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "port: 9090")
	assert.Contains(t, content, "targetPort: 9090")
	assert.Contains(t, content, "app.kubernetes.io/name: my-api")
}

func TestScaffoldApp_CreatesOverlayPerEnvironment(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "repo")
	cfg := testConfig(repoPath)

	s := New(cfg)
	require.NoError(t, s.ScaffoldApp("my-api", 8080))

	for _, env := range []string{"dev", "staging", "production"} {
		kustomization := filepath.Join(repoPath, "environments", env, "my-api", "kustomization.yaml")
		_, err := os.Stat(kustomization)
		require.NoError(t, err, "overlay for %s should exist", env)
	}
}

func TestScaffoldApp_OverlayReplicaCounts(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "repo")
	cfg := testConfig(repoPath)

	s := New(cfg)
	require.NoError(t, s.ScaffoldApp("my-api", 8080))

	tests := []struct {
		env      string
		replicas string
	}{
		{"dev", "count: 1"},
		{"staging", "count: 2"},
		{"production", "count: 3"},
	}

	for _, tt := range tests {
		data, err := os.ReadFile(filepath.Join(repoPath, "environments", tt.env, "my-api", "kustomization.yaml"))
		require.NoError(t, err)
		assert.Contains(t, string(data), tt.replicas, "%s should have %s", tt.env, tt.replicas)
	}
}

func TestScaffoldApp_OverlayReferencesBase(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "repo")
	cfg := testConfig(repoPath)

	s := New(cfg)
	require.NoError(t, s.ScaffoldApp("my-api", 8080))

	data, err := os.ReadFile(filepath.Join(repoPath, "environments/dev/my-api/kustomization.yaml"))
	require.NoError(t, err)

	assert.Contains(t, string(data), "../../base/my-api")
}

func TestScaffoldApp_OverlaySetsNamespace(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "repo")
	cfg := testConfig(repoPath)

	s := New(cfg)
	require.NoError(t, s.ScaffoldApp("my-api", 8080))

	data, err := os.ReadFile(filepath.Join(repoPath, "environments/staging/my-api/kustomization.yaml"))
	require.NoError(t, err)

	assert.Contains(t, string(data), "namespace: staging")
}

func TestScaffoldApp_CustomEnvironmentDefaultsToOneReplica(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "repo")
	cfg := testConfig(repoPath)
	cfg.Environments = []models.EnvironmentConfig{
		{Name: "canary", AutoSync: true},
	}

	s := New(cfg)
	require.NoError(t, s.ScaffoldApp("my-api", 8080))

	data, err := os.ReadFile(filepath.Join(repoPath, "environments/canary/my-api/kustomization.yaml"))
	require.NoError(t, err)

	assert.Contains(t, string(data), "count: 1")
}
