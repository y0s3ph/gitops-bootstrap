package scaffolder

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/y0s3ph/gostrap/internal/models"
)

func setupRepoWithApps(t *testing.T, cfg *models.BootstrapConfig, apps ...string) {
	t.Helper()
	s := New(cfg)
	_, err := s.Scaffold()
	require.NoError(t, err)

	for _, app := range apps {
		s2 := New(cfg)
		require.NoError(t, s2.ScaffoldApp(app, 8080))
	}
}

func TestDiscoverApps_FindsExistingApps(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "repo")
	cfg := testConfig(repoPath)
	cfg.ScaffoldExample = true

	setupRepoWithApps(t, cfg, "payments", "orders")

	s := New(cfg)
	apps, err := s.DiscoverApps()
	require.NoError(t, err)
	assert.Contains(t, apps, "example-api")
	assert.Contains(t, apps, "payments")
	assert.Contains(t, apps, "orders")
	assert.Len(t, apps, 3)
}

func TestDiscoverApps_EmptyRepo(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "repo")
	cfg := testConfig(repoPath)
	cfg.ScaffoldExample = false

	s := New(cfg)
	_, err := s.Scaffold()
	require.NoError(t, err)

	s2 := New(cfg)
	apps, err := s2.DiscoverApps()
	require.NoError(t, err)
	assert.Empty(t, apps)
}

func TestDiscoverApps_NoBaseDir(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "repo")
	cfg := testConfig(repoPath)

	s := New(cfg)
	apps, err := s.DiscoverApps()
	require.NoError(t, err)
	assert.Nil(t, apps)
}

func TestScaffoldEnv_KustomizeCreatesOverlaysAndAppDefs(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "repo")
	cfg := testConfig(repoPath)

	setupRepoWithApps(t, cfg, "api", "web")

	env := models.EnvironmentConfig{Name: "qa", AutoSync: true, Prune: true}
	s := New(cfg)
	require.NoError(t, s.ScaffoldEnv(env))

	assert.FileExists(t, filepath.Join(repoPath, "environments/qa/api/kustomization.yaml"))
	assert.FileExists(t, filepath.Join(repoPath, "environments/qa/web/kustomization.yaml"))
	assert.FileExists(t, filepath.Join(repoPath, "apps/api-qa.yaml"))
	assert.FileExists(t, filepath.Join(repoPath, "apps/web-qa.yaml"))
}

func TestScaffoldEnv_HelmCreatesValuesAndAppDefs(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "repo")
	cfg := testFluxConfig(repoPath)
	cfg.ManifestType = models.ManifestHelm

	setupRepoWithApps(t, cfg, "api")

	env := models.EnvironmentConfig{Name: "qa", AutoSync: true, Prune: false}
	s := New(cfg)
	require.NoError(t, s.ScaffoldEnv(env))

	assert.FileExists(t, filepath.Join(repoPath, "environments/qa/api/values.yaml"))
	assert.FileExists(t, filepath.Join(repoPath, "apps/api-qa.yaml"))

	assert.NoFileExists(t, filepath.Join(repoPath, "environments/qa/api/kustomization.yaml"))
}

func TestScaffoldEnv_AppDefinitionContent(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "repo")
	cfg := testConfig(repoPath)

	setupRepoWithApps(t, cfg, "billing")

	env := models.EnvironmentConfig{Name: "qa", AutoSync: true, Prune: true}
	s := New(cfg)
	require.NoError(t, s.ScaffoldEnv(env))

	data, err := os.ReadFile(filepath.Join(repoPath, "apps/billing-qa.yaml"))
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "billing-qa")
	assert.Contains(t, content, "path: environments/qa/billing")
	assert.Contains(t, content, "namespace: qa")
}

func TestScaffoldEnv_FluxSOPSHasDecryption(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "repo")
	cfg := testSOPSFluxConfig(repoPath)

	setupRepoWithApps(t, cfg, "secure")

	env := models.EnvironmentConfig{Name: "qa", AutoSync: true}
	s := New(cfg)
	require.NoError(t, s.ScaffoldEnv(env))

	data, err := os.ReadFile(filepath.Join(repoPath, "apps/secure-qa.yaml"))
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "decryption:")
	assert.Contains(t, content, "provider: sops")
}

func TestScaffoldEnv_IsIdempotent(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "repo")
	cfg := testConfig(repoPath)

	setupRepoWithApps(t, cfg, "api")

	env := models.EnvironmentConfig{Name: "qa", AutoSync: true}

	s1 := New(cfg)
	require.NoError(t, s1.ScaffoldEnv(env))
	r1 := s1.Result()
	assert.NotEmpty(t, r1.Created)

	s2 := New(cfg)
	require.NoError(t, s2.ScaffoldEnv(env))
	r2 := s2.Result()
	assert.Empty(t, r2.Created, "second run should create nothing")
	assert.NotEmpty(t, r2.Skipped)
}

func TestScaffoldEnv_NoApps(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "repo")
	cfg := testConfig(repoPath)
	cfg.ScaffoldExample = false

	s := New(cfg)
	_, err := s.Scaffold()
	require.NoError(t, err)

	env := models.EnvironmentConfig{Name: "qa", AutoSync: true}
	s2 := New(cfg)
	require.NoError(t, s2.ScaffoldEnv(env))

	r := s2.Result()
	assert.Empty(t, r.Created)
}

func TestScaffoldEnv_MultipleApps(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "repo")
	cfg := testConfig(repoPath)

	setupRepoWithApps(t, cfg, "api", "web", "worker")

	env := models.EnvironmentConfig{Name: "canary", AutoSync: false, Prune: false}
	s := New(cfg)
	require.NoError(t, s.ScaffoldEnv(env))

	for _, app := range []string{"api", "web", "worker"} {
		assert.FileExists(t, filepath.Join(repoPath, "environments/canary", app, "kustomization.yaml"))
		assert.FileExists(t, filepath.Join(repoPath, "apps", app+"-canary.yaml"))
	}
}
