package tui

import (
	"fmt"

	"git.sr.ht/~jakintosh/teller/session"
	tea "github.com/charmbracelet/bubbletea"
)

// handleKey routes keyboard input to the appropriate handler based on the current view
func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.currentView {
	case viewBatch:
		return m, m.updateBatchView(msg)
	case viewTransaction:
		return m, m.updateTransactionView(msg)
	case viewTemplate:
		return m, m.updateTemplateView(msg)
	case viewConfirm:
		return m, m.updateConfirmView(msg)
	default:
		return m, nil
	}
}

// updateBatchView handles keyboard input in the batch summary view
func (m *Model) updateBatchView(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "ctrl+q", "ctrl+c":
		return tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.batch)-1 {
			m.cursor++
		}
	case "n":
		m.startNewTransaction()
	case "e", "enter":
		if len(m.batch) > 0 {
			m.startEditingTransaction(m.cursor)
		}
	case "w":
		if len(m.batch) == 0 {
			m.setStatus("No transactions to write", statusInfo, statusShortDuration)
		} else {
			m.openConfirm(confirmWrite)
		}
	case "q":
		m.openConfirm(confirmQuit)
	}
	return nil
}

// updateTransactionView handles keyboard input in the transaction entry view
func (m *Model) updateTransactionView(msg tea.KeyMsg) tea.Cmd {
	if m.form.focusedField == focusDate {
		if m.handleDateKey(msg) {
			m.recalculateTotals()
			return nil
		}
	}

	switch msg.String() {
	case "ctrl+q":
		return tea.Quit
	case "ctrl+c":
		m.form.cleared = !m.form.cleared
		return nil
	case "ctrl+s":
		m.confirmTransaction()
		return nil
	case "esc":
		m.cancelTransaction()
		return nil
	case "shift+tab":
		m.evaluateAmountField()
		m.retreatFocus()
		return nil
	case "tab":
		if !m.tryAcceptSuggestion() {
			m.evaluateAmountField()
			m.advanceFocus()
		}
		return nil
	case "enter":
		if m.handleEnterKey() {
			return nil
		}
	case "ctrl+a":
		if m.hasActiveLine() {
			m.addLine(m.form.focusedSection, true)
		}
		return nil
	case "ctrl+d":
		if m.hasActiveLine() {
			m.deleteLine(m.form.focusedSection)
		}
		return nil
	case "b":
		if m.balanceAnyLine() {
			m.recalculateTotals()
		}
		return nil
	}

	cmd := m.updateFocusedInput(msg)
	m.refreshSuggestions()
	m.refreshTemplateOptions()
	m.recalculateTotals()
	return cmd
}

// handleEnterKey processes the Enter key based on the currently focused field
// Returns true if the key was handled
func (m *Model) handleEnterKey() bool {
	switch m.form.focusedField {
	case focusDate:
		m.advanceFocus()
		return true
	case focusPayee:
		if !m.tryAcceptSuggestion() {
			m.advanceFocus()
		}
		return true
	case focusComment:
		m.advanceFocus()
		return true
	case focusTemplateButton:
		m.openTemplateSelection()
		return true
	case focusSectionAccount:
		m.advanceFocus()
		return true
	case focusSectionAmount:
		if line := m.currentLine(); line != nil {
			if m.evaluateInput(&line.amountInput) {
				m.advanceFocus()
			}
		}
		return true
	case focusSectionComment:
		m.advanceFocus()
		return true
	}
	return false
}

// updateFocusedInput delegates keyboard input to the currently focused text input
func (m *Model) updateFocusedInput(msg tea.KeyMsg) tea.Cmd {
	switch m.form.focusedField {
	case focusPayee:
		var cmd tea.Cmd
		m.form.payeeInput, cmd = m.form.payeeInput.Update(msg)
		return cmd
	case focusComment:
		var cmd tea.Cmd
		m.form.commentInput, cmd = m.form.commentInput.Update(msg)
		return cmd
	case focusSectionAccount:
		if line := m.currentLine(); line != nil {
			var cmd tea.Cmd
			line.accountInput, cmd = line.accountInput.Update(msg)
			return cmd
		}
	case focusSectionAmount:
		if line := m.currentLine(); line != nil {
			var cmd tea.Cmd
			line.amountInput, cmd = line.amountInput.Update(msg)
			return cmd
		}
	case focusSectionComment:
		if line := m.currentLine(); line != nil {
			var cmd tea.Cmd
			line.commentInput, cmd = line.commentInput.Update(msg)
			return cmd
		}
	}
	return nil
}

// updateConfirmView handles keyboard input in the confirmation view
func (m *Model) updateConfirmView(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "ctrl+q":
		return tea.Quit
	case "enter":
		switch m.pendingConfirm {
		case confirmWrite:
			if err := m.writeTransactionsToLedger(); err != nil {
				m.setStatus(fmt.Sprintf("Failed to write: %v", err), statusError, statusDuration)
			} else {
				count := len(m.batch)
				m.setStatus(fmt.Sprintf("Wrote %d transaction(s) to %s", count, m.ledgerFilePath), statusSuccess, statusShortDuration)
				m.batch = nil
				m.cursor = 0
				if err := session.DeleteSession(); err != nil {
					m.setStatus(fmt.Sprintf("Ledger written but session cleanup failed: %v", err), statusError, statusDuration)
				}
			}
			m.currentView = viewBatch
		case confirmQuit:
			if err := session.DeleteSession(); err != nil {
				m.setStatus(fmt.Sprintf("Failed to clear session: %v", err), statusError, statusDuration)
			}
			return tea.Quit
		}
	case "esc":
		m.currentView = viewBatch
	}
	return nil
}
