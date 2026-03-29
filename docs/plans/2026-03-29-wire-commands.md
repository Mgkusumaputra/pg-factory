# pg-factory CLI – Wire Commands Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Wire all `pg-factory` CLI commands (`create`, `up`, `down`, `list`, `prune`, `connect`, `uninstall`) to the existing infrastructure packages so the tool is fully functional as a cross-platform Postgres instance manager.

**Architecture:** Commands live in `cmd/`, each delegating to `pkg/` packages for Docker operations, global state management (`~/.pgfactory/instances.json`), port allocation, and shell-environment detection. The `state` package is the single source of truth; Docker is always reconciled against it. Output is rendered via `lipgloss`/`lipgloss/list` for a polished TUI experience.

**Tech Stack:** Go 1.25 · Cobra · charm.land/lipgloss/v2 · Docker CLI (exec subprocess) · `encoding/json` · `os/exec`

---

## File Map

| File | Action | Responsibility |
|---|---|---|
| `cmd/root.go` | **Modify** | Move shared types to `pkg/types`; wire global state path init; remove stub flags |
| `pkg/types/types.go` | **Create** | Shared `Instance`, `InstanceList`, `Project`, `ProjectList` structs |
| `pkg/config/config.go` | **Create** | Resolve `~/.pgfactory/` paths; `EnsureDirs()` on startup |
| `cmd/create.go` | **Modify** | Full create logic: validate flags → allocate port → docker run → state write |
| `cmd/up.go` | **Modify** | `docker start <container>` for a stopped instance |
| `cmd/down.go` | **Create** | `docker stop <container>` for a running instance |
| `cmd/list.go` | **Create** | Read state + reconcile with `docker ps` → pretty table |
| `cmd/prune.go` | **Create** | Stop + remove container, volume, and state entry |
| `cmd/connect.go` | **Create** | Run `psql` (or print connection string) for an instance |
| `cmd/uninstall.go` | **Create** | Stop all instances, remove all containers/volumes, wipe `~/.pgfactory/`, remove binary from PATH |
| `pkg/docker/docker.go` | **Modify** | Add `ContainerExists`, `ContainerRunning`, `StartContainer`, `StopContainer`, `RemoveContainer`, `RemoveVolume`, `RunPostgres` helpers |
| `pkg/state/read.go` | **Modify** | Add `ReadInstances() (InstanceList, error)` typed helper |
| `pkg/state/write.go` | **Modify** | Add `WriteInstances(InstanceList) error` typed helper |
| `pkg/port/findPort.go` | **Modify** | Export `FindFreePort(startPort int) int` |
| `utils/customEnumerator.go` | **Modify** | Implement lipgloss custom enumerator for list rendering |

---

## Open Questions

> [!IMPORTANT]
> **Port flag default:** Current stub code defaults the port to `8080`. This plan uses `5432` (the standard Postgres port). Confirm this is acceptable or specify a different default.

> [!IMPORTANT]
> **`connect` command strategy:** Two options:
> 1. `psql <connection-string>` on the host (requires psql installed)
> 2. `docker exec -it <container> psql -U <user> <db>` (no host psql needed)
> The plan uses option 1 with a graceful fallback to printing the connection string if psql is not found. Let me know if you prefer option 2.

> [!IMPORTANT]
> **`uninstall` binary removal:** The binary path depends on how `pg` was installed (e.g., `go install`, copied to `/usr/local/bin`, or placed somewhere on `%PATH%`). The plan uses `os.Executable()` to find the running binary and removes it. Confirm this is the right approach, or specify a fixed install path.

> [!NOTE]
> **`customEnumerator.go`** is currently empty. If you want a custom lipgloss list enumerator style, that can be added as a follow-up task.

---

## Task 1: Extract Shared Types to `pkg/types`

**Files:**
- Create: `pkg/types/types.go`
- Modify: `cmd/root.go`

- [ ] **Step 1: Create the types package**

```go
// pkg/types/types.go
package types

type Instance struct {
	Container string `json:"container"`
	Volume    string `json:"volume"`
	Port      int    `json:"port"`
	User      string `json:"user"`
	Password  string `json:"password"`
	Db        string `json:"db"`
	Version   string `json:"version"`
	CreatedAt string `json:"created_at"`
}

type InstanceList struct {
	Instances []Instance `json:"instances"`
}
```

