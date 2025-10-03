package parser

import (
	"testing"
	"time"
)

func TestParseFile(t *testing.T) {
	transactions, err := ParseFile("../sample.ledger")
	if err != nil {
		t.Fatalf("Failed to parse sample ledger file: %v", err)
	}

	// Check we got the expected number of transactions
	expectedCount := 9
	if len(transactions) != expectedCount {
		t.Errorf("Expected %d transactions, got %d", expectedCount, len(transactions))
	}

	// Test first transaction
	if len(transactions) > 0 {
		tx := transactions[0]
		expectedDate := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		if !tx.Date.Equal(expectedDate) {
			t.Errorf("Expected date %v, got %v", expectedDate, tx.Date)
		}

		if tx.Payee != "Super Grocery Store" {
			t.Errorf("Expected payee 'Super Grocery Store', got '%s'", tx.Payee)
		}

		if len(tx.Postings) != 2 {
			t.Errorf("Expected 2 postings, got %d", len(tx.Postings))
		}

		if len(tx.Postings) >= 2 {
			if tx.Postings[0].Account != "Expenses:Food:Groceries" {
				t.Errorf("Expected first account 'Expenses:Food:Groceries', got '%s'", tx.Postings[0].Account)
			}
			if tx.Postings[0].Amount != "85.42" {
				t.Errorf("Expected first amount '85.42', got '%s'", tx.Postings[0].Amount)
			}
		}
	}

	// Test transaction with different date format (slash separated)
	if len(transactions) > 2 {
		tx := transactions[2]
		expectedDate := time.Date(2025, 1, 17, 0, 0, 0, 0, time.UTC)
		if !tx.Date.Equal(expectedDate) {
			t.Errorf("Expected date %v for slash-separated date, got %v", expectedDate, tx.Date)
		}

		if tx.Payee != "City Market" {
			t.Errorf("Expected payee 'City Market', got '%s'", tx.Payee)
		}
	}

	// Test multi-split transaction (Fine Dining Restaurant is transaction 7, index 6)
	if len(transactions) > 6 {
		tx := transactions[6]
		if tx.Payee != "Fine Dining Restaurant" {
			t.Errorf("Expected payee 'Fine Dining Restaurant', got '%s'", tx.Payee)
		}

		expectedPostings := 4
		if len(tx.Postings) != expectedPostings {
			t.Errorf("Expected %d postings for multi-split transaction, got %d", expectedPostings, len(tx.Postings))
		}
	}
}