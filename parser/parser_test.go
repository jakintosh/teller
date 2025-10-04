package parser

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseFile(t *testing.T) {
	result, err := ParseFile("../sample.ledger")
	if err != nil {
		t.Fatalf("Failed to parse sample ledger file: %v", err)
	}

	transactions := result.Transactions

	if len(result.Issues) != 0 {
		t.Fatalf("expected no parsing issues for sample ledger, got %d: %+v", len(result.Issues), result.Issues)
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

		if !tx.Cleared {
			t.Errorf("expected first transaction to be cleared")
		}

		if tx.Comment != "Weekly groceries run" {
			t.Errorf("unexpected transaction comment: %q", tx.Comment)
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
			if tx.Postings[0].Comment != "Pantry staples" {
				t.Errorf("unexpected posting comment: %q", tx.Postings[0].Comment)
			}
			if tx.Postings[1].Comment != "" {
				t.Errorf("expected second posting comment to be empty, got %q", tx.Postings[1].Comment)
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

	// Test uncleared transaction
	if len(transactions) > 5 {
		tx := transactions[5]
		if tx.Payee != "Online Purchase" {
			t.Fatalf("expected sixth transaction to be Online Purchase, got %s", tx.Payee)
		}
		if tx.Cleared {
			t.Errorf("expected Online Purchase to be not cleared")
		}
		if tx.Comment != "Awaiting shipment" {
			t.Errorf("unexpected comment on Online Purchase: %q", tx.Comment)
		}
	}
}

func TestParsePostingAmountWithCurrencyBeforeSign(t *testing.T) {
	ledger := "2024/08/15 * City Market\n" +
		"    Expenses:Food:Groceries\n" +
		"    Expenses:Food:Alcohol        $14.10\n" +
		"    Liabilities:Apple Card       $-91.41\n"

	dir := t.TempDir()
	path := filepath.Join(dir, "city-market.ledger")
	if err := os.WriteFile(path, []byte(ledger), 0o600); err != nil {
		t.Fatalf("failed to write temp ledger: %v", err)
	}

	result, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile returned error: %v", err)
	}

	if len(result.Issues) != 0 {
		t.Fatalf("expected no parse issues, got %d: %+v", len(result.Issues), result.Issues)
	}

	if len(result.Transactions) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(result.Transactions))
	}

	tx := result.Transactions[0]
	if len(tx.Postings) != 3 {
		t.Fatalf("expected 3 postings, got %d", len(tx.Postings))
	}

	if tx.Postings[0].Amount != "" {
		t.Errorf("expected first posting amount to be elided, got %q", tx.Postings[0].Amount)
	}
	if tx.Postings[1].Amount != "14.10" {
		t.Errorf("expected second posting amount to be 14.10, got %q", tx.Postings[1].Amount)
	}
	if tx.Postings[2].Amount != "-91.41" {
		t.Errorf("expected third posting amount to be -91.41, got %q", tx.Postings[2].Amount)
	}
}

func TestParseAmountWithWhitespace(t *testing.T) {
	tests := []struct {
		name           string
		ledgerContent  string
		expectedAmount string
	}{
		{
			name: "dollar sign with space before digits",
			ledgerContent: "2025/01/01 * Test\n" +
				"    Expenses:Test  $ 123.45\n" +
				"    Assets:Cash\n",
			expectedAmount: "123.45",
		},
		{
			name: "dollar sign without space",
			ledgerContent: "2025/01/01 * Test\n" +
				"    Expenses:Test  $123.45\n" +
				"    Assets:Cash\n",
			expectedAmount: "123.45",
		},
		{
			name: "negative sign before dollar sign with space",
			ledgerContent: "2025/01/01 * Test\n" +
				"    Expenses:Test  -$ 123.45\n" +
				"    Assets:Cash\n",
			expectedAmount: "-123.45",
		},
		{
			name: "dollar sign before negative sign no space",
			ledgerContent: "2025/01/01 * Test\n" +
				"    Expenses:Test  $-123.45\n" +
				"    Assets:Cash\n",
			expectedAmount: "-123.45",
		},
		{
			name: "dollar sign with space before negative sign and digits",
			ledgerContent: "2025/01/01 * Test\n" +
				"    Expenses:Test  $ -123.45\n" +
				"    Assets:Cash\n",
			expectedAmount: "-123.45",
		},
		{
			name: "plain negative amount no dollar sign",
			ledgerContent: "2025/01/01 * Test\n" +
				"    Expenses:Test  -123.45\n" +
				"    Assets:Cash\n",
			expectedAmount: "-123.45",
		},
		{
			name: "plain positive amount no dollar sign",
			ledgerContent: "2025/01/01 * Test\n" +
				"    Expenses:Test  123.45\n" +
				"    Assets:Cash\n",
			expectedAmount: "123.45",
		},
		{
			name: "plus sign with dollar",
			ledgerContent: "2025/01/01 * Test\n" +
				"    Expenses:Test  +$123.45\n" +
				"    Assets:Cash\n",
			expectedAmount: "+123.45",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "test.ledger")
			if err := os.WriteFile(path, []byte(tt.ledgerContent), 0o600); err != nil {
				t.Fatalf("failed to write temp ledger: %v", err)
			}

			result, err := ParseFile(path)
			if err != nil {
				t.Fatalf("ParseFile returned error: %v", err)
			}

			if len(result.Issues) != 0 {
				t.Fatalf("expected no parse issues, got %d: %+v", len(result.Issues), result.Issues)
			}

			if len(result.Transactions) != 1 {
				t.Fatalf("expected 1 transaction, got %d", len(result.Transactions))
			}

			tx := result.Transactions[0]
			if len(tx.Postings) < 1 {
				t.Fatalf("expected at least 1 posting, got %d", len(tx.Postings))
			}

			if tx.Postings[0].Amount != tt.expectedAmount {
				t.Errorf("expected amount %q, got %q", tt.expectedAmount, tx.Postings[0].Amount)
			}
		})
	}
}

func TestParseInvalidAmountsWithWhitespace(t *testing.T) {
	tests := []struct {
		name          string
		ledgerContent string
		shouldFail    bool
		description   string
	}{
		{
			name: "space between negative sign and digits after dollar",
			ledgerContent: "2025/01/01 * Test\n" +
				"    Expenses:Test  $- 123.45\n" +
				"    Assets:Cash\n",
			shouldFail:  true,
			description: "space between - and digits is invalid",
		},
		{
			name: "space within digits",
			ledgerContent: "2025/01/01 * Test\n" +
				"    Expenses:Test  $12 3.45\n" +
				"    Assets:Cash\n",
			shouldFail:  true,
			description: "space within digits is invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "test.ledger")
			if err := os.WriteFile(path, []byte(tt.ledgerContent), 0o600); err != nil {
				t.Fatalf("failed to write temp ledger: %v", err)
			}

			result, err := ParseFile(path)
			if err != nil {
				t.Fatalf("ParseFile returned error: %v", err)
			}

			// For invalid amounts, the parser should either reject the amount
			// or treat the entire line as an account name
			tx := result.Transactions[0]
			if len(tx.Postings) > 0 {
				// If parsed as account with amount, the amount should be empty
				// or it should be reported as an issue
				if tx.Postings[0].Amount != "" && len(result.Issues) == 0 {
					t.Errorf("expected invalid amount to be rejected or treated as account, but got amount: %q", tx.Postings[0].Amount)
				}
			}
		})
	}
}
