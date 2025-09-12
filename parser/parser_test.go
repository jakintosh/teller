package parser

import (
	"reflect"
	"testing"
	"time"

	"git.sr.ht/~jakintosh/teller/core"
)

func TestParseFile(t *testing.T) {
	// Define the expected structure based on sample.ledger
	expectedTransactions := []core.Transaction{
		{
			Date:  time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC),
			Payee: "Opening Balance",
			Postings: []core.Posting{
				{Account: "Assets:Checking", Amount: "$1000.00"},
				{Account: "Equity:Opening Balances", Amount: ""},
			},
		},
		{
			Date:  time.Date(2024, 5, 10, 0, 0, 0, 0, time.UTC),
			Payee: "Coffee Shop",
			Postings: []core.Posting{
				{Account: "Expenses:Food:Coffee", Amount: "$5.50"},
				{Account: "Assets:Checking", Amount: ""},
			},
		},
		{
			Date:  time.Date(2024, 5, 12, 0, 0, 0, 0, time.UTC),
			Payee: "Supermarket",
			Postings: []core.Posting{
				{Account: "Expenses:Groceries", Amount: "$125.45"},
				{Account: "Expenses:Household", Amount: "$30.00"},
				{Account: "Liabilities:Credit Card", Amount: "$-155.45"},
			},
		},
		{
			Date:  time.Date(2024, 5, 15, 0, 0, 0, 0, time.UTC),
			Payee: "Gas Station",
			Postings: []core.Posting{
				{Account: "Expenses:Auto:Gas", Amount: "$45.00"},
				{Account: "Assets:Checking", Amount: ""},
			},
		},
		{
			Date:  time.Date(2024, 5, 20, 0, 0, 0, 0, time.UTC),
			Payee: "Internet Bill",
			Postings: []core.Posting{
				{Account: "Expenses:Utilities:Internet", Amount: "$60.00"},
				{Account: "Assets:Checking", Amount: ""},
			},
		},
		{
			Date:  time.Date(2024, 5, 21, 0, 0, 0, 0, time.UTC),
			Payee: "Paycheck",
			Postings: []core.Posting{
				{Account: "Assets:Checking", Amount: "$2000.00"},
				{Account: "Income:Salary", Amount: "$-2000.00"},
			},
		},
	}

	// Call the parser
	actualTransactions, err := ParseFile("sample.ledger")
	if err != nil {
		t.Fatalf("ParseFile() returned an unexpected error: %v", err)
	}

	// Check if the number of transactions is correct
	if len(actualTransactions) != len(expectedTransactions) {
		t.Fatalf("Expected %d transactions, but got %d", len(expectedTransactions), len(actualTransactions))
	}

	// Deep compare each transaction
	for i := range expectedTransactions {
		if !reflect.DeepEqual(expectedTransactions[i], actualTransactions[i]) {
			t.Errorf("Transaction %d does not match.\nExpected: %+v\nGot:      %+v", i, expectedTransactions[i], actualTransactions[i])
		}
	}
}
