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

// String formats the transaction in ledger-cli format with tab-based alignment.
func (t *Transaction) String() string {
	var builder strings.Builder

	// Date and payee line
	dateStr := t.Date.Format("2006/01/02")
	line := ""
	if t.Cleared {
		line = fmt.Sprintf("%s * %s", dateStr, t.Payee)
	} else {
		line = fmt.Sprintf("%s   %s", dateStr, t.Payee)
	}

	// Add comment if present (aligned to tab 12 = column 48)
	if strings.TrimSpace(t.Comment) != "" {
		line = addTabsToColumn(line, 44) + fmt.Sprintf("; %s", strings.TrimSpace(t.Comment))
	}
	builder.WriteString(line + "\n")

	// Format amounts with decimal alignment
	formattedAmounts := formatAmounts(t.Postings)

	// Postings with proper indentation
	for i, posting := range t.Postings {
		line = "\t" + posting.Account

		if posting.Amount != "" {
			line = addTabsToColumn(line, 44) + formattedAmounts[i]
		}

		if strings.TrimSpace(posting.Comment) != "" {
			line = addTabsToColumn(line, 56) + fmt.Sprintf("; %s", strings.TrimSpace(posting.Comment))
		}
		builder.WriteString(line + "\n")
	}

	return builder.String()
}

// addTabsToColumn adds tabs to a string to reach the target column position.
// Assumes tab width of 4.
func addTabsToColumn(s string, targetCol int) string {
	currentCol := calculateColumnPosition(s)
	if currentCol >= targetCol {
		return s + "\t" // At least one tab
	}

	// Add tabs until we reach or exceed the target column
	for currentCol < targetCol {
		s += "\t"
		currentCol = calculateColumnPosition(s)
	}
	return s
}

// calculateColumnPosition calculates the column position of a string
// accounting for tab width of 4.
func calculateColumnPosition(s string) int {
	col := 0
	for _, ch := range s {
		if ch == '\t' {
			col = ((col / 4) + 1) * 4 // Advance to next multiple of 4
		} else {
			col++
		}
	}
	return col
}

// formatAmounts formats all posting amounts with aligned decimal points.
// The format ensures '$' has at least one space after it, '-' comes immediately
// before the first digit, and all decimal points align vertically.
func formatAmounts(postings []Posting) []string {
	// Parse all amounts to find max widths
	type parsedAmount struct {
		negative bool
		intPart  string
		decPart  string
	}

	parsed := make([]parsedAmount, len(postings))
	maxIntWidth := 0

	for i, posting := range postings {
		if posting.Amount == "" {
			continue
		}

		amount := posting.Amount
		negative := false
		if strings.HasPrefix(amount, "-") {
			negative = true
			amount = amount[1:]
		}

		parts := strings.Split(amount, ".")
		intPart := parts[0]
		decPart := ""
		if len(parts) > 1 {
			decPart = parts[1]
		}

		parsed[i] = parsedAmount{
			negative: negative,
			intPart:  intPart,
			decPart:  decPart,
		}

		// Account for the "-" taking up space
		width := len(intPart)
		if negative {
			width++
		}
		if width > maxIntWidth {
			maxIntWidth = width
		}
	}

	// Format each amount
	result := make([]string, len(postings))
	for i, p := range parsed {
		if postings[i].Amount == "" {
			result[i] = ""
			continue
		}

		var amountBuilder strings.Builder
		amountBuilder.WriteString("$ ")

		// Calculate padding needed
		currentWidth := len(p.intPart)
		if p.negative {
			currentWidth++
		}
		padding := maxIntWidth - currentWidth

		// Add padding and sign
		amountBuilder.WriteString(strings.Repeat(" ", padding))
		if p.negative {
			amountBuilder.WriteString("-")
		}

		amountBuilder.WriteString(p.intPart)
		if p.decPart != "" {
			amountBuilder.WriteString(".")
			amountBuilder.WriteString(p.decPart)
		}

		result[i] = amountBuilder.String()
	}

	return result
}
