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

	db, report, err := intelligence.NewIntelligenceDB(transactions)
	if err != nil {
		t.Fatalf("failed to build intelligence db: %v", err)
	}
	if len(report.Issues) != 0 {
		t.Fatalf("unexpected build issues: %v", report.Issues)
	}
	return db
}

func wait() {
	time.Sleep(10 * time.Millisecond)
}

func TestBatchViewDisplaysLoadSummary(t *testing.T) {
	db := testDB(t)
	summary := core.LoadSummary{Transactions: 9, UniquePayees: 4, UniqueTemplates: 2}
	model := NewModel(db, "ledger.dat", summary)
	view := model.renderBatchView()
	if !strings.Contains(view, "Data load: 9 transactions • 4 payees • 2 templates") {
		t.Fatalf("batch view missing load summary: %q", view)
	}
	if !strings.Contains(view, "Load issues: none") {
		t.Fatalf("batch view missing zero-issue line: %q", view)
	}
}

func TestBatchViewDisplaysLoadIssues(t *testing.T) {
	db := testDB(t)
	summary := core.LoadSummary{
		Transactions:    9,
		UniquePayees:    4,
		UniqueTemplates: 2,
		Issues: []core.LoadIssue{
			{Stage: "parser", Message: "line 3: invalid amount"},
			{Stage: "intelligence", Message: "missing template"},
		},
	}
	model := NewModel(db, "ledger.dat", summary)
	view := model.renderBatchView()
	if !strings.Contains(view, "Load issues: 2 (first: [PARSER] line 3: invalid amount)") {
		t.Fatalf("batch view missing issue summary: %q", view)
	}
}

func TestNewTransactionHighlightsDaySegment(t *testing.T) {
	db := testDB(t)
	model := NewModel(db, "ledger.dat", core.LoadSummary{})

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
	model := NewModel(db, "ledger.dat", core.LoadSummary{})
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

	model := NewModel(db, ledgerPath, core.LoadSummary{})
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
	send(tea.KeyMsg{Type: tea.KeyTab}) // date -> cleared
	send(tea.KeyMsg{Type: tea.KeyTab}) // cleared -> payee
	for _, r := range "Grocery Store" {
		send(keyRunes(r))
	}
	send(tea.KeyMsg{Type: tea.KeyTab}) // payee -> comment
	send(tea.KeyMsg{Type: tea.KeyTab}) // comment -> template button
	send(tea.KeyMsg{Type: tea.KeyTab}) // template button -> debit account
	for _, r := range "Expenses:Food:Groceries" {
		send(keyRunes(r))
	}
	send(tea.KeyMsg{Type: tea.KeyTab}) // debit account -> amount
	for _, r := range "100" {
		send(keyRunes(r))
	}
	send(tea.KeyMsg{Type: tea.KeyTab}) // debit amount -> comment
	send(tea.KeyMsg{Type: tea.KeyTab}) // debit comment -> credit account
	for _, r := range "Assets:Checking" {
		send(keyRunes(r))
	}
	send(tea.KeyMsg{Type: tea.KeyTab}) // credit account -> amount field
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
	if !tx.Cleared {
		t.Fatalf("expected new transactions to default to cleared")
	}
	if tx.Comment != "" {
		t.Fatalf("expected transaction comment to be empty, got %q", tx.Comment)
	}
	if len(tx.Postings) != 2 {
		t.Fatalf("expected 2 postings, got %d", len(tx.Postings))
	}
	if tx.Postings[0].Account != "Expenses:Food:Groceries" || tx.Postings[0].Amount != "100.00" {
		t.Fatalf("unexpected debit posting: %+v", tx.Postings[0])
	}
	if tx.Postings[0].Comment != "" {
		t.Fatalf("expected debit comment to be empty, got %q", tx.Postings[0].Comment)
	}
	if tx.Postings[1].Account != "Assets:Checking" || tx.Postings[1].Amount != "-100.00" {
		t.Fatalf("unexpected credit posting: %+v", tx.Postings[1])
	}
	if tx.Postings[1].Comment != "" {
		t.Fatalf("expected credit comment to be empty, got %q", tx.Postings[1].Comment)
	}
}

