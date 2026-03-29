package cmd

import (
	"os"

	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"

	"github.com/Mgkusumaputra/pg-factory/pkg/config"
)

var rootCmd = &cobra.Command{
	Use:   "pg",
	Short: "Postgres Factory – spin up isolated Postgres instances with Docker",
	Long: lipgloss.NewStyle().
		Foreground(lipgloss.Color("#818cf8")).
		Bold(true).
		Render(`
  ██████╗  ██████╗
  ██╔══██╗██╔════╝
  ██████╔╝██║  ███╗
  ██╔═══╝ ██║   ██║
  ██║     ╚██████╔╝
  ╚═╝      ╚═════╝   factory`) + "\n\n" +
		lipgloss.NewStyle().Foreground(lipgloss.Color("#94a3b8")).Render(
			"  Spin up isolated Postgres instances with Docker.\n"+
				"  State is stored globally in ~/.pgfactory/\n",
		),
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
