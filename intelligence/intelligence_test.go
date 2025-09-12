package intelligence_test

import (
	"git.sr.ht/~jakintosh/teller/core"
	"git.sr.ht/~jakintosh/teller/intelligence"
	"reflect"
	"sort"
	"testing"
	"time"
)

func TestNewIntelligenceDB(t *testing.T) {
	transactions := []core.Transaction{
		{
			Date:  time.Now(),
			Payee: "Coffee Shop",
			Postings: []core.Posting{
				{Account: "Expenses:Food:Coffee"},
				{Account: "Assets:Checking"},
			},
		},
		{
			Date:  time.Now(),
			Payee: "Grocery Store",
			Postings: []core.Posting{
				{Account: "Expenses:Food:Groceries"},
				{Account: "Assets:Credit Card"},
			},
		},
		{Date: time.Now(), Payee: "Coffee Shop"}, // Duplicate Payee
		{
			Date:  time.Now(),
			Payee: "Bookstore",
			Postings: []core.Posting{
				{Account: "Expenses:Books"},
				{Account: "Assets:Checking"},
			},
		},
		{Date: time.Now(), Payee: ""}, // Empty payee
	}

	db := intelligence.New(transactions)

	expectedPayees := []string{"Bookstore", "Coffee Shop", "Grocery Store"}
	if !reflect.DeepEqual(db.Payees, expectedPayees) {
		t.Errorf("expected payees %v, but got %v", expectedPayees, db.Payees)
	}

	expectedAccounts := []string{
		"Assets:Checking",
		"Assets:Credit Card",
		"Expenses:Books",
		"Expenses:Food:Coffee",
		"Expenses:Food:Groceries",
	}
	allAccounts := db.Accounts.Find("")
	sort.Strings(allAccounts)
	if !reflect.DeepEqual(allAccounts, expectedAccounts) {
		t.Errorf("expected accounts %v, but got %v", expectedAccounts, allAccounts)
	}
}

func TestFindPayees(t *testing.T) {
	db := &intelligence.IntelligenceDB{
		Payees: []string{"Bookstore", "Coffee Shop", "Grocery Store"},
	}

	testCases := []struct {
		prefix   string
		expected []string
	}{
		{"coff", []string{"Coffee Shop"}},
		{"Cof", []string{"Coffee Shop"}}, // Case-insensitivity
		{"gro", []string{"Grocery Store"}},
		{"b", []string{"Bookstore"}},
		{"z", []string{}}, // No match
		{"", []string{}},   // Empty prefix
	}

	for _, tc := range testCases {
		t.Run(tc.prefix, func(t *testing.T) {
			results := db.FindPayees(tc.prefix)
			if !reflect.DeepEqual(results, tc.expected) {
				t.Errorf("for prefix '%s', expected %v, but got %v", tc.prefix, tc.expected, results)
			}
		})
	}
}

func TestFindAccounts(t *testing.T) {
	trie := intelligence.NewTrie()
	accounts := []string{
		"Expenses:Food:Groceries",
		"Expenses:Food:Coffee",
		"Expenses:Books",
		"Assets:Checking",
		"Assets:Credit Card",
	}
	for _, acc := range accounts {
		trie.Insert(acc)
	}
	db := &intelligence.IntelligenceDB{Accounts: trie}

	testCases := []struct {
		name     string
		prefix   string
		expected []string
	}{
		{"Partial match", "Expenses:Food", []string{"Expenses:Food:Coffee", "Expenses:Food:Groceries"}},
		{"Full match", "Expenses:Books", []string{"Expenses:Books"}},
		{"Case-insensitive", "assets:c", []string{"Assets:Checking", "Assets:Credit Card"}},
		{"No match", "Liabilities", []string{}},
		{"Empty prefix", "", []string{}},
		{"Root level", "Expenses", []string{"Expenses:Books", "Expenses:Food:Coffee", "Expenses:Food:Groceries"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			results := db.FindAccounts(tc.prefix)
			sort.Strings(results)
			sort.Strings(tc.expected)
			if !reflect.DeepEqual(results, tc.expected) {
				t.Errorf("for prefix '%s', expected %v, but got %v", tc.prefix, tc.expected, results)
			}
		})
	}
}
