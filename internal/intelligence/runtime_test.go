package intelligence

import (
	"testing"
	"time"

	"git.sr.ht/~jakintosh/teller/internal/core"
)

func TestRuntimeBuildFromBatch(t *testing.T) {
	// Create mock transactions
	transactions := []core.Transaction{
		{
			Date:  time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
			Payee: "Brand New Payee",
			Postings: []core.Posting{
				{Account: "Expenses:Food", Amount: "50.00"},
				{Account: "Assets:Checking", Amount: "-50.00"},
			},
		},
		{
			Date:  time.Date(2025, 1, 16, 0, 0, 0, 0, time.UTC),
			Payee: "Brand New Payee", // Same payee, same template
			Postings: []core.Posting{
				{Account: "Expenses:Food", Amount: "75.00"},
				{Account: "Assets:Checking", Amount: "-75.00"},
			},
		},
		{
			Date:  time.Date(2025, 1, 17, 0, 0, 0, 0, time.UTC),
			Payee: "Another Payee", // Different payee
			Postings: []core.Posting{
				{Account: "Expenses:Gas", Amount: "45.00"},
				{Account: "Assets:Checking", Amount: "-45.00"},
			},
		},
	}

	runtime := NewRuntimeIntelligence()
	runtime.BuildFromBatch(transactions)

	// Check payees
	if len(runtime.Payees) != 2 {
		t.Errorf("Expected 2 unique payees, got %d", len(runtime.Payees))
	}

	expectedPayeeFrequencies := map[string]int{
		"Brand New Payee": 2,
		"Another Payee":   1,
	}
	for payee, expected := range expectedPayeeFrequencies {
		if got := runtime.Payees[payee]; got != expected {
			t.Errorf("Expected payee %q frequency %d, got %d", payee, expected, got)
		}
	}

	// Check that templates were created
	brandNewTemplates := runtime.FindTemplates("Brand New Payee")
	if len(brandNewTemplates) != 1 {
		t.Errorf("Expected 1 template for 'Brand New Payee', got %d", len(brandNewTemplates))
	} else {
		// Template should have tracked 2 occurrences (frequency)
		if brandNewTemplates[0].Frequency != 2 {
			t.Errorf("Expected template frequency 2, got %d", brandNewTemplates[0].Frequency)
		}
	}
}

func TestRuntimeBuildFromEmptyBatch(t *testing.T) {
	runtime := NewRuntimeIntelligence()
	runtime.Payees = map[string]int{"Old Payee": 1}
	runtime.Templates["Old Payee"] = []TemplateRecord{{Frequency: 1}}

	// Build from empty batch should clear everything
	runtime.BuildFromBatch([]core.Transaction{})

	if len(runtime.Payees) != 0 {
		t.Errorf("Expected empty payees after building from empty batch, got %d", len(runtime.Payees))
	}
	if len(runtime.Templates) != 0 {
		t.Errorf("Expected empty templates after building from empty batch, got %d", len(runtime.Templates))
	}
}

func TestRuntimeBuildFromNilBatch(t *testing.T) {
	runtime := NewRuntimeIntelligence()
	runtime.Payees = map[string]int{"Old Payee": 1}

	// Build from nil batch should not panic and should clear everything
	runtime.BuildFromBatch(nil)

	if len(runtime.Payees) != 0 {
		t.Errorf("Expected empty payees after building from nil batch, got %d", len(runtime.Payees))
	}
}

func TestRuntimeFindPayees(t *testing.T) {
	transactions := []core.Transaction{
		{Payee: "City Hardware", Postings: []core.Posting{{Account: "Assets:Cash", Amount: "1"}, {Account: "Income:Misc", Amount: "-1"}}},
		{Payee: "City Market", Postings: []core.Posting{{Account: "Assets:Cash", Amount: "1"}, {Account: "Income:Misc", Amount: "-1"}}},
		{Payee: "Coffee Shop", Postings: []core.Posting{{Account: "Assets:Cash", Amount: "1"}, {Account: "Income:Misc", Amount: "-1"}}},
	}

	runtime := NewRuntimeIntelligence()
	runtime.BuildFromBatch(transactions)

	// Test prefix matching
	tests := []struct {
		prefix   string
		expected []string
	}{
		{"City", []string{"City Hardware", "City Market"}},
		{"city", []string{"City Hardware", "City Market"}}, // Case insensitive
		{"Coffee", []string{"Coffee Shop"}},
		{"Nonexistent", []string{}},
		{"", []string{"City Hardware", "City Market", "Coffee Shop"}}, // Empty prefix matches all
	}

	for _, test := range tests {
		results := runtime.FindPayees(test.prefix)
		if len(results) != len(test.expected) {
			t.Errorf("FindPayees(%q): expected %d results, got %d", test.prefix, len(test.expected), len(results))
			continue
		}
		for i, expected := range test.expected {
			if i >= len(results) || results[i] != expected {
				t.Errorf("FindPayees(%q): expected %v, got %v", test.prefix, test.expected, results)
				break
			}
		}
	}
}

func TestRuntimeFindAccounts(t *testing.T) {
	transactions := []core.Transaction{
		{Payee: "Test1", Postings: []core.Posting{
			{Account: "Assets:Checking", Amount: "100"},
			{Account: "Income:Salary", Amount: "-100"},
		}},
		{Payee: "Test2", Postings: []core.Posting{
			{Account: "Assets:Savings", Amount: "50"},
			{Account: "Income:Interest", Amount: "-50"},
		}},
	}

	runtime := NewRuntimeIntelligence()
	runtime.BuildFromBatch(transactions)

	// Test prefix matching
	tests := []struct {
		prefix   string
		expected []string
	}{
		{"Assets", []string{"Assets:Checking", "Assets:Savings"}},
		{"Income", []string{"Income:Interest", "Income:Salary"}},
		{"Assets:", []string{"Assets:Checking", "Assets:Savings"}},
		{"Nonexistent", []string{}},
	}

	for _, test := range tests {
		results := runtime.FindAccounts(test.prefix)
		if len(results) != len(test.expected) {
			t.Errorf("FindAccounts(%q): expected %d results, got %d: %v", test.prefix, len(test.expected), len(results), results)
		}
	}
}

