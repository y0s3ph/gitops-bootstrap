package wizard

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/y0s3ph/gitops-bootstrap/internal/models"
)

// LoadConfig reads a YAML config file and returns a validated BootstrapConfig.
func LoadConfig(path string) (*models.BootstrapConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg models.BootstrapConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	if cfg.Controller.Version == "" {
		cfg.Controller.Version = models.DefaultControllerVersion(cfg.Controller.Type)
	}
	if len(cfg.Environments) == 0 {
		cfg.Environments = models.DefaultEnvironments()
	}
	if cfg.RepoPath == "" {
		cfg.RepoPath = "./gitops-repo"
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config file: %w", err)
	}

	return &cfg, nil
}
