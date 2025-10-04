package intelligence

import (
	"testing"
	"time"

	"git.sr.ht/~jakintosh/teller/core"
)

func TestNewIntelligenceDB(t *testing.T) {
	// Create mock transactions
	transactions := []core.Transaction{
		{
			Date:  time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
			Payee: "Super Grocery Store",
			Postings: []core.Posting{
				{Account: "Expenses:Food:Groceries", Amount: "85.42"},
				{Account: "Assets:Checking", Amount: "-85.42"},
			},
		},
		{
			Date:  time.Date(2025, 1, 16, 0, 0, 0, 0, time.UTC),
			Payee: "Gas Station",
			Postings: []core.Posting{
				{Account: "Expenses:Auto:Gas", Amount: "45.00"},
				{Account: "Assets:Credit Card", Amount: "-45.00"},
			},
		},
		{
			Date:  time.Date(2025, 1, 17, 0, 0, 0, 0, time.UTC),
			Payee: "Super Grocery Store", // Duplicate payee
			Postings: []core.Posting{
				{Account: "Expenses:Food:Groceries", Amount: "125.67"},
				{Account: "Assets:Checking", Amount: "-125.67"},
			},
		},
		{
			Date:  time.Date(2025, 1, 18, 0, 0, 0, 0, time.UTC),
			Payee: "Coffee Shop",
			Postings: []core.Posting{
				{Account: "Expenses:Food:Dining", Amount: "8.50"},
				{Account: "Assets:Checking", Amount: "-8.50"},
			},
		},
	}

	expectedPayeeCount := 3 // Super Grocery Store, Gas Station, Coffee Shop
	db, report, err := NewIntelligenceDB(transactions)
	if err != nil {
		t.Fatalf("Failed to create IntelligenceDB: %v", err)
	}
	if len(report.Issues) != 0 {
		t.Fatalf("expected no build issues, got %d: %v", len(report.Issues), report.Issues)
	}
	if report.UniquePayees != expectedPayeeCount {
		t.Fatalf("expected UniquePayees to be %d, got %d", expectedPayeeCount, report.UniquePayees)
	}

	// Check that we have the correct number of unique payees
	if len(db.Payees) != expectedPayeeCount {
		t.Errorf("Expected %d unique payees, got %d", expectedPayeeCount, len(db.Payees))
	}

	// Check that accounts Trie is populated
	if db.Accounts == nil {
		t.Error("Expected Accounts Trie to be initialized")
	}

	// Check that payees are sorted
	expectedPayees := []string{"Coffee Shop", "Gas Station", "Super Grocery Store"}
	for i, expected := range expectedPayees {
		if i >= len(db.Payees) {
			t.Errorf("Missing payee at index %d: expected '%s'", i, expected)
			continue
		}
		if db.Payees[i] != expected {
			t.Errorf("Expected payee at index %d to be '%s', got '%s'", i, expected, db.Payees[i])
		}
	}
}

func TestFindPayees(t *testing.T) {
	// Create mock transactions
	transactions := []core.Transaction{
		{Payee: "City Hardware", Postings: []core.Posting{{Account: "Assets:Cash", Amount: "1"}, {Account: "Income:Misc", Amount: "-1"}}},
		{Payee: "City Market", Postings: []core.Posting{{Account: "Assets:Cash", Amount: "1"}, {Account: "Income:Misc", Amount: "-1"}}},
		{Payee: "Coffee Shop", Postings: []core.Posting{{Account: "Assets:Cash", Amount: "1"}, {Account: "Income:Misc", Amount: "-1"}}},
		{Payee: "Gas Station", Postings: []core.Posting{{Account: "Assets:Cash", Amount: "1"}, {Account: "Income:Misc", Amount: "-1"}}},
	}

	db, report, err := NewIntelligenceDB(transactions)
	if err != nil {
		t.Fatalf("Failed to create IntelligenceDB: %v", err)
	}
	if len(report.Issues) != 0 {
		t.Fatalf("expected no build issues, got %d: %v", len(report.Issues), report.Issues)
	}

	// Test prefix matching
	tests := []struct {
		prefix   string
		expected []string
	}{
		{
			prefix:   "Cit",
			expected: []string{"City Hardware", "City Market"},
		},
		{
			prefix:   "City H",
			expected: []string{"City Hardware"},
		},
		{
			prefix:   "Coffee",
			expected: []string{"Coffee Shop"},
		},
		{
			prefix:   "Gas",
			expected: []string{"Gas Station"},
		},
		{
			prefix:   "Nonexistent",
			expected: []string{},
		},
		{
			prefix:   "",
			expected: []string{"City Hardware", "City Market", "Coffee Shop", "Gas Station"},
		},
	}

	for _, test := range tests {
		result := db.FindPayees(test.prefix)
		if len(result) != len(test.expected) {
			t.Errorf("For prefix '%s': expected %d results, got %d", test.prefix, len(test.expected), len(result))
			continue
		}
		for i, expected := range test.expected {
			if i >= len(result) || result[i] != expected {
				t.Errorf("For prefix '%s': expected result[%d] = '%s', got '%s'", test.prefix, i, expected, result[i])
			}
		}
	}
}

