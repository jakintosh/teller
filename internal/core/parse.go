package core

// ParseIssue captures a non-fatal problem encountered while reading a ledger file.
type ParseIssue struct {
	Line    int
	Message string
}

// ParseResult contains the parsed transactions along with any issues that occurred.
type ParseResult struct {
	Transactions []Transaction
	Issues       []ParseIssue
}
