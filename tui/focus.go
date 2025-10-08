package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
)

// advanceFocus moves focus to the next field in the transaction form
func (m *Model) advanceFocus() {
	switch m.form.focusedField {
	case focusDate:
		m.form.focusedField = focusCleared
	case focusCleared:
		m.form.focusedField = focusPayee
		m.form.payeeInput.Focus()
	case focusPayee:
		m.form.payeeInput.Blur()
		m.form.focusedField = focusComment
		m.form.commentInput.Focus()
	case focusComment:
		m.focusTemplateButton()
		m.refreshTemplateOptions()
	case focusTemplateButton:
		m.focusSection(sectionCredit, 0, focusSectionAccount)
	case focusSectionAccount:
		if line := m.currentLine(); line != nil {
			line.accountInput.Blur()
			line.amountInput.Focus()
			m.form.focusedField = focusSectionAmount
		}
	case focusSectionAmount:
		if line := m.currentLine(); line != nil {
			line.amountInput.Blur()
			line.commentInput.Focus()
			m.form.focusedField = focusSectionComment
		}
	case focusSectionComment:
		if line := m.currentLine(); line != nil {
			line.commentInput.Blur()
		}
		if m.form.focusedSection == sectionCredit {
			if m.form.focusedIndex < len(m.form.creditLines)-1 {
				m.focusSection(sectionCredit, m.form.focusedIndex+1, focusSectionAccount)
			} else {
				m.focusSection(sectionDebit, 0, focusSectionAccount)
			}
		} else {
			if m.form.focusedIndex < len(m.form.debitLines)-1 {
				m.focusSection(sectionDebit, m.form.focusedIndex+1, focusSectionAccount)
			} else {
				m.addLine(sectionDebit, false)
				m.focusSection(sectionDebit, len(m.form.debitLines)-1, focusSectionAccount)
			}
		}
	}
	m.refreshSuggestions()
}

// retreatFocus moves focus to the previous field in the transaction form
func (m *Model) retreatFocus() {
	switch m.form.focusedField {
	case focusCleared:
		m.form.focusedField = focusDate
	case focusPayee:
		m.form.payeeInput.Blur()
		m.form.focusedField = focusDate
	case focusComment:
		m.form.commentInput.Blur()
		m.form.focusedField = focusPayee
		m.form.payeeInput.Focus()
	case focusTemplateButton:
		m.form.focusedField = focusComment
		m.form.commentInput.Focus()
	case focusSectionAccount:
		if m.form.focusedSection == sectionCredit {
			if m.form.focusedIndex == 0 {
				m.form.focusedField = focusComment
				m.form.commentInput.Focus()
			} else {
				m.focusSection(sectionCredit, m.form.focusedIndex-1, focusSectionAmount)
			}
		} else {
			if m.form.focusedIndex == 0 {
				if len(m.form.creditLines) > 0 {
					m.focusSection(sectionCredit, len(m.form.creditLines)-1, focusSectionAmount)
				} else {
					m.form.focusedField = focusComment
					m.form.commentInput.Focus()
				}
			} else {
				m.focusSection(sectionDebit, m.form.focusedIndex-1, focusSectionAmount)
			}
		}
	case focusSectionAmount:
		m.focusSection(m.form.focusedSection, m.form.focusedIndex, focusSectionAccount)
	case focusSectionComment:
		m.focusSection(m.form.focusedSection, m.form.focusedIndex, focusSectionAmount)
	default:
		m.form.focusedField = focusDate
	}
	m.refreshSuggestions()
}

// focusSection sets focus to a specific posting line field
func (m *Model) focusSection(section sectionType, index int, field focusedField) {
	if section == sectionDebit {
		if index >= len(m.form.debitLines) {
			index = len(m.form.debitLines) - 1
		}
	} else {
		if index >= len(m.form.creditLines) {
			index = len(m.form.creditLines) - 1
		}
	}
	if index < 0 {
		index = 0
	}
	m.blurCurrent()
	m.form.focusedSection = section
	m.form.focusedIndex = index
	m.form.focusedField = field
	if line := m.currentLine(); line != nil {
		line.accountInput.Blur()
		line.amountInput.Blur()
		line.commentInput.Blur()
		switch field {
		case focusSectionAccount:
			line.accountInput.Focus()
		case focusSectionAmount:
			line.amountInput.Focus()
		case focusSectionComment:
			line.commentInput.Focus()
		}
	}
}

// focusTemplateButton sets focus to the template button
func (m *Model) focusTemplateButton() {
	m.blurCurrent()
	m.form.focusedField = focusTemplateButton
}

// blurCurrent removes focus from the currently focused input field
func (m *Model) blurCurrent() {
	if line := m.currentLine(); line != nil {
		line.accountInput.Blur()
		line.amountInput.Blur()
		line.commentInput.Blur()
	}
	if m.form.focusedField == focusPayee {
		m.form.payeeInput.Blur()
	}
	if m.form.focusedField == focusComment {
		m.form.commentInput.Blur()
	}
}

// currentLine returns the currently focused posting line, or nil if no line is focused
func (m *Model) currentLine() *postingLine {
	switch m.form.focusedSection {
	case sectionDebit:
		if m.form.focusedIndex >= 0 && m.form.focusedIndex < len(m.form.debitLines) {
			return &m.form.debitLines[m.form.focusedIndex]
		}
	case sectionCredit:
		if m.form.focusedIndex >= 0 && m.form.focusedIndex < len(m.form.creditLines) {
			return &m.form.creditLines[m.form.focusedIndex]
		}
	}
	return nil
}

// lineHasFocus returns true if the specified posting line currently has focus
func (m *Model) lineHasFocus(section sectionType, index int) bool {
	if m.form.focusedField != focusSectionAccount && m.form.focusedField != focusSectionAmount && m.form.focusedField != focusSectionComment {
		return false
	}
	return m.form.focusedSection == section && m.form.focusedIndex == index
}

// hasActiveLine returns true if a posting line field currently has focus
func (m *Model) hasActiveLine() bool {
	if m.form.focusedField != focusSectionAccount && m.form.focusedField != focusSectionAmount && m.form.focusedField != focusSectionComment {
		return false
	}
	return m.currentLine() != nil
}

// currentTextInput returns the currently focused text input, or nil if no text input is focused
func (m *Model) currentTextInput() *textinput.Model {
	switch m.form.focusedField {
	case focusPayee:
		return &m.form.payeeInput
	case focusComment:
		return &m.form.commentInput
	case focusSectionAccount:
		if line := m.currentLine(); line != nil {
			return &line.accountInput
		}
	case focusSectionAmount:
		if line := m.currentLine(); line != nil {
			return &line.amountInput
		}
	case focusSectionComment:
		if line := m.currentLine(); line != nil {
			return &line.commentInput
		}
	}
	return nil
}

// textInputFocused returns true if any text input currently has focus
func (m *Model) textInputFocused() bool {
	switch m.form.focusedField {
	case focusPayee:
		return m.form.payeeInput.Focused()
	case focusComment:
		return m.form.commentInput.Focused()
	case focusSectionAccount:
		if line := m.currentLine(); line != nil {
			return line.accountInput.Focused()
		}
	case focusSectionAmount:
		if line := m.currentLine(); line != nil {
			return line.amountInput.Focused()
		}
	case focusSectionComment:
		if line := m.currentLine(); line != nil {
			return line.commentInput.Focused()
		}
	}
	return false
}
