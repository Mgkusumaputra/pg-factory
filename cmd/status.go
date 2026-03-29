package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/Mgkusumaputra/pg-factory/pkg/config"
	"github.com/Mgkusumaputra/pg-factory/pkg/docker"
	"github.com/Mgkusumaputra/pg-factory/pkg/state"
)

var statusCmd = &cobra.Command{
	Use:   "status [name]",
	Short: "Show the running status of a Postgres instance",
	Long: `Status prints whether the named Postgres instance is running or stopped,
along with its connection details.

When called without a name argument, pg status resolves the instance from the
current project directory automatically (via ~/.pgfactory/projects.json).

Examples:
  pg status
  pg status myapp`,
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

		type instInfo struct {
			Port      int
			User      string
			Db        string
			Version   string
			CreatedAt string
		}
		var found *instInfo
		for _, inst := range list.Instances {
			if inst.Container == containerName {
				found = &instInfo{inst.Port, inst.User, inst.Db, inst.Version, inst.CreatedAt}
				break
			}
		}
		if found == nil {
			return fmt.Errorf("instance %q not found — run `pg list` to see available instances", name)
		}

		svc := docker.NewDockerService(10 * time.Second)
		running, err := svc.ContainerRunning(containerName)
		if err != nil {
			return fmt.Errorf("docker check failed: %w", err)
		}

		fmt.Println()
		PrintKV("Instance  ", name)
		PrintKV("Version   ", found.Version)
		PrintKV("Port      ", fmt.Sprintf("%d", found.Port))
		PrintKV("User      ", found.User)
		PrintKV("Database  ", found.Db)
		if found.CreatedAt != "" {
			PrintKV("Created   ", found.CreatedAt)
		}
		fmt.Println()
		if running {
			fmt.Println(SuccessStyle.Render("  ● running"))
			fmt.Println()
			PrintInfo(fmt.Sprintf("Connect:  pg connect %s", name))
			PrintInfo(fmt.Sprintf("Stop:     pg down %s", name))
		} else {
			fmt.Println(ErrorStyle.Render("  ○ stopped"))
			fmt.Println()
			PrintInfo(fmt.Sprintf("Start:    pg up %s", name))
			PrintInfo(fmt.Sprintf("Remove:   pg prune %s", name))
		}
		fmt.Println()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
