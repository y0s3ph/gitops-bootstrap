package scaffolder

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScaffold_GeneratesPreCommitConfig(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "repo")
	cfg := testConfig(repoPath)

	s := New(cfg)
	_, err := s.Scaffold()
	require.NoError(t, err)

	preCommitPath := filepath.Join(repoPath, ".pre-commit-config.yaml")
	_, err = os.Stat(preCommitPath)
	require.NoError(t, err, ".pre-commit-config.yaml should exist")

	data, err := os.ReadFile(preCommitPath)
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "pre-commit-hooks")
	assert.Contains(t, content, "check-yaml")
	assert.Contains(t, content, "kubeconform")
	assert.Contains(t, content, "gostrap validate")
	assert.Contains(t, content, "trailing-whitespace")
	assert.Contains(t, content, "end-of-file-fixer")
}

func TestScaffold_PreCommitConfigIsIdempotent(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "repo")
	cfg := testConfig(repoPath)

	s := New(cfg)
	_, err := s.Scaffold()
	require.NoError(t, err)

	preCommitPath := filepath.Join(repoPath, ".pre-commit-config.yaml")
	firstContent, err := os.ReadFile(preCommitPath)
	require.NoError(t, err)

	s2 := New(cfg)
	result, err := s2.Scaffold()
	require.NoError(t, err)

	secondContent, err := os.ReadFile(preCommitPath)
	require.NoError(t, err)
	assert.Equal(t, string(firstContent), string(secondContent), "should not overwrite existing file")

	skipped := false
	for _, f := range result.Skipped {
		if filepath.Base(f) == ".pre-commit-config.yaml" {
			skipped = true
			break
		}
	}
	assert.True(t, skipped, ".pre-commit-config.yaml should be reported as skipped")
}

func TestScaffold_PreCommitConfigExcludesTemplates(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "repo")
	cfg := testConfig(repoPath)

	s := New(cfg)
	_, err := s.Scaffold()
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(repoPath, ".pre-commit-config.yaml"))
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "exclude", "should exclude template files from YAML checks")
	assert.Contains(t, content, ".tmpl")
}

func TestScaffold_PreCommitFluxRepo(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "repo")
	cfg := testFluxConfig(repoPath)

	s := New(cfg)
	_, err := s.Scaffold()
	require.NoError(t, err)

	preCommitPath := filepath.Join(repoPath, ".pre-commit-config.yaml")
	_, err = os.Stat(preCommitPath)
	require.NoError(t, err, ".pre-commit-config.yaml should exist for Flux repos too")

	data, err := os.ReadFile(preCommitPath)
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "gostrap validate")
	assert.Contains(t, content, "Kustomization")
	assert.Contains(t, content, "HelmRelease")
}

func TestScaffold_PreCommitKubeconformSkipsCRDs(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "repo")
	cfg := testConfig(repoPath)

	s := New(cfg)
	_, err := s.Scaffold()
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(repoPath, ".pre-commit-config.yaml"))
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "Application")
	assert.Contains(t, content, "SealedSecret")
	assert.Contains(t, content, "CustomResourceDefinition")
}
