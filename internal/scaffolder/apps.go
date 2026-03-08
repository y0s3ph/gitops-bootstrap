package scaffolder

import (
	"fmt"
	"path/filepath"
)

type applicationData struct {
	AppName  string
	EnvName  string
	AutoSync bool
	Prune    bool
}

func (s *Scaffolder) scaffoldAppDefinitions(appName string) error {
	for _, env := range s.config.Environments {
		data := applicationData{
			AppName:  appName,
			EnvName:  env.Name,
			AutoSync: env.AutoSync,
			Prune:    env.Prune,
		}

		outPath := filepath.Join("apps", fmt.Sprintf("%s-%s.yaml", appName, env.Name))
		if err := s.renderTemplateWithData("apps/application.yaml.tmpl", outPath, data); err != nil {
			return err
		}
	}

	return nil
}

// ScaffoldApp generates the full Kustomize structure and ArgoCD
// Application definitions for a single application across all
// configured environments.
func (s *Scaffolder) ScaffoldApp(name string, port int) error {
	if err := s.scaffoldAppEnvironments(name, port); err != nil {
		return fmt.Errorf("scaffolding environments for %s: %w", name, err)
	}

	if err := s.scaffoldAppDefinitions(name); err != nil {
		return fmt.Errorf("scaffolding app definitions for %s: %w", name, err)
	}

	return nil
}
