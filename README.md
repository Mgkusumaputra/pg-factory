# PG Factory

**Spin up isolated, project-linked PostgreSQL instances with Docker — one command per project.**

`pg` is a CLI tool that manages the full lifecycle of local Postgres containers. It uses Docker under the hood, keeps all state globally in `~/.pgfactory/`, and automatically wires your database connection string into `.env.local` — so you're ready to query within seconds of running `pg create`.

---

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/) installed and running
- [Go](https://go.dev/dl/) 1.21+ (to build from source)
- `psql` (optional — only needed for `pg connect` interactive sessions)

---

## Installation

### End Users

One command installs the binary and launches the setup wizard automatically.

**macOS / Linux / WSL:**
```bash
curl -fsSL https://raw.githubusercontent.com/Mgkusumaputra/pg-factory/main/install.sh | bash
```

**Windows (PowerShell):**
```powershell
irm https://raw.githubusercontent.com/Mgkusumaputra/pg-factory/main/install.ps1 | iex
```

The installer will:
1. Verify Go 1.21+ and Docker are available
2. Install the `pg` binary via `go install`
3. Add the Go bin directory to your `PATH` if needed
4. Automatically launch the first-time setup wizard (`pg init`)

**Requirements:** [Go 1.21+](https://go.dev/dl/) · [Docker](https://docs.docker.com/get-docker/) · (optional) `psql` for `pg connect`

---

### Contributors

Clones the repo, builds from source, and wires everything up:

**macOS / Linux / WSL:**
```bash
curl -fsSL https://raw.githubusercontent.com/Mgkusumaputra/pg-factory/main/dev-install.sh | bash
# or with custom paths:
bash dev-install.sh --dir ~/src/pg-factory --bin-dir ~/.local/bin
```

**Windows (PowerShell):**
```powershell
irm https://raw.githubusercontent.com/Mgkusumaputra/pg-factory/main/dev-install.ps1 | iex
# or with custom paths:
.\dev-install.ps1 -Dir "$HOME\src\pg-factory" -BinDir "$HOME\.local\bin"
```

To rebuild after making changes:
```bash
go build -o pg .        # macOS / Linux / WSL
go build -o pg.exe .    # Windows
```

---


## Quick Start

```bash
# First time (auto-triggered after install, or manually):
pg init            # interactive setup: workstation scope, PG version, port

# From inside your project directory:
pg create          # provisions a Postgres container named after the folder
pg up              # start it again after a reboot
pg down            # stop it (data is preserved)
pg status          # check if it's running and see connection details
pg connect         # open a psql shell (auto-resolved from cwd)
pg list            # see all managed instances
pg rename <old> <new>  # rename an instance
pg prune           # permanently delete the instance and its data
```

`pg create` automatically:
1. Pulls `postgres:16-alpine` (or whichever version you set in `pg init`)
2. Allocates a free port starting from your configured base port
3. Waits for Postgres to accept connections before returning
4. Writes `DATABASE_URL` into `.env.local` in your project directory

---

## Commands

### `pg init`

Interactive first-time setup wizard. Triggered automatically after install; re-run at any time to reconfigure.

```
pg init
```

Sets your global defaults stored in `~/.pgfactory/config.json`:

| Setting | Options | Description |
|---------|---------|-------------|
| **Workstation scope** | `cwd` / `path` / `global` | Controls which directories can auto-link instances |
| **Postgres version** | any Docker Hub tag | Default image tag used by `pg create` |
| **Base port** | 1024–65535 | Starting port for new instances (auto-incremented if busy) |

**Workstation modes:**

| Mode | Behaviour |
|------|-----------|
| `cwd` | Every directory you run `pg` from is its own project context |
| `path` | Set a parent root (e.g. `~/projects`); only sub-directories auto-link |
| `global` | No restriction — any directory on the machine can manage instances |

---

### `pg create`

Provisions a new Postgres Docker container and registers it in global state.

```
pg create [flags]

Flags:
  -n, --name string      instance name (default: current directory name)
  -v, --version string   Postgres Docker image tag (default: from pg init)
  -u, --user string      database username (default: "postgres")
  -s, --pass string      database password (default: "postgres")
  -d, --db string        database name (default: same as --name)
  -p, --port int         preferred host port, auto-incremented if busy (default: from pg init)
```


**Examples:**

```bash
pg create                                          # name = current directory
pg create --name myapp
pg create --name myapp --version 15-alpine --port 5433
pg create --name myapp --user admin --pass secret --db mydb
```

After a successful create you'll see the connection details printed to your terminal and a `DATABASE_URL` written to `.env.local`:

```
DATABASE_URL=postgresql://postgres:postgres@localhost:5432/myapp
```

---

### `pg up`

Starts a stopped instance. When called without a name, the instance linked to the current project directory is resolved automatically.

```
pg up [name]
```

**Examples:**

```bash
pg up              # start the instance linked to the current project
pg up myapp        # start a specific instance by name
```

The command blocks until Postgres is accepting connections, so you can immediately run queries after it returns.

---

### `pg down`

Stops a running instance **without removing any data**. The container and volume are preserved. Use `pg prune` to permanently delete everything.

```
pg down [name]
```

**Examples:**

```bash
pg down            # stop the instance linked to the current project
pg down myapp      # stop a specific instance by name
```

---

### `pg list` / `pg ls`

Displays a styled table of all managed instances and their current status.

```
pg list [flags]

Flags:
  -p, --project   show only instances linked to the current project directory
```

**Output columns:**

| Column   | Description                                      |
|----------|--------------------------------------------------|
| NAME     | Instance name (matches the `--name` used at create) |
| STATUS   | `● running` or `○ stopped`                      |
| PORT     | Host port mapped to `5432` inside the container  |
| DATABASE | Database name                                    |
| VERSION  | Postgres image tag                               |
| PROJECTS | Directories linked to this instance              |

**Examples:**

```bash
pg list             # all instances
pg list --project   # only instances for the current directory
```

---

### `pg connect`

Opens an interactive `psql` shell for the named instance. When called without a name, the instance linked to the current project is resolved automatically from the cwd. If `psql` is not found on `PATH`, the connection string is printed instead.

```
pg connect [name] [flags]

Flags:
  -P, --print   print the connection URL instead of launching psql
```

**Examples:**

```bash
pg connect              # open psql for the current project's instance
pg connect myapp        # open psql session for a named instance
pg connect myapp --print  # just print the URL
pg connect myapp -P     # short flag alias
```

---

### `pg prune`

**Irreversibly** stops and removes a Postgres instance: Docker container, Docker volume (all data), project links, and the `DATABASE_URL` entry in `.env.local`.

```
pg prune [name]
```

> ⚠️ **This action cannot be undone.** Use `pg down` if you only want to stop the container and keep the data.

**Examples:**

```bash
pg prune           # prune the instance linked to the current project
pg prune myapp     # prune a specific instance by name
```

---

### `pg uninstall`

Completely removes pg-factory from the machine:

- Stops and removes **all** managed containers and volumes
- Deletes the `~/.pgfactory/` state directory
- Removes the `pg` binary itself
- Cleans up the `PATH` export added by the installer (bash/zsh)

```
pg uninstall [flags]

Flags:
  --yes       confirm destructive uninstall (required to actually delete)
  --dry-run   preview what would be removed without touching anything
```

**Examples:**

```bash
pg uninstall --dry-run   # safe preview
pg uninstall --yes       # actually uninstall
```

The `--yes` flag is required as an explicit confirmation — this cannot be undone.

---

### `pg status`

Shows the health, port, user, database, and version of a single instance. Auto-resolves from the current project directory when no name is given.

```
pg status [name]
```

**Examples:**

```bash
pg status          # status of the current project's instance
pg status myapp    # status of a specific instance
```

---

### `pg rename`

Renames an instance — updates the Docker container name, `instances.json`, and any project links in `projects.json`.

> **Note:** Docker volumes cannot be renamed. The underlying volume keeps its original Docker name, but all data is fully preserved.

```
pg rename <old-name> <new-name>
```

**Examples:**

```bash
pg rename myapp myapp-v2
pg rename old-project new-project
```

---

## How It Works

```
pg create
  │
  ├─ 1. docker run postgres:<version>   ← spins up a named container (pgf-<name>)
  │                                        with a persistent named volume (pgf-vol-<name>)
  │
  ├─ 2. pg_isready health check          ← waits up to 30s for Postgres to accept connections
  │
  ├─ 3. ~/.pgfactory/instances.json      ← records the instance (port, user, db, version, …)
  │
  ├─ 4. ~/.pgfactory/projects.json       ← links the current directory name → instance name
  │
  └─ 5. .env.local in cwd               ← writes DATABASE_URL=postgresql://…
```

### Global State (`~/.pgfactory/`)

All state lives outside your projects, so you never accidentally commit credentials or instance metadata.

| File                | Contents                                          |
|---------------------|---------------------------------------------------|
| `instances.json`    | Array of all created instances with their metadata |
| `projects.json`     | Map of project directory names → instance names   |

### Project Auto-linking

When you run `pg create` from a directory, pg-factory automatically links that directory's basename to the instance. This lets you run `pg up`, `pg down`, `pg status`, `pg connect`, and `pg prune` **without specifying a name** — it resolves the right instance from your current folder.

### `.env.local` Management

`pg create` writes (or updates) `DATABASE_URL` in the project's `.env.local`. `pg prune` removes it automatically. The file is never touched if the write fails, making this a best-effort convenience — it won't break anything if `.env.local` already has a conflicting structure.

---

## Project Structure

```
pg-factory/
├── main.go                  # entry point
├── cmd/
│   ├── root.go              # CLI root command
│   ├── create.go            # pg create
│   ├── up.go                # pg up + cwd project resolution helper
│   ├── down.go              # pg down
│   ├── list.go              # pg list / pg ls
│   ├── connect.go           # pg connect
│   ├── prune.go             # pg prune
│   ├── status.go            # pg status
│   ├── rename.go            # pg rename
│   ├── uninstall.go         # pg uninstall (+ platform-specific _unix/_windows)
│   ├── envfile.go           # .env.local read/write helpers
│   └── ui.go                # lipgloss styles, spinner, print helpers
└── pkg/
    ├── config/              # ~/.pgfactory/ path helpers and global defaults
    ├── docker/              # docker CLI wrappers (run, start, stop, remove, health)
    ├── port/                # free-port finder
    ├── project/             # project ↔ instance linking store
    ├── state/               # instances.json read/write with file locking
    └── types/               # shared data types (Instance, InstanceList)
```

---

## Configuration Reference

All flags have sensible defaults. The most commonly customised ones are:

| Flag        | Default       | Description                                              |
|-------------|---------------|----------------------------------------------------------|
| `--name`    | directory name | Instance name. Also used as the Docker container prefix  |
| `--version` | `16-alpine`   | Any valid `postgres` Docker Hub image tag                |
| `--port`    | `5432`        | Preferred host port. Incremented automatically if in use |
| `--user`    | `postgres`    | Postgres superuser name                                  |
| `--pass`    | `postgres`    | Postgres superuser password                              |
| `--db`      | `--name` value | Database name created at init                           |

---

## License

MIT — see [LICENSE](LICENSE).
