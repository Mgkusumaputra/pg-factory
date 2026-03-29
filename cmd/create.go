package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Mgkusumaputra/pg-factory/pkg/config"
	"github.com/Mgkusumaputra/pg-factory/pkg/docker"
	"github.com/Mgkusumaputra/pg-factory/pkg/port"
	"github.com/Mgkusumaputra/pg-factory/pkg/project"
	"github.com/Mgkusumaputra/pg-factory/pkg/state"
	"github.com/Mgkusumaputra/pg-factory/pkg/types"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new Postgres instance",
	Long: `Create provisions a new Postgres Docker container and registers it
with pg-factory's global state (~/.pgfactory/instances.json).

When --name is omitted, the current directory name is used as the instance name,
so you can simply run "pg create" from your project folder.

A free host port is automatically allocated starting from --port, incrementing
until one is available. The container will also be health-checked via pg_isready
before the command returns, so you know Postgres is really up.

Examples:
  pg create                            # name = current directory
  pg create --name myapp
  pg create --name myapp --version 15-alpine --port 5433
  pg create --name myapp --user admin --pass secret --db mydb`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load stored defaults; fall back gracefully if config.json is missing.
		defs, _ := config.ReadDefaults()

		name, _ := cmd.Flags().GetString("name")
		version, _ := cmd.Flags().GetString("version")
		if !cmd.Flags().Changed("version") && defs.PGVersion != "" {
			version = defs.PGVersion
		}
		user, _ := cmd.Flags().GetString("user")
		pass, _ := cmd.Flags().GetString("pass")
		db, _ := cmd.Flags().GetString("db")
		requestedPort, _ := cmd.Flags().GetInt("port")
		if !cmd.Flags().Changed("port") && defs.BasePort != 0 {
			requestedPort = defs.BasePort
		}

		// If --name is not provided, default to the current directory basename.
		if name == "" {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("could not determine current directory: %w", err)
			}
			name = filepath.Base(cwd)
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

		// ── Step 1: Start container ─────────────────────────────────────────
		spin := NewSpinner(fmt.Sprintf("Creating container %q on port %d…", name, allocatedPort))
		if err := svc.RunPostgres(containerName, volumeName, version, user, pass, db, allocatedPort); err != nil {
			spin.Stop("Failed to create container", false)
			return err
		}

		// ── Step 2: Wait for Postgres to be ready ───────────────────────────
		spin.UpdateLabel("Waiting for Postgres to accept connections…")
		if err := svc.WaitUntilReady(containerName, user, 30*time.Second); err != nil {
			spin.Stop("Postgres did not become ready in time", false)
			return err
		}
		spin.Stop(fmt.Sprintf("Instance %q is ready", name), true)

		// ── Step 3: Persist to state ────────────────────────────────────────
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

		// ── Step 4: Auto-link project + write .env.local ────────────────────
		connStr := fmt.Sprintf("postgresql://%s:%s@localhost:%d/%s", user, pass, allocatedPort, db)
		linkedSlug, linkedCwd := autoLinkProject(name)
		if linkedSlug != "" {
			created, envErr := writeEnvLocal(linkedCwd, connStr)
			if envErr != nil {
				PrintWarn(envErr.Error())
			} else if created {
				PrintInfo(fmt.Sprintf(".env.local created in %s", linkedCwd))
			} else {
				PrintInfo(fmt.Sprintf("DATABASE_URL updated in %s/.env.local", linkedCwd))
			}
		}

		// ── Step 5: Print connection details ────────────────────────────────
		fmt.Println()
		PrintKV("Host    ", fmt.Sprintf("localhost:%d", allocatedPort))
		PrintKV("User    ", user)
		PrintKV("Password", pass)
		PrintKV("Database", db)
		PrintKV("URL     ", connStr)
		return nil
	},
}

func init() {
	createCmd.Flags().StringP("name", "n", "", "name of the instance (defaults to current directory name)")
	createCmd.Flags().StringP("version", "v", "16-alpine", "Postgres Docker image tag")
	createCmd.Flags().StringP("user", "u", "postgres", "database username")
	createCmd.Flags().StringP("pass", "s", "postgres", "database password")
	createCmd.Flags().StringP("db", "d", "", "database name (defaults to --name)")
	createCmd.Flags().IntP("port", "p", 5432, "preferred host port (auto-incremented if in use)")

	rootCmd.AddCommand(createCmd)
}

// autoLinkProject links the cwd to instanceName in the project store,
// respecting the user's configured workstation scope.
// Returns (projectSlug, cwd) on success, or ("", "") on any error or scope violation.
// Errors are intentionally swallowed — project tracking is best-effort.
func autoLinkProject(instanceName string) (slug string, cwd string) {
	defs, _ := config.ReadDefaults()

	var err error
	cwd, err = os.Getwd()
	if err != nil {
		return "", ""
	}

	switch defs.WorkstationMode {
	case config.WorkstationModeCustomPath:
		// Only link when cwd lives inside the configured workstation root.
		if defs.WorkstationPath == "" {
			break
		}
		rel, err := filepath.Rel(defs.WorkstationPath, cwd)
		if err != nil || strings.HasPrefix(rel, "..") {
			return "", "" // outside configured workstation — skip linking
		}
	case config.WorkstationCWD, config.WorkstationGlobal:
		// No restriction — always link.
	}

	slug = filepath.Base(cwd)

	projectsPath, err := config.ProjectsPath()
	if err != nil {
		return "", ""
	}
	ps := project.New(projectsPath)
	if err := ps.Link(slug, instanceName); err != nil {
		return "", ""
	}
	return slug, cwd
}

// autoUnlinkProject is the inverse — called by prune.
func autoUnlinkProject(instanceName string) {
	cwd, err := os.Getwd()
	if err != nil {
		return
	}
	projectSlug := filepath.Base(cwd)

	projectsPath, err := config.ProjectsPath()
	if err != nil {
		return
	}
	ps := project.New(projectsPath)
	_ = ps.Unlink(projectSlug, instanceName)
}
