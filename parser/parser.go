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

// ParseFile reads a ledger-cli file and converts it into Transaction structs.
func ParseFile(filePath string) ([]core.Transaction, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var transactions []core.Transaction
	scanner := bufio.NewScanner(file)

	// Regex to match transaction start (date at beginning of line)
	dateRegex := regexp.MustCompile(`^(\d{4}[-/]\d{2}[-/]\d{2})\s+(.+)`)
	// Regex to match posting lines (indented with account and optional amount)
	postingRegex := regexp.MustCompile(`^\s+([^;]+?)(?:\s+([+-]?\$?\d+(?:\.\d{2})?))?\s*(?:;.*)?$`)

	var currentTransaction *core.Transaction

	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines and comment lines
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), ";") {
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

			// Try different date formats
			formats := []string{"2006-01-02", "2006/01/02"}
			for _, format := range formats {
				if d, err := time.Parse(format, dateStr); err == nil {
					date = d
					break
				}
			}

			// Start new transaction
			currentTransaction = &core.Transaction{
				Date:     date,
				Payee:    strings.TrimSpace(matches[2]),
				Postings: []core.Posting{},
			}
		} else if currentTransaction != nil {
			// Check if it's a posting line
			if matches := postingRegex.FindStringSubmatch(line); matches != nil {
				account := strings.TrimSpace(matches[1])
				amount := ""
				if len(matches) > 2 && matches[2] != "" {
					// Clean up amount (remove $ sign if present)
					amount = strings.TrimSpace(strings.Replace(matches[2], "$", "", -1))
				}

				posting := core.Posting{
					Account: account,
					Amount:  amount,
				}
				currentTransaction.Postings = append(currentTransaction.Postings, posting)
			}
		}
	}

	// Don't forget the last transaction
	if currentTransaction != nil {
		transactions = append(transactions, *currentTransaction)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return transactions, nil
}