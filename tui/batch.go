package tui

import (
	"fmt"
	"os"
	"sort"
	"time"

	"git.sr.ht/~jakintosh/teller/core"
	"github.com/shopspring/decimal"
)

// SetBatch replaces the current batch with a new set of transactions
func (m *Model) SetBatch(batch []core.Transaction) {
	m.batch = append([]core.Transaction(nil), batch...)
	sort.Slice(m.batch, func(i, j int) bool { return m.batch[i].Date.Before(m.batch[j].Date) })
	if len(m.batch) == 0 {
		m.cursor = 0
		m.lastDate = time.Time{}
		return
	}
	m.cursor = len(m.batch) - 1
	m.lastDate = m.batch[m.cursor].Date
}

// resetForm clears the transaction form and initializes it with the given base date
func (m *Model) resetForm(baseDate time.Time) {
	m.form = newTransactionForm(baseDate)
	m.templateOptions = nil
	m.templateCursor = 0
	m.templateOffset = 0
	m.templatePayee = ""
	m.editingIndex = -1
}

// defaultDate returns the appropriate default date for a new transaction
func (m *Model) defaultDate() time.Time {
	if !m.lastDate.IsZero() {
		return m.lastDate
	}
	if len(m.batch) > 0 {
		return m.batch[len(m.batch)-1].Date
	}
	return time.Now()
}

// startNewTransaction initializes a new transaction form and switches to the transaction view
func (m *Model) startNewTransaction() {
	m.resetForm(m.defaultDate())
	m.currentView = viewTransaction
}

// startEditingTransaction loads an existing transaction into the form for editing
func (m *Model) startEditingTransaction(index int) {
	if index < 0 || index >= len(m.batch) {
		return
	}
	tx := m.batch[index]
	m.resetForm(tx.Date)
	m.editingIndex = index
	m.form.cleared = tx.Cleared
	m.form.payeeInput.SetValue(tx.Payee)
	m.form.payeeInput.CursorEnd()
	m.form.commentInput.SetValue(tx.Comment)
	m.form.commentInput.CursorEnd()
	m.refreshTemplateOptions()

	m.form.debitLines = nil
	m.form.creditLines = nil
	for _, posting := range tx.Postings {
		amount, err := decimal.NewFromString(posting.Amount)
		if err != nil {
			continue
		}
		line := newPostingLine()
		line.accountInput.SetValue(posting.Account)
		line.accountInput.CursorEnd()
		line.amountInput.SetValue(amount.StringFixed(2))
		line.amountInput.CursorEnd()
		line.commentInput.SetValue(posting.Comment)
		line.commentInput.CursorEnd()
		if amount.Sign() >= 0 {
			m.form.debitLines = append(m.form.debitLines, line)
		} else {
			m.form.creditLines = append(m.form.creditLines, line)
		}
	}
	if len(m.form.debitLines) == 0 {
		m.form.debitLines = []postingLine{newPostingLine()}
	}
	if len(m.form.creditLines) == 0 {
		m.form.creditLines = []postingLine{newPostingLine()}
	}
	m.recalculateTotals()
	m.focusSection(sectionCredit, 0, focusSectionAccount)
	m.currentView = viewTransaction
}

// cancelTransaction cancels the current transaction and returns to the batch view
func (m *Model) cancelTransaction() {
	m.resetForm(m.defaultDate())
	m.currentView = viewBatch
}

// openConfirm switches to the confirmation view for the specified action
func (m *Model) openConfirm(kind confirmKind) {
	m.pendingConfirm = kind
	m.currentView = viewConfirm
}

// writeTransactionsToLedger appends all batch transactions to the ledger file
func (m *Model) writeTransactionsToLedger() error {
	if len(m.batch) == 0 {
		return fmt.Errorf("no transactions to write")
	}
	file, err := os.OpenFile(m.ledgerFilePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open ledger: %w", err)
	}
	defer file.Close()
	for _, tx := range m.batch {
		if _, err := file.WriteString("\n"); err != nil {
			return fmt.Errorf("write separator: %w", err)
		}
		if _, err := file.WriteString(tx.String()); err != nil {
			return fmt.Errorf("write transaction: %w", err)
		}
	}
	return nil
}

// setStatus sets a temporary status message with the given duration
func (m *Model) setStatus(message string, duration time.Duration) {
	m.statusMessage = message
	m.statusExpiry = time.Now().Add(duration)
}

// statusLine returns the current status message if it hasn't expired
func (m *Model) statusLine() string {
	if m.statusMessage == "" {
		return ""
	}
	if !m.statusExpiry.IsZero() && time.Now().After(m.statusExpiry) {
		return ""
	}
	return m.statusMessage
}

// refreshAfterLoad updates the UI after loading data
func (m *Model) refreshAfterLoad() {
	m.recalculateTotals()
	if m.currentView == viewTransaction {
		m.refreshSuggestions()
	}
}