- [ ] **Step 2: Update `cmd/root.go` to remove duplicate type definitions and stub flag**

```go
/*
Copyright © 2026 Mgkusumaputra
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/Mgkusumaputra/pg-factory/pkg/config"
)

var rootCmd = &cobra.Command{
	Use:   "pg",
	Short: "Postgres Factory – spin up isolated Postgres instances with Docker",
}

func Execute() {
	if err := config.EnsureDirs(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
```

- [ ] **Step 3: Verify the project still compiles**

```
cd C:\Workspace\Code\pg-factory
go build ./...
```

Expected: no output (success).

- [ ] **Step 4: Commit**

```
git add pkg/types/types.go cmd/root.go
git commit -m "refactor: extract shared types to pkg/types"
```

---

## Task 2: Global Config / State Path Resolution

**Files:**
- Create: `pkg/config/config.go`

- [ ] **Step 1: Create `pkg/config/config.go`**

```go
// pkg/config/config.go
package config

import (
	"os"
	"path/filepath"
)

const dirName = ".pgfactory"

// Dir returns the absolute path to ~/.pgfactory.
func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, dirName), nil
}

// InstancesPath returns the path to ~/.pgfactory/instances.json.
func InstancesPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "instances.json"), nil
}

// EnsureDirs creates ~/.pgfactory if it does not exist.
func EnsureDirs() error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	return os.MkdirAll(dir, 0755)
}
```

- [ ] **Step 2: Compile**

```
go build ./...
```

- [ ] **Step 3: Commit**

```
git add pkg/config/config.go cmd/root.go
git commit -m "feat: add global config path resolution and EnsureDirs"
```

---

## Task 3: Typed State Helpers (`ReadInstances` / `WriteInstances`)

**Files:**
- Modify: `pkg/state/read.go`
- Modify: `pkg/state/write.go`

- [ ] **Step 1: Add `ReadInstances` to `pkg/state/read.go`**

```go
package state

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/Mgkusumaputra/pg-factory/pkg/types"
)

func (s *Store) Read(v any) error {
	f, err := os.Open(s.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.New("state file not found")
		}
		return err
	}
	defer f.Close()
	return json.NewDecoder(f).Decode(v)
}

// ReadInstances reads the instance list from the store.
// Returns an empty list (not an error) when the file doesn't exist yet.
func (s *Store) ReadInstances() (types.InstanceList, error) {
	var list types.InstanceList
	err := s.Read(&list)
	if err != nil && err.Error() == "state file not found" {
		return types.InstanceList{}, nil
	}
	return list, err
}
```

- [ ] **Step 2: Add `WriteInstances` to `pkg/state/write.go`**

Add below the existing `Write` function:

```go
// WriteInstances writes the instance list to the store atomically.
func (s *Store) WriteInstances(list types.InstanceList) error {
	return s.Write(list)
}
```

Add `"github.com/Mgkusumaputra/pg-factory/pkg/types"` to the import block.

- [ ] **Step 3: Compile**

```
go build ./...
```

- [ ] **Step 4: Commit**

```
git add pkg/state/read.go pkg/state/write.go
git commit -m "feat: add typed ReadInstances/WriteInstances state helpers"
```

---

## Task 4: Docker Helper Methods

**Files:**
- Modify: `pkg/docker/docker.go`

- [ ] **Step 1: Append helper methods after `InspectContainer`**

Add `"strings"` to the import block, then append:

