/*
Copyright © 2026 Mgkusumaputra
*/
package cmd

import (
	"fmt"
	"os"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"

	"github.com/Mgkusumaputra/pg-factory/pkg/config"
	"github.com/Mgkusumaputra/pg-factory/pkg/docker"
	"github.com/Mgkusumaputra/pg-factory/pkg/state"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove all pg-factory instances, state, and the pg binary itself",
	Long: `Uninstall completely removes pg-factory from this machine:
  - Stops and removes all managed Docker containers and volumes
  - Deletes the ~/.pgfactory/ state directory
  - Removes this binary from disk

Requires --yes to confirm.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		yes, _ := cmd.Flags().GetBool("yes")
		if !yes {
			return fmt.Errorf("this is destructive and cannot be undone. Re-run with --yes to confirm")
		}

		warn := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#f87171"))
		ok := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#4ade80"))

		// 1. Read current instances from state
		instancesPath, err := config.InstancesPath()
		if err != nil {
			return err
		}
		store := state.New(instancesPath)
		list, err := store.ReadInstances()
		if err != nil {
			return err
		}

		svc := docker.NewDockerService(30 * time.Second)

		// 2. Stop + remove every managed container and volume
		for _, inst := range list.Instances {
			name := inst.Container[4:] // strip "pgf-"

			running, _ := svc.ContainerRunning(inst.Container)
			if running {
				fmt.Printf("  Stopping   %s...\n", name)
				_ = svc.StopContainer(inst.Container)
			}

			exists, _ := svc.ContainerExists(inst.Container)
			if exists {
				fmt.Printf("  Removing   container %s...\n", inst.Container)
				_ = svc.RemoveContainer(inst.Container)
			}

			fmt.Printf("  Removing   volume %s...\n", inst.Volume)
			_ = svc.RemoveVolume(inst.Volume)
		}

		// 3. Wipe ~/.pgfactory/
		pgfDir, err := config.Dir()
		if err != nil {
			return err
		}
		fmt.Printf("  Removing   %s...\n", pgfDir)
		if err := os.RemoveAll(pgfDir); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", warn.Render(fmt.Sprintf("  Warning: could not remove state dir: %v", err)))
		}

		// 4. Remove this binary
		exePath, err := os.Executable()
		if err == nil {
			fmt.Printf("  Removing   binary %s...\n", exePath)
			if err := os.Remove(exePath); err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", warn.Render(fmt.Sprintf("  Warning: could not remove binary: %v", err)))
				fmt.Println("  You may need to manually delete the binary from your PATH.")
			}
		}

		fmt.Println(ok.Render("\n✓ pg-factory uninstalled successfully."))
		return nil
	},
}

func init() {
	uninstallCmd.Flags().Bool("yes", false, "confirm destructive uninstall")
	rootCmd.AddCommand(uninstallCmd)
}