func TestTransactionCapturesCommentsAndCleared(t *testing.T) {
	db := testDB(t)

	tempDir := t.TempDir()
	ledgerPath := filepath.Join(tempDir, "ledger.dat")
	if err := os.WriteFile(ledgerPath, []byte(""), 0644); err != nil {
		t.Fatalf("create ledger file: %v", err)
	}

	model := NewModel(db, ledgerPath, core.LoadSummary{})
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

	send(keyRunes('n'))                           // start new transaction
	send(tea.KeyMsg{Type: tea.KeyTab})            // date -> cleared
	send(tea.KeyMsg{Type: tea.KeySpace})          // toggle cleared off
	send(tea.KeyMsg{Type: tea.KeyTab})            // cleared -> payee
	for _, r := range "Acme Supplies" {
		send(keyRunes(r))
	}
	send(tea.KeyMsg{Type: tea.KeyTab})            // payee -> comment
	for _, r := range "Monthly restock" {
		send(keyRunes(r))
	}
	send(tea.KeyMsg{Type: tea.KeyTab})            // comment -> template button
	send(tea.KeyMsg{Type: tea.KeyTab})            // template -> debit account
	for _, r := range "Expenses:Office:Supplies" {
		send(keyRunes(r))
	}
	send(tea.KeyMsg{Type: tea.KeyTab})            // debit account -> amount
	for _, r := range "123.45" {
		send(keyRunes(r))
	}
	send(tea.KeyMsg{Type: tea.KeyTab})            // debit amount -> comment
	for _, r := range "Office restock" {
		send(keyRunes(r))
	}
	send(tea.KeyMsg{Type: tea.KeyTab})            // debit comment -> credit account
	for _, r := range "Assets:Checking" {
		send(keyRunes(r))
	}
	send(tea.KeyMsg{Type: tea.KeyTab})            // credit account -> amount
	send(keyRunes('b'))                           // balance amount
	send(tea.KeyMsg{Type: tea.KeyTab})            // credit amount -> comment
	for _, r := range "Paid via checking" {
		send(keyRunes(r))
	}
	send(tea.KeyMsg{Type: tea.KeyCtrlC})          // confirm transaction

	wait()
	program.Quit()
	<-done

	if runErr != nil {
		t.Fatalf("program run error: %v", runErr)
	}

	if len(model.batch) != 1 {
		t.Fatalf("expected 1 transaction in batch, got %d", len(model.batch))
	}

	tx := model.batch[0]
	if tx.Cleared {
		t.Fatalf("expected transaction to be marked uncleared")
	}
	if tx.Comment != "Monthly restock" {
		t.Fatalf("unexpected transaction comment: %q", tx.Comment)
	}
	if len(tx.Postings) != 2 {
		t.Fatalf("expected 2 postings, got %d", len(tx.Postings))
	}
	if tx.Postings[0].Comment != "Office restock" {
		t.Fatalf("unexpected debit comment: %q", tx.Postings[0].Comment)
	}
	if tx.Postings[1].Comment != "Paid via checking" {
		t.Fatalf("unexpected credit comment: %q", tx.Postings[1].Comment)
	}
}

func TestDeleteLineKeepsAtLeastOne(t *testing.T) {
	db := testDB(t)
	model := NewModel(db, "test-ledger.dat", core.LoadSummary{})
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
	model := NewModel(db, "ledger.dat", core.LoadSummary{})
	model.startNewTransaction()

	model.form.debitLines[0].amountInput.SetValue("120")
	model.recalculateTotals()
	model.focusSection(sectionCredit, 0, focusSectionAmount)

	model.updateTransactionView(keyRunes('b'))

	got := strings.TrimSpace(model.form.creditLines[0].amountInput.Value())
	if got != "-120.00" {
		t.Fatalf("expected credit amount to balance to -120.00 (to make sum = 0), got %s", got)
	}
}

func TestTemplateSelectionPopulatesSections(t *testing.T) {
	db := testDB(t)
	model := NewModel(db, "ledger.dat", core.LoadSummary{})
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
	model := NewModel(db, "ledger.dat", core.LoadSummary{})
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
	model := NewModel(db, "ledger.dat", core.LoadSummary{})

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
