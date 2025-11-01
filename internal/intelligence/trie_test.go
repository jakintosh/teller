package intelligence

import (
	"reflect"
	"sort"
	"testing"
)

func TestTrieInsertAndFind(t *testing.T) {
	trie := NewTrie()

	// Insert test account names
	accounts := []string{
		"Expenses:Food:Groceries",
		"Expenses:Food:Dining",
		"Expenses:Food:Alcohol",
		"Expenses:Auto:Gas",
		"Expenses:Auto:Maintenance",
		"Assets:Checking",
		"Assets:Credit Card",
		"Income:Salary",
	}

	for _, account := range accounts {
		trie.Insert(account)
	}

	// Test various prefix searches
	tests := []struct {
		prefix   string
		expected []string
	}{
		{
			prefix:   "Expenses",
			expected: []string{"Expenses:Auto:Gas", "Expenses:Auto:Maintenance", "Expenses:Food:Alcohol", "Expenses:Food:Dining", "Expenses:Food:Groceries"},
		},
		{
			prefix:   "Expenses:Food",
			expected: []string{"Expenses:Food:Alcohol", "Expenses:Food:Dining", "Expenses:Food:Groceries"},
		},
		{
			prefix:   "Expenses:Food:",
			expected: []string{"Expenses:Food:Alcohol", "Expenses:Food:Dining", "Expenses:Food:Groceries"},
		},
		{
			prefix:   "Expenses:Food:G",
			expected: []string{"Expenses:Food:Groceries"},
		},
		{
			prefix:   "Assets",
			expected: []string{"Assets:Checking", "Assets:Credit Card"},
		},
		{
			prefix:   "Assets:",
			expected: []string{"Assets:Checking", "Assets:Credit Card"},
		},
		{
			prefix:   "Assets:C",
			expected: []string{"Assets:Checking", "Assets:Credit Card"},
		},
		{
			prefix:   "Assets:Credit",
			expected: []string{"Assets:Credit Card"},
		},
		{
			prefix:   "Income",
			expected: []string{"Income:Salary"},
		},
		{
			prefix:   "Nonexistent",
			expected: []string{},
		},
		{
			prefix:   "",
			expected: accounts, // All accounts
		},
	}

	for _, test := range tests {
		result := trie.Find(test.prefix)

		// Sort both slices for comparison
		sort.Strings(result)
		sort.Strings(test.expected)

		if !reflect.DeepEqual(result, test.expected) {
			t.Errorf("For prefix '%s':\nExpected: %v\nGot:      %v", test.prefix, test.expected, result)
		}
	}
}

func TestTrieHierarchicalCompletion(t *testing.T) {
	trie := NewTrie()

	// Insert hierarchical account names
	accounts := []string{
		"Expenses",
		"Expenses:Food",
		"Expenses:Food:Groceries",
		"Expenses:Food:Dining",
		"Expenses:Auto",
		"Expenses:Auto:Gas",
		"Expenses:Auto:Fuel", // Different from Gas to test completion
	}

	for _, account := range accounts {
		trie.Insert(account)
	}

	// Test segment-by-segment completion
	// Note: Trie returns ALL matches starting with prefix, which is correct behavior
	tests := []struct {
		prefix   string
		expected []string
	}{
		{
			prefix:   "Exp",
			expected: []string{"Expenses", "Expenses:Auto", "Expenses:Auto:Fuel", "Expenses:Auto:Gas", "Expenses:Food", "Expenses:Food:Dining", "Expenses:Food:Groceries"},
		},
		{
			prefix:   "Expenses:",
			expected: []string{"Expenses:Auto", "Expenses:Auto:Fuel", "Expenses:Auto:Gas", "Expenses:Food", "Expenses:Food:Dining", "Expenses:Food:Groceries"},
		},
		{
			prefix:   "Expenses:Fo",
			expected: []string{"Expenses:Food", "Expenses:Food:Dining", "Expenses:Food:Groceries"},
		},
		{
			prefix:   "Expenses:Food:",
			expected: []string{"Expenses:Food:Dining", "Expenses:Food:Groceries"},
		},
		{
			prefix:   "Expenses:Auto:F",
			expected: []string{"Expenses:Auto:Fuel"},
		},
		{
			prefix:   "Expenses:Auto:G",
			expected: []string{"Expenses:Auto:Gas"},
		},
	}

	for _, test := range tests {
		result := trie.Find(test.prefix)

		// Sort both slices for comparison
		sort.Strings(result)
		sort.Strings(test.expected)

		if !reflect.DeepEqual(result, test.expected) {
			t.Errorf("For prefix '%s':\nExpected: %v\nGot:      %v", test.prefix, test.expected, result)
		}
	}
}
