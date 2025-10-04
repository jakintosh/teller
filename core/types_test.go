package core

import (
	"strings"
	"testing"
	"time"
)

func TestTransactionStringIncludesCommentsAndCleared(t *testing.T) {
	tx := Transaction{
		Date:    time.Date(2025, time.February, 10, 0, 0, 0, 0, time.UTC),
		Payee:   "Acme Co",
		Comment: "Invoice 123",
		Cleared: true,
		Postings: []Posting{
			{Account: "Expenses:Office", Amount: "100.00", Comment: "Supplies"},
			{Account: "Assets:Checking", Amount: "-100.00"},
		},
	}

	out := tx.String()
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if len(lines) < 3 {
		t.Fatalf("expected at least 3 lines, got %d: %q", len(lines), out)
	}
	if lines[0] != "2025/02/10 * Acme Co  ; Invoice 123" {
		t.Fatalf("unexpected header line: %q", lines[0])
	}
	if !strings.Contains(lines[1], "  ; Supplies") {
		t.Fatalf("expected debit line to contain comment, got %q", lines[1])
	}
	if strings.Contains(lines[2], ";") {
		t.Fatalf("expected credit line without comment, got %q", lines[2])
	}
}

func TestTransactionStringNotClearedSpacing(t *testing.T) {
	tx := Transaction{
		Date:  time.Date(2025, time.March, 5, 0, 0, 0, 0, time.UTC),
		Payee: "Pending Payee",
		Postings: []Posting{
			{Account: "Assets:Checking", Amount: "-50.00"},
			{Account: "Expenses:Misc", Amount: "50.00"},
		},
	}

	out := tx.String()
	firstLine := strings.Split(out, "\n")[0]
	if firstLine != "2025/03/05   Pending Payee" {
		t.Fatalf("expected uncleared header with triple spaces, got %q", firstLine)
	}
}
