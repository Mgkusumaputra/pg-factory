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
)

var upCmd = &cobra.Command{
	Use:   "up <name>",
	Short: "Start a stopped Postgres instance",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
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
			return fmt.Errorf("instance %q not found. Run `pg list` to see available instances", name)
		}

		svc := docker.NewDockerService(30 * time.Second)
		running, err := svc.ContainerRunning(containerName)
		if err != nil {
			return err
		}
		if running {
			fmt.Printf("Instance %q is already running.\n", name)
			return nil
		}

		if err := svc.StartContainer(containerName); err != nil {
			return err
		}
		fmt.Printf("✓ Instance %q started.\n", name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(upCmd)
}
