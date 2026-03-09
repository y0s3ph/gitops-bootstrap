package cli

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/y0s3ph/gostrap/internal/config"
	"github.com/y0s3ph/gostrap/internal/scaffolder"
)

var addAppFlags struct {
	port     int
	repoPath string
}

var addAppCmd = &cobra.Command{
	Use:   "add-app [name]",
	Short: "Add a new application to an existing GitOps repository",
	Long: `Scaffolds a new application in an existing gostrap-managed repository.

Generates the environment structure (base manifests + per-environment overlays)
and the controller-specific application definitions (ArgoCD Application or Flux
Kustomization/HelmRelease) for every configured environment.

The repository must have been created with "gostrap init" (a .gostrap.yaml file
must exist in the repo root).`,
	Args: cobra.MaximumNArgs(1),
	RunE: runAddApp,
}

func init() {
	f := addAppCmd.Flags()
	f.IntVar(&addAppFlags.port, "port", 0, "Container port for the application (default: 8080)")
	f.StringVar(&addAppFlags.repoPath, "repo-path", ".", "Path to the GitOps repository root")

	rootCmd.AddCommand(addAppCmd)
}

var validAppName = regexp.MustCompile(`^[a-z][a-z0-9-]*[a-z0-9]$`)

func runAddApp(_ *cobra.Command, args []string) error {
	repoPath := addAppFlags.repoPath

	cfg, err := config.Load(repoPath)
	if err != nil {
		return err
	}

	var name string
	if len(args) > 0 {
		name = args[0]
	}

	port := addAppFlags.port

	if name == "" || port == 0 {
		prompted, err := promptAddApp(name, port)
		if err != nil {
			return err
		}
		name = prompted.name
		port = prompted.port
	}

	if port == 0 {
		port = 8080
	}

	if !validAppName.MatchString(name) {
		return fmt.Errorf("invalid app name %q: must be lowercase alphanumeric with hyphens, e.g. my-api", name)
	}

	fmt.Println()
	fmt.Printf("Adding application %s (port %d) to %s...\n", successStyle.Render(name), port, repoPath)
	fmt.Println()

	s := scaffolder.New(cfg)
	if err := s.ScaffoldApp(name, port); err != nil {
		return fmt.Errorf("scaffolding app: %w", err)
	}

	result := s.Result()

	if len(result.Created) > 0 {
		fmt.Println(successStyle.Render("✓ Application scaffolded"))
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

	if len(result.Created) > 0 {
		fmt.Println()
		fmt.Println(dimStyle.Render("Next steps:"))
		fmt.Printf("  1. Edit environments/base/%s/ with your actual manifests\n", name)
		fmt.Println("  2. git add -A && git commit -m \"feat: add " + name + "\"")
		fmt.Println("  3. Push to trigger GitOps sync")
	} else {
		fmt.Println()
		fmt.Println(dimStyle.Render("Nothing to do — application already exists."))
	}

	return nil
}

type addAppInput struct {
	name string
	port int
}

func promptAddApp(existingName string, existingPort int) (*addAppInput, error) {
	name := existingName
	portStr := ""
	if existingPort > 0 {
		portStr = strconv.Itoa(existingPort)
	}

	var fields []huh.Field

	if name == "" {
		fields = append(fields, huh.NewInput().
			Title("Application name").
			Placeholder("my-api").
			Value(&name).
			Validate(func(s string) error {
				if s == "" {
					return fmt.Errorf("name is required")
				}
				if !validAppName.MatchString(s) {
					return fmt.Errorf("must be lowercase alphanumeric with hyphens (e.g. my-api)")
				}
				return nil
			}))
	}

	if existingPort == 0 {
		fields = append(fields, huh.NewInput().
			Title("Container port").
			Placeholder("8080").
			Value(&portStr).
			Validate(func(s string) error {
				if s == "" {
					return nil
				}
				p, err := strconv.Atoi(s)
				if err != nil || p < 1 || p > 65535 {
					return fmt.Errorf("must be a valid port number (1-65535)")
				}
				return nil
			}))
	}

	if len(fields) == 0 {
		return &addAppInput{name: name, port: existingPort}, nil
	}

	form := huh.NewForm(
		huh.NewGroup(fields...),
	).WithTheme(huh.ThemeCatppuccin())

	if err := form.Run(); err != nil {
		return nil, fmt.Errorf("interactive prompt: %w", err)
	}

	port := existingPort
	if portStr != "" {
		p, err := strconv.Atoi(portStr)
		if err == nil {
			port = p
		}
	}
	if port == 0 {
		port = 8080
	}

	return &addAppInput{name: name, port: port}, nil
}
