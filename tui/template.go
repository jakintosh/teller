package tui

import (
	"git.sr.ht/~jakintosh/teller/intelligence"
	tea "github.com/charmbracelet/bubbletea"
)

// openTemplateSelection opens the template selection view
func (m *Model) openTemplateSelection() {
	m.refreshTemplateOptions()
	if m.templatePayee == "" {
		m.setStatus("Enter a payee to view templates", statusShortDuration)
		return
	}
	if len(m.templateOptions) == 0 {
		m.setStatus("No templates available for this payee", statusShortDuration)
		return
	}
	if m.templateCursor >= len(m.templateOptions) {
		m.templateCursor = 0
	}
	m.ensureTemplateCursorVisible()
	m.currentView = viewTemplate
}

// updateTemplateView processes keyboard input in the template selection view
func (m *Model) updateTemplateView(msg tea.KeyMsg) tea.Cmd {
	if len(m.templateOptions) == 0 {
		m.currentView = viewTransaction
		return nil
	}
	switch msg.String() {
	case "up", "k":
		if m.templateCursor > 0 {
			m.templateCursor--
			m.ensureTemplateCursorVisible()
		}
	case "down", "j":
		if m.templateCursor < len(m.templateOptions)-1 {
			m.templateCursor++
			m.ensureTemplateCursorVisible()
		}
	case "enter":
		m.applyTemplate(m.templateOptions[m.templateCursor])
	case "esc":
		m.skipTemplate()
	}
	return nil
}

// applyTemplate populates the transaction form with accounts from the selected template
func (m *Model) applyTemplate(record intelligence.TemplateRecord) {
	m.form.debitLines = nil
	for _, account := range record.DebitAccounts {
		line := newPostingLine()
		line.accountInput.SetValue(account)
		line.accountInput.CursorEnd()
		m.form.debitLines = append(m.form.debitLines, line)
	}
	if len(m.form.debitLines) == 0 {
		m.form.debitLines = []postingLine{newPostingLine()}
	}

	m.form.creditLines = nil
	for _, account := range record.CreditAccounts {
		line := newPostingLine()
		line.accountInput.SetValue(account)
		line.accountInput.CursorEnd()
		m.form.creditLines = append(m.form.creditLines, line)
	}
	if len(m.form.creditLines) == 0 {
		m.form.creditLines = []postingLine{newPostingLine()}
	}

	m.currentView = viewTransaction
	m.focusSection(sectionCredit, 0, focusSectionAccount)
	m.recalculateTotals()
}

// skipTemplate returns to the transaction view without applying a template
func (m *Model) skipTemplate() {
	m.currentView = viewTransaction
	m.focusSection(sectionCredit, 0, focusSectionAccount)
	m.recalculateTotals()
}

// ensureTemplateCursorVisible adjusts the template offset to keep the cursor visible
func (m *Model) ensureTemplateCursorVisible() {
	if len(m.templateOptions) == 0 {
		m.templateOffset = 0
		return
	}
	if m.templateCursor < 0 {
		m.templateCursor = 0
	}
	if m.templateCursor >= len(m.templateOptions) {
		m.templateCursor = len(m.templateOptions) - 1
	}
	if m.templateCursor < m.templateOffset {
		m.templateOffset = m.templateCursor
	}
	visible := maxTemplateDisplay
	if visible <= 0 {
		visible = 1
	}
	if m.templateCursor >= m.templateOffset+visible {
		m.templateOffset = m.templateCursor - visible + 1
	}
	maxOffset := len(m.templateOptions) - visible
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.templateOffset > maxOffset {
		m.templateOffset = maxOffset
	}
	if m.templateOffset < 0 {
		m.templateOffset = 0
	}
}
