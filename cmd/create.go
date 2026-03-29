/*
Copyright © 2026 Mgkusumaputra
*/
package cmd

import (
	"fmt"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"

	"github.com/Mgkusumaputra/pg-factory/pkg/config"
	"github.com/Mgkusumaputra/pg-factory/pkg/docker"
	"github.com/Mgkusumaputra/pg-factory/pkg/port"
	"github.com/Mgkusumaputra/pg-factory/pkg/state"
	"github.com/Mgkusumaputra/pg-factory/pkg/types"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new Postgres instance",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		version, _ := cmd.Flags().GetString("version")
		user, _ := cmd.Flags().GetString("user")
		pass, _ := cmd.Flags().GetString("pass")
		db, _ := cmd.Flags().GetString("db")
		requestedPort, _ := cmd.Flags().GetInt("port")

		if name == "" {
			return fmt.Errorf("--name is required")
		}
		if db == "" {
			db = name
		}

		containerName := "pgf-" + name
		volumeName := "pgf-vol-" + name

		svc := docker.NewDockerService(30 * time.Second)

		exists, err := svc.ContainerExists(containerName)
		if err != nil {
			return fmt.Errorf("docker check failed: %w", err)
		}
		if exists {
			return fmt.Errorf("instance %q already exists", name)
		}

		allocatedPort := port.FindFreePort(requestedPort)

		fmt.Printf("Creating instance %q on port %d...\n", name, allocatedPort)
		if err := svc.RunPostgres(containerName, volumeName, version, user, pass, db, allocatedPort); err != nil {
			return err
		}

		instancesPath, err := config.InstancesPath()
		if err != nil {
			return err
		}
		store := state.New(instancesPath)
		list, err := store.ReadInstances()
		if err != nil {
			return err
		}
		list.Instances = append(list.Instances, types.Instance{
			Container: containerName,
			Volume:    volumeName,
			Port:      allocatedPort,
			User:      user,
			Password:  pass,
			Db:        db,
			Version:   version,
			CreatedAt: time.Now().Format(time.RFC3339),
		})
		if err := store.WriteInstances(list); err != nil {
			return err
		}

		style := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#4ade80"))
		fmt.Println(style.Render(fmt.Sprintf("✓ Instance %q created", name)))
		fmt.Printf("  Host:     localhost:%d\n", allocatedPort)
		fmt.Printf("  User:     %s\n", user)
		fmt.Printf("  Password: %s\n", pass)
		fmt.Printf("  Database: %s\n", db)
		return nil
	},
}

func init() {
	createCmd.Flags().StringP("name", "n", "", "name of the instance (required)")
	createCmd.Flags().StringP("version", "v", "16-alpine", "Postgres Docker image tag")
	createCmd.Flags().StringP("user", "u", "postgres", "database username")
	createCmd.Flags().StringP("pass", "s", "postgres", "database password")
	createCmd.Flags().StringP("db", "d", "", "database name (defaults to --name)")
	createCmd.Flags().IntP("port", "p", 5432, "preferred host port (auto-incremented if in use)")

	rootCmd.AddCommand(createCmd)
}
