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
	// Check that header line starts correctly and contains the comment
	if !strings.HasPrefix(lines[0], "2025/02/10 * Acme Co\t") {
		t.Fatalf("unexpected header line start: %q", lines[0])
	}
	if !strings.Contains(lines[0], "; Invoice 123") {
		t.Fatalf("expected header line to contain comment, got %q", lines[0])
	}
	// Check that posting lines use tabs and contain comments appropriately
	if !strings.HasPrefix(lines[1], "\t") {
		t.Fatalf("expected posting line to start with tab, got %q", lines[1])
	}
	if !strings.Contains(lines[1], "; Supplies") {
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

func TestTransactionStringExactFormat(t *testing.T) {
	tx := Transaction{
		Date:    time.Date(2025, time.January, 15, 0, 0, 0, 0, time.UTC),
		Payee:   "Test Store",
		Comment: "monthly supplies",
		Cleared: true,
		Postings: []Posting{
			{Account: "Expenses:Office", Amount: "100.99", Comment: "pens"},
			{Account: "Expenses:Food", Amount: "1.00"},
			{Account: "Assets:Checking", Amount: "-101.99"},
		},
	}

	expected := "2025/01/15 * Test Store\t\t\t\t\t\t; monthly supplies\n" +
		"\tExpenses:Office\t\t\t\t\t\t\t$  100.99\t; pens\n" +
		"\tExpenses:Food\t\t\t\t\t\t\t$    1.00\n" +
		"\tAssets:Checking\t\t\t\t\t\t\t$ -101.99\n"

	actual := tx.String()
	if actual != expected {
		t.Fatalf("transaction format mismatch\nExpected:\n%q\n\nActual:\n%q", expected, actual)
	}
}

func TestTransactionStringAmountAlignment(t *testing.T) {
	tx := Transaction{
		Date:    time.Date(2025, time.April, 15, 0, 0, 0, 0, time.UTC),
		Payee:   "Test Payee",
		Cleared: true,
		Postings: []Posting{
			{Account: "Expenses:Food", Amount: "100.99"},
			{Account: "Expenses:Travel", Amount: "1.00"},
			{Account: "Assets:Savings", Amount: "-4999.00"},
			{Account: "Assets:Checking", Amount: "4897.01"},
		},
	}

	out := tx.String()
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")

	// Extract just the amount portions from each posting line
	// We expect amounts like: "$ -4999.00", "$   100.99", "$     1.00", "$  4897.01"
	for i := 1; i < len(lines); i++ {
		if !strings.Contains(lines[i], "$ ") {
			t.Fatalf("expected posting line to contain '$ ', got %q", lines[i])
		}
	}

	// Helper function to calculate column position accounting for tabs
	calcColumn := func(s string) int {
		col := 0
		for _, ch := range s {
			if ch == '\t' {
				col = ((col / 4) + 1) * 4
			} else {
				col++
			}
		}
		return col
	}

	// Find the decimal point column positions in each amount
	var decimalColumns []int
	for i := 1; i < len(lines); i++ {
		line := lines[i]
		decimalIdx := strings.Index(line, ".")
		if decimalIdx == -1 {
			continue
		}
		// Calculate column position up to the decimal point
		decimalCol := calcColumn(line[:decimalIdx])
		decimalColumns = append(decimalColumns, decimalCol)
	}

	// All decimal points should be at the same column
	if len(decimalColumns) > 0 {
		firstCol := decimalColumns[0]
		for i, col := range decimalColumns {
			if col != firstCol {
				t.Fatalf("decimal points not aligned: line %d has decimal at column %d, expected %d\nFull output:\n%s",
					i+1, col, firstCol, out)
			}
		}
	}

	// Verify the '-' sign appears immediately before digits
	for i := 1; i < len(lines); i++ {
		line := lines[i]
		if strings.Contains(line, "-") {
			// Find "$ " and check that there's no space between "-" and the digit
			dollarIdx := strings.Index(line, "$ ")
			if dollarIdx != -1 {
				amountPart := line[dollarIdx+2:] // Skip "$ "
				minusIdx := strings.Index(amountPart, "-")
				if minusIdx != -1 && minusIdx+1 < len(amountPart) {
					nextChar := amountPart[minusIdx+1]
					if nextChar < '0' || nextChar > '9' {
						t.Fatalf("expected digit immediately after '-', got %q in line: %q", nextChar, line)
					}
				}
			}
		}
	}
}