func TestRuntimeFindTemplates(t *testing.T) {
	transactions := []core.Transaction{
		{
			Payee: "Test Payee",
			Postings: []core.Posting{
				{Account: "Expenses:Food", Amount: "50"},
				{Account: "Assets:Cash", Amount: "-50"},
			},
		},
		{
			Payee: "Test Payee",
			Postings: []core.Posting{
				{Account: "Expenses:Food", Amount: "30"},
				{Account: "Assets:Credit", Amount: "-30"},
			},
		},
	}

	runtime := NewRuntimeIntelligence()
	runtime.BuildFromBatch(transactions)

	templates := runtime.FindTemplates("Test Payee")
	if len(templates) != 2 {
		t.Errorf("Expected 2 templates for 'Test Payee', got %d", len(templates))
	}

	// Check templates are sorted by frequency (most first)
	if len(templates) > 1 && templates[0].Frequency < templates[1].Frequency {
		t.Errorf("Templates should be sorted by frequency (descending), got %v", templates)
	}
}

func TestRuntimeMutabilityScenario(t *testing.T) {
	// Simulate the user scenario: add typo, then fix it
	runtime := NewRuntimeIntelligence()

	// Step 1: User adds transaction with typo "Payyee"
	batch1 := []core.Transaction{
		{
			Payee: "Brand New Payyee", // Typo
			Postings: []core.Posting{
				{Account: "Expenses:Food", Amount: "50"},
				{Account: "Assets:Cash", Amount: "-50"},
			},
		},
	}
	runtime.BuildFromBatch(batch1)

	payees := runtime.FindPayees("Brand New")
	if !contains(payees, "Brand New Payyee") {
		t.Error("Typo payee should be in suggestions after first transaction")
	}

	// Step 2: User adds another transaction and then edits first one to fix typo
	batch2 := []core.Transaction{
		{
			Payee: "Brand New Payee", // Fixed
			Postings: []core.Posting{
				{Account: "Expenses:Food", Amount: "50"},
				{Account: "Assets:Cash", Amount: "-50"},
			},
		},
		{
			Payee: "Brand New Payyee", // Still has typo
			Postings: []core.Posting{
				{Account: "Expenses:Food", Amount: "75"},
				{Account: "Assets:Cash", Amount: "-75"},
			},
		},
	}
	runtime.BuildFromBatch(batch2)

	// Both payees should exist (one is used, one is typo)
	payees = runtime.FindPayees("Brand New")
	if len(payees) != 2 {
		t.Errorf("Expected 2 payees, got %d: %v", len(payees), payees)
	}

	// Step 3: User fixes the second transaction too
	batch3 := []core.Transaction{
		{
			Payee: "Brand New Payee",
			Postings: []core.Posting{
				{Account: "Expenses:Food", Amount: "50"},
				{Account: "Assets:Cash", Amount: "-50"},
			},
		},
		{
			Payee: "Brand New Payee", // Fixed
			Postings: []core.Posting{
				{Account: "Expenses:Food", Amount: "75"},
				{Account: "Assets:Cash", Amount: "-75"},
			},
		},
	}
	runtime.BuildFromBatch(batch3)

	// Now only the correct payee should exist
	payees = runtime.FindPayees("Brand New")
	if len(payees) != 1 {
		t.Errorf("Expected 1 payee after fix, got %d: %v", len(payees), payees)
	}
	if len(payees) > 0 && payees[0] != "Brand New Payee" {
		t.Errorf("Expected 'Brand New Payee', got '%s'", payees[0])
	}

	// Verify the template frequency was updated
	templates := runtime.FindTemplates("Brand New Payee")
	if len(templates) > 0 && templates[0].Frequency != 2 {
		t.Errorf("Expected template frequency 2, got %d", templates[0].Frequency)
	}
}

// Helper function to check if a slice contains a string
func contains(slice []string, str string) bool {
	for _, v := range slice {
		if v == str {
			return true
		}
	}
	return false
}

func TestRuntimeTemplateFrequency(t *testing.T) {
	// Test that templates track frequency correctly across multiple transactions
	transactions := []core.Transaction{
		{
			Payee: "Grocery Store",
			Postings: []core.Posting{
				{Account: "Expenses:Food", Amount: "50"},
				{Account: "Assets:Checking", Amount: "-50"},
			},
		},
		{
			Payee: "Grocery Store",
			Postings: []core.Posting{
				{Account: "Expenses:Food", Amount: "75"},
				{Account: "Assets:Checking", Amount: "-75"},
			},
		},
		{
			Payee: "Grocery Store",
			Postings: []core.Posting{
				{Account: "Expenses:Food", Amount: "60"},
				{Account: "Assets:CreditCard", Amount: "-60"},
			},
		},
	}

	runtime := NewRuntimeIntelligence()
	runtime.BuildFromBatch(transactions)

	templates := runtime.FindTemplates("Grocery Store")
	if len(templates) == 0 {
		t.Fatal("Expected at least one template")
	}

	// The most common template should be Expenses:Food -> Assets:Checking (appears twice)
	mostFrequent := templates[0]
	if mostFrequent.Frequency != 2 {
		t.Errorf("Most frequent template should have frequency 2, got %d", mostFrequent.Frequency)
	}
}
