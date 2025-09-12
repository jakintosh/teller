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

var (
	// transactionRegex captures the date and payee from the first line of a transaction.
	// It looks for a date (YYYY/MM/DD or YYYY-MM-DD) at the start of a line,
	// followed by the payee description.
	transactionRegex = regexp.MustCompile(`^(\d{4}[/-]\d{2}[/-]\d{2})(?:[=\d/-]*)?\s*(?:[*!])?\s*(?:\(.*\))?\s*(.+)`)

	// postingRegex captures the account and amount from a posting line.
	// It looks for an indented line, captures the account name (can include spaces and colons),
	// followed by at least two spaces, and then the optional amount.
	postingRegex = regexp.MustCompile(`^\s+([\w\s:]+?)\s{2,}(.*?)?\s*(?:;.*)?$`)

	// commentRegex checks if a line is a comment.
	commentRegex = regexp.MustCompile(`^\s*[;#%|*]`)
)

// parseDate tries to parse a date string using a few common ledger formats.
func parseDate(dateStr string) (time.Time, error) {
	layouts := []string{"2006/01/02", "2006-01-02"}
	for _, layout := range layouts {
		t, err := time.Parse(layout, dateStr)
		if err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}

// ParseFile reads a ledger file and returns a slice of transactions.
func ParseFile(filePath string) ([]core.Transaction, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var transactions []core.Transaction
	var currentTransaction *core.Transaction
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()

		if strings.TrimSpace(line) == "" || commentRegex.MatchString(line) {
			continue
		}

		if matches := transactionRegex.FindStringSubmatch(line); len(matches) > 0 {
			// If we are in a middle of a transaction, save it before starting a new one.
			if currentTransaction != nil {
				transactions = append(transactions, *currentTransaction)
			}

			date, err := parseDate(matches[1])
			if err != nil {
				// For now, we skip lines that look like transactions but have unparseable dates.
				continue
			}

			currentTransaction = &core.Transaction{
				Date:     date,
				Payee:    strings.TrimSpace(matches[2]),
				Postings: []core.Posting{},
			}
		} else if matches := postingRegex.FindStringSubmatch(line); len(matches) > 0 && currentTransaction != nil {
			account := strings.TrimSpace(matches[1])
			amount := strings.TrimSpace(matches[2])

			posting := core.Posting{
				Account: account,
				Amount:  amount,
			}
			currentTransaction.Postings = append(currentTransaction.Postings, posting)
		}
	}

	// Append the last transaction if it exists.
	if currentTransaction != nil {
		transactions = append(transactions, *currentTransaction)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return transactions, nil
}
