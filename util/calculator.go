package util

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/Knetic/govaluate"
	"github.com/shopspring/decimal"
)

// EvaluateExpression evaluates a mathematical string like "19.99 * 2" and returns the result.
func EvaluateExpression(expr string) (string, error) {
	// Clean up the expression
	cleanExpr := strings.TrimSpace(expr)
	if cleanExpr == "" {
		return "", fmt.Errorf("empty expression")
	}

	// Clean common currency symbols and formatting
	cleanExpr = cleanCurrencyString(cleanExpr)

	// Check if it's just a number (no calculation needed)
	if isSimpleNumber(cleanExpr) {
		// Validate and format as decimal to ensure precision
		if decimal, err := decimal.NewFromString(cleanExpr); err == nil {
			return decimal.StringFixed(2), nil
		}
		return "", fmt.Errorf("invalid number format")
	}

	// Pre-validate that the expression contains valid characters for a math expression
	if !isValidMathExpression(cleanExpr) {
		return "", fmt.Errorf("invalid expression: contains non-mathematical characters")
	}

	// Try to evaluate as mathematical expression using govaluate
	expression, err := govaluate.NewEvaluableExpression(cleanExpr)
	if err != nil {
		return "", fmt.Errorf("invalid expression: %w", err)
	}

	result, err := expression.Evaluate(nil)
	if err != nil {
		return "", fmt.Errorf("evaluation error: %w", err)
	}

	// Convert result to decimal for precision
	var resultDecimal decimal.Decimal
	switch v := result.(type) {
	case float64:
		// Check for special float values
		if math.IsNaN(v) {
			return "", fmt.Errorf("result is not a number (NaN)")
		}
		if math.IsInf(v, 0) {
			return "", fmt.Errorf("result is infinite")
		}
		resultDecimal = decimal.NewFromFloat(v)
	case int:
		resultDecimal = decimal.NewFromInt(int64(v))
	case int64:
		resultDecimal = decimal.NewFromInt(v)
	default:
		// Try to convert to string and parse as decimal
		resultStr := fmt.Sprintf("%v", result)
		if resultStr == "+Inf" || resultStr == "-Inf" {
			return "", fmt.Errorf("result is infinite")
		}
		if resultStr == "NaN" {
			return "", fmt.Errorf("result is not a number")
		}
		resultDecimal, err = decimal.NewFromString(resultStr)
		if err != nil {
			return "", fmt.Errorf("could not convert result to decimal: %w", err)
		}
	}

	return resultDecimal.StringFixed(2), nil
}

// cleanCurrencyString removes common currency symbols and formatting.
func cleanCurrencyString(s string) string {
	// Remove common currency symbols and formatting
	s = strings.ReplaceAll(s, "$", "")
	s = strings.ReplaceAll(s, ",", "")
	s = strings.TrimSpace(s)
	return s
}

// isSimpleNumber checks if the string is just a number without operators.
func isSimpleNumber(s string) bool {
	// Check if it's a valid number (assuming s is already cleaned)
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

// hasOperators checks if the expression contains mathematical operators.
func hasOperators(expr string) bool {
	operatorPattern := regexp.MustCompile(`[+\-*/()]`)
	return operatorPattern.MatchString(expr)
}

// isValidMathExpression checks if the string contains only valid mathematical characters.
func isValidMathExpression(expr string) bool {
	// Allow digits, decimal points, operators, parentheses, and spaces
	validPattern := regexp.MustCompile(`^[0-9+\-*/.() ]+$`)
	return validPattern.MatchString(expr)
}


