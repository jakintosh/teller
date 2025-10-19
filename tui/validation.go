package tui

import (
	"fmt"
	"sort"
	"strings"

	"git.sr.ht/~jakintosh/teller/core"
	"git.sr.ht/~jakintosh/teller/session"
	"git.sr.ht/~jakintosh/teller/util"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/shopspring/decimal"
)

// recalculateTotals updates the debit, credit, and remaining totals based on current posting lines
func (m *Model) recalculateTotals() {
	debit := decimal.Zero
	for i := range m.form.debitLines {
		debit = debit.Add(lineAmount(&m.form.debitLines[i]))
	}
	credit := decimal.Zero
	for i := range m.form.creditLines {
		credit = credit.Add(lineAmount(&m.form.creditLines[i]))
	}
	m.form.debitTotal = debit
	m.form.creditTotal = credit
	m.form.remaining = debit.Add(credit)
}

// evaluateAmountField evaluates the expression in the currently focused amount field
func (m *Model) evaluateAmountField() {
	if m.form.focusedField != focusSectionAmount {
		return
	}
	if line := m.currentLine(); line != nil {
		m.evaluateInput(&line.amountInput)
	}
}

// evaluateInput evaluates a mathematical expression in the given input field
// Returns true if the evaluation was successful or the field was empty
func (m *Model) evaluateInput(input *textinput.Model) bool {
	value := strings.TrimSpace(input.Value())
	if value == "" {
		return true
	}
	evaluated, err := util.EvaluateExpression(value)
	if err != nil {
		m.setStatus(fmt.Sprintf("Invalid expression: %v", err), statusError, statusDuration)
		return false
	}
	input.SetValue(evaluated)
	input.CursorEnd()
	return true
}

// canBalanceAnyLine returns true if there is exactly one unfilled amount in the entire form
// This check works regardless of which field is currently focused
func (m *Model) canBalanceAnyLine() bool {
	if len(m.form.debitLines)+len(m.form.creditLines) < 2 {
		return false
	}
	unfilled := 0
	for i := range m.form.debitLines {
		if strings.TrimSpace(m.form.debitLines[i].amountInput.Value()) == "" {
			unfilled++
		}
	}
	for i := range m.form.creditLines {
		if strings.TrimSpace(m.form.creditLines[i].amountInput.Value()) == "" {
			unfilled++
		}
	}
	return unfilled == 1
}

// canBalanceCurrentLine returns true if the current line can be auto-balanced
// This is only allowed when on an amount field with exactly one empty amount in the entire form
func (m *Model) canBalanceCurrentLine() bool {
	if m.form.focusedField != focusSectionAmount {
		return false
	}
	line := m.currentLine()
	if line == nil {
		return false
	}
	if strings.TrimSpace(line.amountInput.Value()) != "" {
		return false
	}
	return m.canBalanceAnyLine()
}

// balanceCurrentLine fills the current amount field with the value needed to make the total sum = 0
// Returns true if the line was successfully balanced
func (m *Model) balanceCurrentLine() bool {
	if m.form.focusedField != focusSectionAmount {
		return false
	}
	if line := m.currentLine(); line != nil {
		// Calculate what value makes the sum of all amounts equal to zero
		// New sum = (current remaining) - (current line amount) + X = 0
		// Therefore: X = -(remaining - current line amount)
		difference := m.form.remaining.Sub(lineAmount(line)).Neg()
		if difference.IsZero() {
			return false
		}
		line.amountInput.SetValue(difference.StringFixed(2))
		line.amountInput.CursorEnd()
		return true
	}
	return false
}

// balanceAnyLine finds the single unfilled amount field and fills it to balance the transaction
// Works regardless of which field is currently focused
// Returns true if a line was successfully balanced
func (m *Model) balanceAnyLine() bool {
	if !m.canBalanceAnyLine() {
		return false
	}

	// Find the unfilled line
	for i := range m.form.debitLines {
		line := &m.form.debitLines[i]
		if strings.TrimSpace(line.amountInput.Value()) == "" {
			// Calculate the balancing amount
			difference := m.form.remaining.Sub(lineAmount(line)).Neg()
			if difference.IsZero() {
				return false
			}
			line.amountInput.SetValue(difference.StringFixed(2))
			line.amountInput.CursorEnd()
			return true
		}
	}
	for i := range m.form.creditLines {
		line := &m.form.creditLines[i]
		if strings.TrimSpace(line.amountInput.Value()) == "" {
			// Calculate the balancing amount
			difference := m.form.remaining.Sub(lineAmount(line)).Neg()
			if difference.IsZero() {
				return false
			}
			line.amountInput.SetValue(difference.StringFixed(2))
			line.amountInput.CursorEnd()
			return true
		}
	}
	return false
}

