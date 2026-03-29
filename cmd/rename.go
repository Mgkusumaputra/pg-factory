package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Mgkusumaputra/pg-factory/pkg/config"
	"github.com/Mgkusumaputra/pg-factory/pkg/docker"
	"github.com/Mgkusumaputra/pg-factory/pkg/project"
	"github.com/Mgkusumaputra/pg-factory/pkg/state"
)

var renameCmd = &cobra.Command{
	Use:   "rename <old-name> <new-name>",
	Short: "Rename a Postgres instance",
	Long: `Rename updates the instance name in pg-factory's state and renames the
Docker container. Any project links pointing to the old name are updated
to the new name automatically.

Note: Docker volumes cannot be renamed in-place. The underlying volume will
retain its original Docker name (pgf-vol-<old>), but pg-factory tracks the
new name internally. Your data is fully preserved and accessible.

Examples:
  pg rename myapp myapp-v2
  pg rename old-project new-project`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		oldName := args[0]
		newName := args[1]

		if oldName == newName {
			return fmt.Errorf("old and new names are the same")
		}
		if strings.ContainsAny(newName, " /\\:") {
			return fmt.Errorf("new name must not contain spaces or path separators")
		}

		oldContainer := "pgf-" + oldName
		newContainer := "pgf-" + newName

		instancesPath, err := config.InstancesPath()
		if err != nil {
			return err
		}
		store := state.New(instancesPath)
		list, err := store.ReadInstances()
		if err != nil {
			return err
		}

		// Ensure old instance exists.
		foundIdx := -1
		for i, inst := range list.Instances {
			if inst.Container == oldContainer {
				foundIdx = i
				break
			}
		}
		if foundIdx == -1 {
			return fmt.Errorf("instance %q not found — run `pg list` to see available instances", oldName)
		}

		// Ensure new name is not already taken.
		for _, inst := range list.Instances {
			if inst.Container == newContainer {
				return fmt.Errorf("instance %q already exists — choose a different name", newName)
			}
		}

		svc := docker.NewDockerService(30 * time.Second)

		spin := NewSpinner(fmt.Sprintf("Renaming container %q → %q…", oldContainer, newContainer))
		if err := svc.RenameContainer(oldContainer, newContainer); err != nil {
			spin.Stop("Failed to rename container", false)
			return err
		}
		spin.Stop(fmt.Sprintf("Container renamed to %q", newContainer), true)

		// Update instances.json — only the container name changes; volume keeps
		// its original Docker name (Docker has no rename for volumes).
		list.Instances[foundIdx].Container = newContainer
		if err := store.WriteInstances(list); err != nil {
			return fmt.Errorf("docker container renamed but failed to persist state: %w", err)
		}

		// Update projects.json — any link that pointed to oldName now points to newName.
		projectsPath, err := config.ProjectsPath()
		if err == nil {
			ps := project.New(projectsPath)
			pm, loadErr := ps.Load()
			if loadErr == nil {
				for proj, instances := range pm {
					for i, inst := range instances {
						if inst == oldName {
							pm[proj][i] = newName
						}
					}
				}
				if saveErr := ps.Save(pm); saveErr != nil {
					PrintWarn("project links could not be updated: " + saveErr.Error())
				}
			}
		}

		fmt.Println()
		PrintSuccess(fmt.Sprintf("Instance renamed: %s → %s", oldName, newName))
		PrintInfo("Note: the Docker volume retains its original name in Docker's namespace.")
		PrintInfo("Your data is fully preserved.")
		fmt.Println()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(renameCmd)
}
