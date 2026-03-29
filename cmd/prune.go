/*
Copyright © 2026 Mgkusumaputra
*/
package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/Mgkusumaputra/pg-factory/pkg/config"
	"github.com/Mgkusumaputra/pg-factory/pkg/docker"
	"github.com/Mgkusumaputra/pg-factory/pkg/state"
	"github.com/Mgkusumaputra/pg-factory/pkg/types"
)

var pruneCmd = &cobra.Command{
	Use:   "prune <name>",
	Short: "Stop and permanently remove a Postgres instance and its data",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
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
			return fmt.Errorf("instance %q not found", name)
		}

		svc := docker.NewDockerService(30 * time.Second)

		// Stop if running
		running, _ := svc.ContainerRunning(containerName)
		if running {
			fmt.Printf("Stopping %q...\n", name)
			if err := svc.StopContainer(containerName); err != nil {
				return err
			}
		}

		// Remove container
		exists, _ := svc.ContainerExists(containerName)
		if exists {
			fmt.Printf("Removing container %q...\n", containerName)
			if err := svc.RemoveContainer(containerName); err != nil {
				return err
			}
		}

		// Remove volume (best-effort)
		fmt.Printf("Removing volume %q...\n", volumeName)
		_ = svc.RemoveVolume(volumeName)

		// Remove from state
		remaining := make([]types.Instance, 0, len(list.Instances))
		for _, inst := range list.Instances {
			if inst.Container != containerName {
				remaining = append(remaining, inst)
			}
		}
		list.Instances = remaining
		if err := store.WriteInstances(list); err != nil {
			return err
		}

		fmt.Printf("✓ Instance %q pruned.\n", name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(pruneCmd)
}
