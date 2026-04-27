package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"

	"github.com/Mgkusumaputra/pg-factory/pkg/config"
	"github.com/Mgkusumaputra/pg-factory/pkg/docker"
	"github.com/Mgkusumaputra/pg-factory/pkg/project"
	"github.com/Mgkusumaputra/pg-factory/pkg/state"
)

// column definitions: index, header text, fixed width
var tableCols = []struct {
	header string
	width  int
}{
	{"NAME", 18},
	{"STATUS", 11},
	{"PORT", 7},
	{"DATABASE", 14},
	{"VERSION", 12},
	{"PROJECTS", 22},
}

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all managed Postgres instances",
	Long: `List displays all Postgres instances registered with pg-factory.

Each row shows the instance name, running status, host port, database name,
Postgres version, and the project directories that have linked to it.

Use --project to filter the table to only instances linked to the current
working directory (auto-detected from the directory name).

Examples:
  pg list
  pg ls
  pg list --project`,
	RunE: func(cmd *cobra.Command, args []string) error {
		filterByProject, _ := cmd.Flags().GetBool("project")

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
			PrintInfo("No instances found. Run `pg create` from your project directory to get started.")
			return nil
		}

		// ── Load project store ───────────────────────────────────────────────
		projectsPath, err := config.ProjectsPath()
		if err != nil {
			return err
		}
		ps := project.New(projectsPath)
		pm, _ := ps.Load()

		cwd, _ := os.Getwd()
		currentProject := filepath.Base(cwd)
		currentLinked := map[string]bool{}
		if cwd != "" {
			if linked, err := linkedInstancesForDir(ps, cwd); err == nil {
				for _, instance := range linked {
					currentLinked[instance] = true
				}
			}
		}

		svc := docker.NewDockerService(10 * time.Second)
		runningSet, err := svc.RunningContainerNames()
		if err != nil {
			// non-fatal — fall back to all-stopped display
			runningSet = map[string]bool{}
		}

		// ── Cell builder — lipgloss.Width() pads correctly past ANSI codes ───
		cell := func(s string, w int, style lipgloss.Style) string {
			return style.Width(w).Render(s)
		}

		sep := lipgloss.NewStyle().Foreground(colorBorder).Render(" │ ")

		// ── Header ───────────────────────────────────────────────────────────
		fmt.Println()
		headerParts := make([]string, len(tableCols))
		for i, col := range tableCols {
			headerParts[i] = cell(col.header, col.width, HeaderStyle)
		}
		fmt.Println("  " + strings.Join(headerParts, sep))

		// ── Divider ──────────────────────────────────────────────────────────
		divStyle := lipgloss.NewStyle().Foreground(colorBorder)
		divParts := make([]string, len(tableCols))
		for i, col := range tableCols {
			divParts[i] = divStyle.Render(strings.Repeat("─", col.width))
		}
		fmt.Println("  " + strings.Join(divParts, divStyle.Render("─┼─")))

		// ── Rows ─────────────────────────────────────────────────────────────
		baseStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#e2e8f0"))
		runStyle := lipgloss.NewStyle().Bold(true).Foreground(colorSuccess)
		stopStyle := lipgloss.NewStyle().Foreground(colorError)

		shown := 0
		for _, inst := range list.Instances {
			instName := inst.Container[4:] // strip "pgf-"

			// --project filter
			if filterByProject && !currentLinked[instName] {
				continue
			}

			isRunning := runningSet[inst.Container]

			var statusCell string
			if isRunning {
				statusCell = cell("● running", tableCols[1].width, runStyle)
			} else {
				statusCell = cell("○ stopped", tableCols[1].width, stopStyle)
			}

			// Collect linked project names
			var linked []string
			seenProjects := make(map[string]bool)
			for proj, instances := range pm {
				for _, n := range instances {
					if n == instName {
						display := displayProjectName(proj)
						if !seenProjects[display] {
							seenProjects[display] = true
							linked = append(linked, display)
						}
						break
					}
				}
			}
			projectsVal := "—"
			projectsStyle := DimStyle
			if len(linked) > 0 {
				projectsVal = strings.Join(linked, ", ")
				projectsStyle = lipgloss.NewStyle().Foreground(colorInfo)
			}

			row := []string{
				cell(instName, tableCols[0].width, baseStyle),
				statusCell,
				cell(fmt.Sprintf("%d", inst.Port), tableCols[2].width, DimStyle),
				cell(inst.Db, tableCols[3].width, baseStyle),
				cell(inst.Version, tableCols[4].width, DimStyle),
				cell(projectsVal, tableCols[5].width, projectsStyle),
			}
			fmt.Println("  " + strings.Join(row, sep))
			shown++
		}

		if shown == 0 && filterByProject {
			fmt.Println()
			PrintInfo(fmt.Sprintf("No instances linked to project %q.", currentProject))
		}
		fmt.Println()
		return nil
	},
}

func init() {
	listCmd.Flags().BoolP("project", "p", false, "filter instances linked to the current project directory")
	rootCmd.AddCommand(listCmd)
}
