package scaffolder

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/y0s3ph/gitops-bootstrap/internal/models"
	"github.com/y0s3ph/gitops-bootstrap/internal/templates"
)

type Result struct {
	Created []string
	Skipped []string
}

type Scaffolder struct {
	config *models.BootstrapConfig
	root   string
	result Result
}

func New(cfg *models.BootstrapConfig) *Scaffolder {
	return &Scaffolder{
		config: cfg,
		root:   cfg.RepoPath,
	}
}

// Scaffold generates the full GitOps repository structure.
// It is idempotent: existing files are never overwritten.
func (s *Scaffolder) Scaffold() (*Result, error) {
	if err := s.createDirectories(); err != nil {
		return nil, fmt.Errorf("creating directories: %w", err)
	}

	if err := s.renderBootstrap(); err != nil {
		return nil, fmt.Errorf("rendering bootstrap manifests: %w", err)
	}

	if err := s.renderAppOfApps(); err != nil {
		return nil, fmt.Errorf("rendering App of Apps: %w", err)
	}

	if err := s.renderDocs(); err != nil {
		return nil, fmt.Errorf("rendering documentation: %w", err)
	}

	if s.config.ScaffoldExample {
		if err := s.ScaffoldApp("example-api", 8080); err != nil {
			return nil, fmt.Errorf("scaffolding example app: %w", err)
		}
	}

	return &s.result, nil
}

func (s *Scaffolder) createDirectories() error {
	dirs := []string{
		"bootstrap/argocd",
		"apps",
		"environments/base",
		"platform",
		"policies",
		"docs",
	}

	switch s.config.Secrets.Type {
	case models.SecretsSealedSecrets:
		dirs = append(dirs, "bootstrap/sealed-secrets")
	case models.SecretsESO:
		dirs = append(dirs, "bootstrap/external-secrets")
	}

	for _, env := range s.config.Environments {
		dirs = append(dirs, filepath.Join("environments", env.Name))
	}

	for _, dir := range dirs {
		fullPath := filepath.Join(s.root, dir)
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			return fmt.Errorf("creating %s: %w", dir, err)
		}
	}

	emptyDirs := []string{"platform", "policies", "docs"}
	for _, dir := range emptyDirs {
		gitkeep := filepath.Join(s.root, dir, ".gitkeep")
		if err := s.writeFileIfNotExists(gitkeep, []byte("")); err != nil {
			return err
		}
	}

	return nil
}

func (s *Scaffolder) renderBootstrap() error {
	tmplFiles := []struct {
		tmplPath string
		outPath  string
	}{
		{"bootstrap/argocd/namespace.yaml.tmpl", "bootstrap/argocd/namespace.yaml"},
		{"bootstrap/argocd/kustomization.yaml.tmpl", "bootstrap/argocd/kustomization.yaml"},
		{"bootstrap/argocd/argocd-cm-patch.yaml.tmpl", "bootstrap/argocd/argocd-cm-patch.yaml"},
		{"bootstrap/argocd/argocd-rbac-cm-patch.yaml.tmpl", "bootstrap/argocd/argocd-rbac-cm-patch.yaml"},
		{"bootstrap/argocd/appproject-default.yaml.tmpl", "bootstrap/argocd/appproject-default.yaml"},
	}

	for _, tf := range tmplFiles {
		if err := s.renderTemplate(tf.tmplPath, tf.outPath); err != nil {
			return err
		}
	}

	return nil
}

func (s *Scaffolder) renderAppOfApps() error {
	return s.renderTemplate("apps/_root.yaml.tmpl", "apps/_root.yaml")
}

func (s *Scaffolder) renderTemplate(tmplPath, outPath string) error {
	return s.renderTemplateWithData(tmplPath, outPath, s.config)
}

func (s *Scaffolder) renderTemplateWithData(tmplPath, outPath string, data any) error {
	content, err := templates.FS.ReadFile(tmplPath)
	if err != nil {
		return fmt.Errorf("reading template %s: %w", tmplPath, err)
	}

	tmpl, err := template.New(filepath.Base(tmplPath)).Parse(string(content))
	if err != nil {
		return fmt.Errorf("parsing template %s: %w", tmplPath, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("executing template %s: %w", tmplPath, err)
	}

	fullPath := filepath.Join(s.root, outPath)
	return s.writeFileIfNotExists(fullPath, buf.Bytes())
}

func (s *Scaffolder) writeFileIfNotExists(path string, data []byte) error {
	if _, err := os.Stat(path); err == nil {
		s.result.Skipped = append(s.result.Skipped, path)
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating parent dir for %s: %w", path, err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}

	s.result.Created = append(s.result.Created, path)
	return nil
}
