package cli

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/y0s3ph/gostrap/internal/differ"
)

var (
	diffAddedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))
	diffRemovedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4444"))
	diffHunkStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#00AAFF"))
	diffHeaderStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF"))
)

var diffCmd = &cobra.Command{
	Use:   "diff <source-env> <target-env> [--app <name>]",
	Short: "Show differences between two environments",
	Long: `Compares overlay files and app definitions between two environments.

Shows a unified diff of configuration differences for each application,
helping you understand what changes when promoting from one environment
to another.

Examples:
  gostrap diff dev staging
  gostrap diff dev staging --app my-api
  gostrap diff staging production --repo-path ./gitops-repo`,
	Args: cobra.ExactArgs(2),
	RunE: runDiff,
}

var diffAppFlag string
var diffRepoPath string

func init() {
	diffCmd.Flags().StringVar(&diffAppFlag, "app", "", "Compare only this application")
	diffCmd.Flags().StringVar(&diffRepoPath, "repo-path", ".", "Path to the GitOps repository")
	rootCmd.AddCommand(diffCmd)
}

func runDiff(_ *cobra.Command, args []string) error {
	sourceEnv := args[0]
	targetEnv := args[1]

	results, err := differ.Diff(diffRepoPath, sourceEnv, targetEnv, diffAppFlag)
	if err != nil {
		return err
	}

	anyChanges := false
	for _, r := range results {
		if r.HasChanges() {
			anyChanges = true
			printDiffResult(r)
		}
	}

	if !anyChanges {
		fmt.Printf("\n%s\n", successStyle.Render(
			fmt.Sprintf("✓ No differences between %s and %s", sourceEnv, targetEnv)))
	}

	return nil
}

func printDiffResult(r differ.DiffResult) {
	fmt.Printf("\n%s\n", diffHeaderStyle.Render(
		fmt.Sprintf("═══ %s: %s → %s ═══", r.AppName, r.SourceEnv, r.TargetEnv)))

	for _, f := range r.Files {
		if f.Status == differ.FileIdentical {
			continue
		}

		switch f.Status {
		case differ.FileOnlyInSource:
			fmt.Printf("\n%s %s\n",
				diffRemovedStyle.Render("--- only in "+r.SourceEnv+":"),
				f.RelPath)
		case differ.FileOnlyInTarget:
			fmt.Printf("\n%s %s\n",
				diffAddedStyle.Render("+++ only in "+r.TargetEnv+":"),
				f.RelPath)
		case differ.FileModified:
			fmt.Printf("\n%s\n",
				dimStyle.Render("--- "+r.SourceEnv+"/"+f.RelPath))
			fmt.Printf("%s\n",
				dimStyle.Render("+++ "+r.TargetEnv+"/"+f.RelPath))
		}

		for _, h := range f.Hunks {
			fmt.Println(diffHunkStyle.Render(
				fmt.Sprintf("@@ -%d,%d +%d,%d @@",
					h.SourceStart, h.SourceCount,
					h.TargetStart, h.TargetCount)))

			for _, line := range h.Lines {
				switch line.Kind {
				case differ.DiffContext:
					fmt.Printf(" %s\n", line.Text)
				case differ.DiffRemoved:
					fmt.Println(diffRemovedStyle.Render("-" + line.Text))
				case differ.DiffAdded:
					fmt.Println(diffAddedStyle.Render("+" + line.Text))
				}
			}
		}
	}

	fmt.Println(strings.Repeat("─", 50))
}
