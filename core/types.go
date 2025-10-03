package core

import (
	"fmt"
	"strings"
	"time"
)

// Posting represents a single entry in a transaction.
type Posting struct {
	Account string // e.g., "Expenses:Food:Groceries"
	Amount  string // e.g., "12.34" (stored as string for precision)
}

// Transaction represents a complete financial event.
type Transaction struct {
	Date     time.Time
	Payee    string // e.g., "Super Grocery Store"
	Postings []Posting
}

// String formats the transaction in ledger-cli format.
func (t *Transaction) String() string {
	var builder strings.Builder

	// Date and payee line
	builder.WriteString(fmt.Sprintf("%s %s\n", t.Date.Format("2006/01/02"), t.Payee))

	// Postings with proper indentation
	for _, posting := range t.Postings {
		if posting.Amount == "" {
			// Empty amount posting (balancing entry)
			builder.WriteString(fmt.Sprintf("    %s\n", posting.Account))
		} else {
			// Posting with amount - right-align amount at column 60
			accountPart := fmt.Sprintf("    %s", posting.Account)
			amountPart := fmt.Sprintf("$%s", posting.Amount)

			// Calculate padding to align amount at column 60
			totalLen := len(accountPart) + len(amountPart)
			if totalLen < 60 {
				padding := strings.Repeat(" ", 60-len(accountPart)-len(amountPart))
				builder.WriteString(fmt.Sprintf("%s%s%s\n", accountPart, padding, amountPart))
			} else {
				// If too long, just use two spaces
				builder.WriteString(fmt.Sprintf("%s  %s\n", accountPart, amountPart))
			}
		}
	}

	return builder.String()
}