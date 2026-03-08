package cli

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/y0s3ph/gostrap/internal/installer"
	"github.com/y0s3ph/gostrap/internal/models"
	"github.com/y0s3ph/gostrap/internal/scaffolder"
	"github.com/y0s3ph/gostrap/internal/wizard"
)

var successStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#04B575"))
var dimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#999999"))
var warnStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFAA00"))

var initFlags struct {
	configFile      string
	controller      string
	controllerVer   string
	secrets         string
	environments    string
	repoPath        string
	clusterContext  string
	scaffoldExample bool
	skipInstall     bool
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
	f.BoolVar(&initFlags.skipInstall, "skip-install", false, "Skip cluster installation (only scaffold the repo)")

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

	s := scaffolder.New(cfg)
	result, err := s.Scaffold()
	if err != nil {
		return fmt.Errorf("scaffolding repo: %w", err)
	}

	if len(result.Created) > 0 {
		fmt.Println(successStyle.Render("✓ Repository structure generated"))
		for _, f := range result.Created {
			fmt.Printf("  %s %s\n", successStyle.Render("+"), f)
		}
	}
	if len(result.Skipped) > 0 {
		fmt.Println()
		fmt.Println(dimStyle.Render("Skipped (already exist):"))
		for _, f := range result.Skipped {
			fmt.Printf("  %s %s\n", dimStyle.Render("·"), f)
		}
	}

	shouldInstall := cfg.ClusterContext != "" && !initFlags.skipInstall
	if shouldInstall {
		if err := installController(cfg); err != nil {
			return err
		}

		if err := installSecretsManager(cfg); err != nil {
			return err
		}

		printPostInstallSteps(cfg)
	} else {
		printScaffoldOnlySteps(cfg)
	}

	return nil
}

func installController(cfg *models.BootstrapConfig) error {
	progress := func(step string) {
		fmt.Printf("  %s %s\n", successStyle.Render("✓"), step)
	}

	switch cfg.Controller.Type {
	case models.ControllerFlux:
		fmt.Println()
		fmt.Printf("Installing Flux v%s on cluster %s...\n", cfg.Controller.Version, cfg.ClusterContext)
		fmt.Println()

		fluxInstaller := installer.NewFlux(cfg)
		if err := fluxInstaller.Install(progress); err != nil {
			return fmt.Errorf("installing Flux: %w", err)
		}

		fmt.Println()
		fmt.Println(successStyle.Render("✓ Flux installed and ready"))

	default:
		fmt.Println()
		fmt.Printf("Installing ArgoCD v%s on cluster %s...\n", cfg.Controller.Version, cfg.ClusterContext)
		fmt.Println()

		argoInstaller := installer.NewArgoCD(cfg)
		if err := argoInstaller.Install(progress); err != nil {
			return fmt.Errorf("installing ArgoCD: %w", err)
		}

		fmt.Println()
		fmt.Println(successStyle.Render("✓ ArgoCD installed and ready"))
	}

	return nil
}

func installSecretsManager(cfg *models.BootstrapConfig) error {
	progress := func(step string) {
		fmt.Printf("  %s %s\n", successStyle.Render("✓"), step)
	}

	switch cfg.Secrets.Type {
	case models.SecretsSealedSecrets:
		fmt.Println()
		fmt.Printf("Setting up Sealed Secrets v%s...\n", cfg.Secrets.Version)
		fmt.Println()

		if err := installer.NewSealedSecrets(cfg).Install(progress); err != nil {
			return fmt.Errorf("installing Sealed Secrets: %w", err)
		}

		fmt.Println()
		fmt.Println(successStyle.Render("✓ Sealed Secrets ready"))

	case models.SecretsESO:
		fmt.Println()
		fmt.Printf("Setting up External Secrets Operator v%s...\n", cfg.Secrets.Version)
		fmt.Println()

		if err := installer.NewESO(cfg).Install(progress); err != nil {
			return fmt.Errorf("installing External Secrets Operator: %w", err)
		}

		fmt.Println()
		fmt.Println(successStyle.Render("✓ External Secrets Operator ready"))

	case models.SecretsSOPS:
		fmt.Println()
		fmt.Println("Setting up SOPS with age encryption...")
		fmt.Println()

		if err := installer.NewSOPS(cfg).Install(progress); err != nil {
			return fmt.Errorf("setting up SOPS: %w", err)
		}

		fmt.Println()
		fmt.Println(successStyle.Render("✓ SOPS encryption ready"))
	}

	return nil
}

