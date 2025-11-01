package tui

import (
	"sort"
	"strings"
)

// refreshSuggestions updates the suggestion list for the currently focused input field
func (m *Model) refreshSuggestions() {
	switch m.form.focusedField {
	case focusPayee:
		m.form.payeeInput.SetSuggestions(m.db.FindPayees(m.form.payeeInput.Value()))
	case focusSectionAccount:
		if line := m.currentLine(); line != nil {
			line.accountInput.SetSuggestions(m.accountSuggestions(line.accountInput.Value()))
		}
	case focusSectionAmount:
		if line := m.currentLine(); line != nil {
			line.amountInput.SetSuggestions(nil)
		}
	case focusComment:
		m.form.commentInput.SetSuggestions(nil)
	case focusSectionComment:
		if line := m.currentLine(); line != nil {
			line.commentInput.SetSuggestions(nil)
		}
	}
}

// refreshTemplateOptions updates the template options based on the current payee
func (m *Model) refreshTemplateOptions() {
	payee := strings.TrimSpace(m.form.payeeInput.Value())
	if payee == m.templatePayee {
		return
	}
	m.templatePayee = payee
	if payee == "" {
		m.templateOptions = nil
	} else {
		m.templateOptions = m.db.FindTemplates(payee)
	}
	m.templateCursor = 0
	m.templateOffset = 0
}

// tryAcceptSuggestion attempts to accept the current suggestion for the focused input
// Returns true if a suggestion was accepted
func (m *Model) tryAcceptSuggestion() bool {
	input := m.currentTextInput()
	if input == nil {
		return false
	}

	// Only accept suggestions from inputs that actually have focus.
	// This prevents accessing stale suggestion state from unfocused inputs,
	// which can happen during rapid focus changes or state transitions.
	if !input.Focused() {
		return false
	}

	suggestion := input.CurrentSuggestion()
	if suggestion == "" {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(input.Value()), suggestion) {
		return false
	}
	input.SetValue(suggestion)
	input.CursorEnd()
	m.refreshSuggestions()
	if input == &m.form.payeeInput {
		m.templatePayee = ""
		m.refreshTemplateOptions()
	}
	return true
}

// accountSuggestions generates hierarchical account suggestions based on the prefix
func (m *Model) accountSuggestions(prefix string) []string {
	raw := m.db.FindAccounts(prefix)
	if len(raw) == 0 {
		return nil
	}
	seen := make(map[string]struct{})
	suggestions := make([]string, 0, len(raw))
	for _, account := range raw {
		suggestion := nextHierarchicalSuggestion(prefix, account)
		if suggestion == "" {
			continue
		}
		key := strings.ToLower(suggestion)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		suggestions = append(suggestions, suggestion)
	}
	sort.Strings(suggestions)
	return suggestions
}

// nextHierarchicalSuggestion calculates the next hierarchical account suggestion
// For example, given prefix "Exp" and account "Expenses:Food:Groceries", returns "Expenses"
// Given prefix "Expenses:" and the same account, returns "Expenses:Food"
func nextHierarchicalSuggestion(prefix, account string) string {
	if prefix == "" {
		parts := strings.Split(account, ":")
		if len(parts) > 0 {
			return parts[0]
		}
		return account
	}
	lowerPrefix := strings.ToLower(prefix)
	lowerAccount := strings.ToLower(account)
	if !strings.HasPrefix(lowerAccount, lowerPrefix) {
		return ""
	}
	if len(account) == len(prefix) {
		return account
	}
	if strings.HasSuffix(prefix, ":") {
		remainder := account[len(prefix):]
		idx := strings.IndexRune(remainder, ':')
		if idx == -1 {
			return account
		}
		return account[:len(prefix)+idx]
	}
	prefixSegments := strings.Split(prefix, ":")
	accountSegments := strings.Split(account, ":")
	if len(prefixSegments) > len(accountSegments) {
		return account
	}
	for i := 0; i < len(prefixSegments)-1; i++ {
		if !strings.EqualFold(prefixSegments[i], accountSegments[i]) {
			return ""
		}
	}
	lastPrefix := prefixSegments[len(prefixSegments)-1]
	accountSegment := accountSegments[len(prefixSegments)-1]
	if !strings.HasPrefix(strings.ToLower(accountSegment), strings.ToLower(lastPrefix)) {
		return ""
	}
	return strings.Join(accountSegments[:len(prefixSegments)], ":")
}
