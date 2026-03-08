package scaffolder

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/y0s3ph/gostrap/internal/models"
)

func testConfig(repoPath string) *models.BootstrapConfig {
	return &models.BootstrapConfig{
		Controller: models.ControllerConfig{
			Type:    models.ControllerArgoCD,
			Version: "2.13.1",
		},
		Secrets: models.SecretsConfig{
			Type: models.SecretsSealedSecrets,
		},
		Environments: models.DefaultEnvironments(),
		RepoPath:     repoPath,
	}
}

func testFluxConfig(repoPath string) *models.BootstrapConfig {
	return &models.BootstrapConfig{
		Controller: models.ControllerConfig{
			Type:    models.ControllerFlux,
			Version: "2.8.1",
		},
		Secrets: models.SecretsConfig{
			Type: models.SecretsSealedSecrets,
		},
		Environments: models.DefaultEnvironments(),
		RepoPath:     repoPath,
	}
}

func TestScaffold_CreatesDirectoryTree(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "gitops-repo")
	cfg := testConfig(repoPath)

	s := New(cfg)
	_, err := s.Scaffold()
	require.NoError(t, err)

	expectedDirs := []string{
		"bootstrap/argocd",
		"bootstrap/sealed-secrets",
		"apps",
		"environments/base",
		"environments/dev",
		"environments/staging",
		"environments/production",
		"platform",
		"policies",
		"docs",
	}

	for _, dir := range expectedDirs {
		fullPath := filepath.Join(repoPath, dir)
		info, err := os.Stat(fullPath)
		require.NoError(t, err, "directory %s should exist", dir)
		assert.True(t, info.IsDir(), "%s should be a directory", dir)
	}
}

func TestScaffold_CreatesBootstrapManifests(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "gitops-repo")
	cfg := testConfig(repoPath)

	s := New(cfg)
	_, err := s.Scaffold()
	require.NoError(t, err)

	expectedFiles := []string{
		"bootstrap/argocd/namespace.yaml",
		"bootstrap/argocd/kustomization.yaml",
		"bootstrap/argocd/argocd-cm-patch.yaml",
		"bootstrap/argocd/argocd-rbac-cm-patch.yaml",
		"bootstrap/argocd/appproject-default.yaml",
	}

	for _, f := range expectedFiles {
		fullPath := filepath.Join(repoPath, f)
		_, err := os.Stat(fullPath)
		require.NoError(t, err, "file %s should exist", f)
	}
}

func TestScaffold_CreatesRootApplication(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "gitops-repo")
	cfg := testConfig(repoPath)

	s := New(cfg)
	_, err := s.Scaffold()
	require.NoError(t, err)

	rootApp := filepath.Join(repoPath, "apps/_root.yaml")
	data, err := os.ReadFile(rootApp)
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "kind: Application")
	assert.Contains(t, content, "name: root")
	assert.Contains(t, content, "path: apps")
	assert.Contains(t, content, "selfHeal: true")
}

func TestScaffold_InjectsControllerVersion(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "gitops-repo")
	cfg := testConfig(repoPath)
	cfg.Controller.Version = "2.10.0"

	s := New(cfg)
	_, err := s.Scaffold()
	require.NoError(t, err)

	kustomization := filepath.Join(repoPath, "bootstrap/argocd/kustomization.yaml")
	data, err := os.ReadFile(kustomization)
	require.NoError(t, err)

	assert.Contains(t, string(data), "v2.10.0/manifests/install.yaml")
}

func TestScaffold_ESOBootstrapDirectory(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "gitops-repo")
	cfg := testConfig(repoPath)
	cfg.Secrets.Type = models.SecretsESO

	s := New(cfg)
	_, err := s.Scaffold()
	require.NoError(t, err)

	esoDir := filepath.Join(repoPath, "bootstrap/external-secrets")
	info, err := os.Stat(esoDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	sealedDir := filepath.Join(repoPath, "bootstrap/sealed-secrets")
	_, err = os.Stat(sealedDir)
	assert.True(t, os.IsNotExist(err), "sealed-secrets dir should not exist when ESO is selected")
}

