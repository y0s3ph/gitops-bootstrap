package validator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/y0s3ph/gostrap/internal/config"
	"github.com/y0s3ph/gostrap/internal/models"
)

type Severity int

const (
	SeverityError Severity = iota
	SeverityWarning
)

type Issue struct {
	Severity Severity
	Path     string
	Message  string
}

func (i Issue) String() string {
	sev := "ERROR"
	if i.Severity == SeverityWarning {
		sev = "WARN"
	}
	if i.Path != "" {
		return fmt.Sprintf("[%s] %s: %s", sev, i.Path, i.Message)
	}
	return fmt.Sprintf("[%s] %s", sev, i.Message)
}

type Result struct {
	Issues []Issue
}

func (r *Result) Errors() []Issue {
	var errs []Issue
	for _, i := range r.Issues {
		if i.Severity == SeverityError {
			errs = append(errs, i)
		}
	}
	return errs
}

func (r *Result) Warnings() []Issue {
	var warns []Issue
	for _, i := range r.Issues {
		if i.Severity == SeverityWarning {
			warns = append(warns, i)
		}
	}
	return warns
}

func (r *Result) IsValid() bool {
	return len(r.Errors()) == 0
}

func (r *Result) addError(path, msg string) {
	r.Issues = append(r.Issues, Issue{Severity: SeverityError, Path: path, Message: msg})
}

func (r *Result) addWarning(path, msg string) {
	r.Issues = append(r.Issues, Issue{Severity: SeverityWarning, Path: path, Message: msg})
}

// Validate checks the integrity of a gostrap-managed repository.
func Validate(repoPath string) *Result {
	result := &Result{}

	cfg, err := config.Load(repoPath)
	if err != nil {
		result.addError(config.FileName, fmt.Sprintf("cannot load config: %v", err))
		return result
	}

	checkRootApp(repoPath, result)
	checkBootstrapDir(repoPath, cfg, result)

	apps := discoverApps(repoPath)
	if len(apps) == 0 {
		result.addWarning("environments/base/", "no applications found")
		return result
	}

	for _, app := range apps {
		checkAppBase(repoPath, app, cfg, result)
		for _, env := range cfg.Environments {
			checkAppOverlay(repoPath, app, env, cfg, result)
			checkAppDefinition(repoPath, app, env, result)
		}
	}

	return result
}

func checkRootApp(repoPath string, result *Result) {
	rootApp := filepath.Join(repoPath, "apps", "_root.yaml")
	if _, err := os.Stat(rootApp); os.IsNotExist(err) {
		result.addError("apps/_root.yaml", "missing root application")
		return
	}
	validateYAML(rootApp, "apps/_root.yaml", result)
}

func checkBootstrapDir(repoPath string, cfg *models.BootstrapConfig, result *Result) {
	var bootstrapDir string
	switch cfg.Controller.Type {
	case models.ControllerFlux:
		bootstrapDir = "bootstrap/flux-system"
	default:
		bootstrapDir = "bootstrap/argocd"
	}

	fullPath := filepath.Join(repoPath, bootstrapDir)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		result.addError(bootstrapDir, "missing bootstrap directory for "+string(cfg.Controller.Type))
	}
}

func discoverApps(repoPath string) []string {
	baseDir := filepath.Join(repoPath, "environments", "base")
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil
	}

	var apps []string
	for _, e := range entries {
		if e.IsDir() {
			apps = append(apps, e.Name())
		}
	}
	return apps
}

func checkAppBase(repoPath, app string, cfg *models.BootstrapConfig, result *Result) {
	basePath := filepath.Join(repoPath, "environments", "base", app)

	if cfg.ManifestType == models.ManifestHelm {
		checkFile(basePath, "Chart.yaml", result)
		checkFile(basePath, "values.yaml", result)
	} else {
		checkFile(basePath, "kustomization.yaml", result)
		checkFile(basePath, "deployment.yaml", result)
		checkFile(basePath, "service.yaml", result)
	}

	yamlFiles := findYAMLFiles(basePath)
	for _, f := range yamlFiles {
		rel, _ := filepath.Rel(repoPath, f)
		validateYAML(f, rel, result)
	}
}

func checkAppOverlay(repoPath, app string, env models.EnvironmentConfig, cfg *models.BootstrapConfig, result *Result) {
	envAppDir := filepath.Join(repoPath, "environments", env.Name, app)

	if cfg.ManifestType == models.ManifestHelm {
		overlayFile := filepath.Join(envAppDir, "values.yaml")
		rel := filepath.Join("environments", env.Name, app, "values.yaml")
		if _, err := os.Stat(overlayFile); os.IsNotExist(err) {
			result.addError(rel, "missing Helm values overlay")
			return
		}
		validateYAML(overlayFile, rel, result)
	} else {
		overlayFile := filepath.Join(envAppDir, "kustomization.yaml")
		rel := filepath.Join("environments", env.Name, app, "kustomization.yaml")
		if _, err := os.Stat(overlayFile); os.IsNotExist(err) {
			result.addError(rel, "missing Kustomize overlay")
			return
		}
		validateYAML(overlayFile, rel, result)
	}
}

func checkAppDefinition(repoPath, app string, env models.EnvironmentConfig, result *Result) {
	defFile := filepath.Join(repoPath, "apps", fmt.Sprintf("%s-%s.yaml", app, env.Name))
	rel := filepath.Join("apps", fmt.Sprintf("%s-%s.yaml", app, env.Name))

	if _, err := os.Stat(defFile); os.IsNotExist(err) {
		result.addError(rel, "missing app definition")
		return
	}
	validateYAML(defFile, rel, result)
}

func checkFile(dir, name string, result *Result) {
	fullPath := filepath.Join(dir, name)
	parent := filepath.Base(filepath.Dir(fullPath))
	rel := filepath.Join("environments/base", parent, name)

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		result.addError(rel, "missing file")
	}
}

func validateYAML(path, relPath string, result *Result) {
	data, err := os.ReadFile(path)
	if err != nil {
		result.addError(relPath, fmt.Sprintf("cannot read: %v", err))
		return
	}

	content := string(data)

	if len(strings.TrimSpace(content)) == 0 {
		result.addWarning(relPath, "file is empty")
		return
	}

	if strings.Contains(content, "{{") {
		return
	}

	var doc any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		result.addError(relPath, fmt.Sprintf("invalid YAML: %v", err))
	}
}

func findYAMLFiles(dir string) []string {
	var files []string
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		ext := filepath.Ext(path)
		if ext == ".yaml" || ext == ".yml" {
			files = append(files, path)
		}
		return nil
	})
	return files
}