// confirmTransaction validates and saves the current transaction to the batch
// Returns true if the transaction was successfully confirmed
func (m *Model) confirmTransaction() bool {
	date := m.form.date.time()
	if date.IsZero() {
		m.setStatus("Invalid date", statusError, statusShortDuration)
		m.form.focusedField = focusDate
		return false
	}
	if strings.TrimSpace(m.form.payeeInput.Value()) == "" {
		m.setStatus("Payee is required", statusError, statusShortDuration)
		m.form.focusedField = focusPayee
		m.form.payeeInput.Focus()
		return false
	}

	// Evaluate all amount fields before validation
	for i := range m.form.debitLines {
		_ = m.evaluateInput(&m.form.debitLines[i].amountInput)
	}
	for i := range m.form.creditLines {
		_ = m.evaluateInput(&m.form.creditLines[i].amountInput)
	}
	m.recalculateTotals()

	// Validate structure
	if len(m.form.debitLines) == 0 || len(m.form.creditLines) == 0 {
		m.setStatus("At least one debit and credit leg required", statusError, statusShortDuration)
		return false
	}
	if m.form.debitTotal.IsZero() || m.form.creditTotal.IsZero() {
		m.setStatus("Amounts required in both sections", statusError, statusShortDuration)
		return false
	}

	// Validate balance
	difference := m.form.debitTotal.Add(m.form.creditTotal).Abs()
	if difference.GreaterThan(decimal.NewFromFloat(balanceTolerance)) {
		m.setStatus("Transaction must balance (sum of all amounts = 0)", statusError, statusDuration)
		return false
	}

	// Build postings list
	// Debits and Credits are purely organizational - amounts stored as-entered with no sign manipulation
	postings := make([]core.Posting, 0, len(m.form.debitLines)+len(m.form.creditLines))
	for i := range m.form.debitLines {
		line := &m.form.debitLines[i]
		account := strings.TrimSpace(line.accountInput.Value())
		amount := lineAmount(line)
		if account == "" || amount.IsZero() {
			continue
		}
		postings = append(postings, core.Posting{
			Account: account,
			Amount:  amount.StringFixed(2),
			Comment: strings.TrimSpace(line.commentInput.Value()),
		})
	}
	for i := range m.form.creditLines {
		line := &m.form.creditLines[i]
		account := strings.TrimSpace(line.accountInput.Value())
		amount := lineAmount(line)
		if account == "" || amount.IsZero() {
			continue
		}
		postings = append(postings, core.Posting{
			Account: account,
			Amount:  amount.StringFixed(2),
			Comment: strings.TrimSpace(line.commentInput.Value()),
		})
	}

	if len(postings) < 2 {
		m.setStatus("Incomplete transaction", statusError, statusShortDuration)
		return false
	}

	// Create transaction
	tx := core.Transaction{
		Date:     date,
		Payee:    m.form.payeeInput.Value(),
		Comment:  strings.TrimSpace(m.form.commentInput.Value()),
		Cleared:  m.form.cleared,
		Postings: postings,
	}

	// Add or update transaction in batch
	wasEdit := m.editingIndex >= 0 && m.editingIndex < len(m.batch)
	if wasEdit {
		m.batch[m.editingIndex] = tx
	} else {
		m.batch = append(m.batch, tx)
	}

	// Sort batch by date and payee
	sort.SliceStable(m.batch, func(i, j int) bool {
		if m.batch[i].Date.Equal(m.batch[j].Date) {
			return m.batch[i].Payee < m.batch[j].Payee
		}
		return m.batch[i].Date.Before(m.batch[j].Date)
	})

	// Update cursor to the confirmed transaction
	m.cursor = m.findTransactionIndex(tx)

	// Save session
	if err := session.SaveBatch(m.batch); err != nil {
		m.setStatus(fmt.Sprintf("Saved but session write failed: %v", err), statusError, statusDuration)
	} else {
		action := "added"
		if wasEdit {
			action = "updated"
		}
		m.setStatus(fmt.Sprintf("Transaction %s (%d total)", action, len(m.batch)), statusSuccess, statusShortDuration)
	}

	m.lastDate = date
	m.resetForm(date)
	m.ensureBatchCursorVisible()
	m.currentView = viewBatch
	return true
}

// findTransactionIndex locates the index of a transaction in the batch
// Returns the last index if no exact match is found
func (m *Model) findTransactionIndex(tx core.Transaction) int {
	for i, candidate := range m.batch {
		if candidate.Date.Equal(tx.Date) && candidate.Payee == tx.Payee && len(candidate.Postings) == len(tx.Postings) {
			match := true
			for j := range candidate.Postings {
				if candidate.Postings[j] != tx.Postings[j] {
					match = false
					break
				}
			}
			if match {
				return i
			}
		}
	}
	return len(m.batch) - 1
}
