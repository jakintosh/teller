package intelligence

import (
	"git.sr.ht/~jakintosh/teller/core"
	"sort"
	"strings"
)

// IntelligenceDB is an in-memory database that provides queryable information
// derived from a user's transaction history. It powers features like
// auto-completion and template suggestions.
type IntelligenceDB struct {
	// Payees is a sorted list of unique payee names encountered in the user's
	// transaction history.
	Payees []string
}

// New creates and initializes a new IntelligenceDB from a slice of transactions.
// It parses the transactions to populate the database's queryable fields.
func New(transactions []core.Transaction) *IntelligenceDB {
	// Use a map to efficiently track unique payee names. The value can be an
	// empty struct to minimize memory usage.
	payeeSet := make(map[string]struct{})
	for _, t := range transactions {
		// Ensure we don't add empty payee strings to our set.
		if t.Payee != "" {
			payeeSet[t.Payee] = struct{}{}
		}
	}

	// Convert the set of payees into a slice.
	payees := make([]string, 0, len(payeeSet))
	for payee := range payeeSet {
		payees = append(payees, payee)
	}

	// Sort the slice of payees alphabetically. This ensures a consistent and
	// predictable order for presentation and querying.
	sort.Strings(payees)

	return &IntelligenceDB{
		Payees: payees,
	}
}

// FindPayees searches for payees with a given prefix. The search is
// case-insensitive. It returns a slice of matching payees.
func (db *IntelligenceDB) FindPayees(prefix string) []string {
	// If the prefix is empty, there are no payees to find.
	if prefix == "" {
		return []string{}
	}

	// Initialize a non-nil slice to hold the matches. This prevents issues
	// with reflect.DeepEqual in tests when no matches are found.
	matches := make([]string, 0)
	lowerPrefix := strings.ToLower(prefix)

	for _, payee := range db.Payees {
		if strings.HasPrefix(strings.ToLower(payee), lowerPrefix) {
			matches = append(matches, payee)
		}
	}

	return matches
}