```go
// ContainerExists returns true if a container with the given name exists (running or stopped).
func (d *DockerService) ContainerExists(name string) (bool, error) {
	stdout, _, err := d.RunCommand("ps", "-a", "--filter", "name=^"+name+"$", "--format", "{{.Names}}")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(stdout) == name, nil
}

// ContainerRunning returns true if the container is currently running.
func (d *DockerService) ContainerRunning(name string) (bool, error) {
	stdout, _, err := d.RunCommand("ps", "--filter", "name=^"+name+"$", "--format", "{{.Names}}")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(stdout) == name, nil
}

// StartContainer starts a stopped container by name.
func (d *DockerService) StartContainer(name string) error {
	_, stderr, err := d.RunCommand("start", name)
	if err != nil {
		return fmt.Errorf("docker start failed: %s", stderr)
	}
	return nil
}

// StopContainer stops a running container by name.
func (d *DockerService) StopContainer(name string) error {
	_, stderr, err := d.RunCommand("stop", name)
	if err != nil {
		return fmt.Errorf("docker stop failed: %s", stderr)
	}
	return nil
}

// RemoveContainer removes a container by name (must be stopped first).
func (d *DockerService) RemoveContainer(name string) error {
	_, stderr, err := d.RunCommand("rm", name)
	if err != nil {
		return fmt.Errorf("docker rm failed: %s", stderr)
	}
	return nil
}

// RemoveVolume removes a Docker volume by name.
func (d *DockerService) RemoveVolume(name string) error {
	_, stderr, err := d.RunCommand("volume", "rm", name)
	if err != nil {
		return fmt.Errorf("docker volume rm failed: %s", stderr)
	}
	return nil
}

// RunPostgres creates and starts a new Postgres container.
func (d *DockerService) RunPostgres(containerName, volumeName, version, user, password, db string, port int) error {
	args := []string{
		"run", "-d",
		"--name", containerName,
		"-e", fmt.Sprintf("POSTGRES_USER=%s", user),
		"-e", fmt.Sprintf("POSTGRES_PASSWORD=%s", password),
		"-e", fmt.Sprintf("POSTGRES_DB=%s", db),
		"-p", fmt.Sprintf("%d:5432", port),
		"-v", fmt.Sprintf("%s:/var/lib/postgresql/data", volumeName),
		fmt.Sprintf("postgres:%s", version),
	}
	_, stderr, err := d.RunCommand(args...)
	if err != nil {
		return fmt.Errorf("docker run failed: %s", stderr)
	}
	return nil
}
```

- [ ] **Step 2: Compile**

```
go build ./...
```

- [ ] **Step 3: Commit**

```
git add pkg/docker/docker.go
git commit -m "feat: add ContainerExists, RunPostgres, Stop/Start/Remove docker helpers"
```

---

## Task 5: Export `FindFreePort`

**Files:**
- Modify: `pkg/port/findPort.go`

- [ ] **Step 1: Rename `findFreePort` → `FindFreePort`**

```go
package port

// FindFreePort returns the first port >= startPort that is not in use
// locally (via netstat) or by any Docker container.
func FindFreePort(startPort int) int {
	port := startPort
	for {
		inUse := false
		if checkLocalPort(port) {
			inUse = true
		}
		if checkDockerPort(port) {
			inUse = true
		}
		if !inUse {
			return port
		}
		port++
	}
}
```

- [ ] **Step 2: Compile**

```
go build ./...
```

- [ ] **Step 3: Commit**

```
git add pkg/port/findPort.go
git commit -m "refactor: export FindFreePort"
```

---

## Task 6: Wire `create` Command

**Files:**
- Modify: `cmd/create.go`

- [ ] **Step 1: Replace `cmd/create.go` with full implementation**

```go
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
```

- [ ] **Step 2: Compile and smoke test**

```
go build ./...
go run . create --name mydb
```

Expected: docker container `pgf-mydb` starts; `~/.pgfactory/instances.json` created with one entry.

- [ ] **Step 3: Commit**

```
git add cmd/create.go
git commit -m "feat: wire create command to docker + state"
```

---

## Task 7: Wire `up` Command

**Files:**
- Modify: `cmd/up.go`

- [ ] **Step 1: Replace stub `cmd/up.go`**

```go
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
```

- [ ] **Step 2: Compile and smoke test**

```
go build ./...
go run . up mydb
```

Expected: `✓ Instance "mydb" started.` (or "already running").

- [ ] **Step 3: Commit**

```
git add cmd/up.go
git commit -m "feat: wire up command to start a stopped instance"
```

---

## Task 8: Add `down` Command

**Files:**
- Create: `cmd/down.go`

- [ ] **Step 1: Create `cmd/down.go`**

```go
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

var downCmd = &cobra.Command{
	Use:   "down <name>",
	Short: "Stop a running Postgres instance",
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
			return fmt.Errorf("instance %q not found", name)
		}

		svc := docker.NewDockerService(30 * time.Second)
		running, err := svc.ContainerRunning(containerName)
		if err != nil {
			return err
		}
		if !running {
			fmt.Printf("Instance %q is already stopped.\n", name)
			return nil
		}

		if err := svc.StopContainer(containerName); err != nil {
			return err
		}
		fmt.Printf("✓ Instance %q stopped.\n", name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(downCmd)
}
```

