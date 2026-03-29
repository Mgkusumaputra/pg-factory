package cmd

import (
	"fmt"
	"os"
	"sync"
	"time"

	"charm.land/lipgloss/v2"
)

// ── Colour palette ───────────────────────────────────────────────────────────

var (
	colorSuccess = lipgloss.Color("#4ade80")
	colorError   = lipgloss.Color("#f87171")
	colorWarn    = lipgloss.Color("#facc15")
	colorInfo    = lipgloss.Color("#67e8f9")
	colorDim     = lipgloss.Color("#6b7280")
	colorBorder  = lipgloss.Color("#374151")
	colorAccent  = lipgloss.Color("#818cf8")
)

// ── Shared styles ────────────────────────────────────────────────────────────

var (
	SuccessStyle = lipgloss.NewStyle().Bold(true).Foreground(colorSuccess)
	ErrorStyle   = lipgloss.NewStyle().Bold(true).Foreground(colorError)
	WarnStyle    = lipgloss.NewStyle().Bold(true).Foreground(colorWarn)
	InfoStyle    = lipgloss.NewStyle().Foreground(colorInfo)
	HeaderStyle  = lipgloss.NewStyle().Bold(true).Underline(true).Foreground(lipgloss.Color("#e2e8f0"))
	DimStyle     = lipgloss.NewStyle().Foreground(colorDim)
	AccentStyle  = lipgloss.NewStyle().Bold(true).Foreground(colorAccent)

	labelStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#cbd5e1"))
	valueStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#f1f5f9"))
)

// PrintKV prints a key/value pair in a styled format.
func PrintKV(key, value string) {
	fmt.Printf("  %s  %s\n", labelStyle.Render(key+":"), valueStyle.Render(value))
}

// ── Error panel ──────────────────────────────────────────────────────────────

var errorPanel = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(colorError).
	Foreground(colorError).
	Bold(true).
	Padding(0, 1).
	Width(50)

var warnPanel = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(colorWarn).
	Foreground(colorWarn).
	Padding(0, 1).
	Width(50)

// PrintError renders a red bordered error box to stderr.
func PrintError(msg string) {
	title := ErrorStyle.Render("✗ Error")
	body := fmt.Sprintf("%s\n%s", title, msg)
	fmt.Fprintln(os.Stderr, errorPanel.Render(body))
}

// PrintSuccess prints a green success line.
func PrintSuccess(msg string) {
	fmt.Println(SuccessStyle.Render("✓ " + msg))
}

// PrintInfo prints a dim info line.
func PrintInfo(msg string) {
	fmt.Println(InfoStyle.Render("  " + msg))
}

// PrintWarn prints a yellow warning panel.
func PrintWarn(msg string) {
	fmt.Println(warnPanel.Render(WarnStyle.Render("⚠ Warning\n") + msg))
}

// ── Raw-ANSI braille spinner ─────────────────────────────────────────────────

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Spinner is a terminal spinner using raw ANSI escape codes. It runs in a
// background goroutine and is safe to update the label from any goroutine.
type Spinner struct {
	mu    sync.Mutex
	label string
	done  chan struct{}
	wg    sync.WaitGroup
}

// NewSpinner starts a spinner with the given initial label and returns it.
func NewSpinner(label string) *Spinner {
	s := &Spinner{
		label: label,
		done:  make(chan struct{}),
	}
	s.wg.Add(1)
	go s.run()
	return s
}

func (s *Spinner) run() {
	defer s.wg.Done()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	i := 0
	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			s.mu.Lock()
			lbl := s.label
			s.mu.Unlock()
			frame := InfoStyle.Render(spinnerFrames[i%len(spinnerFrames)])
			fmt.Printf("\r%s %s   ", frame, DimStyle.Render(lbl))
			i++
		}
	}
}

// UpdateLabel changes the spinner label while it's running.
func (s *Spinner) UpdateLabel(label string) {
	s.mu.Lock()
	s.label = label
	s.mu.Unlock()
}

// Stop halts the spinner and prints a final success (ok=true) or failure line.
func (s *Spinner) Stop(label string, ok bool) {
	close(s.done)
	s.wg.Wait()
	// Clear the spinner line
	fmt.Print("\r\033[K")
	if ok {
		fmt.Println(SuccessStyle.Render("✓ " + label))
	} else {
		fmt.Println(ErrorStyle.Render("✗ " + label))
	}
}
