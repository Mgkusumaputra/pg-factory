package cmd

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

// RunInitWizard presents the charmbracelet/huh onboarding form and persists the
// resulting Defaults to ~/.pgfactory/config.json.
//
// It is called automatically the first time any pg command runs (when config.json
// is absent) and is also wired directly to "pg init" for reconfiguration.
func RunInitWizard() error {
	cwd, _ := os.Getwd()
	cwdLabel := filepath.Base(cwd)
	if cwd == "" || cwdLabel == "." {
		cwdLabel = "current directory"
	}

	// ── Print banner / current-config summary ───────────────────────────────
	fmt.Println()
	if config.DefaultsExist() {
		d, _ := config.ReadDefaults()
		fmt.Println(AccentStyle.Render("  ◆ pg factory — reconfigure"))
		fmt.Println(DimStyle.Render("  Current settings:"))
		PrintKV("  Version    ", d.PGVersion)
		PrintKV("  Base port  ", strconv.Itoa(d.BasePort))
		PrintKV("  Workstation", string(d.WorkstationMode)+ifNotEmpty(" → "+d.WorkstationPath, d.WorkstationPath))
		fmt.Println()
	} else {
		fmt.Println(AccentStyle.Render("  ◆ pg factory — first-time setup"))
		fmt.Println(DimStyle.Render("  Configure your defaults. This only takes a moment."))
		fmt.Println()
	}

	// ── Pre-fill from stored config ──────────────────────────────────────────
	defs := config.FallbackDefaults()
	if config.DefaultsExist() {
		if d, err := config.ReadDefaults(); err == nil {
			defs = d
		}
	}

	var (
		wsMode      = string(defs.WorkstationMode)
		wsPath      = defs.WorkstationPath
		pgVersion   = defs.PGVersion
		basePortStr = strconv.Itoa(defs.BasePort)
	)
	if wsPath == "" {
		wsPath = cwd
	}

	// ── Build huh form ───────────────────────────────────────────────────────
	form := huh.NewForm(
		// Group 1: workstation scope
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Workstation scope").
				Description("Where should pg-factory look for your projects?").
				Options(
					huh.NewOption(
						fmt.Sprintf("Current directory  (%s)", cwdLabel),
						"cwd",
					),
					huh.NewOption(
						"Custom path        (you'll enter it next)",
						"path",
					),
					huh.NewOption(
						"Global             (any directory on this machine)",
						"global",
					),
				).
				Value(&wsMode),
		),

		// Group 2: custom path — only shown when scope == "path"
		huh.NewGroup(
			huh.NewInput().
				Title("Projects root path").
				Description("Absolute path — only sub-directories here will auto-link instances.").
				Placeholder("/home/you/projects").
				Value(&wsPath).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("path cannot be empty")
					}
					if _, err := os.Stat(s); os.IsNotExist(err) {
						return fmt.Errorf("path does not exist: %s", s)
					}
					return nil
				}),
		).WithHideFunc(func() bool { return wsMode != "path" }),

		// Group 3: Postgres defaults
		huh.NewGroup(
			huh.NewInput().
				Title("Default Postgres version").
				Description("Any valid postgres Docker Hub tag  (e.g. 16-alpine, 15, 14-alpine)").
				Placeholder("16-alpine").
				Value(&pgVersion).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("version cannot be empty")
					}
					return nil
				}),

			huh.NewInput().
				Title("Preferred base port").
				Description("pg-factory auto-increments if this port is already in use.").
				Placeholder("5432").
				Value(&basePortStr).
				Validate(func(s string) error {
					p, err := strconv.Atoi(s)
					if err != nil || p < 1024 || p > 65535 {
						return fmt.Errorf("must be a number between 1024 and 65535")
					}
					return nil
				}),
		),
	).WithTheme(huh.ThemeCatppuccin())

	if err := form.Run(); err != nil {
		// User hit Ctrl-C / ESC — treat as intentional cancel, not an error crash.
		if errors.Is(err, huh.ErrUserAborted) {
			fmt.Println()
			PrintWarn("Setup cancelled. Run \"pg init\" whenever you're ready.")
			fmt.Println()
			return nil
		}
		return err
	}

	// ── Persist ──────────────────────────────────────────────────────────────
	basePort, _ := strconv.Atoi(basePortStr)
	finalPath := ""
	if wsMode == "path" {
		finalPath = wsPath
	}

	d := config.Defaults{
		PGVersion:       pgVersion,
		BasePort:        basePort,
		WorkstationMode: config.WorkstationMode(wsMode),
		WorkstationPath: finalPath,
	}
	if err := config.WriteDefaults(d); err != nil {
		return fmt.Errorf("could not save config: %w", err)
	}

	fmt.Println()
	PrintSuccess("Config saved  →  ~/.pgfactory/config.json")
	PrintInfo("Run  pg create  from your project folder to spin up your first instance.")
	fmt.Println()
	return nil
}

// ifNotEmpty returns s when cond is non-empty, otherwise returns "".
func ifNotEmpty(s, cond string) string {
	if cond != "" {
		return s
	}
	return ""
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Configure pg-factory defaults (version, port, workstation scope)",
	Long: `init runs an interactive wizard to set your global defaults.

Stored in ~/.pgfactory/config.json, these values are used as fallbacks for every
pg command. Re-run "pg init" at any time to change them.

Workstation modes:
  cwd     Each directory you run pg from becomes its own project context.
  path    Set a fixed parent path (e.g. ~/projects). Only sub-directories there
          can auto-link instances.
  global  No restriction — any directory on this machine can manage instances.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunInitWizard()
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
