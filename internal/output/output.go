package output

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var (
	Success   = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	Error     = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	Grey      = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	Highlight = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
	Bold      = lipgloss.NewStyle().Bold(true)

	SymbolCheckmark = Success.Render("✓")
	SymbolX         = Error.Render("✗")
)

func Pass(msg string) string {
	return fmt.Sprintf("%s %s", SymbolCheckmark, Success.Render(msg))
}

func Fail(msg string) string {
	return fmt.Sprintf("%s %s", SymbolX, Grey.Render(msg))
}

func Successf(format string, args ...any) string {
	return Success.Render(fmt.Sprintf(format, args...))
}

func Errorf(format string, args ...any) string {
	return Error.Render(fmt.Sprintf(format, args...))
}

func Boldf(format string, args ...any) string {
	return Bold.Render(fmt.Sprintf(format, args...))
}

func File(path string) string {
	return Highlight.Render(path)
}
