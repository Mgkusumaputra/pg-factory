# pg-factory Audit Fixes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix all 24 issues identified in the CLI audit — bugs, bad patterns, dead code removal, and missing commands.

**Architecture:** Fixes are grouped into independent tasks ordered by dependency: infrastructure fixes first (pkg layer), then cmd layer fixes, then dead code removal, then new commands. Each task compiles cleanly on its own.

**Tech Stack:** Go 1.24, cobra, charm.land/lipgloss/v2, charmbracelet/huh

---

## Task 1: Fix `go.mod` Go version + cross-platform port check

**Files:**
- Modify: `go.mod`
- Modify: `pkg/port/checkPort.go`

### Context

`go.mod` declares `go 1.25.4` which doesn't exist (latest is 1.24.x). Must be corrected.

`checkPort.go` uses `netstat -ano` (Windows-only). On macOS/Linux it silently returns `false`, meaning ports are never detected as in-use and `pg create` can trample live ports. Fix with a pure-Go net.Listen probe (cross-platform, no external binary).

- [ ] **Step 1: Fix go.mod version**

Replace line 3 of `go.mod`:
```
go 1.24.4
```

- [ ] **Step 2: Rewrite checkPort.go with pure-Go port probe**

Replace entire content of `pkg/port/checkPort.go`:
```go
package port

import (
	"fmt"
	"net"
	"time"
)

// checkLocalPort returns true when the given port is already bound on the
// local machine. Uses a pure-Go net.Listen probe — no external binary needed,
// works identically on Windows, macOS, and Linux.
func checkLocalPort(p int) bool {
	addr := fmt.Sprintf(":%d", p)
	ln, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1%s", addr), 200*time.Millisecond)
	if err == nil {
		ln.Close()
		return true // something answered — port is in use
	}
	// Also try binding: if Listen succeeds the port is free
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return true // couldn't bind → in use
	}
	l.Close()
	return false
}
```

- [ ] **Step 3: Verify the project builds**

```
cd c:\Workspace\Code\pg-factory
go build ./...
```

Expected: no errors.

- [ ] **Step 4: Commit**

```
git add go.mod pkg/port/checkPort.go
git commit -m "fix: cross-platform port check (replace Windows-only netstat) + fix go.mod version"
```

---

## Task 2: Fix `pkg/docker/health.go` — pass actual user to `pg_isready`

**Files:**
- Modify: `pkg/docker/health.go`
- Modify: `cmd/create.go` (call site)
- Modify: `cmd/up.go` (call site)

### Context

`WaitUntilReady` hard-codes `-U postgres` in `pg_isready`. Any container created with a custom `--user` will time out. The user must be threaded through from the call sites.

- [ ] **Step 1: Update `WaitUntilReady` signature and `isReady` helper**

Replace entire `pkg/docker/health.go`:
```go
// pkg/docker/health.go
// Readiness polling for Postgres containers.
package docker

import (
	"context"
	"fmt"
	"os/exec"
	"time"
)

// WaitUntilReady polls `docker exec <container> pg_isready -U <user>` every
// second until Postgres accepts connections or the timeout is reached.
func (d *DockerService) WaitUntilReady(containerName, user string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if isReady(containerName, user) {
			return nil
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("timed out waiting for Postgres in %q to become ready", containerName)
}

// isReady runs pg_isready inside the container and returns true on exit code 0.
func isReady(containerName, user string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "exec", containerName,
		"pg_isready", "-U", user)
	return cmd.Run() == nil
}
```

- [ ] **Step 2: Update `cmd/create.go` call site**

In `cmd/create.go`, find the `WaitUntilReady` call (line ~91) and add the `user` argument:
```go
if err := svc.WaitUntilReady(containerName, user, 30*time.Second); err != nil {
```

- [ ] **Step 3: Update `cmd/up.go` call site**

In `cmd/up.go`, the `up` command restarts a stopped container. It doesn't currently have the user handy — read it from state. Replace the `WaitUntilReady` call block (starting around line 76):

First, capture the user from the found instance. Change the state-lookup loop in `upCmd.RunE`:
```go
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
```

Then update the `WaitUntilReady` call:
```go
if err := svc.WaitUntilReady(containerName, foundUser, 30*time.Second); err != nil {
```

