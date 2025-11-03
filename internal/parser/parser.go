package parser

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
	"unicode"

	"git.sr.ht/~jakintosh/teller/internal/core"
)

// ParseFile reads a ledger-cli file and converts it into Transaction structs.
func ParseFile(filePath string) (core.ParseResult, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return core.ParseResult{}, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var (
		transactions       []core.Transaction
		issues             []core.ParseIssue
		scanner            = bufio.NewScanner(file)
		lineNumber         = 0
		currentTransaction *core.Transaction
	)

	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()

		trimmed := strings.TrimSpace(line)
		// Skip empty lines and comment lines
		if trimmed == "" || strings.HasPrefix(trimmed, ";") {
			continue
		}

		// Check if line starts a new transaction (starts with a digit)
		if len(line) > 0 && unicode.IsDigit(rune(line[0])) {
			// Save previous transaction if exists
			if currentTransaction != nil {
				transactions = append(transactions, *currentTransaction)
			}

			// Parse transaction line
			tx, err := parseTransactionLine(line)
			if err != nil {
				issues = append(issues, core.ParseIssue{
					Line:    lineNumber,
					Message: err.Error(),
				})
				currentTransaction = nil
				continue
			}

			currentTransaction = tx
			continue
		}

		// Otherwise, it should be a posting line
		if currentTransaction == nil {
			issues = append(issues, core.ParseIssue{
				Line:    lineNumber,
				Message: "encountered posting before any transaction date",
			})
			continue
		}

		posting, err := parsePostingLine(line)
		if err != nil {
			issues = append(issues, core.ParseIssue{
				Line:    lineNumber,
				Message: err.Error(),
			})
			continue
		}

		currentTransaction.Postings = append(currentTransaction.Postings, *posting)
	}

	if currentTransaction != nil {
		transactions = append(transactions, *currentTransaction)
	}

	if err := scanner.Err(); err != nil {
		return core.ParseResult{}, fmt.Errorf("error reading file: %w", err)
	}

	return core.ParseResult{Transactions: transactions, Issues: issues}, nil
}

// parseTransactionLine parses a transaction header line.
// Expected format: DATE [*] PAYEE [; COMMENT]
func parseTransactionLine(line string) (*core.Transaction, error) {
	s := line

	// Parse date
	date, rest, err := parseDate(s)
	if err != nil {
		return nil, err
	}
	s = rest

	// Parse cleared marker
	cleared, rest := parseCleared(s)
	s = rest

	// Parse payee and comment
	payee, comment := parsePayeeAndComment(s)

	return &core.Transaction{
		Date:     date,
		Payee:    payee,
		Comment:  comment,
		Cleared:  cleared,
		Postings: []core.Posting{},
	}, nil
}

// parseDate extracts a date from the beginning of a string.
// Returns the parsed date and the remaining string.
func parseDate(s string) (time.Time, string, error) {
	s = strings.TrimSpace(s)

	// Find the date portion (YYYY-MM-DD or YYYY/MM/DD)
	if len(s) < 10 {
		return time.Time{}, s, fmt.Errorf("line too short to contain a date")
	}

	dateStr := s[:10]
	rest := s[10:]

	// Try both date formats
	formats := []string{"2006-01-02", "2006/01/02"}
	for _, format := range formats {
		if date, err := time.Parse(format, dateStr); err == nil {
			return date, rest, nil
		}
	}

	return time.Time{}, s, fmt.Errorf("unrecognized date format '%s'", dateStr)
}

// parseCleared checks if the string starts with a cleared marker (*).
// Returns whether it's cleared and the remaining string.
func parseCleared(s string) (bool, string) {
	s = strings.TrimSpace(s)

	if strings.HasPrefix(s, "*") {
		return true, strings.TrimSpace(s[1:])
	}

	return false, s
}

// parsePayeeAndComment extracts the payee and optional comment from a string.
// Expected format: PAYEE [; COMMENT]
func parsePayeeAndComment(s string) (payee, comment string) {
	s = strings.TrimSpace(s)

	// Look for comment separator
	if idx := strings.Index(s, ";"); idx != -1 {
		payee = strings.TrimSpace(s[:idx])
		comment = strings.TrimSpace(s[idx+1:])
		return
	}

	payee = strings.TrimSpace(s)
	return
}

