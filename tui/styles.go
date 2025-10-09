package tui

import "github.com/charmbracelet/lipgloss"

// Color definitions for the TUI
var (
	// Balance colors
	balancedColor   = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))  // Green
	unbalancedColor = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))   // Red

	// Status message colors
	successColor = lipgloss.NewStyle().Foreground(lipgloss.Color("10")) // Green
	errorColor   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))  // Red
	infoColor    = lipgloss.NewStyle().Foreground(lipgloss.Color("12")) // Blue

	// Section colors
	creditColor = lipgloss.NewStyle().Foreground(lipgloss.Color("14")) // Cyan
	debitColor  = lipgloss.NewStyle().Foreground(lipgloss.Color("11")) // Yellow

	// UI element colors
	cursorColor     = lipgloss.NewStyle().Foreground(lipgloss.Color("13")).Bold(true) // Bright magenta
	noIssuesColor   = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))            // Green
	issuesColor     = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))             // Red
	frequencyColor  = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))            // Cyan
	dimmedColor     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))             // Dark grey for disabled commands
	activeColor     = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))            // White for active commands
)

// formatBalanced returns a colored string for the remaining balance
func formatBalanced(amount string, isBalanced bool) string {
	if isBalanced {
		return balancedColor.Render("$" + amount)
	}
	return unbalancedColor.Render("$" + amount)
}

// formatStatus returns a colored status message based on the status kind
func formatStatus(message string, kind statusKind) string {
	switch kind {
	case statusSuccess:
		return successColor.Render(message)
	case statusError:
		return errorColor.Render(message)
	case statusInfo:
		return infoColor.Render(message)
	default:
		return message
	}
}

// formatCursor returns a colored cursor marker
func formatCursor(marker string) string {
	return cursorColor.Render(marker)
}

// formatCreditTotal returns a colored credit total
func formatCreditTotal(amount string) string {
	return creditColor.Render(amount)
}

// formatDebitTotal returns a colored debit total
func formatDebitTotal(amount string) string {
	return debitColor.Render(amount)
}

// formatNoIssues returns a colored "no issues" message
func formatNoIssues(message string) string {
	return noIssuesColor.Render(message)
}

// formatIssues returns a colored issues message
func formatIssues(message string) string {
	return issuesColor.Render(message)
}

// formatFrequency returns a colored frequency count
func formatFrequency(text string) string {
	return frequencyColor.Render(text)
}

// formatCommand returns a command hint, dimmed if disabled, white if enabled
func formatCommand(text string, enabled bool) string {
	if enabled {
		return activeColor.Render(text)
	}
	return dimmedColor.Render(text)
}