- [ ] **Step 4: Build**

```
go build ./...
```

Expected: no errors.

- [ ] **Step 5: Commit**

```
git add pkg/docker/health.go cmd/create.go cmd/up.go
git commit -m "fix: thread actual pg user into WaitUntilReady / pg_isready"
```

---

## Task 3: Fix prune — swallowed error + false "removed" message

**Files:**
- Modify: `cmd/prune.go`
- Modify: `cmd/envfile.go`

### Context

Two bugs in prune:
1. `ContainerRunning` error is silently swallowed (`running, _ := ...`).
2. `removeFromEnvLocal` returns nil when the file doesn't exist, so prune always prints "DATABASE_URL removed" even when there was no `.env.local`.

- [ ] **Step 1: Fix `removeFromEnvLocal` to report whether it actually removed anything**

Replace the function signature and return value in `cmd/envfile.go`. Change `removeFromEnvLocal` to return `(removed bool, err error)`:

```go
// removeFromEnvLocal removes the DATABASE_URL line from <dir>/.env.local.
// Returns (true, nil) when a line was actually removed.
// Returns (false, nil) when the file doesn't exist or the key wasn't present.
// If the file becomes empty after removal it is deleted entirely.
func removeFromEnvLocal(dir string) (removed bool, err error) {
	envPath := filepath.Join(dir, ".env.local")

	data, readErr := os.ReadFile(envPath)
	if readErr != nil {
		if os.IsNotExist(readErr) {
			return false, nil // nothing to do
		}
		return false, fmt.Errorf("could not read .env.local: %w", readErr)
	}

	var kept []string
	found := false
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, envKey+"=") {
			found = true
		} else {
			kept = append(kept, line)
		}
	}

	if !found {
		return false, nil
	}

	// Trim trailing empty lines left behind
	for len(kept) > 0 && strings.TrimSpace(kept[len(kept)-1]) == "" {
		kept = kept[:len(kept)-1]
	}

	// If nothing remains, remove the file
	if len(kept) == 0 {
		return true, os.Remove(envPath)
	}

	content := strings.Join(kept, "\n") + "\n"
	if writeErr := os.WriteFile(envPath, []byte(content), 0644); writeErr != nil {
		return false, fmt.Errorf("could not update .env.local: %w", writeErr)
	}
	return true, nil
}
```

- [ ] **Step 2: Fix `cmd/prune.go` — propagate ContainerRunning error + use new return value**

In the `pruneCmd.RunE` function:

Fix the swallowed error (line 65):
```go
running, err := svc.ContainerRunning(containerName)
if err != nil {
    spin.Stop("Docker check failed", false)
    return fmt.Errorf("docker check failed: %w", err)
}
```

Fix the env cleanup block (lines 103–112):
```go
// Clean up project link and .env.local DATABASE_URL
cwd, cwdErr := os.Getwd()
autoUnlinkProject(name)
if cwdErr == nil {
    if removed, err := removeFromEnvLocal(cwd); err != nil {
        PrintWarn("could not clean .env.local: " + err.Error())
    } else if removed {
        PrintInfo("DATABASE_URL removed from .env.local")
    }
}
```

- [ ] **Step 3: Build**

```
go build ./...
```

Expected: no errors.

- [ ] **Step 4: Commit**

```
git add cmd/prune.go cmd/envfile.go
git commit -m "fix: propagate ContainerRunning error in prune; only print env removal when it actually happened"
```

---

## Task 4: Fix state package — sentinel error + tmp file leak

**Files:**
- Modify: `pkg/state/read.go`
- Modify: `pkg/state/write.go`

### Context

