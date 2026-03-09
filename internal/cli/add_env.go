package cli

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/y0s3ph/gostrap/internal/config"
	"github.com/y0s3ph/gostrap/internal/models"
	"github.com/y0s3ph/gostrap/internal/scaffolder"
)

var addEnvFlags struct {
	autoSync bool
	prune    bool
	repoPath string
}

var addEnvCmd = &cobra.Command{
	Use:   "add-env [name]",
	Short: "Add a new environment to an existing GitOps repository",
	Long: `Scaffolds a new environment in an existing gostrap-managed repository.

For every application already in the repo, generates the environment overlay
(Kustomize overlay or Helm values file) and the controller-specific application
definition (ArgoCD Application or Flux Kustomization/HelmRelease).

The repository must have been created with "gostrap init" (a .gostrap.yaml file
must exist in the repo root).`,
	Args: cobra.MaximumNArgs(1),
	RunE: runAddEnv,
}

func init() {
	f := addEnvCmd.Flags()
	f.BoolVar(&addEnvFlags.autoSync, "auto-sync", true, "Enable automatic sync for this environment")
	f.BoolVar(&addEnvFlags.prune, "prune", false, "Enable automatic pruning of removed resources")
	f.StringVar(&addEnvFlags.repoPath, "repo-path", ".", "Path to the GitOps repository root")

	rootCmd.AddCommand(addEnvCmd)
}

func runAddEnv(cmd *cobra.Command, args []string) error {
	repoPath := addEnvFlags.repoPath

	cfg, err := config.Load(repoPath)
	if err != nil {
		return err
	}

	var name string
	if len(args) > 0 {
		name = args[0]
	}

	autoSync := addEnvFlags.autoSync
	prune := addEnvFlags.prune

	if name == "" {
		prompted, err := promptAddEnv()
		if err != nil {
			return err
		}
		name = prompted.name
		autoSync = prompted.autoSync
		prune = prompted.prune
	}

	if !validAppName.MatchString(name) {
		return fmt.Errorf("invalid environment name %q: must be lowercase alphanumeric with hyphens, e.g. qa", name)
	}

	for _, env := range cfg.Environments {
		if env.Name == name {
			return fmt.Errorf("environment %q already exists in .gostrap.yaml", name)
		}
	}

	if cmd.Flags().Changed("auto-sync") {
		autoSync = addEnvFlags.autoSync
	}
	if cmd.Flags().Changed("prune") {
		prune = addEnvFlags.prune
	}

	env := models.EnvironmentConfig{
		Name:     name,
		AutoSync: autoSync,
		Prune:    prune,
	}

	fmt.Println()
	fmt.Printf("Adding environment %s to %s...\n", successStyle.Render(name), repoPath)
	fmt.Println()

	s := scaffolder.New(cfg)

	apps, err := s.DiscoverApps()
	if err != nil {
		return fmt.Errorf("discovering apps: %w", err)
	}

	if err := s.ScaffoldEnv(env); err != nil {
		return fmt.Errorf("scaffolding environment: %w", err)
	}

	result := s.Result()

	if len(result.Created) > 0 {
		fmt.Println(successStyle.Render("✓ Environment scaffolded"))
		for _, f := range result.Created {
			fmt.Printf("  %s %s\n", successStyle.Render("+"), f)
		}
		if len(apps) > 0 {
			fmt.Printf("\n  Apps configured: %s\n", dimStyle.Render(fmt.Sprintf("%v", apps)))
		}
	}
	if len(result.Skipped) > 0 {
		fmt.Println()
		fmt.Println(dimStyle.Render("Skipped (already exist):"))
		for _, f := range result.Skipped {
			fmt.Printf("  %s %s\n", dimStyle.Render("·"), f)
		}
	}

	cfg.Environments = append(cfg.Environments, env)
	if err := config.Save(repoPath, cfg); err != nil {
		return fmt.Errorf("updating config: %w", err)
	}

	if len(result.Created) > 0 {
		fmt.Println()
		fmt.Println(dimStyle.Render("Next steps:"))
		fmt.Printf("  1. Review and customize environments/%s/ overlays\n", name)
		fmt.Println("  2. git add -A && git commit -m \"feat: add " + name + " environment\"")
		fmt.Println("  3. Push to trigger GitOps sync")
	} else {
		fmt.Println()
		fmt.Println(dimStyle.Render("Nothing to do — environment already exists."))
	}

	return nil
}

type addEnvInput struct {
	name     string
	autoSync bool
	prune    bool
}

func promptAddEnv() (*addEnvInput, error) {
	var name string
	var autoSync bool
	var prune bool

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Environment name").
				Placeholder("qa").
				Value(&name).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("name is required")
					}
					if !validAppName.MatchString(s) {
						return fmt.Errorf("must be lowercase alphanumeric with hyphens (e.g. qa)")
					}
					return nil
				}),

			huh.NewConfirm().
				Title("Enable automatic sync?").
				Description("Automatically apply changes when pushed to Git").
				Affirmative("Yes").
				Negative("No").
				Value(&autoSync),

			huh.NewConfirm().
				Title("Enable automatic pruning?").
				Description("Remove resources that are no longer in Git").
				Affirmative("Yes").
				Negative("No").
				Value(&prune),
		),
	).WithTheme(huh.ThemeCatppuccin())

	if err := form.Run(); err != nil {
		return nil, fmt.Errorf("interactive prompt: %w", err)
	}

	return &addEnvInput{name: name, autoSync: autoSync, prune: prune}, nil
}
