/*
Copyright © 2026 Mgkusumaputra
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/Mgkusumaputra/pg-factory/pkg/config"
)

var rootCmd = &cobra.Command{
	Use:   "pg",
	Short: "Postgres Factory – spin up isolated Postgres instances with Docker",
}

func Execute() {
	if err := config.EnsureDirs(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
