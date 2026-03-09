package scaffolder

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/y0s3ph/gostrap/internal/models"
)

// DiscoverApps returns the names of applications found under environments/base/.
func (s *Scaffolder) DiscoverApps() ([]string, error) {
	baseDir := filepath.Join(s.root, "environments", "base")
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading %s: %w", baseDir, err)
	}

	var apps []string
	for _, e := range entries {
		if e.IsDir() {
			apps = append(apps, e.Name())
		}
	}
	return apps, nil
}

// ScaffoldEnv creates the environment overlay and controller-specific app
// definitions for every existing application in the repo.
func (s *Scaffolder) ScaffoldEnv(env models.EnvironmentConfig) error {
	envDir := filepath.Join(s.root, "environments", env.Name)
	if err := os.MkdirAll(envDir, 0755); err != nil {
		return fmt.Errorf("creating environment directory: %w", err)
	}

	apps, err := s.DiscoverApps()
	if err != nil {
		return err
	}

	for _, appName := range apps {
		if err := s.scaffoldEnvOverlay(appName, env); err != nil {
			return fmt.Errorf("scaffolding overlay for %s/%s: %w", env.Name, appName, err)
		}

		if err := s.scaffoldEnvAppDefinition(appName, env); err != nil {
			return fmt.Errorf("scaffolding app definition for %s/%s: %w", env.Name, appName, err)
		}
	}

	return nil
}

func (s *Scaffolder) scaffoldEnvOverlay(appName string, env models.EnvironmentConfig) error {
	replicas := defaultReplicas[env.Name]
	if replicas == 0 {
		replicas = 1
	}

	data := overlayData{
		AppName:  appName,
		EnvName:  env.Name,
		Replicas: replicas,
	}

	if s.config.ManifestType == models.ManifestHelm {
		outPath := filepath.Join("environments", env.Name, appName, "values.yaml")
		return s.renderTemplateWithData("environments/helm-overlay/values.yaml.tmpl", outPath, data)
	}

	outPath := filepath.Join("environments", env.Name, appName, "kustomization.yaml")
	return s.renderTemplateWithData("environments/overlay/kustomization.yaml.tmpl", outPath, data)
}

func (s *Scaffolder) scaffoldEnvAppDefinition(appName string, env models.EnvironmentConfig) error {
	tmpl := s.appDefinitionTemplate()

	replicas := defaultReplicas[env.Name]
	if replicas == 0 {
		replicas = 1
	}

	data := applicationData{
		AppName:     appName,
		EnvName:     env.Name,
		AutoSync:    env.AutoSync,
		Prune:       env.Prune,
		Replicas:    replicas,
		SecretsType: string(s.config.Secrets.Type),
	}

	outPath := filepath.Join("apps", fmt.Sprintf("%s-%s.yaml", appName, env.Name))
	return s.renderTemplateWithData(tmpl, outPath, data)
}
