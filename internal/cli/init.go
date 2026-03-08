package cli

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/y0s3ph/gitops-bootstrap/internal/models"
	"github.com/y0s3ph/gitops-bootstrap/internal/wizard"
)

var successStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#04B575"))
var dimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#999999"))

var initFlags struct {
	configFile     string
	controller     string
	controllerVer  string
	secrets        string
	environments   string
	repoPath       string
	clusterContext string
	scaffoldExample bool
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Bootstrap a GitOps workflow on a Kubernetes cluster",
	Long: `Interactive wizard to set up a production-ready GitOps workflow.

Guides you through selecting a GitOps controller, secrets management,
environments, and repository structure. Produces a complete, ready-to-use
GitOps repository and installs the controller on your cluster.

Use --config to load from a YAML file, or pass individual flags for
non-interactive (CI/automation) usage.`,
	RunE: runInit,
}

func init() {
	f := initCmd.Flags()
	f.StringVar(&initFlags.configFile, "config", "", "Path to a YAML config file (skips interactive wizard)")
	f.StringVar(&initFlags.controller, "controller", "", "GitOps controller: argocd or flux")
	f.StringVar(&initFlags.controllerVer, "controller-version", "", "Controller version (default: latest stable)")
	f.StringVar(&initFlags.secrets, "secrets", "", "Secrets management: sealed-secrets, external-secrets, or sops")
	f.StringVar(&initFlags.environments, "environments", "", "Comma-separated list of environments (default: dev,staging,production)")
	f.StringVar(&initFlags.repoPath, "repo-path", "", "Target repository path (default: ./gitops-repo)")
	f.StringVar(&initFlags.clusterContext, "cluster-context", "", "Kubernetes cluster context")
	f.BoolVar(&initFlags.scaffoldExample, "scaffold-example", false, "Scaffold an example application")

	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	var cfg *models.BootstrapConfig
	var err error

	switch {
	case initFlags.configFile != "":
		cfg, err = wizard.LoadConfig(initFlags.configFile)
	case isNonInteractive():
		cfg, err = buildConfigFromFlags()
	default:
		cfg, err = wizard.Run()
	}

	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Println(successStyle.Render("✓ Configuration complete"))
	fmt.Println()

	out, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshalling config: %w", err)
	}
	fmt.Println(dimStyle.Render("Generated configuration:"))
	fmt.Println(string(out))

	fmt.Println(dimStyle.Render("Next: scaffolder and installer will use this configuration."))
	fmt.Println(dimStyle.Render("(not yet implemented — coming in Phase 1)"))

	return nil
}

// isNonInteractive returns true if enough flags were provided to skip the wizard.
func isNonInteractive() bool {
	return initFlags.controller != "" && initFlags.secrets != ""
}

func buildConfigFromFlags() (*models.BootstrapConfig, error) {
	ct := models.ControllerType(initFlags.controller)
	ver := initFlags.controllerVer
	if ver == "" {
		ver = models.DefaultControllerVersion(ct)
	}

	repoPath := initFlags.repoPath
	if repoPath == "" {
		repoPath = "./gitops-repo"
	}

	var envs []models.EnvironmentConfig
	if initFlags.environments != "" {
		for _, name := range strings.Split(initFlags.environments, ",") {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			envs = append(envs, models.EnvironmentConfig{
				Name:      name,
				AutoSync:  name != "production",
				Prune:     name == "dev",
				RequirePR: name == "production",
			})
		}
	}
	if len(envs) == 0 {
		envs = models.DefaultEnvironments()
	}

	cfg := &models.BootstrapConfig{
		Controller: models.ControllerConfig{
			Type:    ct,
			Version: ver,
		},
		Secrets: models.SecretsConfig{
			Type: models.SecretsType(initFlags.secrets),
		},
		Environments:    envs,
		RepoPath:        repoPath,
		ClusterContext:  initFlags.clusterContext,
		ScaffoldExample: initFlags.scaffoldExample,
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid flags: %w", err)
	}

	return cfg, nil
}
