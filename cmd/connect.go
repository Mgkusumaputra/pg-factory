package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/spf13/cobra"

	"github.com/Mgkusumaputra/pg-factory/pkg/config"
	"github.com/Mgkusumaputra/pg-factory/pkg/docker"
	"github.com/Mgkusumaputra/pg-factory/pkg/state"
)

var connectCmd = &cobra.Command{
	Use:   "connect [name]",
	Short: "Open a psql shell or print the connection string for an instance",
	Long: `Connect opens an interactive psql session for the named instance.

When called without a name argument, pg connect resolves the instance from the
current project directory automatically (via ~/.pgfactory/projects.json).

If psql is not found on your PATH, the connection string is printed instead.
Use --print to always print the URL without launching psql.

Examples:
  pg connect
  pg connect myapp
  pg connect myapp --print
  pg connect myapp -P   # short flag for --print`,
	Args: cobra.RangeArgs(0, 1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name, err := resolveInstanceName(args)
		if err != nil {
			return err
		}
		containerName := "pgf-" + name
		printOnly, _ := cmd.Flags().GetBool("print")

		instancesPath, err := config.InstancesPath()
		if err != nil {
			return err
		}
		store := state.New(instancesPath)
		list, err := store.ReadInstances()
		if err != nil {
			return err
		}

		type connInfo struct {
			Port     int
			User     string
			Password string
			Db       string
		}
		var found *connInfo
		for _, inst := range list.Instances {
			if inst.Container == containerName {
				found = &connInfo{inst.Port, inst.User, inst.Password, inst.Db}
				break
			}
		}
		if found == nil {
			return fmt.Errorf("instance %q not found — run `pg list` to see available instances", name)
		}

		connStr := buildPostgresURL(found.User, found.Password, found.Port, found.Db)

		if printOnly {
			fmt.Println(connStr)
			return nil
		}

		// Check the container is actually running before attempting psql.
		svc := docker.NewDockerService(10 * time.Second)
		running, err := svc.ContainerRunning(containerName)
		if err != nil {
			return fmt.Errorf("docker check failed: %w", err)
		}
		if !running {
			return fmt.Errorf("instance %q is not running — start it with: pg up %s", name, name)
		}

		psqlPath, err := exec.LookPath("psql")
		if err != nil {
			PrintInfo("psql not found on PATH. Connection string:")
			fmt.Println(AccentStyle.Render(connStr))
			return nil
		}

		psqlCmd := exec.Command(psqlPath, "-d", connStr)
		psqlCmd.Stdin = os.Stdin
		psqlCmd.Stdout = os.Stdout
		psqlCmd.Stderr = os.Stderr
		return psqlCmd.Run()
	},
}

func init() {
	connectCmd.Flags().BoolP("print", "P", false, "print the connection string instead of launching psql")
	rootCmd.AddCommand(connectCmd)
}
