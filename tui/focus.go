package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
)

// buildFocusPath creates an ordered list of all focusable elements based on visual layout
// Order: Date → Payee → Comment → Template → Debits → Credits
func (m *Model) buildFocusPath() []focusPosition {
	path := []focusPosition{
		{field: focusDate},
		{field: focusPayee},
		{field: focusComment},
		{field: focusTemplateButton},
	}

	// Add all debit lines (account → amount → comment for each)
	for i := range m.form.debitLines {
		path = append(path,
			focusPosition{field: focusSectionAccount, section: sectionDebit, index: i},
			focusPosition{field: focusSectionAmount, section: sectionDebit, index: i},
			focusPosition{field: focusSectionComment, section: sectionDebit, index: i},
		)
	}

	// Add all credit lines (account → amount → comment for each)
	for i := range m.form.creditLines {
		path = append(path,
			focusPosition{field: focusSectionAccount, section: sectionCredit, index: i},
			focusPosition{field: focusSectionAmount, section: sectionCredit, index: i},
			focusPosition{field: focusSectionComment, section: sectionCredit, index: i},
		)
	}

	return path
}

// currentPosition returns the current focus position
func (m *Model) currentPosition() focusPosition {
	return focusPosition{
		field:   m.form.focusedField,
		section: m.form.focusedSection,
		index:   m.form.focusedIndex,
	}
}

// findPositionInPath returns the index of the given position in the path, or -1 if not found
func findPositionInPath(path []focusPosition, pos focusPosition) int {
	for i, p := range path {
		// For header fields, only compare the field itself
		if p.field == pos.field {
			switch p.field {
			case focusDate, focusPayee, focusComment, focusTemplateButton:
				return i
			case focusSectionAccount, focusSectionAmount, focusSectionComment:
				// For posting line fields, also compare section and index
				if p.section == pos.section && p.index == pos.index {
					return i
				}
			}
		}
	}
	return -1
}

// moveFocusToPosition moves focus to the specified position, handling all blur/focus transitions
func (m *Model) moveFocusToPosition(pos focusPosition) {
	m.blurCurrent()
	m.form.focusedField = pos.field
	m.form.focusedSection = pos.section
	m.form.focusedIndex = pos.index

	// Focus the appropriate input field
	switch pos.field {
	case focusPayee:
		m.form.payeeInput.Focus()
	case focusComment:
		m.form.commentInput.Focus()
	case focusSectionAccount, focusSectionAmount, focusSectionComment:
		if line := m.currentLine(); line != nil {
			switch pos.field {
			case focusSectionAccount:
				line.accountInput.Focus()
			case focusSectionAmount:
				line.amountInput.Focus()
			case focusSectionComment:
				line.commentInput.Focus()
			}
		}
	}
}

// advanceFocus moves focus to the next field in the transaction form
func (m *Model) advanceFocus() {
	// Build the current focus path
	path := m.buildFocusPath()
	current := m.currentPosition()
	currentIdx := findPositionInPath(path, current)

	// Special case: refresh templates when leaving comment field
	if current.field == focusComment {
		m.refreshTemplateOptions()
	}

	// Special case: at end of last credit line, add a new line
	if current.field == focusSectionComment &&
		current.section == sectionCredit &&
		current.index == len(m.form.creditLines)-1 {
		m.addLine(sectionCredit, false)
		// Rebuild path after adding line
		path = m.buildFocusPath()
		currentIdx = findPositionInPath(path, current)
	}

	// Move to next position in path
	if currentIdx >= 0 && currentIdx < len(path)-1 {
		m.moveFocusToPosition(path[currentIdx+1])
	}

	m.refreshSuggestions()
}

// retreatFocus moves focus to the previous field in the transaction form
func (m *Model) retreatFocus() {
	// Build the current focus path
	path := m.buildFocusPath()
	current := m.currentPosition()
	currentIdx := findPositionInPath(path, current)

	// Move to previous position in path
	if currentIdx > 0 {
		m.moveFocusToPosition(path[currentIdx-1])
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

// focusFirstPostingLine moves focus to the first posting line in the focus path
// This is the field that comes after the template button
func (m *Model) focusFirstPostingLine() {
	path := m.buildFocusPath()
	// Find the template button position
	templatePos := focusPosition{field: focusTemplateButton}
	templateIdx := findPositionInPath(path, templatePos)

	// Move to the next position after the template button
	if templateIdx >= 0 && templateIdx < len(path)-1 {
		m.moveFocusToPosition(path[templateIdx+1])
	}
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
