package core

// LoadIssue describes a non-fatal problem encountered while building startup data.
type LoadIssue struct {
	Stage   string
	Message string
}

// LoadSummary aggregates metrics about the parsed ledger file and intelligence database.
type LoadSummary struct {
	Transactions    int
	UniquePayees    int
	UniqueTemplates int
	Issues          []LoadIssue
}

// HasIssues reports whether any issues were recorded during startup processing.
func (s LoadSummary) HasIssues() bool {
	return len(s.Issues) > 0
}
