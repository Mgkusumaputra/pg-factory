/*
Copyright © 2026 Mgkusumaputra
*/
package cmd

import (
	"fmt"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"

	"github.com/Mgkusumaputra/pg-factory/pkg/config"
	"github.com/Mgkusumaputra/pg-factory/pkg/docker"
	"github.com/Mgkusumaputra/pg-factory/pkg/state"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all managed Postgres instances",
	RunE: func(cmd *cobra.Command, args []string) error {
		instancesPath, err := config.InstancesPath()
		if err != nil {
			return err
		}
		store := state.New(instancesPath)
		list, err := store.ReadInstances()
		if err != nil {
			return err
		}

		if len(list.Instances) == 0 {
			fmt.Println("No instances found. Run `pg create --name <name>` to get started.")
			return nil
		}

		svc := docker.NewDockerService(10 * time.Second)

		header := lipgloss.NewStyle().Bold(true).Underline(true)
		runningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#4ade80"))
		stoppedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#f87171"))

		fmt.Printf("%-20s %-10s %-8s %-12s %-10s\n",
			header.Render("NAME"),
			header.Render("STATUS"),
			header.Render("PORT"),
			header.Render("DB"),
			header.Render("VERSION"),
		)

		for _, inst := range list.Instances {
			name := inst.Container[4:] // strip "pgf-" prefix
			isRunning, _ := svc.ContainerRunning(inst.Container)

			statusStr := stoppedStyle.Render("stopped")
			if isRunning {
				statusStr = runningStyle.Render("running")
			}

			fmt.Printf("%-20s %-10s %-8d %-12s %-10s\n",
				name, statusStr, inst.Port, inst.Db, inst.Version,
			)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
