package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"git.sr.ht/~jakintosh/teller/core"
	"git.sr.ht/~jakintosh/teller/intelligence"
	tea "github.com/charmbracelet/bubbletea"
)

func keyRunes(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

func testDB(t *testing.T) *intelligence.IntelligenceDB {
	t.Helper()
	transactions := []core.Transaction{
		{
			Payee: "Sample Market",
			Postings: []core.Posting{
				{Account: "Expenses:Food:Groceries", Amount: "50.00"},
				{Account: "Assets:Checking", Amount: "-50.00"},
			},
		},
		{
			Payee: "Fuel Station",
			Postings: []core.Posting{
				{Account: "Expenses:Auto:Gas", Amount: "40.00"},
				{Account: "Assets:Credit Card", Amount: "-40.00"},
			},
		},
	}

	db, err := intelligence.NewIntelligenceDB(transactions)
	if err != nil {
		t.Fatalf("failed to build intelligence db: %v", err)
	}
	return db
}

func wait() {
	time.Sleep(10 * time.Millisecond)
}

func TestNewTransactionHighlightsDaySegment(t *testing.T) {
	db := testDB(t)
	model := NewModel(db, "ledger.dat")

	if model.form.date.segment != dateSegmentDay {
		t.Fatalf("expected initial date segment to default to day, got %v", model.form.date.segment)
	}

	model.startNewTransaction()
	if model.form.date.segment != dateSegmentDay {
		t.Fatalf("expected new transaction date segment to be day, got %v", model.form.date.segment)
	}
}

func TestTransactionActionsStackedVertically(t *testing.T) {
	db := testDB(t)
	model := NewModel(db, "ledger.dat")
	model.startNewTransaction()

	view := model.renderTransactionView()
	actionsStart := strings.Index(view, "[tab]next")
	if actionsStart == -1 {
		t.Fatalf("failed to locate actions in transaction view: %q", view)
	}
	actions := view[actionsStart:]
	if !strings.Contains(actions, "\n[shift+tab]prev") {
		t.Fatalf("expected actions to be stacked vertically, got %q", actions)
	}
}

func TestTransactionFlowAddsBatchEntry(t *testing.T) {
	db := testDB(t)

	tempDir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer func() { _ = os.Chdir(wd) }()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	ledgerPath := filepath.Join(tempDir, "ledger.dat")
	if err := os.WriteFile(ledgerPath, []byte(""), 0644); err != nil {
		t.Fatalf("create ledger file: %v", err)
	}

	model := NewModel(db, ledgerPath)
	program := tea.NewProgram(model, tea.WithoutRenderer())

	done := make(chan struct{})
	var runErr error
	go func() {
		_, runErr = program.Run()
		close(done)
	}()

	wait()
	send := func(msg tea.KeyMsg) {
		program.Send(msg)
		wait()
	}

	send(keyRunes('n')) // start new transaction
	send(tea.KeyMsg{Type: tea.KeyTab})
	for _, r := range "Grocery Store" {
		send(keyRunes(r))
	}
	send(tea.KeyMsg{Type: tea.KeyTab}) // template button
	send(tea.KeyMsg{Type: tea.KeyTab}) // debit account
	for _, r := range "Expenses:Food:Groceries" {
		send(keyRunes(r))
	}
	send(tea.KeyMsg{Type: tea.KeyTab}) // debit amount
	for _, r := range "100" {
		send(keyRunes(r))
	}
	send(tea.KeyMsg{Type: tea.KeyTab}) // move to credit account
	for _, r := range "Assets:Checking" {
		send(keyRunes(r))
	}
	send(tea.KeyMsg{Type: tea.KeyTab}) // credit amount field
	send(keyRunes('b'))                // balance shortcut fills value
	send(tea.KeyMsg{Type: tea.KeyCtrlC})

	wait()
	program.Quit()
	<-done

	if runErr != nil {
		t.Fatalf("program run error: %v", runErr)
	}

	if len(model.batch) != 1 {
		t.Fatalf("expected 1 transaction in batch, got %d (status=%q)", len(model.batch), model.statusMessage)
	}

	tx := model.batch[0]
	if tx.Payee != "Grocery Store" {
		t.Fatalf("unexpected payee: %s", tx.Payee)
	}
	if len(tx.Postings) != 2 {
		t.Fatalf("expected 2 postings, got %d", len(tx.Postings))
	}
	if tx.Postings[0].Account != "Expenses:Food:Groceries" || tx.Postings[0].Amount != "100.00" {
		t.Fatalf("unexpected debit posting: %+v", tx.Postings[0])
	}
	if tx.Postings[1].Account != "Assets:Checking" || tx.Postings[1].Amount != "-100.00" {
		t.Fatalf("unexpected credit posting: %+v", tx.Postings[1])
	}
}

func TestDeleteLineKeepsAtLeastOne(t *testing.T) {
	db := testDB(t)
	model := NewModel(db, "test-ledger.dat")
	model.startNewTransaction()
	model.addLine(sectionDebit, false)
	model.focusSection(sectionDebit, 1, focusSectionAccount)

	model.updateTransactionView(tea.KeyMsg{Type: tea.KeyCtrlD})

	if len(model.form.debitLines) != 1 {
		t.Fatalf("expected debit lines to reduce to 1, got %d", len(model.form.debitLines))
	}
}

func TestBalanceShortcutFillsCreditDifference(t *testing.T) {
	db := testDB(t)
	model := NewModel(db, "ledger.dat")
	model.startNewTransaction()

	model.form.debitLines[0].amountInput.SetValue("120")
	model.recalculateTotals()
	model.focusSection(sectionCredit, 0, focusSectionAmount)

	model.updateTransactionView(keyRunes('b'))

	got := strings.TrimSpace(model.form.creditLines[0].amountInput.Value())
	if got != "120.00" {
		t.Fatalf("expected credit amount to balance to 120.00, got %s", got)
	}
}

func TestTemplateSelectionPopulatesSections(t *testing.T) {
	db := testDB(t)
	model := NewModel(db, "ledger.dat")
	model.startNewTransaction()
	model.templateOptions = []intelligence.TemplateRecord{{
		DebitAccounts:  []string{"Expenses:Rent", "Expenses:Utilities"},
		CreditAccounts: []string{"Assets:Checking"},
		Frequency:      5,
	}}
	model.templateOffset = 0
	model.currentView = viewTemplate

	model.updateTemplateView(tea.KeyMsg{Type: tea.KeyEnter})

	if len(model.form.debitLines) != 2 {
		t.Fatalf("expected 2 debit lines, got %d", len(model.form.debitLines))
	}
	if len(model.form.creditLines) != 1 {
		t.Fatalf("expected 1 credit line, got %d", len(model.form.creditLines))
	}
	if model.form.debitLines[0].accountInput.Value() != "Expenses:Rent" {
		t.Fatalf("unexpected first debit account: %s", model.form.debitLines[0].accountInput.Value())
	}
	if model.form.creditLines[0].accountInput.Value() != "Assets:Checking" {
		t.Fatalf("unexpected credit account: %s", model.form.creditLines[0].accountInput.Value())
	}
}

func TestTemplateViewDisplaysAccountsVertically(t *testing.T) {
	db := testDB(t)
	model := NewModel(db, "ledger.dat")
	model.templateOptions = []intelligence.TemplateRecord{{
		DebitAccounts:  []string{"Expenses:Rent", "Expenses:Utilities"},
		CreditAccounts: []string{"Assets:Checking"},
		Frequency:      2,
	}}
	model.templateCursor = 0
	model.templateOffset = 0

	view := model.renderTemplateView()
	if !strings.Contains(view, "\n    Debit Accounts:\n      Expenses:Rent\n      Expenses:Utilities\n") {
		t.Fatalf("expected debit accounts to be listed vertically, got %q", view)
	}
	if !strings.Contains(view, "\n    Credit Accounts:\n      Assets:Checking\n") {
		t.Fatalf("expected credit accounts to be listed vertically, got %q", view)
	}
	if !strings.Contains(view, "\n[enter]apply\n[esc]skip") {
		t.Fatalf("expected template actions to be vertical, got %q", view)
	}
}

func TestTemplateViewScrollsWithCursor(t *testing.T) {
	db := testDB(t)
	model := NewModel(db, "ledger.dat")

	for i := 0; i < maxTemplateDisplay+1; i++ {
		model.templateOptions = append(model.templateOptions, intelligence.TemplateRecord{
			DebitAccounts:  []string{fmt.Sprintf("Expenses:Category:%d", i+1)},
			CreditAccounts: []string{"Assets:Checking"},
			Frequency:      i + 1,
		})
	}
	model.templateCursor = 0
	model.templateOffset = 0
	model.ensureTemplateCursorVisible()

	view := model.renderTemplateView()
	if strings.Contains(view, fmt.Sprintf("%d.", maxTemplateDisplay+1)) {
		t.Fatalf("expected initial template view to exclude item %d, got %q", maxTemplateDisplay+1, view)
	}

	for i := 0; i < maxTemplateDisplay; i++ {
		model.updateTemplateView(keyRunes('j'))
	}

	view = model.renderTemplateView()
	if !strings.Contains(view, fmt.Sprintf("%d.", maxTemplateDisplay+1)) {
		t.Fatalf("expected template view to include item %d after scrolling, got %q", maxTemplateDisplay+1, view)
	}
	if strings.Contains(view, "1.") {
		t.Fatalf("expected template view to scroll past first item, got %q", view)
	}
}