func TestFindAccounts(t *testing.T) {
	// Create mock transactions with varied account structures
	transactions := []core.Transaction{
		{
			Payee: "Super Grocery Store",
			Postings: []core.Posting{
				{Account: "Expenses:Food:Groceries", Amount: "85.42"},
				{Account: "Assets:Checking", Amount: "-85.42"},
			},
		},
		{
			Payee: "Gas Station",
			Postings: []core.Posting{
				{Account: "Expenses:Auto:Gas", Amount: "45.00"},
				{Account: "Assets:Credit Card", Amount: "-45.00"},
			},
		},
		{
			Payee: "Coffee Shop",
			Postings: []core.Posting{
				{Account: "Expenses:Food:Dining", Amount: "8.50"},
				{Account: "Assets:Checking", Amount: "-8.50"},
			},
		},
	}

	db, report, err := NewIntelligenceDB(transactions)
	if err != nil {
		t.Fatalf("Failed to create IntelligenceDB: %v", err)
	}
	if len(report.Issues) != 0 {
		t.Fatalf("expected no build issues, got %d: %v", len(report.Issues), report.Issues)
	}

	// Test account prefix searches
	tests := []struct {
		prefix   string
		expected []string
	}{
		{
			prefix:   "Expenses:Fo",
			expected: []string{"Expenses:Food:Dining", "Expenses:Food:Groceries"},
		},
		{
			prefix:   "Assets:",
			expected: []string{"Assets:Checking", "Assets:Credit Card"},
		},
		{
			prefix:   "Expenses:Auto:",
			expected: []string{"Expenses:Auto:Gas"},
		},
		{
			prefix:   "Nonexistent",
			expected: []string{},
		},
	}

	for _, test := range tests {
		result := db.FindAccounts(test.prefix)
		if len(result) != len(test.expected) {
			t.Errorf("For account prefix '%s': expected %d results, got %d", test.prefix, len(test.expected), len(result))
			continue
		}
		for _, expected := range test.expected {
			found := false
			for _, actual := range result {
				if actual == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("For account prefix '%s': expected to find '%s' in results %v", test.prefix, expected, result)
			}
		}
	}
}