1. `read.go` compares errors by string value (`err.Error() == "state file not found"`) — fragile. Use a sentinel var + `errors.Is`.
2. `write.go` leaks a `state-*.tmp` file when `tmp.Sync()` fails (the `Encode` error path cleans up, but the Sync path doesn't).

- [ ] **Step 1: Add sentinel error and fix read.go**

Replace entire `pkg/state/read.go`:
```go
package state

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/Mgkusumaputra/pg-factory/pkg/types"
)

// ErrNotFound is returned by Read when the state file does not exist yet.
var ErrNotFound = errors.New("state file not found")

func (s *Store) Read(v any) error {
	f, err := os.Open(s.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrNotFound
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
	if errors.Is(err, ErrNotFound) {
		return types.InstanceList{}, nil
	}
	return list, err
}
```

- [ ] **Step 2: Fix tmp file leak in write.go**

In `pkg/state/write.go`, fix the `Sync` error path to also remove the temp file:
```go
if err := tmp.Sync(); err != nil {
    tmp.Close()
    os.Remove(tmp.Name()) // clean up the temp file
    return err
}
```

- [ ] **Step 3: Build**

```
go build ./...
```

Expected: no errors.

- [ ] **Step 4: Commit**

```
git add pkg/state/read.go pkg/state/write.go
git commit -m "fix: use sentinel ErrNotFound in state store; clean up tmp file on Sync failure"
```

---

## Task 5: Fix project store — atomic write to prevent race corruption

**Files:**
- Modify: `pkg/project/project.go`

### Context

`project.Save()` writes directly with `os.WriteFile` — no lock, no atomic rename. Two simultaneous `pg create` calls can corrupt `projects.json`. Apply the same lock-file + temp-rename pattern used in `pkg/state/write.go`.

- [ ] **Step 1: Rewrite Save with atomic write**

Replace the entire `Save` function in `pkg/project/project.go`:
```go
// Save writes the project map to disk atomically (temp file + rename + lock).
func (s *Store) Save(m ProjectMap) error {
	lockPath := s.Path + ".lock"
	lf, err := lockFile(lockPath)
	if err != nil {
		return err
	}
	defer unlockFile(lf)

	dir := filepath.Dir(s.Path)
	tmp, err := os.CreateTemp(dir, "projects-*.tmp")
	if err != nil {
		return err
	}

	enc := json.NewEncoder(tmp)
	enc.SetIndent("", "  ")
	if err := enc.Encode(m); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), s.Path)
}
```

- [ ] **Step 2: Add the lock helpers to project package**

Create a new file `pkg/project/lock.go` with the same lock primitives (can't import from state — avoid circular deps):
```go
package project

import (
	"errors"
	"os"
	"time"
)

func lockFile(lockPath string) (*os.File, error) {
	for range 50 {
		f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0644)
		if err == nil {
			return f, nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return nil, errors.New("failed to acquire lock")
}

func unlockFile(f *os.File) error {
	name := f.Name()
	err := f.Close()
	if err != nil {
		return err
	}
	return os.Remove(name)
}
```

- [ ] **Step 3: Add missing imports to project.go**

`project.go` now needs `path/filepath` and `os`. Add them to the import block:
```go
import (
	"encoding/json"
	"os"
	"path/filepath"
)
```

- [ ] **Step 4: Build**

```
go build ./...
```

Expected: no errors.

- [ ] **Step 5: Commit**

```
git add pkg/project/project.go pkg/project/lock.go
git commit -m "fix: atomic write for projects.json to prevent race corruption"
```

---

## Task 6: Fix `cmd/connect.go` — auto-resolve from cwd + running-state check

**Files:**
- Modify: `cmd/connect.go`

### Context

Two issues:
1. `pg connect` requires an explicit name — unlike `up`/`down`/`prune` which auto-resolve from cwd. Fix by changing to `cobra.RangeArgs(0,1)` and using `resolveInstanceName`.
2. No check that the instance is running before launching `psql`. A stopped instance gives a cryptic "connection refused" with no hint.

- [ ] **Step 1: Rewrite connect.go**

Replace entire `cmd/connect.go`:
```go
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

		connStr := fmt.Sprintf("postgresql://%s:%s@localhost:%d/%s",
			found.User, found.Password, found.Port, found.Db)

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
			return fmt.Errorf("instance %q is not running — start it first with: pg up %s", name, name)
		}

		psqlPath, err := exec.LookPath("psql")
		if err != nil {
			PrintInfo("psql not found on PATH. Connection string:")
			fmt.Println(AccentStyle.Render(connStr))
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

- [ ] **Step 2: Build**

```
go build ./...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```
git add cmd/connect.go
git commit -m "fix: connect auto-resolves from cwd; checks running state before psql"
```

---

## Task 7: Fix `cmd/list.go` — single batched Docker status check

**Files:**
- Modify: `pkg/docker/docker.go`
- Modify: `cmd/list.go`

### Context

`pg list` spawns one `docker ps` subprocess per instance (O(n)). With 10 instances that's 10 Docker calls. Fix: add a `RunningContainerNames() (map[string]bool, error)` method that makes one call and returns a set.

- [ ] **Step 1: Add `RunningContainerNames` to DockerService**

Add to `pkg/docker/docker.go`:
```go
// RunningContainerNames returns a set of container names that are currently running.
// One docker ps call for all containers — O(1) regardless of instance count.
func (d *DockerService) RunningContainerNames() (map[string]bool, error) {
	stdout, _, err := d.RunCommand("ps", "--format", "{{.Names}}")
	if err != nil {
		return nil, err
	}
	running := make(map[string]bool)
	for _, line := range strings.Split(strings.TrimSpace(stdout), "\n") {
		name := strings.TrimSpace(line)
		if name != "" {
			running[name] = true
		}
	}
	return running, nil
}
```

- [ ] **Step 2: Use it in `cmd/list.go`**

Replace the per-instance `ContainerRunning` call:

Remove:
```go
svc := docker.NewDockerService(10 * time.Second)
```

And the per-row call:
```go
isRunning, _ := svc.ContainerRunning(inst.Container)
```

Add before the rows loop:
```go
svc := docker.NewDockerService(10 * time.Second)
runningSet, err := svc.RunningContainerNames()
if err != nil {
    // non-fatal — fall back to all-stopped display
    runningSet = map[string]bool{}
}
```

In the rows loop, change the status check to:
```go
isRunning := runningSet[inst.Container]
```

- [ ] **Step 3: Build**

```
go build ./...
```

Expected: no errors.

- [ ] **Step 4: Commit**

```
git add pkg/docker/docker.go cmd/list.go
git commit -m "perf: single batched docker ps for pg list instead of O(n) individual calls"
```

---

## Task 8: Fix `cmd/init.go` — use `errors.Is` for ErrUserAborted

**Files:**
- Modify: `cmd/init.go`

### Context

`err == huh.ErrUserAborted` should be `errors.Is(err, huh.ErrUserAborted)` per Go sentinel error idiom.

- [ ] **Step 1: Fix the comparison**

In `cmd/init.go`, find the error check after `form.Run()` (line ~130):
```go
if err := form.Run(); err != nil {
    if errors.Is(err, huh.ErrUserAborted) {
        fmt.Println()
        PrintWarn("Setup cancelled. Run \"pg init\" whenever you're ready.")
        fmt.Println()
        return nil
    }
    return err
}
```

- [ ] **Step 2: Add `"errors"` to the import block if not already present**

The import block in `init.go` should include `"errors"`:
```go
import (
    "errors"
    "fmt"
    "os"
    "path/filepath"
    "strconv"

    "github.com/charmbracelet/huh"
    "github.com/spf13/cobra"

    "github.com/Mgkusumaputra/pg-factory/pkg/config"
)
```

- [ ] **Step 3: Build**

```
go build ./...
```

Expected: no errors.

- [ ] **Step 4: Commit**

```
git add cmd/init.go
git commit -m "fix: use errors.Is for huh.ErrUserAborted sentinel check"
```

---

## Task 9: Fix `cmd/create.go` — rename mode constant collision + remove redundant up re-link

**Files:**
- Modify: `pkg/config/defaults.go`
- Modify: `cmd/create.go`
- Modify: `cmd/up.go`
- Modify: `cmd/init.go`

### Context

1. `config.WorkstationPath` is used as both the mode constant (`"path"`) and refers to the `.WorkstationPath` string field. Rename the constant to `WorkstationModeCustomPath` to eliminate the confusion.
2. `cmd/up.go` calls `autoLinkProject(name)` on every `pg up`, re-linking unnecessarily. Remove it.

- [ ] **Step 1: Rename constant in `pkg/config/defaults.go`**

Change:
```go
WorkstationPath WorkstationMode = "path"
```
To:
```go
// WorkstationModeCustomPath uses a fixed parent directory set by the user (e.g. ~/projects).
// Only sub-directories of that path can auto-link instances.
WorkstationModeCustomPath WorkstationMode = "path"
```

Also update the `WorkstationPath` field comment to clarify it's only used when mode == `WorkstationModeCustomPath`.

- [ ] **Step 2: Update the switch in `cmd/create.go`**

In `autoLinkProject`:
```go
switch defs.WorkstationMode {
case config.WorkstationModeCustomPath:
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
```

- [ ] **Step 3: Update `cmd/init.go` wsMode option**

The `huh` form uses `"path"` as a value for the select — this is the serialized form, not the constant. The constant rename doesn't affect persisted JSON, so the `huh.NewOption` value string stays `"path"`. No change needed in init.go options.

However, update the `Long` description in `initCmd` to match:
```
  path    Set a fixed parent path (e.g. ~/projects). Only sub-directories there
          can auto-link instances.
```
(no change needed — already matches the "path" value, not the constant name)

- [ ] **Step 4: Remove the `autoLinkProject` re-link from `cmd/up.go`**

In `upCmd.RunE`, delete line:
```go
autoLinkProject(name)
```

- [ ] **Step 5: Build**

```
go build ./...
```

Expected: no errors.

- [ ] **Step 6: Commit**

```
git add pkg/config/defaults.go cmd/create.go cmd/up.go
git commit -m "refactor: rename WorkstationPath constant to WorkstationModeCustomPath; remove redundant re-link on pg up"
```

---

## Task 10: Add `--dry-run` to `pg uninstall`

**Files:**
- Modify: `cmd/uninstall.go`

### Context

`pg uninstall --yes` is irreversible with no preview. Add `--dry-run` that lists what would be removed.

- [ ] **Step 1: Rewrite uninstall.go with dry-run support**

Replace the `uninstallCmd.RunE` body:

```go
RunE: func(cmd *cobra.Command, args []string) error {
    yes, _ := cmd.Flags().GetBool("yes")
    dryRun, _ := cmd.Flags().GetBool("dry-run")

    if !yes && !dryRun {
        return fmt.Errorf("this operation is destructive and cannot be undone — re-run with --yes to confirm, or use --dry-run to preview")
    }

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

    pgfDir, _ := config.Dir()
    exePath, _ := os.Executable()

    if dryRun {
        fmt.Println()
        PrintInfo("Dry run — nothing will be removed. The following would be deleted:")
        fmt.Println()
        for _, inst := range list.Instances {
            instName := inst.Container[4:] // strip "pgf-"
            PrintKV("  Container", inst.Container)
            PrintKV("  Volume   ", inst.Volume)
            PrintKV("  Instance ", instName)
            fmt.Println()
        }
        PrintKV("  State dir", pgfDir)
        if exePath != "" {
            PrintKV("  Binary   ", exePath)
        }
        PrintInfo("Re-run with --yes to actually uninstall.")
        fmt.Println()
        return nil
    }

    svc := docker.NewDockerService(30 * time.Second)

    // 2. Stop + remove every managed container and volume
    for _, inst := range list.Instances {
        instName := inst.Container[4:] // strip "pgf-"
        spin := NewSpinner(fmt.Sprintf("Removing instance %q…", instName))

        running, _ := svc.ContainerRunning(inst.Container)
        if running {
            spin.UpdateLabel(fmt.Sprintf("Stopping %q…", inst.Container))
            _ = svc.StopContainer(inst.Container)
        }

        exists, _ := svc.ContainerExists(inst.Container)
        if exists {
            spin.UpdateLabel(fmt.Sprintf("Removing container %q…", inst.Container))
            _ = svc.RemoveContainer(inst.Container)
        }

        spin.UpdateLabel(fmt.Sprintf("Removing volume %q…", inst.Volume))
        _ = svc.RemoveVolume(inst.Volume)

        spin.Stop(fmt.Sprintf("Instance %q removed", instName), true)
    }

    // 3. Wipe ~/.pgfactory/
    spin := NewSpinner(fmt.Sprintf("Removing state directory %s…", pgfDir))
    if err := os.RemoveAll(pgfDir); err != nil {
        spin.Stop("Could not remove state dir: "+err.Error(), false)
    } else {
        spin.Stop("State directory removed", true)
    }

    // 4. Remove this binary
    if exePath != "" {
        spin2 := NewSpinner("Removing binary…")
        if err := os.Remove(exePath); err != nil {
            spin2.Stop("Could not remove binary — delete it manually from your PATH", false)
        } else {
            spin2.Stop("Binary removed", true)
        }
    }

    // 5. Clean up PATH export line added by install.sh / dev-install.sh
    removePathExport()

    fmt.Println()
    PrintSuccess("pg-factory uninstalled successfully.")
    PrintInfo("If you installed on Windows, remove the Go bin path from your user PATH via System Settings.")
    return nil
},
```

- [ ] **Step 2: Register the new flag**

In the `init()` function:
```go
func init() {
    uninstallCmd.Flags().Bool("yes", false, "confirm destructive uninstall")
    uninstallCmd.Flags().Bool("dry-run", false, "preview what would be removed without deleting anything")
    rootCmd.AddCommand(uninstallCmd)
}
```

- [ ] **Step 3: Build**

```
go build ./...
```

Expected: no errors.

- [ ] **Step 4: Commit**

```
git add cmd/uninstall.go
git commit -m "feat: add --dry-run to pg uninstall for safe preview before destruction"
```

---

## Task 11: Remove dead code — `pkg/environment`, `utils`, `pkg/state/update.go`

**Files:**
- Delete: `pkg/environment/detectShell.go`
- Delete: `pkg/environment/detectDevNull.go`
- Delete: `utils/pathNormalize.go`
- Delete: `utils/customEnumerator.go`
- Delete: `pkg/state/update.go`

### Context

`pkg/environment` functions are never called. `utils/pathNormalize.go` and `utils/customEnumerator.go` are never imported. `pkg/state/update.go` is unused and silently swallows errors.

Verify no callers exist before deleting:
```
grep -r "detectShell\|detectDevNull\|PathNormalize\|state\.Update" --include="*.go" c:\Workspace\Code\pg-factory
```
Expected: no results (all dead).

- [ ] **Step 1: Delete dead files**

```
cd c:\Workspace\Code\pg-factory
del pkg\environment\detectShell.go
del pkg\environment\detectDevNull.go
del utils\pathNormalize.go
del utils\customEnumerator.go
del pkg\state\update.go
```

If the `pkg/environment` directory becomes empty, also remove it:
```
rmdir pkg\environment
```

If the `utils` directory becomes empty, also remove it:
```
rmdir utils
```

- [ ] **Step 2: Build**

```
go build ./...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```
git add -A
git commit -m "chore: remove dead code (pkg/environment, utils, state.Update)"
```

---

## Task 12: Add `pg status` command

**Files:**
- Create: `cmd/status.go`

### Context

No quick way exists to check whether a single instance is running/stopped without parsing the full `pg list` table. `pg status [name]` (or `pg status` via cwd auto-resolve) fills this gap.

- [ ] **Step 1: Create `cmd/status.go`**

```go
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

		var found *struct {
			Port    int
			User    string
			Db      string
			Version string
		}
		for _, inst := range list.Instances {
			if inst.Container == containerName {
				found = &struct {
					Port    int
					User    string
					Db      string
					Version string
				}{inst.Port, inst.User, inst.Db, inst.Version}
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
		PrintKV("Instance", name)
		PrintKV("Version ", found.Version)
		PrintKV("Port    ", fmt.Sprintf("%d", found.Port))
		PrintKV("User    ", found.User)
		PrintKV("Database", found.Db)
		if running {
			fmt.Println()
			fmt.Println(SuccessStyle.Render("● running"))
			PrintInfo(fmt.Sprintf("Connect: pg connect %s", name))
		} else {
			fmt.Println()
			fmt.Println(ErrorStyle.Render("○ stopped"))
			PrintInfo(fmt.Sprintf("Start:   pg up %s", name))
		}
		fmt.Println()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
```

- [ ] **Step 2: Build**

```
go build ./...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```
git add cmd/status.go
git commit -m "feat: add pg status command for quick single-instance health check"
```

---

## Task 13: Add `pg rename` command

**Files:**
- Create: `cmd/rename.go`

### Context

Instances are permanently named at creation. No way to rename without editing raw JSON. `pg rename <old> <new>` updates `instances.json`, `projects.json`, and renames the Docker container + volume (or at minimum warns that the Docker assets can't be renamed in-place and provides instructions).

Note: Docker doesn't support renaming volumes. The rename command will:
1. Rename the container via `docker rename`
2. Update state in `instances.json` and `projects.json`
3. Warn that the volume name in Docker's namespace remains `pgf-vol-<old>` but is now tracked internally as `pgf-vol-<new>` (Docker volume rename is not supported; the volume continues to work, just with the old Docker name)

- [ ] **Step 1: Add `RenameContainer` to DockerService**

Add to `pkg/docker/docker.go`:
```go
// RenameContainer renames a Docker container. The container can be running or stopped.
func (d *DockerService) RenameContainer(oldName, newName string) error {
	_, stderr, err := d.RunCommand("rename", oldName, newName)
	if err != nil {
		return fmt.Errorf("docker rename failed: %s", stderr)
	}
	return nil
}
```

- [ ] **Step 2: Create `cmd/rename.go`**

```go
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
	"github.com/Mgkusumaputra/pg-factory/pkg/types"
)

var renameCmd = &cobra.Command{
	Use:   "rename <old-name> <new-name>",
	Short: "Rename a Postgres instance",
	Long: `Rename updates the instance name in pg-factory's state and renames the
Docker container. Any project links pointing to the old name are updated
to the new name automatically.

Note: Docker volumes cannot be renamed — the underlying volume will retain
its original Docker name (pgf-vol-<old>) but pg-factory tracks the new name.
The database data is fully preserved.

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

		// Find old instance
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

		// Check new name not already taken
		for _, inst := range list.Instances {
			if inst.Container == newContainer {
				return fmt.Errorf("instance %q already exists", newName)
			}
		}

		svc := docker.NewDockerService(30 * time.Second)

		spin := NewSpinner(fmt.Sprintf("Renaming container %q → %q…", oldContainer, newContainer))
		if err := svc.RenameContainer(oldContainer, newContainer); err != nil {
			spin.Stop("Failed to rename container", false)
			return err
		}
		spin.Stop(fmt.Sprintf("Container renamed to %q", newContainer), true)

		// Update state
		inst := list.Instances[foundIdx]
		inst.Container = newContainer
		// Volume name in Docker stays pgf-vol-<old> — we keep the real Docker name tracked
		// so prune can still find it. We do NOT update inst.Volume.
		list.Instances[foundIdx] = inst
		if err := store.WriteInstances(list); err != nil {
			return fmt.Errorf("state updated in Docker but failed to persist: %w", err)
		}

		// Update project links
		projectsPath, err := config.ProjectsPath()
		if err != nil {
			return err
		}
		ps := project.New(projectsPath)
		pm, err := ps.Load()
		if err == nil {
			changed := false
			for proj, instances := range pm {
				for i, inst := range instances {
					if inst == oldName {
						pm[proj][i] = newName
						changed = true
					}
				}
			}
			if changed {
				_ = ps.Save(pm)
			}
		}

		PrintSuccess(fmt.Sprintf("Instance renamed: %s → %s", oldName, newName))
		PrintInfo("Note: the Docker volume retains its original name in Docker's namespace.")
		PrintInfo("Your data is fully preserved.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(renameCmd)
}
```

- [ ] **Step 3: Build**

```
go build ./...
```

Expected: no errors.

- [ ] **Step 4: Commit**

```
git add cmd/rename.go pkg/docker/docker.go
git commit -m "feat: add pg rename command to rename instances"
```

---

## Final Build Verification

- [ ] **Run full build**

```
cd c:\Workspace\Code\pg-factory
go build ./...
```

Expected: zero errors.

- [ ] **Run vet**

```
go vet ./...
```

Expected: zero warnings.

- [ ] **Check all commands appear in help**

```
.\pg.exe --help
```

Expected: `create`, `up`, `down`, `prune`, `connect`, `list`, `status`, `rename`, `init`, `uninstall` all listed.