func controllerBootstrapPath(cfg *models.BootstrapConfig) string {
	if cfg.Controller.Type == models.ControllerFlux {
		return cfg.RepoPath + "/bootstrap/flux-system/"
	}
	return cfg.RepoPath + "/bootstrap/argocd/"
}

func printPostInstallSteps(cfg *models.BootstrapConfig) {
	fmt.Println()
	fmt.Println(dimStyle.Render("Next steps:"))
	fmt.Printf("  1. cd %s && git init && git add -A && git commit -m \"feat: initial gitops structure\"\n", cfg.RepoPath)
	fmt.Println("  2. Push to your Git provider")
	fmt.Println("  3. Update apps/_root.yaml with your Git repo URL")

	if cfg.Controller.Type == models.ControllerArgoCD {
		fmt.Println()
		fmt.Println(dimStyle.Render("Access ArgoCD UI:"))
		fmt.Println("  kubectl -n argocd port-forward svc/argocd-server 8080:443")
		fmt.Println("  kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath='{.data.password}' | base64 -d")
	} else {
		fmt.Println()
		fmt.Println(dimStyle.Render("Check Flux status:"))
		fmt.Println("  kubectl -n flux-system get all")
		fmt.Println("  kubectl -n flux-system get gitrepositories")
		fmt.Println("  kubectl -n flux-system get kustomizations")
	}

	switch cfg.Secrets.Type {
	case models.SecretsSealedSecrets:
		fmt.Println()
		fmt.Println(dimStyle.Render("Sealed Secrets — backup your key (critical!):"))
		fmt.Println("  kubectl -n kube-system get secret -l sealedsecrets.bitnami.com/sealed-secrets-key -o yaml > sealed-secrets-key-backup.yaml")
		fmt.Println("  # Store this backup securely — losing it means losing all sealed secrets")
		fmt.Println()
		fmt.Println(dimStyle.Render("Seal a secret:"))
		fmt.Printf("  kubeseal --cert %s/bootstrap/sealed-secrets/pub-cert.pem --format yaml < secret.yaml\n", cfg.RepoPath)

	case models.SecretsESO:
		fmt.Println()
		fmt.Println(dimStyle.Render("External Secrets Operator — configure your provider:"))
		fmt.Printf("  1. Edit %s/bootstrap/external-secrets/clustersecretstore-example.yaml\n", cfg.RepoPath)
		fmt.Println("  2. Uncomment and configure your secrets provider (AWS SM, Vault, GCP, Azure)")
		fmt.Println("  3. kubectl apply -f bootstrap/external-secrets/clustersecretstore-example.yaml")
		fmt.Println()
		fmt.Println(dimStyle.Render("Check ESO status:"))
		fmt.Println("  kubectl get deployments -l app.kubernetes.io/name=external-secrets")
		fmt.Println("  kubectl get clustersecretstores")
		fmt.Println("  kubectl get externalsecrets --all-namespaces")

	case models.SecretsSOPS:
		fmt.Println()
		fmt.Println(dimStyle.Render("SOPS — backup your age key (critical!):"))
		fmt.Printf("  cp %s/bootstrap/sops/age.agekey ~/.config/sops/age/keys.txt\n", cfg.RepoPath)
		fmt.Println("  # Store this key securely — losing it means losing all encrypted secrets")
		fmt.Println()
		fmt.Println(warnStyle.Render("  ⚠ Never commit age.agekey to Git"))
		fmt.Println()
		fmt.Println(dimStyle.Render("Encrypt a secret:"))
		fmt.Println("  sops --encrypt --in-place environments/<env>/<app>/secret.yaml")
		fmt.Println()
		fmt.Println(dimStyle.Render("Edit an encrypted secret:"))
		fmt.Println("  sops environments/<env>/<app>/secret.yaml")
	}
}

func printScaffoldOnlySteps(cfg *models.BootstrapConfig) {
	fmt.Println()
	fmt.Println(dimStyle.Render("Next steps:"))
	fmt.Printf("  1. cd %s && git init && git add -A && git commit -m \"feat: initial gitops structure\"\n", cfg.RepoPath)
	fmt.Println("  2. Push to your Git provider")
	fmt.Println("  3. Update apps/_root.yaml with your Git repo URL")
	if cfg.ClusterContext == "" {
		fmt.Println()
		fmt.Println(warnStyle.Render("  No cluster context provided — skipped installation."))
		fmt.Println(dimStyle.Render("  To install later: kubectl apply -k " + controllerBootstrapPath(cfg)))
	}
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
			Type:    models.SecretsType(initFlags.secrets),
			Version: models.DefaultSecretsVersion(models.SecretsType(initFlags.secrets)),
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
