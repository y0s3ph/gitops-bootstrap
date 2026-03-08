package cli

import (
	"github.com/spf13/cobra"
)

var version = "dev"

var rootCmd = &cobra.Command{
	Use:     "gostrap",
	Short:   "From zero to GitOps in one command",
	Long:    "Opinionated CLI to bootstrap a production-ready GitOps workflow on any Kubernetes cluster.",
	Version: version,
}

func Execute() error {
	return rootCmd.Execute()
}
