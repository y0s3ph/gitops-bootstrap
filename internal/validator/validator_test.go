package validator

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

func setupValidRepo(t *testing.T) string {
	t.Helper()
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
		ManifestType:    models.ManifestKustomize,
		Environments:    models.DefaultEnvironments(),
		RepoPath:        repoPath,
		ScaffoldExample: true,
	}

	s := scaffolder.New(cfg)
	_, err := s.Scaffold()
	require.NoError(t, err)
	require.NoError(t, config.Save(repoPath, cfg))

	return repoPath
}

func setupValidHelmRepo(t *testing.T) string {
	t.Helper()
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
		ManifestType:    models.ManifestHelm,
		Environments:    models.DefaultEnvironments(),
		RepoPath:        repoPath,
		ScaffoldExample: true,
	}

	s := scaffolder.New(cfg)
	_, err := s.Scaffold()
	require.NoError(t, err)
	require.NoError(t, config.Save(repoPath, cfg))

	return repoPath
}

func TestValidate_ValidKustomizeRepo(t *testing.T) {
	repoPath := setupValidRepo(t)

	result := Validate(repoPath)
	assert.True(t, result.IsValid(), "valid repo should pass: %v", result.Issues)
	assert.Empty(t, result.Errors())
}

func TestValidate_ValidHelmRepo(t *testing.T) {
	repoPath := setupValidHelmRepo(t)

	result := Validate(repoPath)
	assert.True(t, result.IsValid(), "valid helm repo should pass: %v", result.Issues)
	assert.Empty(t, result.Errors())
}

func TestValidate_MissingConfig(t *testing.T) {
	repoPath := t.TempDir()

	result := Validate(repoPath)
	assert.False(t, result.IsValid())
	assert.Len(t, result.Errors(), 1)
	assert.Contains(t, result.Errors()[0].Message, "cannot load config")
}

func TestValidate_MissingRootApp(t *testing.T) {
	repoPath := setupValidRepo(t)
	require.NoError(t, os.Remove(filepath.Join(repoPath, "apps/_root.yaml")))

	result := Validate(repoPath)
	assert.False(t, result.IsValid())

	found := false
	for _, e := range result.Errors() {
		if e.Path == "apps/_root.yaml" {
			found = true
			break
		}
	}
	assert.True(t, found, "should report missing root app")
}

func TestValidate_MissingBootstrapDir(t *testing.T) {
	repoPath := setupValidRepo(t)
	require.NoError(t, os.RemoveAll(filepath.Join(repoPath, "bootstrap/argocd")))

	result := Validate(repoPath)
	assert.False(t, result.IsValid())

	found := false
	for _, e := range result.Errors() {
		if e.Path == "bootstrap/argocd" {
			found = true
			break
		}
	}
	assert.True(t, found, "should report missing bootstrap dir")
}

func TestValidate_MissingOverlay(t *testing.T) {
	repoPath := setupValidRepo(t)
	require.NoError(t, os.RemoveAll(filepath.Join(repoPath, "environments/staging/example-api")))

	result := Validate(repoPath)
	assert.False(t, result.IsValid())

	found := false
	for _, e := range result.Errors() {
		if e.Path == "environments/staging/example-api/kustomization.yaml" {
			found = true
			break
		}
	}
	assert.True(t, found, "should report missing staging overlay")
}

func TestValidate_MissingAppDefinition(t *testing.T) {
	repoPath := setupValidRepo(t)
	require.NoError(t, os.Remove(filepath.Join(repoPath, "apps/example-api-production.yaml")))

	result := Validate(repoPath)
	assert.False(t, result.IsValid())

	found := false
	for _, e := range result.Errors() {
		if e.Path == "apps/example-api-production.yaml" {
			found = true
			break
		}
	}
	assert.True(t, found, "should report missing app definition")
}

func TestValidate_InvalidYAML(t *testing.T) {
	repoPath := setupValidRepo(t)

	badFile := filepath.Join(repoPath, "apps/example-api-dev.yaml")
	require.NoError(t, os.WriteFile(badFile, []byte("invalid: yaml: [broken"), 0644))

	result := Validate(repoPath)
	assert.False(t, result.IsValid())

	found := false
	for _, e := range result.Errors() {
		if e.Path == "apps/example-api-dev.yaml" {
			found = true
			assert.Contains(t, e.Message, "invalid YAML")
			break
		}
	}
	assert.True(t, found, "should report invalid YAML")
}

func TestValidate_MissingBaseFiles(t *testing.T) {
	repoPath := setupValidRepo(t)
	require.NoError(t, os.Remove(filepath.Join(repoPath, "environments/base/example-api/deployment.yaml")))

	result := Validate(repoPath)
	assert.False(t, result.IsValid())

	found := false
	for _, e := range result.Errors() {
		if e.Path == "environments/base/example-api/deployment.yaml" {
			found = true
			break
		}
	}
	assert.True(t, found, "should report missing deployment.yaml")
}

func TestValidate_NoAppsWarning(t *testing.T) {
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

	s := scaffolder.New(cfg)
	_, err := s.Scaffold()
	require.NoError(t, err)
	require.NoError(t, config.Save(repoPath, cfg))

	result := Validate(repoPath)
	assert.True(t, result.IsValid(), "no apps is a warning, not an error")
	assert.Len(t, result.Warnings(), 1)
	assert.Contains(t, result.Warnings()[0].Message, "no applications found")
}

func TestValidate_HelmMissingChart(t *testing.T) {
	repoPath := setupValidHelmRepo(t)
	require.NoError(t, os.Remove(filepath.Join(repoPath, "environments/base/example-api/Chart.yaml")))

	result := Validate(repoPath)
	assert.False(t, result.IsValid())

	found := false
	for _, e := range result.Errors() {
		if e.Path == "environments/base/example-api/Chart.yaml" {
			found = true
			break
		}
	}
	assert.True(t, found, "should report missing Chart.yaml")
}

func TestValidate_HelmMissingValuesOverlay(t *testing.T) {
	repoPath := setupValidHelmRepo(t)
	require.NoError(t, os.RemoveAll(filepath.Join(repoPath, "environments/dev/example-api")))

	result := Validate(repoPath)
	assert.False(t, result.IsValid())

	found := false
	for _, e := range result.Errors() {
		if e.Path == "environments/dev/example-api/values.yaml" {
			found = true
			break
		}
	}
	assert.True(t, found, "should report missing Helm values overlay")
}

func TestValidate_MultipleErrors(t *testing.T) {
	repoPath := setupValidRepo(t)

	require.NoError(t, os.Remove(filepath.Join(repoPath, "apps/example-api-dev.yaml")))
	require.NoError(t, os.Remove(filepath.Join(repoPath, "apps/example-api-staging.yaml")))
	require.NoError(t, os.RemoveAll(filepath.Join(repoPath, "environments/production/example-api")))

	result := Validate(repoPath)
	assert.False(t, result.IsValid())
	assert.GreaterOrEqual(t, len(result.Errors()), 3, "should report multiple errors")
}

func TestResult_IsValid(t *testing.T) {
	r := &Result{}
	assert.True(t, r.IsValid())

	r.addWarning("", "just a warning")
	assert.True(t, r.IsValid(), "warnings alone should not fail")

	r.addError("", "an error")
	assert.False(t, r.IsValid())
}
