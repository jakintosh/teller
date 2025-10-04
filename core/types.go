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
	Comment string // optional inline comment written after the amount
}

// Transaction represents a complete financial event.
type Transaction struct {
	Date     time.Time
	Payee    string // e.g., "Super Grocery Store"
	Comment  string // optional comment appended to the payee line
	Cleared  bool   // true when the transaction is cleared ("*")
	Postings []Posting
}

// String formats the transaction in ledger-cli format.
func (t *Transaction) String() string {
	var builder strings.Builder

	// Date and payee line
	dateStr := t.Date.Format("2006/01/02")
	if t.Cleared {
		builder.WriteString(fmt.Sprintf("%s * %s", dateStr, t.Payee))
	} else {
		builder.WriteString(fmt.Sprintf("%s   %s", dateStr, t.Payee))
	}
	if strings.TrimSpace(t.Comment) != "" {
		builder.WriteString(fmt.Sprintf("  ; %s", strings.TrimSpace(t.Comment)))
	}
	builder.WriteString("\n")

	// Postings with proper indentation
	for _, posting := range t.Postings {
		line := fmt.Sprintf("    %s", posting.Account)
		if posting.Amount != "" {
			amountPart := fmt.Sprintf("$%s", posting.Amount)
			totalLen := len(line) + len(amountPart)
			if totalLen < 60 {
				padding := strings.Repeat(" ", 60-len(line)-len(amountPart))
				line = line + padding + amountPart
			} else {
				line = line + "  " + amountPart
			}
		}
		if strings.TrimSpace(posting.Comment) != "" {
			line = line + fmt.Sprintf("  ; %s", strings.TrimSpace(posting.Comment))
		}
		builder.WriteString(line + "\n")
	}

	return builder.String()
}
