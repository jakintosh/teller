package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
)

// renderBatchView displays the batch summary screen
func (m *Model) renderBatchView() string {
	var b strings.Builder
	fmt.Fprintf(&b, "-- Batch Summary (%d transactions) --\n", len(m.batch))
	for _, line := range m.loadSummaryLines() {
		fmt.Fprintf(&b, "%s\n", line)
	}
	b.WriteString("\n")
	if len(m.batch) == 0 {
		b.WriteString("No transactions in current batch.\n\n")
	} else {
		for i, tx := range m.batch {
			cursor := " "
			if i == m.cursor {
				cursor = ">"
			}
			payee := tx.Payee
			if len(payee) > 28 {
				payee = payee[:25] + "..."
			}
			primary := ""
			if len(tx.Postings) > 0 {
				primary = tx.Postings[0].Account
				parts := strings.Split(primary, ":")
				primary = parts[len(parts)-1]
			}
			fmt.Fprintf(&b, "%s %s %-28s (%s)\n", cursor, tx.Date.Format("2006-01-02"), payee, primary)
		}
		b.WriteString("\n")
	}
	if msg := m.statusLine(); msg != "" {
		fmt.Fprintf(&b, "%s\n\n", msg)
	}
	b.WriteString("[n]ew  [e]dit  [w]rite  [q]uit  [enter]edit selected")
	return b.String()
}

// loadSummaryLines generates status lines describing the data load results
func (m *Model) loadSummaryLines() []string {
	line := fmt.Sprintf(
		"Data load: %d transactions • %d payees • %d templates",
		m.loadSummary.Transactions,
		m.loadSummary.UniquePayees,
		m.loadSummary.UniqueTemplates,
	)
	lines := []string{line}
	if m.loadSummary.HasIssues() {
		first := m.loadSummary.Issues[0]
		stage := strings.ToUpper(first.Stage)
		if stage == "" {
			stage = "GENERAL"
		}
		if len(m.loadSummary.Issues) == 1 {
			lines = append(lines, fmt.Sprintf("Load issue: [%s] %s", stage, first.Message))
		} else {
			lines = append(lines, fmt.Sprintf("Load issues: %d (first: [%s] %s)", len(m.loadSummary.Issues), stage, first.Message))
		}
		return lines
	}
	return append(lines, "Load issues: none")
}

