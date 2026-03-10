package cli

import (
	"runtime/debug"

	"github.com/spf13/cobra"
)

var version = buildVersion()

var rootCmd = &cobra.Command{
	Use:     "gostrap",
	Short:   "From zero to GitOps in one command",
	Long:    "Opinionated CLI to bootstrap a production-ready GitOps workflow on any Kubernetes cluster.",
	Version: version,
}

func Execute() error {
	return rootCmd.Execute()
}

func buildVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "dev"
	}

	v := info.Main.Version
	if v == "" || v == "(devel)" {
		return "dev"
	}

	return v
}
