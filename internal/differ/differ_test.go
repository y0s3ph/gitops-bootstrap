package differ

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

func TestDiff_IdenticalEnvironments(t *testing.T) {
	repoPath := setupRepo(t)

	devOverlay := filepath.Join(repoPath, "environments", "dev", "example-api", "kustomization.yaml")
	stagingOverlay := filepath.Join(repoPath, "environments", "staging", "example-api", "kustomization.yaml")

	devContent, err := os.ReadFile(devOverlay)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(stagingOverlay, devContent, 0644))

	stagingDef := filepath.Join(repoPath, "apps", "example-api-staging.yaml")
	devDef := filepath.Join(repoPath, "apps", "example-api-dev.yaml")
	devDefContent, err := os.ReadFile(devDef)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(stagingDef, devDefContent, 0644))

	results, err := Diff(repoPath, "dev", "staging", "example-api")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.False(t, results[0].HasChanges())
}

func TestDiff_DetectsModifiedOverlay(t *testing.T) {
	repoPath := setupRepo(t)

	results, err := Diff(repoPath, "dev", "staging", "example-api")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.True(t, results[0].HasChanges(), "dev and staging should differ in replicas")

	found := false
	for _, f := range results[0].Files {
		if f.Status == FileModified {
			found = true
			break
		}
	}
	assert.True(t, found, "should have at least one modified file")
}

func TestDiff_AllApps(t *testing.T) {
	repoPath := setupRepo(t)

	results, err := Diff(repoPath, "dev", "production", "")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "example-api", results[0].AppName)
}

func TestDiff_InvalidSourceEnv(t *testing.T) {
	repoPath := setupRepo(t)

	_, err := Diff(repoPath, "nonexistent", "staging", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found in config")
}

func TestDiff_InvalidTargetEnv(t *testing.T) {
	repoPath := setupRepo(t)

	_, err := Diff(repoPath, "dev", "nonexistent", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found in config")
}

func TestDiff_InvalidApp(t *testing.T) {
	repoPath := setupRepo(t)

	_, err := Diff(repoPath, "dev", "staging", "nonexistent-app")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestDiff_MissingConfig(t *testing.T) {
	repoPath := t.TempDir()

	_, err := Diff(repoPath, "dev", "staging", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "loading config")
}

func TestDiff_FileOnlyInSource(t *testing.T) {
	repoPath := setupRepo(t)

	extraFile := filepath.Join(repoPath, "environments", "dev", "example-api", "extra.yaml")
	require.NoError(t, os.WriteFile(extraFile, []byte("key: value\n"), 0644))

	results, err := Diff(repoPath, "dev", "staging", "example-api")
	require.NoError(t, err)
	require.Len(t, results, 1)

	found := false
	for _, f := range results[0].Files {
		if f.RelPath == "extra.yaml" && f.Status == FileOnlyInSource {
			found = true
			break
		}
	}
	assert.True(t, found, "should detect file only in source")
}

func TestDiff_FileOnlyInTarget(t *testing.T) {
	repoPath := setupRepo(t)

	extraFile := filepath.Join(repoPath, "environments", "staging", "example-api", "extra.yaml")
	require.NoError(t, os.WriteFile(extraFile, []byte("key: value\n"), 0644))

	results, err := Diff(repoPath, "dev", "staging", "example-api")
	require.NoError(t, err)
	require.Len(t, results, 1)

	found := false
	for _, f := range results[0].Files {
		if f.RelPath == "extra.yaml" && f.Status == FileOnlyInTarget {
			found = true
			break
		}
	}
	assert.True(t, found, "should detect file only in target")
}

func TestDiff_HelmRepo(t *testing.T) {
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

	results, err := Diff(repoPath, "dev", "production", "example-api")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.True(t, results[0].HasChanges(), "dev and production Helm values should differ")
}

func TestDiff_MultipleApps(t *testing.T) {
	repoPath := setupRepo(t)

	cfg, err := config.Load(repoPath)
	require.NoError(t, err)

	s := scaffolder.New(cfg)
	require.NoError(t, s.ScaffoldApp("api-two", 8081))

	results, err := Diff(repoPath, "dev", "staging", "")
	require.NoError(t, err)
	assert.Len(t, results, 2, "should diff both apps")

	appNames := make(map[string]bool)
	for _, r := range results {
		appNames[r.AppName] = true
	}
	assert.True(t, appNames["example-api"])
	assert.True(t, appNames["api-two"])
}

func TestMyersDiff_Basic(t *testing.T) {
	src := []string{"a", "b", "c"}
	tgt := []string{"a", "d", "c"}

	ops := myersDiff(src, tgt)
	assert.NotEmpty(t, ops)

	// Verify ops correctly transform src into tgt
	var result []string
	si, ti := 0, 0
	for _, op := range ops {
		switch op {
		case opEqual:
			result = append(result, src[si])
			si++
			ti++
		case opDelete:
			si++
		case opInsert:
			result = append(result, tgt[ti])
			ti++
		}
	}
	assert.Equal(t, tgt, result)
}

func TestMyersDiff_EmptySource(t *testing.T) {
	ops := myersDiff(nil, []string{"a", "b"})
	assert.Len(t, ops, 2)
	for _, op := range ops {
		assert.Equal(t, opInsert, op)
	}
}

func TestMyersDiff_EmptyTarget(t *testing.T) {
	ops := myersDiff([]string{"a", "b"}, nil)
	assert.Len(t, ops, 2)
	for _, op := range ops {
		assert.Equal(t, opDelete, op)
	}
}

func TestMyersDiff_Identical(t *testing.T) {
	src := []string{"a", "b", "c"}
	ops := myersDiff(src, src)
	for _, op := range ops {
		assert.Equal(t, opEqual, op)
	}
}

func TestDiffResult_HasChanges(t *testing.T) {
	r := DiffResult{Files: []FileDiff{{Status: FileIdentical}}}
	assert.False(t, r.HasChanges())

	r.Files = append(r.Files, FileDiff{Status: FileModified})
	assert.True(t, r.HasChanges())
}