- [ ] **Step 2: Compile and smoke test**

```
go build ./...
go run . down mydb
```

Expected: `✓ Instance "mydb" stopped.`

- [ ] **Step 3: Commit**

```
git add cmd/down.go
git commit -m "feat: add down command to stop a running instance"
```

---

## Task 9: Add `list` Command

**Files:**
- Create: `cmd/list.go`

- [ ] **Step 1: Create `cmd/list.go`**

```go
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
	"github.com/Mgkusumaputra/pg-factory/pkg/state"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all managed Postgres instances",
	RunE: func(cmd *cobra.Command, args []string) error {
		instancesPath, err := config.InstancesPath()
		if err != nil {
			return err
		}
		store := state.New(instancesPath)
		list, err := store.ReadInstances()
		if err != nil {
			return err
		}

		if len(list.Instances) == 0 {
			fmt.Println("No instances found. Run `pg create --name <name>` to get started.")
			return nil
		}

		svc := docker.NewDockerService(10 * time.Second)

		header := lipgloss.NewStyle().Bold(true).Underline(true)
		runningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#4ade80"))
		stoppedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#f87171"))

		fmt.Printf("%-20s %-10s %-8s %-12s %-10s\n",
			header.Render("NAME"),
			header.Render("STATUS"),
			header.Render("PORT"),
			header.Render("DB"),
			header.Render("VERSION"),
		)

		for _, inst := range list.Instances {
			name := inst.Container[4:] // strip "pgf-" prefix
			isRunning, _ := svc.ContainerRunning(inst.Container)

			statusStr := stoppedStyle.Render("stopped")
			if isRunning {
				statusStr = runningStyle.Render("running")
			}

			fmt.Printf("%-20s %-10s %-8d %-12s %-10s\n",
				name, statusStr, inst.Port, inst.Db, inst.Version,
			)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
```

- [ ] **Step 2: Compile and smoke test**

```
go build ./...
go run . list
```

Expected: a formatted table showing all instances with their status.

- [ ] **Step 3: Commit**

```
git add cmd/list.go
git commit -m "feat: add list command with live Docker status"
```

---

## Task 10: Add `prune` Command

**Files:**
- Create: `cmd/prune.go`

- [ ] **Step 1: Create `cmd/prune.go`**

```go
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
```

- [ ] **Step 2: Compile and smoke test**

```
go build ./...
go run . prune mydb
go run . list
```

Expected: `mydb` no longer appears in `list`.

- [ ] **Step 3: Commit**

```
git add cmd/prune.go
git commit -m "feat: add prune command to destroy instance, volume, and state"
```

---

## Task 11: Add `connect` Command

**Files:**
- Create: `cmd/connect.go`

- [ ] **Step 1: Create `cmd/connect.go`**

```go
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
```

- [ ] **Step 2: Compile and smoke test**

```
go build ./...
go run . create --name mydb
go run . connect mydb --print
```

Expected: `postgresql://postgres:postgres@localhost:5432/mydb`

- [ ] **Step 3: Commit**

```
git add cmd/connect.go
git commit -m "feat: add connect command with psql launch or --print flag"
```

---

## Task 12: Add `uninstall` Command

**Files:**
- Create: `cmd/uninstall.go`

The `uninstall` command performs a full teardown:
1. Stops and removes every managed container and its volume
2. Wipes `~/.pgfactory/` (the entire state directory)
3. Removes the `pg` binary itself from disk (using `os.Executable()`)

It requires an explicit `--yes` confirmation flag to prevent accidents.

- [ ] **Step 1: Create `cmd/uninstall.go`**

```go
/*
Copyright © 2026 Mgkusumaputra
*/
package cmd

import (
	"fmt"
	"os"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"

	"github.com/Mgkusumaputra/pg-factory/pkg/config"
	"github.com/Mgkusumaputra/pg-factory/pkg/docker"
	"github.com/Mgkusumaputra/pg-factory/pkg/state"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove all pg-factory instances, state, and the pg binary itself",
	Long: `Uninstall completely removes pg-factory from this machine:
  - Stops and removes all managed Docker containers and volumes
  - Deletes the ~/.pgfactory/ state directory
  - Removes this binary from disk

