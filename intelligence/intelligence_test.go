package intelligence_test

import (
	"git.sr.ht/~jakintosh/teller/core"
	"git.sr.ht/~jakintosh/teller/intelligence"
	"reflect"
	"testing"
	"time"
)

func TestNewIntelligenceDB(t *testing.T) {
	transactions := []core.Transaction{
		{Date: time.Now(), Payee: "Coffee Shop"},
		{Date: time.Now(), Payee: "Grocery Store"},
		{Date: time.Now(), Payee: "Coffee Shop"}, // Duplicate
		{Date: time.Now(), Payee: "Bookstore"},
		{Date: time.Now(), Payee: ""}, // Empty payee
	}

	db := intelligence.New(transactions)

	expectedPayees := []string{"Bookstore", "Coffee Shop", "Grocery Store"}
	if !reflect.DeepEqual(db.Payees, expectedPayees) {
		t.Errorf("expected payees %v, but got %v", expectedPayees, db.Payees)
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
