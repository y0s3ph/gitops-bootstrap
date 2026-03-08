package wizard

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/y0s3ph/gitops-bootstrap/internal/models"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 2)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#999999")).
			Italic(true)

	successStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#04B575"))
)

func banner() string {
	title := titleStyle.Render("gitops-bootstrap")
	subtitle := subtitleStyle.Render("From zero to GitOps in one command")
	return fmt.Sprintf("\n  %s\n  %s\n", title, subtitle)
}

// Run launches the interactive wizard and returns the resulting configuration.
func Run() (*models.BootstrapConfig, error) {
	fmt.Println(banner())

	var (
		controllerType string
		controllerVer  string
		secretsType    string
		envsRaw        string
		scaffoldExample bool
		repoPath       string
		clusterCtx     string
	)

	controllerForm := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("GitOps controller").
				Options(
					huh.NewOption("ArgoCD (recommended)", "argocd"),
					huh.NewOption("Flux CD", "flux"),
				).
				Value(&controllerType),

			huh.NewInput().
				Title("Controller version").
				Placeholder(models.DefaultArgoCDVersion).
				Value(&controllerVer),
		),
	).WithTheme(huh.ThemeCatppuccin())

	if err := controllerForm.Run(); err != nil {
		return nil, fmt.Errorf("controller selection: %w", err)
	}

	ct := models.ControllerType(controllerType)
	if controllerVer == "" {
		controllerVer = models.DefaultControllerVersion(ct)
	}

	secretsForm := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Secrets management").
				Options(
					huh.NewOption("Sealed Secrets (simple, self-contained)", "sealed-secrets"),
					huh.NewOption("External Secrets Operator (AWS SM, Vault, etc.)", "external-secrets"),
					huh.NewOption("SOPS (git-native encryption)", "sops"),
				).
				Value(&secretsType),
		),
	).WithTheme(huh.ThemeCatppuccin())

	if err := secretsForm.Run(); err != nil {
		return nil, fmt.Errorf("secrets selection: %w", err)
	}

	envForm := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Environments to create (comma-separated)").
				Placeholder("dev,staging,production").
				Value(&envsRaw),

			huh.NewConfirm().
				Title("Scaffold an example application?").
				Affirmative("Yes").
				Negative("No").
				Value(&scaffoldExample),
		),
	).WithTheme(huh.ThemeCatppuccin())

	if err := envForm.Run(); err != nil {
		return nil, fmt.Errorf("environment setup: %w", err)
	}

	outputForm := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Target repo path").
				Placeholder("./gitops-repo").
				Value(&repoPath),

			huh.NewInput().
				Title("Cluster context (leave empty for current)").
				Value(&clusterCtx),
		),
	).WithTheme(huh.ThemeCatppuccin())

	if err := outputForm.Run(); err != nil {
		return nil, fmt.Errorf("output configuration: %w", err)
	}

	if repoPath == "" {
		repoPath = "./gitops-repo"
	}

	envs := parseEnvironments(envsRaw)

	st := models.SecretsType(secretsType)
	cfg := &models.BootstrapConfig{
		Controller: models.ControllerConfig{
			Type:    ct,
			Version: controllerVer,
		},
		Secrets: models.SecretsConfig{
			Type:    st,
			Version: models.DefaultSecretsVersion(st),
		},
		Environments:    envs,
		RepoPath:        repoPath,
		ClusterContext:  clusterCtx,
		ScaffoldExample: scaffoldExample,
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

func parseEnvironments(raw string) []models.EnvironmentConfig {
	if strings.TrimSpace(raw) == "" {
		return models.DefaultEnvironments()
	}

	parts := strings.Split(raw, ",")
	envs := make([]models.EnvironmentConfig, 0, len(parts))
	for _, p := range parts {
		name := strings.TrimSpace(p)
		if name == "" {
			continue
		}
		env := models.EnvironmentConfig{
			Name:     name,
			AutoSync: name != "production",
			Prune:    name == "dev",
		}
		if name == "production" {
			env.RequirePR = true
		}
		envs = append(envs, env)
	}

	if len(envs) == 0 {
		return models.DefaultEnvironments()
	}
	return envs
}