Requires --yes to confirm.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		yes, _ := cmd.Flags().GetBool("yes")
		if !yes {
			return fmt.Errorf("this is destructive and cannot be undone. Re-run with --yes to confirm")
		}

		warn := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#f87171"))
		ok   := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#4ade80"))

		// 1. Read current instances from state
		instancesPath, err := config.InstancesPath()
		if err != nil {
			return err
		}
		store := state.New(instancesPath)
		list, err := store.ReadInstances()
		if err != nil {
			return err
		}

		svc := docker.NewDockerService(30 * time.Second)

		// 2. Stop + remove every managed container and volume
		for _, inst := range list.Instances {
			name := inst.Container[4:] // strip "pgf-"

			running, _ := svc.ContainerRunning(inst.Container)
			if running {
				fmt.Printf("  Stopping   %s...\n", name)
				_ = svc.StopContainer(inst.Container)
			}

			exists, _ := svc.ContainerExists(inst.Container)
			if exists {
				fmt.Printf("  Removing   container %s...\n", inst.Container)
				_ = svc.RemoveContainer(inst.Container)
			}

			fmt.Printf("  Removing   volume %s...\n", inst.Volume)
			_ = svc.RemoveVolume(inst.Volume)
		}

		// 3. Wipe ~/.pgfactory/
		pgfDir, err := config.Dir()
		if err != nil {
			return err
		}
		fmt.Printf("  Removing   %s...\n", pgfDir)
		if err := os.RemoveAll(pgfDir); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", warn.Render(fmt.Sprintf("  Warning: could not remove state dir: %v", err)))
		}

		// 4. Remove this binary
		exePath, err := os.Executable()
		if err == nil {
			fmt.Printf("  Removing   binary %s...\n", exePath)
			if err := os.Remove(exePath); err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", warn.Render(fmt.Sprintf("  Warning: could not remove binary: %v", err)))
				fmt.Println("  You may need to manually delete the binary from your PATH.")
			}
		}

		fmt.Println(ok.Render("\n✓ pg-factory uninstalled successfully."))
		return nil
	},
}

func init() {
	uninstallCmd.Flags().Bool("yes", false, "confirm destructive uninstall")
	rootCmd.AddCommand(uninstallCmd)
}
```

- [ ] **Step 2: Compile**

```
go build ./...
```

- [ ] **Step 3: Smoke test (dry-run, without --yes)**

```
go run . uninstall
```

Expected error: `this is destructive and cannot be undone. Re-run with --yes to confirm`

- [ ] **Step 4: Commit**

```
git add cmd/uninstall.go
git commit -m "feat: add uninstall command to remove all instances, state, and binary"
```

---

## Task 13: Final Integration Smoke Test

- [ ] **Step 1: Run the full lifecycle**

```
go run . create --name smoketest --port 15432
go run . list
go run . down smoketest
go run . list
go run . up smoketest
go run . connect smoketest --print
go run . prune smoketest
go run . list
```

Expected output sequence:
1. `✓ Instance "smoketest" created`
2. Table shows `smoketest` as `running` on port 15432
3. `✓ Instance "smoketest" stopped`
4. Table shows `smoketest` as `stopped`
5. `✓ Instance "smoketest" started`
6. `postgresql://postgres:postgres@localhost:15432/smoketest`
7. `✓ Instance "smoketest" pruned`
8. `No instances found.`

- [ ] **Step 2: Test uninstall guard**

```
go run . uninstall
```

Expected: error message requiring `--yes`.

- [ ] **Step 3: ⚠️ MANUAL — Build release binary and test full uninstall (delegated to user)**

> **This step is for the user to run manually.** It destroys Docker containers and removes the binary from disk.

```
go build -o pg-test.exe .
./pg-test.exe create --name finaltest
./pg-test.exe uninstall --yes
```

Expected:
- Container `pgf-finaltest` stopped and removed
- Volume `pgf-vol-finaltest` removed
- `~/.pgfactory/` directory deleted
- `pg-test.exe` binary deleted from disk

- [ ] **Step 4: Final commit**

```
git add -A
git commit -m "chore: verify full lifecycle + uninstall smoke test, build release binary"
```