func TestScaffold_CustomEnvironments(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "gitops-repo")
	cfg := testConfig(repoPath)
	cfg.Environments = []models.EnvironmentConfig{
		{Name: "test"},
		{Name: "canary"},
	}

	s := New(cfg)
	_, err := s.Scaffold()
	require.NoError(t, err)

	for _, env := range []string{"test", "canary"} {
		envDir := filepath.Join(repoPath, "environments", env)
		info, err := os.Stat(envDir)
		require.NoError(t, err, "environment dir %s should exist", env)
		assert.True(t, info.IsDir())
	}
}

func TestScaffold_Idempotent(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "gitops-repo")
	cfg := testConfig(repoPath)

	s1 := New(cfg)
	result1, err := s1.Scaffold()
	require.NoError(t, err)
	assert.NotEmpty(t, result1.Created)
	assert.Empty(t, result1.Skipped)

	s2 := New(cfg)
	result2, err := s2.Scaffold()
	require.NoError(t, err)
	assert.Empty(t, result2.Created, "second run should not create any files")
	assert.NotEmpty(t, result2.Skipped, "second run should skip all files")
	assert.Equal(t, len(result1.Created), len(result2.Skipped), "all created files should be skipped on re-run")
}

func TestScaffold_DoesNotOverwriteExisting(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "gitops-repo")
	cfg := testConfig(repoPath)

	s := New(cfg)
	_, err := s.Scaffold()
	require.NoError(t, err)

	customContent := []byte("# customized by user\n")
	rootApp := filepath.Join(repoPath, "apps/_root.yaml")
	require.NoError(t, os.WriteFile(rootApp, customContent, 0644))

	s2 := New(cfg)
	_, err = s2.Scaffold()
	require.NoError(t, err)

	data, err := os.ReadFile(rootApp)
	require.NoError(t, err)
	assert.Equal(t, customContent, data, "user customizations should be preserved")
}

func TestScaffold_GitkeepInEmptyDirs(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "gitops-repo")
	cfg := testConfig(repoPath)

	s := New(cfg)
	_, err := s.Scaffold()
	require.NoError(t, err)

	for _, dir := range []string{"platform", "policies", "docs"} {
		gitkeep := filepath.Join(repoPath, dir, ".gitkeep")
		_, err := os.Stat(gitkeep)
		require.NoError(t, err, ".gitkeep should exist in %s", dir)
	}
}

func TestScaffold_RBACPolicyContent(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "gitops-repo")
	cfg := testConfig(repoPath)

	s := New(cfg)
	_, err := s.Scaffold()
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(repoPath, "bootstrap/argocd/argocd-rbac-cm-patch.yaml"))
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "name: argocd-rbac-cm")
	assert.Contains(t, content, "policy.default: role:readonly")
	assert.Contains(t, content, "role:admin")
	assert.Contains(t, content, "role:developer")
	assert.Contains(t, content, "argocd-admins")
}

func TestScaffold_AppProjectScopedToEnvironments(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "gitops-repo")
	cfg := testConfig(repoPath)

	s := New(cfg)
	_, err := s.Scaffold()
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(repoPath, "bootstrap/argocd/appproject-default.yaml"))
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "kind: AppProject")
	assert.Contains(t, content, "name: default")
	assert.Contains(t, content, "namespace: dev")
	assert.Contains(t, content, "namespace: staging")
	assert.Contains(t, content, "namespace: production")
	assert.Contains(t, content, "namespace: argocd")
	assert.Contains(t, content, "warn: true", "orphaned resources should be enabled")
}

func TestScaffold_AppProjectCustomEnvironments(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "gitops-repo")
	cfg := testConfig(repoPath)
	cfg.Environments = []models.EnvironmentConfig{
		{Name: "qa"},
		{Name: "canary"},
	}

	s := New(cfg)
	_, err := s.Scaffold()
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(repoPath, "bootstrap/argocd/appproject-default.yaml"))
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "namespace: qa")
	assert.Contains(t, content, "namespace: canary")
	assert.NotContains(t, content, "namespace: dev")
	assert.NotContains(t, content, "namespace: production")
}

