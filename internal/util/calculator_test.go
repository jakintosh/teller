package util

import "testing"

func TestEvaluateExpression(t *testing.T) {
	tests := []struct {
		expression string
		expected   string
		shouldErr  bool
	}{
		// Simple numbers
		{"42", "42.00", false},
		{"42.50", "42.50", false},
		{"$42.50", "42.50", false},
		{"1,234.56", "1234.56", false},

		// Basic arithmetic
		{"10 + 5", "15.00", false},
		{"20 - 8", "12.00", false},
		{"6 * 7", "42.00", false},
		{"100 / 4", "25.00", false},

		// Order of operations
		{"10 + 5 * 2", "20.00", false},
		{"(10 + 5) * 2", "30.00", false},
		{"100 / (5 + 5)", "10.00", false},

		// Decimal arithmetic
		{"19.99 * 2", "39.98", false},
		{"123.45 + 67.89", "191.34", false},
		{"50.00 - 12.34", "37.66", false},

		// Complex expressions
		{"(15.50 * 2) + 7.99", "38.99", false},
		{"100 - (25 + 15) * 1.5", "40.00", false},

		// Error cases
		{"", "", true},
		{"10 +", "", true},
		{"invalid", "", true},
		{"10 / 0", "", true},
	}

	for _, test := range tests {
		result, err := EvaluateExpression(test.expression)

		if test.shouldErr {
			if err == nil {
				t.Errorf("Expected error for expression '%s', but got result: %s", test.expression, result)
			}
		} else {
			if err != nil {
				t.Errorf("Unexpected error for expression '%s': %v", test.expression, err)
				continue
			}
			if result != test.expected {
				t.Errorf("For expression '%s': expected '%s', got '%s'", test.expression, test.expected, result)
			}
		}
	}
}

func TestIsSimpleNumber(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"42", true},
		{"42.50", true},
		{"$42.50", false},   // isSimpleNumber expects cleaned input
		{"1,234.56", false}, // isSimpleNumber expects cleaned input
		{" 123.45 ", false}, // isSimpleNumber expects cleaned input
		{"123.45", true},    // cleaned version should work
		{"10 + 5", false},
		{"(42)", false},
		{"invalid", false},
		{"", false},
	}

	for _, test := range tests {
		result := isSimpleNumber(test.input)
		if result != test.expected {
			t.Errorf("For input '%s': expected %t, got %t", test.input, test.expected, result)
		}
	}
}
