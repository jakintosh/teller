package core

import "time"

// Posting represents a single entry in a transaction, either a credit or a
// debit. It associates an account with a specific amount. The Amount is
// stored as a string to preserve precision and will be parsed by a decimal
// library for calculations.
type Posting struct {
	Account string
	Amount  string
}

// Transaction represents a complete financial event. It consists of a date,
// a payee (a description of the transaction), and a series of balanced
// postings that must sum to zero.
type Transaction struct {
	Date     time.Time
	Payee    string
	Postings []Posting
}
