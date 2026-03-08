package models

import "fmt"

type ControllerType string

const (
	ControllerArgoCD ControllerType = "argocd"
	ControllerFlux   ControllerType = "flux"
)

const DefaultArgoCDVersion = "2.13.1"
const DefaultFluxVersion = "2.4.0"
const DefaultSealedSecretsVersion = "0.27.3"

func DefaultControllerVersion(ct ControllerType) string {
	switch ct {
	case ControllerArgoCD:
		return DefaultArgoCDVersion
	case ControllerFlux:
		return DefaultFluxVersion
	default:
		return ""
	}
}

func DefaultSecretsVersion(st SecretsType) string {
	switch st {
	case SecretsSealedSecrets:
		return DefaultSealedSecretsVersion
	default:
		return ""
	}
}

type SecretsType string

const (
	SecretsSealedSecrets SecretsType = "sealed-secrets"
	SecretsESO           SecretsType = "external-secrets"
	SecretsSOPS          SecretsType = "sops"
)

type ControllerConfig struct {
	Type    ControllerType `yaml:"type"`
	Version string         `yaml:"version"`
	Ingress IngressConfig  `yaml:"ingress"`
}

type IngressConfig struct {
	Enabled bool   `yaml:"enabled"`
	Host    string `yaml:"host"`
	TLS     bool   `yaml:"tls"`
}

type SecretsConfig struct {
	Type    SecretsType `yaml:"type"`
	Version string      `yaml:"version,omitempty"`
}

type EnvironmentConfig struct {
	Name      string `yaml:"name"`
	AutoSync  bool   `yaml:"auto_sync"`
	Prune     bool   `yaml:"prune"`
	RequirePR bool   `yaml:"require_pr"`
}

type ApplicationConfig struct {
	Name       string         `yaml:"name"`
	Type       string         `yaml:"type"`
	Port       int            `yaml:"port"`
	Replicas   map[string]int `yaml:"replicas"`
	HasIngress bool           `yaml:"has_ingress"`
	HasHPA     bool           `yaml:"has_hpa"`
	HPA        HPAConfig      `yaml:"hpa,omitempty"`
}

type HPAConfig struct {
	MinReplicas int `yaml:"min_replicas"`
	MaxReplicas int `yaml:"max_replicas"`
	TargetCPU   int `yaml:"target_cpu"`
}

type PlatformServicesConfig struct {
	CertManager  bool `yaml:"cert_manager"`
	ExternalDNS  bool `yaml:"external_dns"`
	IngressNginx bool `yaml:"ingress_nginx"`
	Monitoring   bool `yaml:"monitoring"`
}

type PoliciesConfig struct {
	Enabled bool   `yaml:"enabled"`
	Engine  string `yaml:"engine,omitempty"`
}

type BootstrapConfig struct {
	Controller       ControllerConfig       `yaml:"controller"`
	Secrets          SecretsConfig          `yaml:"secrets"`
	Environments     []EnvironmentConfig    `yaml:"environments"`
	Applications     []ApplicationConfig    `yaml:"applications,omitempty"`
	PlatformServices PlatformServicesConfig `yaml:"platform_services,omitempty"`
	Policies         PoliciesConfig         `yaml:"policies,omitempty"`
	RepoPath         string                 `yaml:"repo_path"`
	ClusterContext   string                 `yaml:"cluster_context,omitempty"`
	ScaffoldExample  bool                   `yaml:"scaffold_example"`
}

func DefaultEnvironments() []EnvironmentConfig {
	return []EnvironmentConfig{
		{Name: "dev", AutoSync: true, Prune: true},
		{Name: "staging", AutoSync: true, Prune: false},
		{Name: "production", AutoSync: false, Prune: false, RequirePR: true},
	}
}

func (c *BootstrapConfig) Validate() error {
	if c.Controller.Type == "" {
		return fmt.Errorf("controller type is required")
	}
	if c.Controller.Type != ControllerArgoCD && c.Controller.Type != ControllerFlux {
		return fmt.Errorf("invalid controller type: %s (must be 'argocd' or 'flux')", c.Controller.Type)
	}
	if c.Controller.Version == "" {
		return fmt.Errorf("controller version is required")
	}
	if c.Secrets.Type == "" {
		return fmt.Errorf("secrets type is required")
	}
	if c.Secrets.Type != SecretsSealedSecrets && c.Secrets.Type != SecretsESO && c.Secrets.Type != SecretsSOPS {
		return fmt.Errorf("invalid secrets type: %s", c.Secrets.Type)
	}
	if len(c.Environments) == 0 {
		return fmt.Errorf("at least one environment is required")
	}
	for i, env := range c.Environments {
		if env.Name == "" {
			return fmt.Errorf("environment %d: name is required", i)
		}
	}
	if c.RepoPath == "" {
		return fmt.Errorf("repo path is required")
	}
	return nil
}
