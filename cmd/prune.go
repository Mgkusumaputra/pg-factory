package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/Mgkusumaputra/pg-factory/pkg/config"
	"github.com/Mgkusumaputra/pg-factory/pkg/docker"
	"github.com/Mgkusumaputra/pg-factory/pkg/state"
	"github.com/Mgkusumaputra/pg-factory/pkg/types"
)

var pruneCmd = &cobra.Command{
	Use:   "prune [name]",
	Short: "Stop and permanently remove a Postgres instance and its data",
	Long: `Prune stops a running instance, removes its Docker container, and deletes
the associated Docker volume — permanently destroying all data.

When called without a name argument, pg prune resolves the instance from the
current project directory automatically (via ~/.pgfactory/projects.json).

This action is irreversible. Use pg down to only stop the instance while
keeping the data intact.

Examples:
  pg prune          # prune the instance linked to the current project
  pg prune myapp    # prune a specific instance by name`,
	Args: cobra.RangeArgs(0, 1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name, err := resolveInstanceName(args)
		if err != nil {
			return err
		}
		containerName := "pgf-" + name
		volumeName := "pgf-vol-" + name

		instancesPath, err := config.InstancesPath()
		if err != nil {
			return err
		}
		store := state.New(instancesPath)
		list, err := store.ReadInstances()
		if err != nil {
			return err
		}

		found := false
		for _, inst := range list.Instances {
			if inst.Container == containerName {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("instance %q not found — run `pg list` to see available instances", name)
		}

		svc := docker.NewDockerService(30 * time.Second)
		spin := NewSpinner(fmt.Sprintf("Pruning instance %q…", name))

		// ── 1. Stop if running ─────────────────────────────────────────────────────────────────────
		running, err := svc.ContainerRunning(containerName)
		if err != nil {
			spin.Stop("Docker check failed", false)
			return fmt.Errorf("docker check failed: %w", err)
		}
		if running {
			spin.UpdateLabel(fmt.Sprintf("Stopping container %q…", containerName))
			if err := svc.StopContainer(containerName); err != nil {
				spin.Stop("Failed to stop container", false)
				return err
			}
		}

		// ── 2. Remove container ──────────────────────────────────────────────
		exists, _ := svc.ContainerExists(containerName)
		if exists {
			spin.UpdateLabel(fmt.Sprintf("Removing container %q…", containerName))
			if err := svc.RemoveContainer(containerName); err != nil {
				spin.Stop("Failed to remove container", false)
				return err
			}
		}

		// ── 3. Remove volume (best-effort) ───────────────────────────────────
		spin.UpdateLabel(fmt.Sprintf("Removing volume %q…", volumeName))
		_ = svc.RemoveVolume(volumeName)

		// ── 4. Remove from state ─────────────────────────────────────────────
		remaining := make([]types.Instance, 0, len(list.Instances))
		for _, inst := range list.Instances {
			if inst.Container != containerName {
				remaining = append(remaining, inst)
			}
		}
		list.Instances = remaining
		if err := store.WriteInstances(list); err != nil {
			spin.Stop("Failed to update state", false)
			return err
		}

		spin.Stop(fmt.Sprintf("Instance %q pruned", name), true)

		// Clean up project link and .env.local DATABASE_URL
		cwd, cwdErr := os.Getwd()
		autoUnlinkProject(name)
		if cwdErr == nil {
			if removed, envErr := removeFromEnvLocal(cwd); envErr != nil {
				PrintWarn("could not clean .env.local: " + envErr.Error())
			} else if removed {
				PrintInfo("DATABASE_URL removed from .env.local")
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(pruneCmd)
}
