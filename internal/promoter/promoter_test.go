package promoter

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

func setupRepo(t *testing.T) string {
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

func TestPromote_CopiesOverlayFiles(t *testing.T) {
	repoPath := setupRepo(t)

	devOverlay := filepath.Join(repoPath, "environments", "dev", "example-api", "kustomization.yaml")
	require.NoError(t, os.WriteFile(devOverlay, []byte("modified: true\n"), 0644))

	results, err := Promote(PromoteOptions{
		RepoPath:  repoPath,
		AppName:   "example-api",
		SourceEnv: "dev",
		TargetEnv: "staging",
	})
	require.NoError(t, err)
	require.Len(t, results, 1)

	assert.Equal(t, "example-api", results[0].AppName)
	assert.Len(t, results[0].Copied, 1)

	stagingOverlay := filepath.Join(repoPath, "environments", "staging", "example-api", "kustomization.yaml")
	content, err := os.ReadFile(stagingOverlay)
	require.NoError(t, err)
	assert.Equal(t, "modified: true\n", string(content))
}

func TestPromote_SkipsIdenticalFiles(t *testing.T) {
	repoPath := setupRepo(t)

	devOverlay := filepath.Join(repoPath, "environments", "dev", "example-api", "kustomization.yaml")
	stagingOverlay := filepath.Join(repoPath, "environments", "staging", "example-api", "kustomization.yaml")

	devContent, err := os.ReadFile(devOverlay)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(stagingOverlay, devContent, 0644))

	results, err := Promote(PromoteOptions{
		RepoPath:  repoPath,
		AppName:   "example-api",
		SourceEnv: "dev",
		TargetEnv: "staging",
	})
	require.NoError(t, err)
	require.Len(t, results, 1)

	assert.Empty(t, results[0].Copied, "identical files should be skipped")
	assert.Len(t, results[0].Skipped, 1)
}

func TestPromote_DryRunDoesNotModify(t *testing.T) {
	repoPath := setupRepo(t)

	devOverlay := filepath.Join(repoPath, "environments", "dev", "example-api", "kustomization.yaml")
	require.NoError(t, os.WriteFile(devOverlay, []byte("dry-run-test: true\n"), 0644))

	stagingOverlay := filepath.Join(repoPath, "environments", "staging", "example-api", "kustomization.yaml")
	originalContent, err := os.ReadFile(stagingOverlay)
	require.NoError(t, err)

	results, err := Promote(PromoteOptions{
		RepoPath:  repoPath,
		AppName:   "example-api",
		SourceEnv: "dev",
		TargetEnv: "staging",
		DryRun:    true,
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Len(t, results[0].Copied, 1, "dry-run should still report files to be copied")

	afterContent, err := os.ReadFile(stagingOverlay)
	require.NoError(t, err)
	assert.Equal(t, string(originalContent), string(afterContent), "dry-run should not modify target")
}

func TestPromote_AllApps(t *testing.T) {
	repoPath := setupRepo(t)

	cfg, err := config.Load(repoPath)
	require.NoError(t, err)

	s := scaffolder.New(cfg)
	require.NoError(t, s.ScaffoldApp("api-two", 8081))

	results, err := Promote(PromoteOptions{
		RepoPath:  repoPath,
		SourceEnv: "dev",
		TargetEnv: "staging",
	})
	require.NoError(t, err)
	assert.Len(t, results, 2, "should promote both apps")

	appNames := make(map[string]bool)
	for _, r := range results {
		appNames[r.AppName] = true
	}
	assert.True(t, appNames["example-api"])
	assert.True(t, appNames["api-two"])
}

func TestPromote_InvalidSourceEnv(t *testing.T) {
	repoPath := setupRepo(t)

	_, err := Promote(PromoteOptions{
		RepoPath:  repoPath,
		AppName:   "example-api",
		SourceEnv: "nonexistent",
		TargetEnv: "staging",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found in config")
}

func TestPromote_InvalidTargetEnv(t *testing.T) {
	repoPath := setupRepo(t)

	_, err := Promote(PromoteOptions{
		RepoPath:  repoPath,
		AppName:   "example-api",
		SourceEnv: "dev",
		TargetEnv: "nonexistent",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found in config")
}

func TestPromote_InvalidApp(t *testing.T) {
	repoPath := setupRepo(t)

	_, err := Promote(PromoteOptions{
		RepoPath:  repoPath,
		AppName:   "nonexistent-app",
		SourceEnv: "dev",
		TargetEnv: "staging",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestPromote_MissingConfig(t *testing.T) {
	repoPath := t.TempDir()

	_, err := Promote(PromoteOptions{
		RepoPath:  repoPath,
		SourceEnv: "dev",
		TargetEnv: "staging",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "loading config")
}

func TestPromote_CopiesNewFiles(t *testing.T) {
	repoPath := setupRepo(t)

	extraFile := filepath.Join(repoPath, "environments", "dev", "example-api", "configmap.yaml")
	require.NoError(t, os.WriteFile(extraFile, []byte("apiVersion: v1\nkind: ConfigMap\n"), 0644))

	results, err := Promote(PromoteOptions{
		RepoPath:  repoPath,
		AppName:   "example-api",
		SourceEnv: "dev",
		TargetEnv: "staging",
	})
	require.NoError(t, err)
	require.Len(t, results, 1)

	copiedNewFile := false
	for _, f := range results[0].Copied {
		if filepath.Base(f) == "configmap.yaml" {
			copiedNewFile = true
			break
		}
	}
	assert.True(t, copiedNewFile, "should copy new file to target")

	tgtFile := filepath.Join(repoPath, "environments", "staging", "example-api", "configmap.yaml")
	content, err := os.ReadFile(tgtFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), "ConfigMap")
}

func TestPromote_HelmRepo(t *testing.T) {
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

	devValues := filepath.Join(repoPath, "environments", "dev", "example-api", "values.yaml")
	require.NoError(t, os.WriteFile(devValues, []byte("image:\n  tag: v1.2.3\n"), 0644))

	results, err := Promote(PromoteOptions{
		RepoPath:  repoPath,
		AppName:   "example-api",
		SourceEnv: "dev",
		TargetEnv: "staging",
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.NotEmpty(t, results[0].Copied)

	tgtValues := filepath.Join(repoPath, "environments", "staging", "example-api", "values.yaml")
	content, err := os.ReadFile(tgtValues)
	require.NoError(t, err)
	assert.Contains(t, string(content), "v1.2.3")
}

func TestPreview_ShowsDiff(t *testing.T) {
	repoPath := setupRepo(t)

	diffs, err := Preview(PromoteOptions{
		RepoPath:  repoPath,
		AppName:   "example-api",
		SourceEnv: "dev",
		TargetEnv: "production",
	})
	require.NoError(t, err)
	require.Len(t, diffs, 1)
	assert.True(t, diffs[0].HasChanges(), "dev and production should differ")
}

func TestFileContentEqual(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.yaml")

	content := []byte("key: value\n")
	require.NoError(t, os.WriteFile(path, content, 0644))

	assert.True(t, fileContentEqual(path, content))
	assert.False(t, fileContentEqual(path, []byte("different\n")))
	assert.False(t, fileContentEqual(filepath.Join(dir, "missing.yaml"), content))
}