// parsePostingLine parses a posting line.
// Expected format: WHITESPACE ACCOUNT [AMOUNT] [; COMMENT]
func parsePostingLine(line string) (*core.Posting, error) {
	// Posting lines must start with whitespace
	if len(line) == 0 || !unicode.IsSpace(rune(line[0])) {
		return nil, fmt.Errorf("posting line must start with whitespace")
	}

	s := strings.TrimSpace(line)

	// Extract comment if present
	text, comment := extractComment(s)

	// Parse account and amount
	account, amount := parseAccountAndAmount(text)

	if account == "" {
		return nil, fmt.Errorf("posting missing account name")
	}

	return &core.Posting{
		Account: account,
		Amount:  amount,
		Comment: comment,
	}, nil
}

// extractComment separates a line into text and comment parts.
// Returns the text before the comment and the comment itself.
func extractComment(s string) (text, comment string) {
	if idx := strings.Index(s, ";"); idx != -1 {
		text = strings.TrimSpace(s[:idx])
		comment = strings.TrimSpace(s[idx+1:])
		return
	}

	text = s
	return
}

// parseAccountAndAmount extracts account and amount from a posting line text.
// Splits on the first occurrence of either: (1) two or more spaces, or (2) one or more tabs.
// This allows account names to contain single spaces.
func parseAccountAndAmount(s string) (string, string) {
	s = strings.TrimSpace(s)

	// Find the first occurrence of either 2+ spaces or 1+ tabs
	splitIdx := -1
	for i := 0; i < len(s); i++ {
		if s[i] == '\t' {
			// Found a tab - this is the split point
			splitIdx = i
			break
		}
		if s[i] == ' ' && i+1 < len(s) && s[i+1] == ' ' {
			// Found two consecutive spaces - this is the split point
			splitIdx = i
			break
		}
	}

	if splitIdx == -1 {
		// No separator found, entire string is the account
		return s, ""
	}

	potentialAccount := strings.TrimSpace(s[:splitIdx])
	potentialAmount := strings.TrimSpace(s[splitIdx:])

	// Check if what we found after the separator looks like an amount
	if isAmount(potentialAmount) {
		return potentialAccount, cleanAmount(potentialAmount)
	}

	// Otherwise, entire string is the account
	return s, ""
}

// isAmount checks if a string looks like a monetary amount.
// Valid formats:
//   - $ 123.45, $123.45 (dollar with optional space before digits)
//   - -$ 123.45, -$123.45 (negative before dollar)
//   - $ -123.45, $-123.45 (dollar before negative)
//   - 123.45, -123.45, +123.45 (plain numbers)
//
// Invalid formats:
//   - $- 123.45 (space between sign and digits)
//   - $12 3.45 (space within digits)
func isAmount(s string) bool {
	if s == "" {
		return false
	}

	s = strings.TrimSpace(s)
	i := 0

	// Track if we've seen a sign or dollar
	hasLeadingSign := false
	hasDollar := false

	// Check for optional leading sign
	if i < len(s) && (s[i] == '-' || s[i] == '+') {
		hasLeadingSign = true
		i++
	}

	// Check for optional dollar sign
	if i < len(s) && s[i] == '$' {
		hasDollar = true
		i++
	}

	// Skip optional whitespace after dollar sign
	if hasDollar {
		for i < len(s) && s[i] == ' ' {
			i++
		}
	}

	// Check for optional sign (if not already present)
	if !hasLeadingSign && i < len(s) && (s[i] == '-' || s[i] == '+') {
		i++
	}

	// Now we must have digits, and no more whitespace is allowed
	hasDigit := false
	for i < len(s) {
		r := rune(s[i])
		if unicode.IsDigit(r) || r == '.' {
			hasDigit = true
			i++
		} else {
			// Invalid character (including whitespace within digits)
			return false
		}
	}

	return hasDigit
}

// cleanAmount removes currency symbols from an amount string.
func cleanAmount(s string) string {
	clean := strings.ReplaceAll(s, "$", "")
	return strings.ReplaceAll(clean, " ", "")
}
