package main

import (
	"os"

	"github.com/y0s3ph/gostrap/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
