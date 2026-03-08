package scaffolder

import (
	"fmt"
	"path/filepath"

	"github.com/y0s3ph/gostrap/internal/models"
)

type applicationData struct {
	AppName     string
	EnvName     string
	AutoSync    bool
	Prune       bool
	SecretsType string
}

func (s *Scaffolder) appDefinitionTemplate() string {
	if s.config.Controller.Type == models.ControllerFlux {
		return "apps/flux-kustomization.yaml.tmpl"
	}
	return "apps/application.yaml.tmpl"
}

func (s *Scaffolder) scaffoldAppDefinitions(appName string) error {
	tmpl := s.appDefinitionTemplate()

	for _, env := range s.config.Environments {
		data := applicationData{
			AppName:     appName,
			EnvName:     env.Name,
			AutoSync:    env.AutoSync,
			Prune:       env.Prune,
			SecretsType: string(s.config.Secrets.Type),
		}

		outPath := filepath.Join("apps", fmt.Sprintf("%s-%s.yaml", appName, env.Name))
		if err := s.renderTemplateWithData(tmpl, outPath, data); err != nil {
			return err
		}
	}

	return nil
}

// ScaffoldApp generates the full Kustomize structure and controller-specific
// definitions (ArgoCD Application or Flux Kustomization) for a single
// application across all configured environments.
func (s *Scaffolder) ScaffoldApp(name string, port int) error {
	if err := s.scaffoldAppEnvironments(name, port); err != nil {
		return fmt.Errorf("scaffolding environments for %s: %w", name, err)
	}

	if err := s.scaffoldAppDefinitions(name); err != nil {
		return fmt.Errorf("scaffolding app definitions for %s: %w", name, err)
	}

	return nil
}
