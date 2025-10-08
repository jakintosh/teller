package tui

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/shopspring/decimal"
)

// newTextInput creates a configured text input with standard settings
func newTextInput(placeholder string) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.Prompt = ""
	ti.CharLimit = 256
	ti.Width = 40
	ti.ShowSuggestions = true
	return ti
}

// newPostingLine creates a new posting line with account, amount, and comment inputs
func newPostingLine() postingLine {
	account := newTextInput("Account")
	amount := newTextInput("Amount")
	amount.ShowSuggestions = false
	comment := newTextInput("Comment")
	comment.ShowSuggestions = false
	comment.Width = 30
	return postingLine{accountInput: account, amountInput: amount, commentInput: comment}
}

// lineAmount extracts the decimal amount from a posting line
// Returns zero if the amount is empty or invalid
func lineAmount(line *postingLine) decimal.Decimal {
	value := strings.TrimSpace(line.amountInput.Value())
	if value == "" {
		return decimal.Zero
	}
	amount, err := decimal.NewFromString(value)
	if err != nil {
		return decimal.Zero
	}
	return amount
}

// categorySeed extracts the category prefix from an account value
// Used when cloning a line to suggest a related account
func categorySeed(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if !strings.Contains(value, ":") {
		return value + ":"
	}
	idx := strings.LastIndex(value, ":")
	if idx == len(value)-1 {
		return value
	}
	return value[:idx+1]
}

// addLine adds a new posting line to the specified section
// If cloneCategory is true, seeds the account field with the current line's category
func (m *Model) addLine(section sectionType, cloneCategory bool) {
	var lines *[]postingLine
	switch section {
	case sectionDebit:
		lines = &m.form.debitLines
	case sectionCredit:
		lines = &m.form.creditLines
	}
	seed := ""
	if cloneCategory {
		if line := m.currentLine(); line != nil {
			seed = categorySeed(line.accountInput.Value())
		}
	}
	newLine := newPostingLine()
	if seed != "" {
		newLine.accountInput.SetValue(seed)
	}
	*lines = append(*lines, newLine)
}

// deleteLine removes the currently focused posting line from the specified section
// Ensures at least one line remains in the section
func (m *Model) deleteLine(section sectionType) {
	var lines *[]postingLine
	switch section {
	case sectionDebit:
		lines = &m.form.debitLines
	case sectionCredit:
		lines = &m.form.creditLines
	}
	if lines == nil || len(*lines) <= 1 {
		m.setStatus("At least one line required", statusError, statusShortDuration)
		return
	}
	idx := m.form.focusedIndex
	if idx < 0 || idx >= len(*lines) {
		return
	}
	*lines = append((*lines)[:idx], (*lines)[idx+1:]...)
	if idx >= len(*lines) {
		idx = len(*lines) - 1
	}
	m.focusSection(section, idx, focusSectionAccount)
	m.recalculateTotals()
}

// newTransactionForm creates a new transaction form initialized with the given base date
func newTransactionForm(baseDate time.Time) transactionForm {
	if baseDate.IsZero() {
		baseDate = time.Now()
	}
	date := dateField{}
	date.setTime(baseDate)
	date.segment = dateSegmentDay

	payee := newTextInput("Payee")
	comment := newTextInput("Comment")
	comment.ShowSuggestions = false
	comment.Width = 60

	debit := []postingLine{newPostingLine()}
	credit := []postingLine{newPostingLine()}

	return transactionForm{
		date:           date,
		cleared:        true,
		payeeInput:     payee,
		commentInput:   comment,
		debitLines:     debit,
		creditLines:    credit,
		focusedField:   focusDate,
		focusedSection: sectionCredit,
		focusedIndex:   0,
		remaining:      decimal.Zero,
		debitTotal:     decimal.Zero,
		creditTotal:    decimal.Zero,
	}
}
