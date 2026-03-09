package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/y0s3ph/gostrap/internal/config"
	"github.com/y0s3ph/gostrap/internal/models"
	"github.com/y0s3ph/gostrap/internal/scaffolder"
)

func setupRepo(t *testing.T, cfg *models.BootstrapConfig) string {
	t.Helper()

	s := scaffolder.New(cfg)
	_, err := s.Scaffold()
	require.NoError(t, err)

	require.NoError(t, config.Save(cfg.RepoPath, cfg))

	return cfg.RepoPath
}

func TestAddApp_ScaffoldsKustomizeApp(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "repo")

	cfg := &models.BootstrapConfig{
		Controller: models.ControllerConfig{
			Type:    models.ControllerArgoCD,
			Version: "2.13.1",
		},
		Secrets: models.SecretsConfig{
			Type: models.SecretsSealedSecrets,
		},
		ManifestType: models.ManifestKustomize,
		Environments: models.DefaultEnvironments(),
		RepoPath:     repoPath,
	}

	setupRepo(t, cfg)

	loaded, err := config.Load(repoPath)
	require.NoError(t, err)

	s := scaffolder.New(loaded)
	require.NoError(t, s.ScaffoldApp("payments", 3000))

	assert.FileExists(t, filepath.Join(repoPath, "environments/base/payments/deployment.yaml"))
	assert.FileExists(t, filepath.Join(repoPath, "environments/base/payments/service.yaml"))
	assert.FileExists(t, filepath.Join(repoPath, "environments/base/payments/kustomization.yaml"))

	for _, env := range []string{"dev", "staging", "production"} {
		assert.FileExists(t, filepath.Join(repoPath, "environments", env, "payments/kustomization.yaml"))
		assert.FileExists(t, filepath.Join(repoPath, "apps", "payments-"+env+".yaml"))
	}
}

func TestAddApp_ScaffoldsHelmApp(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "repo")

	cfg := &models.BootstrapConfig{
		Controller: models.ControllerConfig{
			Type:    models.ControllerFlux,
			Version: "2.8.1",
		},
		Secrets: models.SecretsConfig{
			Type: models.SecretsSealedSecrets,
		},
		ManifestType: models.ManifestHelm,
		Environments: models.DefaultEnvironments(),
		RepoPath:     repoPath,
	}

	setupRepo(t, cfg)

	loaded, err := config.Load(repoPath)
	require.NoError(t, err)

	s := scaffolder.New(loaded)
	require.NoError(t, s.ScaffoldApp("orders", 8080))

	assert.FileExists(t, filepath.Join(repoPath, "environments/base/orders/Chart.yaml"))
	assert.FileExists(t, filepath.Join(repoPath, "environments/base/orders/values.yaml"))
	assert.FileExists(t, filepath.Join(repoPath, "environments/base/orders/templates/deployment.yaml"))
	assert.FileExists(t, filepath.Join(repoPath, "environments/base/orders/templates/service.yaml"))

	for _, env := range []string{"dev", "staging", "production"} {
		assert.FileExists(t, filepath.Join(repoPath, "environments", env, "orders/values.yaml"))
		assert.FileExists(t, filepath.Join(repoPath, "apps", "orders-"+env+".yaml"))
	}
}

func TestAddApp_IsIdempotent(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "repo")

	cfg := &models.BootstrapConfig{
		Controller: models.ControllerConfig{
			Type:    models.ControllerArgoCD,
			Version: "2.13.1",
		},
		Secrets: models.SecretsConfig{
			Type: models.SecretsSealedSecrets,
		},
		ManifestType: models.ManifestKustomize,
		Environments: models.DefaultEnvironments(),
		RepoPath:     repoPath,
	}

	setupRepo(t, cfg)

	loaded, err := config.Load(repoPath)
	require.NoError(t, err)

	s1 := scaffolder.New(loaded)
	require.NoError(t, s1.ScaffoldApp("my-api", 8080))
	firstResult := s1.Result()
	assert.NotEmpty(t, firstResult.Created)

	loaded2, _ := config.Load(repoPath)
	s2 := scaffolder.New(loaded2)
	require.NoError(t, s2.ScaffoldApp("my-api", 8080))
	secondResult := s2.Result()
	assert.Empty(t, secondResult.Created, "second run should create nothing")
	assert.NotEmpty(t, secondResult.Skipped)
}

func TestAddApp_AppDefinitionsContainCorrectContent(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "repo")

	cfg := &models.BootstrapConfig{
		Controller: models.ControllerConfig{
			Type:    models.ControllerArgoCD,
			Version: "2.13.1",
		},
		Secrets: models.SecretsConfig{
			Type: models.SecretsSealedSecrets,
		},
		ManifestType: models.ManifestKustomize,
		Environments: models.DefaultEnvironments(),
		RepoPath:     repoPath,
	}

	setupRepo(t, cfg)

	loaded, err := config.Load(repoPath)
	require.NoError(t, err)

	s := scaffolder.New(loaded)
	require.NoError(t, s.ScaffoldApp("billing", 9090))

	data, err := os.ReadFile(filepath.Join(repoPath, "apps/billing-dev.yaml"))
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "billing-dev")
	assert.Contains(t, content, "path: environments/dev/billing")

	baseDeployment, err := os.ReadFile(filepath.Join(repoPath, "environments/base/billing/deployment.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(baseDeployment), "containerPort: 9090")
}

func TestAddApp_FluxSOPSHasDecryption(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "repo")

	cfg := &models.BootstrapConfig{
		Controller: models.ControllerConfig{
			Type:    models.ControllerFlux,
			Version: "2.8.1",
		},
		Secrets: models.SecretsConfig{
			Type: models.SecretsSOPS,
		},
		ManifestType: models.ManifestKustomize,
		Environments: models.DefaultEnvironments(),
		RepoPath:     repoPath,
	}

	setupRepo(t, cfg)

	loaded, err := config.Load(repoPath)
	require.NoError(t, err)

	s := scaffolder.New(loaded)
	require.NoError(t, s.ScaffoldApp("secure-api", 8080))

	data, err := os.ReadFile(filepath.Join(repoPath, "apps/secure-api-dev.yaml"))
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "decryption:")
	assert.Contains(t, content, "provider: sops")
}

func TestValidAppName(t *testing.T) {
	tests := []struct {
		name  string
		valid bool
	}{
		{"my-api", true},
		{"payments", true},
		{"my-cool-service-v2", true},
		{"a1", true},
		{"My-Api", false},
		{"-bad", false},
		{"bad-", false},
		{"", false},
		{"a", false},
		{"has space", false},
		{"has_underscore", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.valid, validAppName.MatchString(tt.name), "name: %q", tt.name)
		})
	}
}
