package cli

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/y0s3ph/gostrap/internal/promoter"
)

var promoteCmd = &cobra.Command{
	Use:   "promote <app> --from <env> --to <env>",
	Short: "Promote an application from one environment to another",
	Long: `Copies overlay files from a source environment to a target environment.

Shows a diff preview before applying changes. App definitions (ArgoCD
Application / Flux Kustomization) are not modified since they already
point to the correct environment path.

Examples:
  gostrap promote my-api --from dev --to staging
  gostrap promote my-api --from staging --to production --dry-run
  gostrap promote --from dev --to staging                          # all apps
  gostrap promote my-api --from dev --to staging --yes             # skip confirm`,
	Args: cobra.MaximumNArgs(1),
	RunE: runPromote,
}

var (
	promoteFromFlag     string
	promoteToFlag       string
	promoteDryRunFlag   bool
	promoteYesFlag      bool
	promoteRepoPathFlag string
)

func init() {
	promoteCmd.Flags().StringVar(&promoteFromFlag, "from", "", "Source environment (required)")
	promoteCmd.Flags().StringVar(&promoteToFlag, "to", "", "Target environment (required)")
	promoteCmd.Flags().BoolVar(&promoteDryRunFlag, "dry-run", false, "Show what would be promoted without making changes")
	promoteCmd.Flags().BoolVar(&promoteYesFlag, "yes", false, "Skip confirmation prompt")
	promoteCmd.Flags().StringVar(&promoteRepoPathFlag, "repo-path", ".", "Path to the GitOps repository")
	rootCmd.AddCommand(promoteCmd)
}

func runPromote(_ *cobra.Command, args []string) error {
	appName := ""
	if len(args) > 0 {
		appName = args[0]
	}

	if promoteFromFlag == "" || promoteToFlag == "" {
		return fmt.Errorf("both --from and --to flags are required")
	}

	if promoteFromFlag == promoteToFlag {
		return fmt.Errorf("source and target environments must be different")
	}

	opts := promoter.PromoteOptions{
		RepoPath:  promoteRepoPathFlag,
		AppName:   appName,
		SourceEnv: promoteFromFlag,
		TargetEnv: promoteToFlag,
		DryRun:    promoteDryRunFlag,
	}

	scope := appName
	if scope == "" {
		scope = "all apps"
	}
	fmt.Printf("\n  Promoting %s: %s → %s\n", scope, promoteFromFlag, promoteToFlag)

	diffs, err := promoter.Preview(opts)
	if err != nil {
		return err
	}

	anyChanges := false
	for _, d := range diffs {
		if d.HasChanges() {
			anyChanges = true
			printDiffResult(d)
		}
	}

	if !anyChanges {
		fmt.Printf("\n  %s\n\n", successStyle.Render(
			fmt.Sprintf("✓ %s and %s are already identical — nothing to promote",
				promoteFromFlag, promoteToFlag)))
		return nil
	}

	if promoteDryRunFlag {
		fmt.Printf("\n  %s\n\n", warnStyle.Render("dry-run: no files were modified"))
		return nil
	}

	if !promoteYesFlag {
		var confirmed bool
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title(fmt.Sprintf("Apply promotion %s → %s?", promoteFromFlag, promoteToFlag)).
					Value(&confirmed),
			),
		)
		if err := form.Run(); err != nil {
			return fmt.Errorf("prompt cancelled")
		}
		if !confirmed {
			fmt.Println("\n  Promotion cancelled.")
			return nil
		}
	}

	results, err := promoter.Promote(opts)
	if err != nil {
		return err
	}

	fmt.Println()
	for _, r := range results {
		if len(r.Copied) == 0 && len(r.Skipped) == 0 {
			continue
		}

		fmt.Printf("  %s %s\n", successStyle.Render("✓"), r.AppName)
		for _, f := range r.Copied {
			fmt.Printf("    %s %s\n", diffAddedStyle.Render("→"), f)
		}
		for _, f := range r.Skipped {
			fmt.Printf("    %s %s (unchanged)\n", dimStyle.Render("·"), f)
		}
	}

	fmt.Printf("\n  %s\n", successStyle.Render(
		fmt.Sprintf("✓ Promotion complete: %s → %s", promoteFromFlag, promoteToFlag)))
	fmt.Printf("  %s\n\n", dimStyle.Render(
		"Next: review the changes, adjust env-specific values if needed, then commit and push"))

	return nil
}

