package parser

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"git.sr.ht/~jakintosh/teller/core"
)

// ParseIssue captures a non-fatal problem encountered while reading the ledger file.
type ParseIssue struct {
	Line    int
	Message string
}

// ParseResult contains the parsed transactions along with any issues that occurred.
type ParseResult struct {
	Transactions []core.Transaction
	Issues       []ParseIssue
}

// ParseFile reads a ledger-cli file and converts it into Transaction structs.
func ParseFile(filePath string) (ParseResult, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return ParseResult{}, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var (
		transactions       []core.Transaction
		issues             []ParseIssue
		scanner            = bufio.NewScanner(file)
		lineNumber         = 0
		currentTransaction *core.Transaction
	)

	// Regex to match transaction start (date at beginning of line, optional cleared marker, optional comment)
	dateRegex := regexp.MustCompile(`^(\d{4}[-/]\d{2}[-/]\d{2})\s+(?:(\*)\s+)?(.+?)(?:\s*;\s*(.*))?$`)
	// Regex to match posting lines (indented with account, optional amount, optional comment)
	postingRegex := regexp.MustCompile(`^\s+(.+?)(?:\s+([+-]?\$?\d+(?:\.\d{2})?))?(?:\s*;\s*(.*))?$`)

	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()

		trimmed := strings.TrimSpace(line)
		// Skip empty lines and comment lines
		if trimmed == "" || strings.HasPrefix(trimmed, ";") {
			continue
		}

		// Check if line starts a new transaction
		if matches := dateRegex.FindStringSubmatch(line); matches != nil {
			// Save previous transaction if exists
			if currentTransaction != nil {
				transactions = append(transactions, *currentTransaction)
			}

			// Parse date
			dateStr := matches[1]
			var date time.Time
			formats := []string{"2006-01-02", "2006/01/02"}
			for _, format := range formats {
				if d, err := time.Parse(format, dateStr); err == nil {
					date = d
					break
				}
			}
			if date.IsZero() {
				issues = append(issues, ParseIssue{
					Line:    lineNumber,
					Message: fmt.Sprintf("unrecognized date format '%s'", dateStr),
				})
			}

			payee := strings.TrimSpace(matches[3])
			comment := ""
			if len(matches) > 4 {
				comment = strings.TrimSpace(matches[4])
			}
			currentTransaction = &core.Transaction{
				Date:     date,
				Payee:    payee,
				Comment:  comment,
				Cleared:  matches[2] == "*",
				Postings: []core.Posting{},
			}
			continue
		}

		if currentTransaction == nil {
			issues = append(issues, ParseIssue{
				Line:    lineNumber,
				Message: "encountered posting before any transaction date",
			})
			continue
		}

		if matches := postingRegex.FindStringSubmatch(line); matches != nil {
			account := strings.TrimSpace(matches[1])
			if account == "" {
				issues = append(issues, ParseIssue{Line: lineNumber, Message: "posting missing account name"})
				continue
			}
			amount := ""
			if len(matches) > 2 && matches[2] != "" {
				amount = strings.TrimSpace(strings.Replace(matches[2], "$", "", -1))
			}
			comment := ""
			if len(matches) > 3 {
				comment = strings.TrimSpace(matches[3])
			}

			posting := core.Posting{
				Account: account,
				Amount:  amount,
				Comment: comment,
			}
			currentTransaction.Postings = append(currentTransaction.Postings, posting)
			continue
		}

		issues = append(issues, ParseIssue{
			Line:    lineNumber,
			Message: fmt.Sprintf("unsupported or malformed line: %q", trimmed),
		})
	}

	if currentTransaction != nil {
		transactions = append(transactions, *currentTransaction)
	}

	if err := scanner.Err(); err != nil {
		return ParseResult{}, fmt.Errorf("error reading file: %w", err)
	}

	return ParseResult{Transactions: transactions, Issues: issues}, nil
}