// renderTransactionView displays the transaction entry form
func (m *Model) renderTransactionView() string {
	var b strings.Builder
	fmt.Fprintf(&b, "-- Transaction Entry -- Remaining: $%s --\n\n", m.form.remaining.StringFixed(2))

	dateDisplay := m.form.date.display(m.form.focusedField == focusDate)
	clearedCursor := " "
	if m.form.focusedField == focusCleared {
		clearedCursor = ">"
	}
	clearedMark := " "
	if m.form.cleared {
		clearedMark = "x"
	}
	fmt.Fprintf(&b, "Date    %s  %sCleared [%s]\n", dateDisplay, clearedCursor, clearedMark)
	fmt.Fprintf(&b, "Payee   %s", m.form.payeeInput.View())
	if m.form.focusedField == focusPayee {
		b.WriteString(renderSuggestionList(m.form.payeeInput))
	}
	b.WriteString("\n")
	fmt.Fprintf(&b, "Comment %s\n\n", m.form.commentInput.View())
	buttonCursor := " "
	if m.form.focusedField == focusTemplateButton {
		buttonCursor = ">"
	}
	fmt.Fprintf(&b, "        %s[%s]\n\n", buttonCursor, templateAvailabilityLabel(len(m.templateOptions)))

	fmt.Fprintf(&b, "Credits  (total %s)\n", m.form.creditTotal.StringFixed(2))
	for i, line := range m.form.creditLines {
		cursor := " "
		if m.lineHasFocus(sectionCredit, i) {
			cursor = ">"
		}
		fmt.Fprintf(&b, "%s [%s] [%s] [%s]", cursor, line.accountInput.View(), line.amountInput.View(), line.commentInput.View())
		if m.lineHasFocus(sectionCredit, i) && m.form.focusedField == focusSectionAccount {
			b.WriteString(renderSuggestionList(line.accountInput))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")

	fmt.Fprintf(&b, "Debits   (total %s)\n", m.form.debitTotal.StringFixed(2))
	for i, line := range m.form.debitLines {
		cursor := " "
		if m.lineHasFocus(sectionDebit, i) {
			cursor = ">"
		}
		fmt.Fprintf(&b, "%s [%s] [%s] [%s]", cursor, line.accountInput.View(), line.amountInput.View(), line.commentInput.View())
		if m.lineHasFocus(sectionDebit, i) && m.form.focusedField == focusSectionAccount {
			b.WriteString(renderSuggestionList(line.accountInput))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")

	if msg := m.statusLine(); msg != "" {
		fmt.Fprintf(&b, "%s\n\n", msg)
	}

	commands := []string{"[tab]next", "[shift+tab]prev"}
	if m.hasActiveLine() {
		commands = append(commands, "[ctrl+n]add line", "[ctrl+d]delete line")
	}
	if m.canBalanceCurrentLine() {
		commands = append(commands, "[b]alance")
	}
	if m.form.focusedField == focusCleared {
		commands = append(commands, "[space]toggle cleared")
	}
	commands = append(commands, "[ctrl+c]confirm", "[esc]cancel", "[ctrl+q]quit")
	b.WriteString(strings.Join(commands, "\n"))
	return b.String()
}

// templateAvailabilityLabel returns a label describing the number of templates available
func templateAvailabilityLabel(count int) string {
	if count == 1 {
		return "1 template available"
	}
	return fmt.Sprintf("%d templates available", count)
}

// renderTemplateView displays the template selection screen
func (m *Model) renderTemplateView() string {
	var b strings.Builder
	fmt.Fprintf(&b, "-- Templates for %s --\n\n", m.form.payeeInput.Value())
	if len(m.templateOptions) == 0 {
		b.WriteString("No templates available\n\n[esc]skip")
		return b.String()
	}
	start := m.templateOffset
	if start < 0 {
		start = 0
	}
	if start >= len(m.templateOptions) {
		start = len(m.templateOptions) - 1
	}
	end := start + maxTemplateDisplay
	if end > len(m.templateOptions) {
		end = len(m.templateOptions)
	}
	for i := start; i < end; i++ {
		tpl := m.templateOptions[i]
		cursor := " "
		if i == m.templateCursor {
			cursor = ">"
		}
		usageLabel := "times"
		if tpl.Frequency == 1 {
			usageLabel = "time"
		}
		fmt.Fprintf(&b, "%s %d. Used %d %s\n", cursor, i+1, tpl.Frequency, usageLabel)
		b.WriteString("    Debit Accounts:\n")
		if len(tpl.DebitAccounts) == 0 {
			b.WriteString("      (none)\n")
		} else {
			for _, account := range tpl.DebitAccounts {
				fmt.Fprintf(&b, "      %s\n", account)
			}
		}
		b.WriteString("    Credit Accounts:\n")
		if len(tpl.CreditAccounts) == 0 {
			b.WriteString("      (none)\n")
		} else {
			for _, account := range tpl.CreditAccounts {
				fmt.Fprintf(&b, "      %s\n", account)
			}
		}
		if i < end-1 {
			b.WriteString("\n")
		}
	}
	b.WriteString("\n[enter]apply\n[esc]skip")
	return b.String()
}

// renderConfirmView displays the confirmation dialog
func (m *Model) renderConfirmView() string {
	var b strings.Builder
	switch m.pendingConfirm {
	case confirmWrite:
		fmt.Fprintf(&b, "Write %d transaction(s) to %s?\n\n", len(m.batch), m.ledgerFilePath)
	case confirmQuit:
		if len(m.batch) > 0 {
			fmt.Fprintf(&b, "Quit without writing %d pending transaction(s)?\n\n", len(m.batch))
		} else {
			b.WriteString("Quit the application?\n\n")
		}
	}
	b.WriteString("[enter]confirm  [esc]cancel  [ctrl+q]quit immediately")
	return b.String()
}

// renderSuggestionList displays autocomplete suggestions below an input field
func renderSuggestionList(input textinput.Model) string {
	matches := input.MatchedSuggestions()
	if len(matches) <= 1 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n")
	display := len(matches)
	if display > maxSuggestionDisplay {
		display = maxSuggestionDisplay
	}
	for i := 0; i < display; i++ {
		cursor := " "
		if i == input.CurrentSuggestionIndex() {
			cursor = ">"
		}
		fmt.Fprintf(&b, "      %s %s\n", cursor, matches[i])
	}
	if len(matches) > display {
		fmt.Fprintf(&b, "      ... and %d more\n", len(matches)-display)
	}
	return b.String()
}
