package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/y0s3ph/gostrap/internal/models"
)

func TestSaveAndLoad_RoundTrip(t *testing.T) {
	dir := t.TempDir()

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
		RepoPath:     dir,
	}

	require.NoError(t, Save(dir, cfg))

	loaded, err := Load(dir)
	require.NoError(t, err)

	assert.Equal(t, models.ControllerArgoCD, loaded.Controller.Type)
	assert.Equal(t, "2.13.1", loaded.Controller.Version)
	assert.Equal(t, models.SecretsSealedSecrets, loaded.Secrets.Type)
	assert.Equal(t, models.ManifestKustomize, loaded.ManifestType)
	assert.Len(t, loaded.Environments, 3)
	assert.Equal(t, "dev", loaded.Environments[0].Name)
	assert.Equal(t, dir, loaded.RepoPath)
}

func TestSaveAndLoad_FluxHelm(t *testing.T) {
	dir := t.TempDir()

	cfg := &models.BootstrapConfig{
		Controller: models.ControllerConfig{
			Type:    models.ControllerFlux,
			Version: "2.8.1",
		},
		Secrets: models.SecretsConfig{
			Type: models.SecretsSOPS,
		},
		ManifestType: models.ManifestHelm,
		Environments: []models.EnvironmentConfig{
			{Name: "dev", AutoSync: true},
			{Name: "prod", AutoSync: false},
		},
		RepoPath: dir,
	}

	require.NoError(t, Save(dir, cfg))

	loaded, err := Load(dir)
	require.NoError(t, err)

	assert.Equal(t, models.ControllerFlux, loaded.Controller.Type)
	assert.Equal(t, models.SecretsSOPS, loaded.Secrets.Type)
	assert.Equal(t, models.ManifestHelm, loaded.ManifestType)
	assert.Len(t, loaded.Environments, 2)
}

func TestLoad_MissingFile(t *testing.T) {
	dir := t.TempDir()

	_, err := Load(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "gostrap init")
}

func TestLoad_DefaultsManifestType(t *testing.T) {
	dir := t.TempDir()
	content := `controller:
  type: argocd
  version: "2.13.1"
secrets:
  type: sealed-secrets
environments:
  - name: dev
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, FileName), []byte(content), 0644))

	loaded, err := Load(dir)
	require.NoError(t, err)

	assert.Equal(t, models.ManifestKustomize, loaded.ManifestType)
}

func TestSave_CreatesFile(t *testing.T) {
	dir := t.TempDir()

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
		RepoPath:     dir,
	}

	require.NoError(t, Save(dir, cfg))

	_, err := os.Stat(filepath.Join(dir, FileName))
	require.NoError(t, err)
}
