package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validConfig() BootstrapConfig {
	return BootstrapConfig{
		Controller: ControllerConfig{
			Type:    ControllerArgoCD,
			Version: "2.13.1",
		},
		Secrets: SecretsConfig{
			Type: SecretsSealedSecrets,
		},
		Environments: DefaultEnvironments(),
		RepoPath:     "./gitops-repo",
	}
}

func TestValidate_ValidConfig(t *testing.T) {
	cfg := validConfig()
	assert.NoError(t, cfg.Validate())
}

func TestValidate_MissingControllerType(t *testing.T) {
	cfg := validConfig()
	cfg.Controller.Type = ""
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "controller type is required")
}

func TestValidate_InvalidControllerType(t *testing.T) {
	cfg := validConfig()
	cfg.Controller.Type = "jenkins"
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid controller type")
}

func TestValidate_MissingControllerVersion(t *testing.T) {
	cfg := validConfig()
	cfg.Controller.Version = ""
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "controller version is required")
}

func TestValidate_MissingSecretsType(t *testing.T) {
	cfg := validConfig()
	cfg.Secrets.Type = ""
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "secrets type is required")
}

func TestValidate_InvalidSecretsType(t *testing.T) {
	cfg := validConfig()
	cfg.Secrets.Type = "plaintext"
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid secrets type")
}

func TestValidate_EmptyEnvironments(t *testing.T) {
	cfg := validConfig()
	cfg.Environments = nil
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one environment")
}

func TestValidate_EnvironmentWithoutName(t *testing.T) {
	cfg := validConfig()
	cfg.Environments = []EnvironmentConfig{{Name: ""}}
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

func TestValidate_MissingRepoPath(t *testing.T) {
	cfg := validConfig()
	cfg.RepoPath = ""
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "repo path is required")
}

func TestValidate_FluxController(t *testing.T) {
	cfg := validConfig()
	cfg.Controller.Type = ControllerFlux
	cfg.Controller.Version = "2.4.0"
	assert.NoError(t, cfg.Validate())
}

func TestValidate_AllSecretsTypes(t *testing.T) {
	for _, st := range []SecretsType{SecretsSealedSecrets, SecretsESO, SecretsSOPS} {
		cfg := validConfig()
		cfg.Secrets.Type = st
		assert.NoError(t, cfg.Validate(), "secrets type %s should be valid", st)
	}
}

func TestDefaultControllerVersion(t *testing.T) {
	assert.Equal(t, DefaultArgoCDVersion, DefaultControllerVersion(ControllerArgoCD))
	assert.Equal(t, DefaultFluxVersion, DefaultControllerVersion(ControllerFlux))
	assert.Equal(t, "", DefaultControllerVersion("unknown"))
}

func TestDefaultEnvironments(t *testing.T) {
	envs := DefaultEnvironments()
	require.Len(t, envs, 3)

	assert.Equal(t, "dev", envs[0].Name)
	assert.True(t, envs[0].AutoSync)
	assert.True(t, envs[0].Prune)
	assert.False(t, envs[0].RequirePR)

	assert.Equal(t, "staging", envs[1].Name)
	assert.True(t, envs[1].AutoSync)
	assert.False(t, envs[1].Prune)

	assert.Equal(t, "production", envs[2].Name)
	assert.False(t, envs[2].AutoSync)
	assert.False(t, envs[2].Prune)
	assert.True(t, envs[2].RequirePR)
}
