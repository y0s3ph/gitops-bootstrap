package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/y0s3ph/gostrap/internal/validator"
)

var validateCmd = &cobra.Command{
	Use:   "validate [repo-path]",
	Short: "Validate the structure of a GitOps repository",
	Long: `Checks the integrity of a gostrap-managed repository.

Verifies that:
  - .gostrap.yaml exists and is valid
  - Bootstrap directory exists for the configured controller
  - Root application definition exists
  - Each application has base manifests (deployment, service, kustomization or Helm chart)
  - Each application has overlays for every configured environment
  - Each application has a controller definition for every environment
  - All YAML files are syntactically valid

Exit code is 0 if the repo is valid, 1 if there are errors.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runValidate,
}

func init() {
	rootCmd.AddCommand(validateCmd)
}

func runValidate(_ *cobra.Command, args []string) error {
	repoPath := "."
	if len(args) > 0 {
		repoPath = args[0]
	}

	fmt.Printf("Validating %s...\n\n", repoPath)

	result := validator.Validate(repoPath)

	warnings := result.Warnings()
	errors := result.Errors()

	for _, w := range warnings {
		fmt.Printf("  %s %s\n", warnStyle.Render("⚠"), w.String())
	}
	for _, e := range errors {
		fmt.Printf("  %s %s\n", lipglossErrorStyle.Render("✗"), e.String())
	}

	fmt.Println()

	if result.IsValid() {
		if len(warnings) > 0 {
			fmt.Printf("%s (%d warnings)\n", successStyle.Render("✓ Repository is valid"), len(warnings))
		} else {
			fmt.Println(successStyle.Render("✓ Repository is valid — no issues found"))
		}
		return nil
	}

	fmt.Printf("%s: %d errors, %d warnings\n",
		lipglossErrorStyle.Render("✗ Validation failed"),
		len(errors), len(warnings))
	os.Exit(1)
	return nil
}
