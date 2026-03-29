package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/Mgkusumaputra/pg-factory/pkg/config"
	"github.com/Mgkusumaputra/pg-factory/pkg/docker"
	"github.com/Mgkusumaputra/pg-factory/pkg/state"
)

var downCmd = &cobra.Command{
	Use:   "down [name]",
	Short: "Stop a running Postgres instance",
	Long: `Stop a running Postgres container without removing its data.

When called without a name argument, pg down resolves the instance from the
current project directory automatically (via ~/.pgfactory/projects.json).

The container and volume are preserved; use pg up to restart later.
Use pg prune to permanently delete the instance and all its data.

Examples:
  pg down           # stop the instance linked to the current project
  pg down myapp     # stop a specific instance by name`,
	Args: cobra.RangeArgs(0, 1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name, err := resolveInstanceName(args)
		if err != nil {
			return err
		}
		containerName := "pgf-" + name

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
		running, err := svc.ContainerRunning(containerName)
		if err != nil {
			return err
		}
		if !running {
			PrintInfo(fmt.Sprintf("Instance %q is already stopped.", name))
			return nil
		}

		spin := NewSpinner(fmt.Sprintf("Stopping instance %q…", name))
		if err := svc.StopContainer(containerName); err != nil {
			spin.Stop("Failed to stop container", false)
			return err
		}
		spin.Stop(fmt.Sprintf("Instance %q stopped", name), true)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(downCmd)
}
