package tui

import (
	"os"
	"path/filepath"
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
				{Account: "Assets:Checking", Amount: "-50.00"},
				{Account: "Expenses:Food:Groceries", Amount: "50.00"},
			},
		},
		{
			Payee: "Fuel Station",
			Postings: []core.Posting{
				{Account: "Assets:Credit Card", Amount: "-40.00"},
				{Account: "Expenses:Auto:Gas", Amount: "40.00"},
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

func TestTransactionFlowAddsBatchEntry(t *testing.T) {
	db := testDB(t)

	tempDir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer func() {
		_ = os.Chdir(wd)
	}()
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

	// Start transaction entry
	send(keyRunes('n'))
	send(tea.KeyMsg{Type: tea.KeyTab})
	for _, r := range "New Payee" {
		send(keyRunes(r))
	}
	send(tea.KeyMsg{Type: tea.KeyTab})
	for _, r := range "100" {
		send(keyRunes(r))
	}
	send(tea.KeyMsg{Type: tea.KeyEnter})
	for _, r := range "Assets:Checking" {
		send(keyRunes(r))
	}
	send(tea.KeyMsg{Type: tea.KeyEnter})
	for _, r := range "Expenses:Food:Groceries" {
		send(keyRunes(r))
	}
	send(tea.KeyMsg{Type: tea.KeyTab})
	for _, r := range "100" {
		send(keyRunes(r))
	}
	// Move focus back to the date field so confirm key is active
	for i := 0; i < 5; i++ {
		send(tea.KeyMsg{Type: tea.KeyShiftTab})
	}
	send(keyRunes('c'))

	wait()
	program.Quit()
	<-done

	if runErr != nil {
		t.Fatalf("program run error: %v", runErr)
	}

	if len(model.batch) != 1 {
		t.Fatalf("expected 1 transaction in batch, got %d (status=%q, view=%d, focus=%d)", len(model.batch), model.statusMessage, model.currentView, model.form.focusedField)
	}

	tx := model.batch[0]
	if tx.Payee != "New Payee" {
		t.Fatalf("unexpected payee: %s", tx.Payee)
	}
	if len(tx.Postings) != 2 {
		t.Fatalf("expected 2 postings, got %d", len(tx.Postings))
	}
	if tx.Postings[0].Account != "Assets:Checking" {
		t.Fatalf("unexpected primary account: %s", tx.Postings[0].Account)
	}
	if tx.Postings[0].Amount != "-100.00" {
		t.Fatalf("unexpected primary amount: %s", tx.Postings[0].Amount)
	}
	if tx.Postings[1].Account != "Expenses:Food:Groceries" {
		t.Fatalf("unexpected allocation account: %s", tx.Postings[1].Account)
	}
	if tx.Postings[1].Amount != "100.00" {
		t.Fatalf("unexpected allocation amount: %s", tx.Postings[1].Amount)
	}
}

func TestBalanceShortcut(t *testing.T) {
	db := testDB(t)
	model := NewModel(db, "test-ledger.dat")
	model.startNewTransaction()
	model.form.headerConfirmed = true
	model.form.totalInput.SetValue("100.00")
	model.form.postings = []postingLine{
		newPostingLine(""),
		newPostingLine(""),
	}
	model.form.postings[0].amountInput.SetValue("60.00")
	model.form.focusedField = focusPostingAmount
	model.form.focusedPosting = 1
	model.recalculateRemaining()

	model.updateTransactionView(keyRunes('b'))

	got := model.form.postings[1].amountInput.Value()
	if got != "40.00" {
		t.Fatalf("expected balance fill to be 40.00, got %s", got)
	}
}

func TestTemplateSelectionAppliesAccounts(t *testing.T) {
	db := testDB(t)
	model := NewModel(db, "test-ledger.dat")
	model.startNewTransaction()
	model.form.headerConfirmed = true
	model.form.primaryInput.SetValue("Assets:Checking")
	model.templateOptions = []intelligence.TemplateRecord{{
		Accounts:  []string{"Assets:Checking", "Expenses:Rent", "Expenses:Utilities"},
		Frequency: 3,
	}}
	model.currentView = viewTemplate

	model.updateTemplateView(tea.KeyMsg{Type: tea.KeyEnter})

	if model.currentView != viewTransaction {
		t.Fatalf("expected to return to transaction view after applying template")
	}

	if len(model.form.postings) != 2 {
		t.Fatalf("expected 2 postings from template, got %d", len(model.form.postings))
	}
	accounts := []string{
		model.form.postings[0].accountInput.Value(),
		model.form.postings[1].accountInput.Value(),
	}
	expected := []string{"Expenses:Rent", "Expenses:Utilities"}
	for i, want := range expected {
		if accounts[i] != want {
			t.Fatalf("posting %d expected account %s, got %s", i, want, accounts[i])
		}
	}
}
