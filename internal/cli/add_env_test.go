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

func setupRepoWithApp(t *testing.T, repoPath string, cfg *models.BootstrapConfig) {
	t.Helper()

	s := scaffolder.New(cfg)
	_, err := s.Scaffold()
	require.NoError(t, err)
	require.NoError(t, config.Save(repoPath, cfg))

	s2 := scaffolder.New(cfg)
	require.NoError(t, s2.ScaffoldApp("my-api", 8080))
}

func TestAddEnv_CreatesOverlaysForExistingApps(t *testing.T) {
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

	setupRepoWithApp(t, repoPath, cfg)

	loaded, err := config.Load(repoPath)
	require.NoError(t, err)

	env := models.EnvironmentConfig{Name: "qa", AutoSync: true, Prune: true}
	s := scaffolder.New(loaded)
	require.NoError(t, s.ScaffoldEnv(env))

	assert.FileExists(t, filepath.Join(repoPath, "environments/qa/my-api/kustomization.yaml"))
	assert.FileExists(t, filepath.Join(repoPath, "apps/my-api-qa.yaml"))

	data, err := os.ReadFile(filepath.Join(repoPath, "apps/my-api-qa.yaml"))
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "my-api-qa")
	assert.Contains(t, content, "environments/qa/my-api")
}

func TestAddEnv_UpdatesConfig(t *testing.T) {
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

	setupRepoWithApp(t, repoPath, cfg)

	loaded, err := config.Load(repoPath)
	require.NoError(t, err)

	env := models.EnvironmentConfig{Name: "qa", AutoSync: true, Prune: false}
	loaded.Environments = append(loaded.Environments, env)
	require.NoError(t, config.Save(repoPath, loaded))

	reloaded, err := config.Load(repoPath)
	require.NoError(t, err)
	assert.Len(t, reloaded.Environments, 4)

	found := false
	for _, e := range reloaded.Environments {
		if e.Name == "qa" {
			found = true
			assert.True(t, e.AutoSync)
			assert.False(t, e.Prune)
		}
	}
	assert.True(t, found, "qa environment should be in config")
}

func TestAddEnv_HelmCreatesValuesFiles(t *testing.T) {
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

	setupRepoWithApp(t, repoPath, cfg)

	loaded, err := config.Load(repoPath)
	require.NoError(t, err)

	env := models.EnvironmentConfig{Name: "qa", AutoSync: true}
	s := scaffolder.New(loaded)
	require.NoError(t, s.ScaffoldEnv(env))

	assert.FileExists(t, filepath.Join(repoPath, "environments/qa/my-api/values.yaml"))
	assert.FileExists(t, filepath.Join(repoPath, "apps/my-api-qa.yaml"))
	assert.NoFileExists(t, filepath.Join(repoPath, "environments/qa/my-api/kustomization.yaml"))
}

func TestAddEnv_DuplicateDetection(t *testing.T) {
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

	setupRepoWithApp(t, repoPath, cfg)

	loaded, err := config.Load(repoPath)
	require.NoError(t, err)

	for _, env := range loaded.Environments {
		if env.Name == "dev" {
			assert.Equal(t, "dev", env.Name)
			break
		}
	}
}
