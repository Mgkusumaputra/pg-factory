package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/Mgkusumaputra/pg-factory/pkg/config"
	"github.com/Mgkusumaputra/pg-factory/pkg/docker"
	"github.com/Mgkusumaputra/pg-factory/pkg/project"
	"github.com/Mgkusumaputra/pg-factory/pkg/state"
)

var upCmd = &cobra.Command{
	Use:   "up [name]",
	Short: "Start a stopped Postgres instance",
	Long: `Start a Postgres instance that was previously stopped with pg down.

When called without a name argument, pg up resolves the instance from the
current project directory automatically (via ~/.pgfactory/projects.json).

The command waits for Postgres to be accepting connections before returning,
so you can immediately run queries after it completes.

Examples:
  pg up             # start the instance linked to the current project
  pg up myapp       # start a specific instance by name`,
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

		var foundUser string
		found := false
		for _, inst := range list.Instances {
			if inst.Container == containerName {
				found = true
				foundUser = inst.User
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
		if running {
			PrintInfo(fmt.Sprintf("Instance %q is already running.", name))
			return nil
		}

		spin := NewSpinner(fmt.Sprintf("Starting instance %q…", name))
		if err := svc.StartContainer(containerName); err != nil {
			spin.Stop("Failed to start container", false)
			return err
		}

		spin.UpdateLabel("Waiting for Postgres to accept connections…")
		if err := svc.WaitUntilReady(containerName, foundUser, 30*time.Second); err != nil {
			spin.Stop("Postgres did not become ready in time", false)
			return err
		}
		spin.Stop(fmt.Sprintf("Instance %q started", name), true)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(upCmd)
}

// resolveInstanceName returns the instance name from args or, when args is
// empty, from the current project's linked instances in the project store.
func resolveInstanceName(args []string) (string, error) {
	if len(args) == 1 {
		return args[0], nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("could not determine current directory: %w", err)
	}
	projectSlug := filepath.Base(cwd)

	projectsPath, err := config.ProjectsPath()
	if err != nil {
		return "", err
	}
	ps := project.New(projectsPath)
	instances, err := ps.InstancesFor(projectSlug)
	if err != nil {
		return "", err
	}

	switch len(instances) {
	case 0:
		return "", fmt.Errorf(
			"no instance linked to project %q — run `pg create` to create one, or specify a name explicitly",
			projectSlug,
		)
	case 1:
		return instances[0], nil
	default:
		return "", fmt.Errorf(
			"multiple instances linked to project %q: %v — specify which one to use",
			projectSlug, instances,
		)
	}
}
