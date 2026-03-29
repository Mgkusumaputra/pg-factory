package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/Mgkusumaputra/pg-factory/pkg/config"
)

var rootCmd = &cobra.Command{
	Use:   "pg",
	Short: "Postgres Factory – spin up isolated Postgres instances with Docker",
	Long: `pg-factory manages ephemeral, isolated Postgres instances backed by Docker.

Each instance gets its own container, volume, port, and credentials.
State is stored globally in ~/.pgfactory/ so instances persist across reboots
and are accessible from any directory on your machine.

Commands:
  pg create   Provision a new Postgres instance
  pg up       Start a stopped instance
  pg down     Stop a running instance
  pg prune    Permanently delete an instance and its data
  pg list     List all instances and their status
  pg connect  Open a psql shell (or print the connection string)
  pg status   Show health and details of one instance
  pg rename   Rename an instance
  pg init     (Re)configure global defaults
  pg uninstall Remove everything pg-factory installed`,
}

func Execute() {
	if err := config.EnsureDirs(); err != nil {
		PrintError(err.Error())
		os.Exit(1)
	}

	// Auto-trigger the first-run wizard when config.json is absent.
	// Skip when the user explicitly typed "pg init" — it will run itself.
	if !config.DefaultsExist() {
		args := os.Args[1:]
		isInitCmd := len(args) > 0 && args[0] == "init"
		if !isInitCmd {
			if err := RunInitWizard(); err != nil {
				PrintError(err.Error())
				os.Exit(1)
			}
		}
	}

	rootCmd.SilenceErrors = true
	rootCmd.SilenceUsage = true

	if err := rootCmd.Execute(); err != nil {
		PrintError(err.Error())
		os.Exit(1)
	}
}
