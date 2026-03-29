/*
Copyright © 2026 Mgkusumaputra
*/
package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/Mgkusumaputra/pg-factory/pkg/config"
	"github.com/Mgkusumaputra/pg-factory/pkg/state"
)

var connectCmd = &cobra.Command{
	Use:   "connect <name>",
	Short: "Open a psql shell or print the connection string for an instance",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
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
			return fmt.Errorf("instance %q not found", name)
		}

		connStr := fmt.Sprintf("postgresql://%s:%s@localhost:%d/%s",
			found.User, found.Password, found.Port, found.Db)

		if printOnly {
			fmt.Println(connStr)
			return nil
		}

		psqlPath, err := exec.LookPath("psql")
		if err != nil {
			fmt.Printf("psql not found on PATH. Connection string:\n%s\n", connStr)
			return nil
		}

		psqlCmd := exec.Command(psqlPath, connStr)
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