func TestScaffold_KustomizationIncludesRBAC(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "gitops-repo")
	cfg := testConfig(repoPath)

	s := New(cfg)
	_, err := s.Scaffold()
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(repoPath, "bootstrap/argocd/kustomization.yaml"))
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "appproject-default.yaml")
	assert.Contains(t, content, "argocd-rbac-cm-patch.yaml")
}

// --- Flux-specific scaffolding tests ---

func TestScaffoldFlux_CreatesFluxDirectoryTree(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "gitops-repo")
	cfg := testFluxConfig(repoPath)

	s := New(cfg)
	_, err := s.Scaffold()
	require.NoError(t, err)

	expectedDirs := []string{
		"bootstrap/flux-system",
		"bootstrap/sealed-secrets",
		"apps",
		"environments/base",
		"environments/dev",
		"environments/staging",
		"environments/production",
	}

	for _, dir := range expectedDirs {
		fullPath := filepath.Join(repoPath, dir)
		info, err := os.Stat(fullPath)
		require.NoError(t, err, "directory %s should exist", dir)
		assert.True(t, info.IsDir(), "%s should be a directory", dir)
	}

	_, err = os.Stat(filepath.Join(repoPath, "bootstrap/argocd"))
	assert.True(t, os.IsNotExist(err), "bootstrap/argocd should not exist when Flux is selected")
}

func TestScaffoldFlux_CreatesBootstrapManifests(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "gitops-repo")
	cfg := testFluxConfig(repoPath)

	s := New(cfg)
	_, err := s.Scaffold()
	require.NoError(t, err)

	expectedFiles := []string{
		"bootstrap/flux-system/namespace.yaml",
		"bootstrap/flux-system/kustomization.yaml",
		"bootstrap/flux-system/gotk-sync.yaml",
	}

	for _, f := range expectedFiles {
		fullPath := filepath.Join(repoPath, f)
		_, err := os.Stat(fullPath)
		require.NoError(t, err, "file %s should exist", f)
	}
}

func TestScaffoldFlux_KustomizationReferencesUpstream(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "gitops-repo")
	cfg := testFluxConfig(repoPath)

	s := New(cfg)
	_, err := s.Scaffold()
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(repoPath, "bootstrap/flux-system/kustomization.yaml"))
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "fluxcd/flux2/releases/download/v2.8.1/install.yaml")
}

func TestScaffoldFlux_GotKSyncContent(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "gitops-repo")
	cfg := testFluxConfig(repoPath)

	s := New(cfg)
	_, err := s.Scaffold()
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(repoPath, "bootstrap/flux-system/gotk-sync.yaml"))
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "kind: GitRepository")
	assert.Contains(t, content, "kind: Kustomization")
	assert.Contains(t, content, "name: gitops-repo")
	assert.Contains(t, content, "path: ./apps")
}

func TestScaffoldFlux_CreatesRootWithFluxCRDs(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "gitops-repo")
	cfg := testFluxConfig(repoPath)

	s := New(cfg)
	_, err := s.Scaffold()
	require.NoError(t, err)

	rootApp := filepath.Join(repoPath, "apps/_root.yaml")
	data, err := os.ReadFile(rootApp)
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "kind: GitRepository")
	assert.Contains(t, content, "kind: Kustomization")
	assert.Contains(t, content, "path: ./apps")
	assert.NotContains(t, content, "argoproj.io")
}

func TestScaffoldFlux_InjectsControllerVersion(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "gitops-repo")
	cfg := testFluxConfig(repoPath)
	cfg.Controller.Version = "2.7.0"

	s := New(cfg)
	_, err := s.Scaffold()
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(repoPath, "bootstrap/flux-system/kustomization.yaml"))
	require.NoError(t, err)

	assert.Contains(t, string(data), "v2.7.0/install.yaml")
}
