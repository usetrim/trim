// Package cliout provides TTY-aware, professional CLI presentation helpers.
// Colors and spinners activate only when stdout is a terminal and NO_COLOR is unset.
package cliout

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

// Writer is where human output goes (tests can override).
var Writer io.Writer = os.Stdout

func enabled() bool {
	if strings.TrimSpace(os.Getenv("NO_COLOR")) != "" {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(os.Getenv("TRIM_CLI_COLOR")), "0") {
		return false
	}
	f, ok := Writer.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}

func style(s lipgloss.Style, text string) string {
	if !enabled() || text == "" {
		return text
	}
	return s.Render(text)
}

var (
	stSuccess = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)  // bright green
	stWarn    = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true)  // yellow
	stError   = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)   // red
	stInfo    = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)  // blue
	stMuted   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))              // gray
	stKey     = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))             // cyan
	stValue   = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))             // white
	stAccent  = lipgloss.NewStyle().Foreground(lipgloss.Color("13")).Bold(true)  // magenta
	stHeader  = lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Bold(true)
)

// Success prints a green success line (● message).
func Success(msg string) {
	if msg == "" {
		return
	}
	fmt.Fprintln(Writer, style(stSuccess, "●")+" "+style(stHeader, msg))
}

// Warn prints a yellow warning line.
func Warn(msg string) {
	if msg == "" {
		return
	}
	fmt.Fprintln(Writer, style(stWarn, "!")+" "+msg)
}

// ErrorLine prints a red error line (does not exit).
func ErrorLine(msg string) {
	if msg == "" {
		return
	}
	fmt.Fprintln(Writer, style(stError, "✗")+" "+msg)
}

// Info prints a blue info line.
func Info(msg string) {
	if msg == "" {
		return
	}
	fmt.Fprintln(Writer, style(stInfo, "ℹ")+" "+msg)
}

// Header prints a bold section header.
func Header(msg string) {
	if msg == "" {
		return
	}
	fmt.Fprintln(Writer, style(stHeader, msg))
}

// Muted prints gray secondary text.
func Muted(msg string) {
	if msg == "" {
		return
	}
	fmt.Fprintln(Writer, style(stMuted, msg))
}

// Row is one label/value pair for a scannable table.
type Row struct {
	Label string
	Value string
	// Kind tints the value: "", "ok", "warn", "bad", "accent"
	Kind string
}

// Table prints aligned label/value rows (human default).
func Table(rows []Row) {
	width := 0
	for _, r := range rows {
		if r.Label == "" && r.Value == "" {
			continue
		}
		if w := displayWidth(r.Label); w > width {
			width = w
		}
	}
	if width < 10 {
		width = 10
	}
	for _, r := range rows {
		if r.Label == "" && r.Value == "" {
			continue
		}
		pad := width - displayWidth(r.Label)
		if pad < 0 {
			pad = 0
		}
		label := style(stKey, r.Label)
		val := colorValue(r.Value, r.Kind)
		fmt.Fprintf(Writer, "  %s%s  %s\n", label, strings.Repeat(" ", pad), val)
	}
}

func colorValue(v, kind string) string {
	switch kind {
	case "ok":
		return style(stSuccess, v)
	case "warn":
		return style(stWarn, v)
	case "bad":
		return style(stError, v)
	case "accent":
		return style(stAccent, v)
	default:
		return style(stValue, v)
	}
}

func displayWidth(s string) int {
	n := 0
	for _, r := range s {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		n++
	}
	return n
}

// TitleCaseWord uppercases the first letter (plan_tier display).
func TitleCaseWord(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// Spinner shows a brief animated progress line while fn-less long work runs.
// Returns a stop function (idempotent). No-op when not a TTY or JSON mode.
func Spinner(label string) (stop func()) {
	if !enabled() || strings.TrimSpace(label) == "" {
		return func() {}
	}
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	done := make(chan struct{})
	var once sync.Once
	go func() {
		i := 0
		for {
			select {
			case <-done:
				// Clear spinner line.
				fmt.Fprintf(Writer, "\r\033[K")
				return
			default:
				frame := style(stInfo, frames[i%len(frames)])
				fmt.Fprintf(Writer, "\r%s %s", frame, style(stMuted, label))
				i++
				time.Sleep(80 * time.Millisecond)
			}
		}
	}()
	return func() {
		once.Do(func() {
			close(done)
			time.Sleep(90 * time.Millisecond)
		})
	}
}
