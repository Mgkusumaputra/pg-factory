package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/Mgkusumaputra/pg-factory/pkg/config"
	"github.com/Mgkusumaputra/pg-factory/pkg/docker"
	"github.com/Mgkusumaputra/pg-factory/pkg/state"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove all pg-factory instances, state, and the pg binary itself",
	Long: `Uninstall completely removes pg-factory from this machine:

  • Stops and removes all managed Docker containers and volumes
  • Deletes ~/.pgfactory/ (instances, projects, and config)
  • Removes this binary from disk
  • Cleans up the PATH export line added by the installer (bash/zsh)

This action is IRREVERSIBLE. You must pass --yes to confirm.

Examples:
  pg uninstall --yes`,
	RunE: func(cmd *cobra.Command, args []string) error {
		yes, _ := cmd.Flags().GetBool("yes")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		if !yes && !dryRun {
			return fmt.Errorf("this operation is destructive and cannot be undone — re-run with --yes to confirm, or use --dry-run to preview")
		}

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

		pgfDir, dirErr := config.Dir()
		if dirErr != nil {
			return dirErr
		}
		exePath, _ := os.Executable()

		if dryRun {
			fmt.Println()
			PrintInfo("Dry run — nothing will be removed. The following would be deleted:")
			fmt.Println()
			for _, inst := range list.Instances {
				instName := inst.Container[4:] // strip "pgf-"
				PrintKV("  Container", inst.Container)
				PrintKV("  Volume   ", inst.Volume)
				PrintKV("  Instance ", instName)
				fmt.Println()
			}
			PrintKV("  State dir", pgfDir)
			if exePath != "" {
				PrintKV("  Binary   ", exePath)
			}
			PrintInfo("Re-run with --yes to actually uninstall.")
			fmt.Println()
			return nil
		}

		svc := docker.NewDockerService(30 * time.Second)

		// 2. Stop + remove every managed container and volume
		for _, inst := range list.Instances {
			instName := inst.Container[4:] // strip "pgf-"
			spin := NewSpinner(fmt.Sprintf("Removing instance %q…", instName))

			running, _ := svc.ContainerRunning(inst.Container)
			if running {
				spin.UpdateLabel(fmt.Sprintf("Stopping %q…", inst.Container))
				_ = svc.StopContainer(inst.Container)
			}

			exists, _ := svc.ContainerExists(inst.Container)
			if exists {
				spin.UpdateLabel(fmt.Sprintf("Removing container %q…", inst.Container))
				_ = svc.RemoveContainer(inst.Container)
			}

			spin.UpdateLabel(fmt.Sprintf("Removing volume %q…", inst.Volume))
			_ = svc.RemoveVolume(inst.Volume)

			spin.Stop(fmt.Sprintf("Instance %q removed", instName), true)
		}

		// 3. Wipe ~/.pgfactory/
		spin := NewSpinner(fmt.Sprintf("Removing state directory %s…", pgfDir))
		if err := os.RemoveAll(pgfDir); err != nil {
			spin.Stop("Could not remove state dir: "+err.Error(), false)
		} else {
			spin.Stop("State directory removed", true)
		}

		// 4. Remove this binary
		if exePath != "" {
			spin2 := NewSpinner("Removing binary…")
			if err := os.Remove(exePath); err != nil {
				spin2.Stop("Could not remove binary — delete it manually from your PATH", false)
			} else {
				spin2.Stop("Binary removed", true)
			}
		}

		// 5. Clean up PATH export line added by install.sh / dev-install.sh
		// We attempt all common shell rc files; missing files are silently skipped.
		removePathExport()

		fmt.Println()
		PrintSuccess("pg-factory uninstalled successfully.")
		PrintInfo("If you installed on Windows, remove the Go bin path from your user PATH via System Settings.")
		return nil
	},
}

func init() {
	uninstallCmd.Flags().Bool("yes", false, "confirm destructive uninstall")
	uninstallCmd.Flags().Bool("dry-run", false, "preview what would be removed without deleting anything")
	rootCmd.AddCommand(uninstallCmd)
}
