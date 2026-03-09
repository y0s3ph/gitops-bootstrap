package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/y0s3ph/gostrap/internal/models"
)

const FileName = ".gostrap.yaml"

// RepoConfig is the subset of BootstrapConfig persisted in the generated repo.
// It stores only the choices that downstream commands (add-app, add-env) need.
type RepoConfig struct {
	Controller   controllerRef          `yaml:"controller"`
	Secrets      secretsRef             `yaml:"secrets"`
	ManifestType models.ManifestType    `yaml:"manifest_type"`
	Environments []models.EnvironmentConfig `yaml:"environments"`
}

type controllerRef struct {
	Type    models.ControllerType `yaml:"type"`
	Version string                `yaml:"version"`
}

type secretsRef struct {
	Type models.SecretsType `yaml:"type"`
}

// Save writes a .gostrap.yaml into the repo root.
func Save(repoPath string, cfg *models.BootstrapConfig) error {
	rc := RepoConfig{
		Controller: controllerRef{
			Type:    cfg.Controller.Type,
			Version: cfg.Controller.Version,
		},
		Secrets: secretsRef{
			Type: cfg.Secrets.Type,
		},
		ManifestType: cfg.ManifestType,
		Environments: cfg.Environments,
	}

	data, err := yaml.Marshal(rc)
	if err != nil {
		return fmt.Errorf("marshalling repo config: %w", err)
	}

	path := filepath.Join(repoPath, FileName)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing %s: %w", FileName, err)
	}

	return nil
}

// Load reads .gostrap.yaml from a repo path and returns a BootstrapConfig
// populated with enough information for add-app / add-env operations.
func Load(repoPath string) (*models.BootstrapConfig, error) {
	path := filepath.Join(repoPath, FileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w (was this repo created with gostrap init?)", FileName, err)
	}

	var rc RepoConfig
	if err := yaml.Unmarshal(data, &rc); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", FileName, err)
	}

	if rc.ManifestType == "" {
		rc.ManifestType = models.ManifestKustomize
	}

	cfg := &models.BootstrapConfig{
		Controller: models.ControllerConfig{
			Type:    rc.Controller.Type,
			Version: rc.Controller.Version,
		},
		Secrets: models.SecretsConfig{
			Type: rc.Secrets.Type,
		},
		ManifestType: rc.ManifestType,
		Environments: rc.Environments,
		RepoPath:     repoPath,
	}

	return cfg, nil
}