func TestFindTemplates(t *testing.T) {
	// Create mock transactions where "City Market" has two transactions with the same template
	// and one with a different template
	transactions := []core.Transaction{
		{
			Payee: "City Market",
			Postings: []core.Posting{
				{Account: "Assets:Checking", Amount: "-85.42"},
				{Account: "Expenses:Groceries", Amount: "85.42"},
			},
		},
		{
			Payee: "City Market",
			Postings: []core.Posting{
				{Account: "Assets:Checking", Amount: "-125.67"},
				{Account: "Expenses:Groceries", Amount: "125.67"},
			},
		},
		{
			Payee: "City Market",
			Postings: []core.Posting{
				{Account: "Assets:Credit Card", Amount: "-45.00"},
				{Account: "Expenses:Alcohol", Amount: "25.00"},
				{Account: "Expenses:Groceries", Amount: "20.00"},
			},
		},
		{
			Payee: "Gas Station",
			Postings: []core.Posting{
				{Account: "Assets:Credit Card", Amount: "-40.00"},
				{Account: "Expenses:Auto:Gas", Amount: "40.00"},
			},
		},
	}

	db, report, err := NewIntelligenceDB(transactions)
	if err != nil {
		t.Fatalf("Failed to create IntelligenceDB: %v", err)
	}
	if len(report.Issues) != 0 {
		t.Fatalf("expected no build issues, got %d: %v", len(report.Issues), report.Issues)
	}

	// Test City Market templates
	cityMarketTemplates := db.FindTemplates("City Market")
	if len(cityMarketTemplates) != 2 {
		t.Errorf("Expected 2 templates for City Market, got %d", len(cityMarketTemplates))
	}

	if len(cityMarketTemplates) >= 1 {
		firstTemplate := cityMarketTemplates[0]
		if firstTemplate.Frequency != 2 {
			t.Errorf("Expected first template frequency to be 2, got %d", firstTemplate.Frequency)
		}
		expectedDebit := []string{"Expenses:Groceries"}
		expectedCredit := []string{"Assets:Checking"}
		if !equalSlices(firstTemplate.DebitAccounts, expectedDebit) {
			t.Errorf("Expected debit accounts %v, got %v", expectedDebit, firstTemplate.DebitAccounts)
		}
		if !equalSlices(firstTemplate.CreditAccounts, expectedCredit) {
			t.Errorf("Expected credit accounts %v, got %v", expectedCredit, firstTemplate.CreditAccounts)
		}
	}

	if len(cityMarketTemplates) >= 2 {
		secondTemplate := cityMarketTemplates[1]
		if secondTemplate.Frequency != 1 {
			t.Errorf("Expected second template frequency to be 1, got %d", secondTemplate.Frequency)
		}
		expectedDebit := []string{"Expenses:Alcohol", "Expenses:Groceries"}
		expectedCredit := []string{"Assets:Credit Card"}
		if !equalSlices(secondTemplate.DebitAccounts, expectedDebit) {
			t.Errorf("Expected second debit accounts %v, got %v", expectedDebit, secondTemplate.DebitAccounts)
		}
		if !equalSlices(secondTemplate.CreditAccounts, expectedCredit) {
			t.Errorf("Expected second credit accounts %v, got %v", expectedCredit, secondTemplate.CreditAccounts)
		}
	}

	// Test Gas Station templates
	gasStationTemplates := db.FindTemplates("Gas Station")
	if len(gasStationTemplates) != 1 {
		t.Errorf("Expected 1 template for Gas Station, got %d", len(gasStationTemplates))
	}

	// Test non-existent payee
	nonExistentTemplates := db.FindTemplates("Nonexistent Payee")
	if len(nonExistentTemplates) != 0 {
		t.Errorf("Expected 0 templates for non-existent payee, got %d", len(nonExistentTemplates))
	}
}

func TestTemplatesIncludeElidedAmounts(t *testing.T) {
	transactions := []core.Transaction{
		{
			Payee: "City Market",
			Postings: []core.Posting{
				{Account: "Expenses:Food:Groceries", Amount: "125.67"},
				{Account: "Expenses:Food:Alcohol", Amount: "19.99"},
				{Account: "Assets:Checking", Amount: ""},
			},
		},
	}

	db, report, err := NewIntelligenceDB(transactions)
	if err != nil {
		t.Fatalf("Failed to create IntelligenceDB: %v", err)
	}
	if len(report.Issues) != 0 {
		t.Fatalf("expected no build issues, got %d: %v", len(report.Issues), report.Issues)
	}

	templates := db.FindTemplates("City Market")
	if len(templates) != 1 {
		t.Fatalf("Expected 1 template, got %d", len(templates))
	}

	tpl := templates[0]
	expectedDebit := []string{"Expenses:Food:Alcohol", "Expenses:Food:Groceries"}
	if !equalSlices(tpl.DebitAccounts, expectedDebit) {
		t.Fatalf("expected debit accounts %v, got %v", expectedDebit, tpl.DebitAccounts)
	}

	expectedCredit := []string{"Assets:Checking"}
	if !equalSlices(tpl.CreditAccounts, expectedCredit) {
		t.Fatalf("expected credit accounts %v, got %v", expectedCredit, tpl.CreditAccounts)
	}
}

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
